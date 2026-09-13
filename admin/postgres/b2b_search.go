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
func (s *store) RecordSearchJob(ctx context.Context, jobID, tenantID int64, rubro, specialty string) error {
	if jobID == 0 || tenantID == 0 {
		return fmt.Errorf("missing job or tenant")
	}

	const q = `
INSERT INTO b2b_search_jobs (job_id, tenant_id, rubro, specialty)
VALUES ($1, $2, $3, $4)
ON CONFLICT (job_id) DO NOTHING`

	_, err := s.db.Exec(ctx, q, jobID, tenantID, rubro, specialty)

	return err
}

// IngestPendingSearchJobs files completed scrapes that still need a CRM overlay.
func (s *store) IngestPendingSearchJobs(ctx context.Context) error {
	const q = `
SELECT j.job_id
FROM b2b_search_jobs j
INNER JOIN scrape_results sr ON sr.job_id = j.job_id
WHERE NOT j.ingested`

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

// IngestSearchJob upserts qualifying leads from a finished scrape into the
// tenant CRM, tagging category and specialty. Existing status / category /
// specialty are left alone.
func (s *store) IngestSearchJob(ctx context.Context, jobID int64) error {
	if jobID == 0 {
		return nil
	}

	var (
		tenantID  int64
		rubro     string
		specialty string
		ingested  bool
	)

	err := s.db.QueryRow(ctx,
		`SELECT tenant_id, rubro, specialty, ingested FROM b2b_search_jobs WHERE job_id = $1`,
		jobID,
	).Scan(&tenantID, &rubro, &specialty, &ingested)
	if err != nil {
		return nil // no tagged job — nothing to file
	}

	if ingested {
		return nil
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

	keys := make([]string, 0, len(entries))
	titles := make([]string, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))

	for i := range entries {
		e := entries[i]
		if !admin.QualifiesAsLead(e.Title, e.ReviewCount) {
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

	canon := admin.InferFixedCategory(rubro)
	if canon == "" {
		canon = admin.InferFixedCategory(specialty)
	}

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
    updated_at  = NOW()`

		if _, err := s.db.Exec(ctx, upsert, keys, tenantID, admin.StatusProspect, categoryID, specialty, titles); err != nil {
			return err
		}
	}

	_, err = s.db.Exec(ctx, `UPDATE b2b_search_jobs SET ingested = TRUE WHERE job_id = $1`, jobID)

	return err
}

// HideBusinessesByStatus sends every tenant business with the given status to
// the trash (hidden), so they leave the map and counters.
func (s *store) HideBusinessesByStatus(ctx context.Context, tenantID int64, status string) (int64, error) {
	if !admin.ValidBusinessStatus(status) {
		return 0, fmt.Errorf("invalid status: %s", status)
	}

	ct, err := s.db.Exec(ctx, `
UPDATE b2b_business_crm
SET hidden = TRUE, updated_at = NOW()
WHERE tenant_id = $1 AND status = $2 AND NOT hidden`, tenantID, status)
	if err != nil {
		return 0, err
	}

	return ct.RowsAffected(), nil
}
