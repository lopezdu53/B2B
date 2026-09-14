package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gosom/google-maps-scraper/admin"
	"github.com/gosom/google-maps-scraper/gmaps"
)

// RecordSearchJob remembers which tenant launched a scrape and which rubro /
// specialty it should file under once results land.
func (s *store) RecordSearchJob(ctx context.Context, jobID, tenantID int64, rubro, specialty, ratingBand string) error {
	if jobID == 0 || tenantID == 0 {
		return fmt.Errorf("missing job or tenant")
	}

	const q = `
INSERT INTO b2b_search_jobs (job_id, tenant_id, rubro, specialty, rating_band)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (job_id) DO NOTHING`

	_, err := s.db.Exec(ctx, q, jobID, tenantID, rubro, specialty, ratingBand)

	return err
}

// DismissSearchJob hides a scrape from the B2B recent-searches list.
func (s *store) DismissSearchJob(ctx context.Context, jobID int64) error {
	if jobID == 0 {
		return fmt.Errorf("missing job")
	}

	const q = `
INSERT INTO b2b_dismissed_jobs (job_id, dismissed_at)
VALUES ($1, NOW())
ON CONFLICT (job_id) DO NOTHING`

	_, err := s.db.Exec(ctx, q, jobID)

	return err
}

// ListDismissedJobIDs returns River job ids hidden from the recent list.
func (s *store) ListDismissedJobIDs(ctx context.Context) ([]int64, error) {
	rows, err := s.db.Query(ctx, `SELECT job_id FROM b2b_dismissed_jobs`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64

	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}

		ids = append(ids, id)
	}

	return ids, rows.Err()
}

// IngestPendingSearchJobs files completed scrapes that still need a CRM overlay.
func (s *store) IngestPendingSearchJobs(ctx context.Context) error {
	const q = `
SELECT j.job_id
FROM b2b_search_jobs j
INNER JOIN scrape_results sr ON sr.job_id = j.job_id
WHERE NOT j.ingested OR j.created_at > NOW() - INTERVAL '14 days'
ORDER BY j.ingested ASC, j.created_at DESC
LIMIT 200`

	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return err
		}

		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	for _, id := range ids {
		if err := s.IngestSearchJob(ctx, id); err != nil {
			return err
		}
	}

	return nil
}

// restoreTrashOnIngest reports whether ingest should pull matching CRM rows
// out of the papelera. Only the first ingest of a job (a new search) does
// that, so "Eliminar todos" + buscar de nuevo vuelve a mostrar resultados.
// Reloading Negocios re-files jobs from the last 14 days and must not undo
// an explicit "enviar a papelera".
func restoreTrashOnIngest(alreadyIngested bool) bool {
	return !alreadyIngested
}

// IngestSearchJob upserts qualifying leads from a finished scrape into the
// tenant CRM, tagging category and specialty. Existing status / category /
// specialty are left alone. The first ingest of a job un-hides matching keys
// so a brand-new search can refill the map; later re-ingests keep hidden.
func (s *store) IngestSearchJob(ctx context.Context, jobID int64) error {
	if jobID == 0 {
		return nil
	}

	var (
		tenantID   int64
		rubro      string
		specialty  string
		ingested   bool
		ratingBand string
	)

	err := s.db.QueryRow(ctx,
		`SELECT tenant_id, rubro, specialty, ingested, rating_band FROM b2b_search_jobs WHERE job_id = $1`,
		jobID,
	).Scan(&tenantID, &rubro, &specialty, &ingested, &ratingBand)
	if err != nil {
		return nil // no tagged job — nothing to file
	}

	var raw []byte
	err = s.db.QueryRow(ctx, `SELECT results FROM scrape_results WHERE job_id = $1`, jobID).Scan(&raw)
	if err != nil || len(raw) == 0 {
		return nil
	}

	var entries []gmaps.Entry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return fmt.Errorf("decode scrape results: %w", err)
	}

	ratingMin, ratingMaxExcl, _, _ := admin.RatingBandBounds(ratingBand)

	keys := make([]string, 0, len(entries))
	titles := make([]string, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))

	for i := range entries {
		e := entries[i]
		if !admin.QualifiesAsLead(e.Title, e.ReviewCount) {
			continue
		}

		if !gmaps.MatchesRating(e.ReviewRating, ratingMin, ratingMaxExcl) {
			continue
		}

		key := e.PlaceID
		if key == "" {
			key = e.Cid
		}

		if key == "" {
			continue
		}

		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}
		keys = append(keys, key)
		titles = append(titles, e.Title)
	}

	canon := admin.ParentCategoryName(rubro, specialty)

	var categoryID *int64

	if canon != "" && tenantID != 0 {
		var id int64
		if err := s.db.QueryRow(ctx,
			`SELECT id FROM b2b_categories WHERE tenant_id = $1 AND name = $2`,
			tenantID, canon,
		).Scan(&id); err == nil {
			categoryID = &id
		}
	}

	if len(keys) > 0 {
		const upsert = `
INSERT INTO b2b_business_crm (place_id, tenant_id, status, category_id, specialty, title)
SELECT k, $2, $3, $4, $5, t
FROM unnest($1::text[], $6::text[]) AS x(k, t)
ON CONFLICT (place_id, tenant_id) DO UPDATE SET
    category_id = COALESCE(b2b_business_crm.category_id, EXCLUDED.category_id),
    specialty   = CASE WHEN COALESCE(b2b_business_crm.specialty, '') = '' THEN EXCLUDED.specialty ELSE b2b_business_crm.specialty END,
    title       = COALESCE(NULLIF(EXCLUDED.title, ''), b2b_business_crm.title),
    hidden      = CASE WHEN $7 THEN FALSE ELSE b2b_business_crm.hidden END,
    updated_at  = NOW()`

		if _, err := s.db.Exec(ctx, upsert, keys, tenantID, admin.StatusProspect, categoryID, specialty, titles, restoreTrashOnIngest(ingested)); err != nil {
			return err
		}
	}

	_, err = s.db.Exec(ctx, `UPDATE b2b_search_jobs SET ingested = TRUE WHERE job_id = $1`, jobID)

	return err
}

// HideAllVisibleBusinesses sends every lead currently shown on the map to the
// trash. Most businesses have no CRM row yet, so we upsert hidden=true for
// every qualifying scrape key — not only existing "client" rows.
func (s *store) HideAllVisibleBusinesses(ctx context.Context, tenantID int64) (int64, error) {
	if tenantID == 0 {
		return 0, fmt.Errorf("missing tenant")
	}

	q := `
INSERT INTO b2b_business_crm (place_id, tenant_id, hidden, title, updated_at)
SELECT bkey, $1, TRUE, title, NOW()
FROM (
    SELECT DISTINCT ON (bkey) bkey, title
    FROM (
        SELECT
            COALESCE(NULLIF(elem->>'place_id', ''), NULLIF(elem->>'cid', '')) AS bkey,
            COALESCE(elem->>'title', '') AS title
        FROM scrape_results sr
        CROSS JOIN LATERAL jsonb_array_elements(` + scrapeResultsArraySQL + `) AS elem
        WHERE COALESCE(NULLIF(elem->>'place_id', ''), NULLIF(elem->>'cid', '')) IS NOT NULL
          AND (elem->>'latitude') ~ '^-?[0-9]'
          AND (elem->>'latitude')::float8 <> 0
` + adminLeadQualitySQL + `
    ) raw
    ORDER BY bkey
) t
ON CONFLICT (place_id, tenant_id) DO UPDATE SET
    hidden = TRUE,
    title = COALESCE(NULLIF(EXCLUDED.title, ''), b2b_business_crm.title),
    updated_at = NOW()
WHERE NOT b2b_business_crm.hidden`

	ct, err := s.db.Exec(ctx, q, tenantID)
	if err != nil {
		return 0, err
	}

	return ct.RowsAffected(), nil
}

// ResetLeads wipes CRM overlays (including papelera), tagged search jobs,
// and the shared scrape pool so the map and KPI tiles start at zero.
func (s *store) ResetLeads(ctx context.Context, tenantID int64) (*admin.ResetLeadsResult, error) {
	if tenantID == 0 {
		return nil, fmt.Errorf("missing tenant")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var out admin.ResetLeadsResult

	ct, err := tx.Exec(ctx, `DELETE FROM b2b_business_crm WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("reset crm: %w", err)
	}

	out.CRM = ct.RowsAffected()

	ct, err = tx.Exec(ctx, `DELETE FROM b2b_search_jobs WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("reset search jobs: %w", err)
	}

	out.SearchJobs = ct.RowsAffected()

	ct, err = tx.Exec(ctx, `DELETE FROM scrape_results`)
	if err != nil {
		return nil, fmt.Errorf("reset scrapes: %w", err)
	}

	out.Scrapes = ct.RowsAffected()

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &out, nil
}
