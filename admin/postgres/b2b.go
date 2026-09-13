package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gosom/google-maps-scraper/admin"
)

func jsonbOrNil(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}

	return raw
}

// defaultBusinessLimit caps how many businesses we return for the map so the
// browser stays responsive even with very large scrape datasets.
const defaultBusinessLimit = 5000

// cityMatchSQL compares the first letter-token of a scraped city to $1 so
// "Bogotá, BOGOTÁ D.C." and "BOGOTÁ" both match the filter "Bogotá".
const cityMatchSQL = `split_part(translate(lower(regexp_replace(COALESCE(city, ''), '[^A-Za-zÁÉÍÓÚáéíóúÜüÑñ]+', ' ', 'g')), 'áéíóúüñÁÉÍÓÚÜÑ', 'aeiouunAEIOUUN'), ' ', 1) = split_part(translate(lower(regexp_replace($1, '[^A-Za-zÁÉÍÓÚáéíóúÜüÑñ]+', ' ', 'g')), 'áéíóúüñÁÉÍÓÚÜÑ', 'aeiouunAEIOUUN'), ' ', 1)`

// listBusinessesQuery returns scraped businesses (shared lead pool) joined with
// the requesting tenant's CRM overlay. All filters are parameterized so the
// query stays a compile-time constant. $6 is the tenant id.
const listBusinessesQuery = `
SELECT bkey, title, category, address, city, phone, website, maps_url, email, specialty, lat, lng, rating, review_count, status, advisor_id, zone_id, category_id, notes
FROM (
    SELECT DISTINCT ON (bkey)
        bkey, title, category, address, city, phone, website, maps_url, email, specialty, lat, lng, rating, review_count,
        status, advisor_id, zone_id, category_id, category_name, notes, added_at
    FROM (
        SELECT
            COALESCE(NULLIF(elem->>'place_id', ''), NULLIF(elem->>'cid', '')) AS bkey,
            elem->>'title'    AS title,
            COALESCE(elem->>'category', '') AS category,
            COALESCE(elem->>'address', '')  AS address,
            COALESCE(elem->'complete_address'->>'city', '') AS city,
            COALESCE(elem->>'phone', '')    AS phone,
            COALESCE(elem->>'web_site', '') AS website,
            COALESCE(elem->>'link', '') AS maps_url,
            COALESCE(elem->'emails'->>0, '') AS email,
            COALESCE(NULLIF(crm.specialty, ''), elem->>'category', '') AS specialty,
            (elem->>'latitude')::float8 AS lat,
            COALESCE(NULLIF(elem->>'longitude', '')::float8, NULLIF(elem->>'longtitude', '')::float8) AS lng,
            COALESCE(NULLIF(elem->>'review_rating', '')::float8, 0) AS rating,
            COALESCE(NULLIF(elem->>'review_count', '')::int, 0) AS review_count,
            COALESCE(crm.status, 'prospect') AS status,
            crm.advisor_id,
            crm.zone_id,
            crm.category_id,
            cat.name AS category_name,
            COALESCE(crm.notes, '') AS notes,
            sr.created_at AS added_at
        FROM scrape_results sr
        CROSS JOIN LATERAL jsonb_array_elements(sr.results) AS elem
        LEFT JOIN b2b_business_crm crm
            ON crm.place_id = COALESCE(NULLIF(elem->>'place_id', ''), NULLIF(elem->>'cid', ''))
           AND crm.tenant_id = $6
        LEFT JOIN b2b_categories cat ON cat.id = crm.category_id AND cat.tenant_id = $6
        WHERE COALESCE(NULLIF(elem->>'place_id', ''), NULLIF(elem->>'cid', '')) IS NOT NULL
          AND (elem->>'latitude') ~ '^-?[0-9]'
          AND (elem->>'latitude')::float8 <> 0
          AND COALESCE(crm.hidden, false) = $10
` + adminLeadQualitySQL + `
    ) raw
    WHERE ($1 = '' OR ` + cityMatchSQL + `)
      AND ($2 = '' OR status = $2)
      AND ($3 = 0 OR advisor_id = $3)
      AND ($4 = '' OR title ILIKE '%' || $4 || '%' OR category ILIKE '%' || $4 || '%' OR address ILIKE '%' || $4 || '%' OR specialty ILIKE '%' || $4 || '%')
      AND ($7 = 0 OR category_id = $7)
    ORDER BY bkey, added_at DESC
) t
ORDER BY
    (CASE WHEN $9 = 'added' THEN added_at END) DESC NULLS LAST,
    (CASE WHEN $9 = 'category' THEN lower(coalesce(category_name, '~')) END) ASC NULLS LAST,
    lower(title) ASC
LIMIT $5 OFFSET $8`

// adminLeadQualitySQL is the shared review/chain filter (kept next to the queries).
const adminLeadQualitySQL = `
          AND COALESCE(NULLIF(elem->>'review_count', '')::int, 0) > 50
          AND NOT ((' ' || translate(lower(regexp_replace(COALESCE(elem->>'title', ''), '[^A-Za-z0-9ÁÉÍÓÚáéíóúÜüÑñ]+', ' ', 'g')), 'áéíóúüñÁÉÍÓÚÜÑ', 'aeiouunAEIOUUN') || ' ') ~ ' (exito|olimpica|d1|ara|carulla|oxxo|falabella|isimo|homecenter|easy|justo y bueno|justo bueno) ')`

// ListBusinesses returns scraped businesses with the tenant's CRM overlay.
func (s *store) ListBusinesses(ctx context.Context, tenantID int64, f admin.BusinessFilter) ([]admin.MapBusiness, error) {
	limit := f.Limit
	if limit <= 0 || limit > defaultBusinessLimit {
		limit = defaultBusinessLimit
	}

	var advisorID int64
	if f.AdvisorID != nil {
		advisorID = *f.AdvisorID
	}

	var categoryID int64
	if f.CategoryID != nil {
		categoryID = *f.CategoryID
	}

	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	sort := f.Sort
	if sort != "category" && sort != "added" {
		sort = "name"
	}

	rows, err := s.db.Query(ctx, listBusinessesQuery, f.City, f.Status, advisorID, f.Search, limit, tenantID, categoryID, offset, sort, f.Hidden)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []admin.MapBusiness

	for rows.Next() {
		var mb admin.MapBusiness
		if err := rows.Scan(
			&mb.Key, &mb.Title, &mb.Category, &mb.Address, &mb.City,
			&mb.Phone, &mb.Website, &mb.MapsURL, &mb.Email, &mb.Specialty,
			&mb.Lat, &mb.Lng, &mb.Rating, &mb.ReviewCount,
			&mb.Status, &mb.AdvisorID, &mb.ZoneID, &mb.CategoryID, &mb.Notes,
		); err != nil {
			return nil, err
		}

		out = append(out, mb)
	}

	return out, rows.Err()
}

// countBusinessesQuery counts the distinct businesses matching the filters
// (same shape as listBusinessesQuery, without paging). $6 is the tenant id.
const countBusinessesQuery = `
SELECT COUNT(*) FROM (
    SELECT DISTINCT ON (bkey) bkey FROM (
        SELECT
            COALESCE(NULLIF(elem->>'place_id', ''), NULLIF(elem->>'cid', '')) AS bkey,
            elem->>'title'    AS title,
            COALESCE(elem->>'category', '') AS category,
            COALESCE(elem->>'address', '')  AS address,
            COALESCE(elem->'complete_address'->>'city', '') AS city,
            COALESCE(NULLIF(crm.specialty, ''), elem->>'category', '') AS specialty,
            COALESCE(crm.status, 'prospect') AS status,
            crm.advisor_id,
            crm.category_id
        FROM scrape_results sr
        CROSS JOIN LATERAL jsonb_array_elements(sr.results) AS elem
        LEFT JOIN b2b_business_crm crm
            ON crm.place_id = COALESCE(NULLIF(elem->>'place_id', ''), NULLIF(elem->>'cid', ''))
           AND crm.tenant_id = $6
        WHERE COALESCE(NULLIF(elem->>'place_id', ''), NULLIF(elem->>'cid', '')) IS NOT NULL
          AND (elem->>'latitude') ~ '^-?[0-9]'
          AND (elem->>'latitude')::float8 <> 0
          AND COALESCE(crm.hidden, false) = $7
` + adminLeadQualitySQL + `
    ) t
    WHERE ($1 = '' OR ` + cityMatchSQL + `)
      AND ($2 = '' OR status = $2)
      AND ($3 = 0 OR advisor_id = $3)
      AND ($4 = '' OR title ILIKE '%' || $4 || '%' OR category ILIKE '%' || $4 || '%' OR address ILIKE '%' || $4 || '%' OR COALESCE(specialty, '') ILIKE '%' || $4 || '%')
      AND ($5 = 0 OR category_id = $5)
) c`

// CountBusinesses returns how many businesses match the filter (for paging).
func (s *store) CountBusinesses(ctx context.Context, tenantID int64, f admin.BusinessFilter) (int, error) {
	var advisorID int64
	if f.AdvisorID != nil {
		advisorID = *f.AdvisorID
	}

	var categoryID int64
	if f.CategoryID != nil {
		categoryID = *f.CategoryID
	}

	var n int
	err := s.db.QueryRow(ctx, countBusinessesQuery,
		f.City, f.Status, advisorID, f.Search, categoryID, tenantID, f.Hidden).Scan(&n)

	return n, err
}

// ListBusinessCities returns the predefined city list (Bogotá first).
// Scraped raw values are not used: they produce duplicates like
// "Bogotá, BOGOTÁ D.C." and leftover towns from old searches.
func (s *store) ListBusinessCities(_ context.Context) ([]string, error) {
	return admin.CityList(), nil
}

// SetBusinessCRM upserts the tenant's CRM overlay for a business.
func (s *store) SetBusinessCRM(ctx context.Context, tenantID int64, key, status string, advisorID, zoneID *int64, notes, title string) error {
	if key == "" {
		return fmt.Errorf("empty business key")
	}

	if !admin.ValidBusinessStatus(status) {
		return fmt.Errorf("invalid status: %s", status)
	}

	const q = `
INSERT INTO b2b_business_crm (place_id, tenant_id, status, advisor_id, zone_id, notes, title, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
ON CONFLICT (place_id, tenant_id) DO UPDATE SET
    status     = EXCLUDED.status,
    advisor_id = EXCLUDED.advisor_id,
    zone_id    = EXCLUDED.zone_id,
    notes      = EXCLUDED.notes,
    title      = COALESCE(NULLIF(EXCLUDED.title, ''), b2b_business_crm.title),
    updated_at = NOW()`

	_, err := s.db.Exec(ctx, q, key, tenantID, status, advisorID, zoneID, notes, title)

	return err
}

// businessTotalQuery counts distinct scraped businesses that have coordinates,
// excluding the ones the tenant sent to the trash (hidden). $1 is the tenant id.
const businessTotalQuery = `
SELECT COUNT(*) FROM (
    SELECT DISTINCT COALESCE(NULLIF(elem->>'place_id', ''), NULLIF(elem->>'cid', '')) AS bkey
    FROM scrape_results sr
    CROSS JOIN LATERAL jsonb_array_elements(sr.results) AS elem
    LEFT JOIN b2b_business_crm crm
        ON crm.place_id = COALESCE(NULLIF(elem->>'place_id', ''), NULLIF(elem->>'cid', ''))
       AND crm.tenant_id = $1
    WHERE COALESCE(NULLIF(elem->>'place_id', ''), NULLIF(elem->>'cid', '')) IS NOT NULL
      AND (elem->>'latitude') ~ '^-?[0-9]'
      AND (elem->>'latitude')::float8 <> 0
      AND NOT COALESCE(crm.hidden, false)
` + adminLeadQualitySQL + `
) t`

// B2BSummary returns aggregate counters for the tenant's dashboard header.
// When advisorID is non-nil the business counts are limited to that advisor's
// assigned businesses (so an advisor only sees their own numbers).
func (s *store) B2BSummary(ctx context.Context, tenantID int64, advisorID *int64) (*admin.B2BSummary, error) {
	var sum admin.B2BSummary

	// Trashed (hidden) businesses never count toward any totalizer.
	statusQ := `SELECT status, COUNT(*) FROM b2b_business_crm WHERE tenant_id = $1 AND NOT hidden`
	args := []any{tenantID}

	if advisorID != nil {
		statusQ += ` AND advisor_id = $2`
		args = append(args, *advisorID)
	} else {
		// Tenant-wide total counts every scraped business (shared lead pool),
		// minus the ones this tenant sent to the trash.
		if err := s.db.QueryRow(ctx, businessTotalQuery, tenantID).Scan(&sum.Total); err != nil {
			return nil, err
		}
	}

	statusQ += ` GROUP BY status`

	statusRows, err := s.db.Query(ctx, statusQ, args...)
	if err != nil {
		return nil, err
	}
	defer statusRows.Close()

	for statusRows.Next() {
		var (
			st string
			n  int
		)

		if err := statusRows.Scan(&st, &n); err != nil {
			return nil, err
		}

		switch st {
		case admin.StatusClient:
			sum.Clients = n
		case admin.StatusInProgress:
			sum.InProgress = n
		case admin.StatusDiscarded:
			sum.Discarded = n
		case admin.StatusProspect:
			sum.Prospects = n
		case admin.StatusFeatured:
			sum.Featured = n
		}
	}

	if err := statusRows.Err(); err != nil {
		return nil, err
	}

	if advisorID != nil {
		// The advisor only sees their assigned businesses; total is their sum.
		sum.Total = sum.Clients + sum.InProgress + sum.Discarded + sum.Prospects + sum.Featured
	} else if tracked := sum.Clients + sum.InProgress + sum.Discarded + sum.Prospects + sum.Featured; sum.Total > tracked {
		// Every business without a CRM row yet counts as a prospect.
		sum.Prospects = sum.Total - sum.Clients - sum.InProgress - sum.Discarded - sum.Featured
	}

	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM b2b_advisors WHERE active AND tenant_id = $1`, tenantID).Scan(&sum.Advisors); err != nil {
		return nil, err
	}

	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM b2b_zones WHERE tenant_id = $1`, tenantID).Scan(&sum.Zones); err != nil {
		return nil, err
	}

	return &sum, nil
}

// ListAdvisors returns the tenant's advisors ordered by name.
func (s *store) ListAdvisors(ctx context.Context, tenantID int64) ([]admin.Advisor, error) {
	const q = `SELECT id, name, COALESCE(email, ''), COALESCE(phone, ''), COALESCE(city, ''), active, created_at
FROM b2b_advisors WHERE tenant_id = $1 ORDER BY name`

	rows, err := s.db.Query(ctx, q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []admin.Advisor

	for rows.Next() {
		var a admin.Advisor
		if err := rows.Scan(&a.ID, &a.Name, &a.Email, &a.Phone, &a.City, &a.Active, &a.CreatedAt); err != nil {
			return nil, err
		}

		out = append(out, a)
	}

	return out, rows.Err()
}

// CreateAdvisor inserts a new advisor for the tenant.
func (s *store) CreateAdvisor(ctx context.Context, tenantID int64, name, email, phone, city string) (*admin.Advisor, error) {
	const q = `INSERT INTO b2b_advisors (tenant_id, name, email, phone, city) VALUES ($1, $2, $3, $4, $5)
RETURNING id, name, COALESCE(email, ''), COALESCE(phone, ''), COALESCE(city, ''), active, created_at`

	var a admin.Advisor
	if err := s.db.QueryRow(ctx, q, tenantID, name, email, phone, city).Scan(
		&a.ID, &a.Name, &a.Email, &a.Phone, &a.City, &a.Active, &a.CreatedAt,
	); err != nil {
		return nil, err
	}

	return &a, nil
}

// UpdateAdvisor edits one of the tenant's advisors.
func (s *store) UpdateAdvisor(ctx context.Context, tenantID, id int64, name, email, phone, city string) error {
	ct, err := s.db.Exec(ctx,
		`UPDATE b2b_advisors SET name = $1, email = $2, phone = $3, city = $4 WHERE id = $5 AND tenant_id = $6`,
		name, email, phone, city, id, tenantID)
	if err != nil {
		return err
	}

	if ct.RowsAffected() == 0 {
		return admin.ErrResourceNotFound
	}

	return nil
}

// DeleteAdvisor removes one of the tenant's advisors.
func (s *store) DeleteAdvisor(ctx context.Context, tenantID, id int64) error {
	_, err := s.db.Exec(ctx, `DELETE FROM b2b_advisors WHERE id = $1 AND tenant_id = $2`, id, tenantID)

	return err
}

// ListZones returns the tenant's zones ordered by city then name.
func (s *store) ListZones(ctx context.Context, tenantID int64) ([]admin.Zone, error) {
	const q = `SELECT id, name, city, advisor_id, color, geometry, created_at FROM b2b_zones WHERE tenant_id = $1 ORDER BY city, name`

	rows, err := s.db.Query(ctx, q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []admin.Zone

	for rows.Next() {
		var z admin.Zone
		var geom []byte
		if err := rows.Scan(&z.ID, &z.Name, &z.City, &z.AdvisorID, &z.Color, &geom, &z.CreatedAt); err != nil {
			return nil, err
		}

		if len(geom) > 0 {
			z.Geometry = geom
		}

		out = append(out, z)
	}

	return out, rows.Err()
}

// CreateZone inserts a new zone for the tenant.
func (s *store) CreateZone(ctx context.Context, tenantID int64, name, city string, advisorID *int64, color string, geometry json.RawMessage) (*admin.Zone, error) {
	if color == "" {
		color = "#2563eb"
	}

	const q = `INSERT INTO b2b_zones (tenant_id, name, city, advisor_id, color, geometry) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, name, city, advisor_id, color, geometry, created_at`

	var z admin.Zone
	var geom []byte
	if err := s.db.QueryRow(ctx, q, tenantID, name, city, advisorID, color, jsonbOrNil(geometry)).Scan(
		&z.ID, &z.Name, &z.City, &z.AdvisorID, &z.Color, &geom, &z.CreatedAt,
	); err != nil {
		return nil, err
	}

	if len(geom) > 0 {
		z.Geometry = geom
	}

	return &z, nil
}

// UpdateZone edits one of the tenant's zones.
func (s *store) UpdateZone(ctx context.Context, tenantID, id int64, name, city string, advisorID *int64, color string, geometry json.RawMessage) error {
	if color == "" {
		color = "#2563eb"
	}

	const q = `
UPDATE b2b_zones
SET name = $1, city = $2, advisor_id = $3, color = $4,
    geometry = CASE WHEN $5::jsonb IS NULL THEN geometry ELSE $5::jsonb END
WHERE id = $6 AND tenant_id = $7`

	ct, err := s.db.Exec(ctx, q, name, city, advisorID, color, jsonbOrNil(geometry), id, tenantID)
	if err != nil {
		return err
	}

	if ct.RowsAffected() == 0 {
		return admin.ErrResourceNotFound
	}

	return nil
}

// DeleteZone removes one of the tenant's zones.
func (s *store) DeleteZone(ctx context.Context, tenantID, id int64) error {
	_, err := s.db.Exec(ctx, `DELETE FROM b2b_zones WHERE id = $1 AND tenant_id = $2`, id, tenantID)

	return err
}

// OwnedBusinessKeys returns the subset of keys that belong to advisorID and
// are not in the tenant trash.
func (s *store) OwnedBusinessKeys(ctx context.Context, tenantID, advisorID int64, keys []string) ([]string, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	const q = `
SELECT place_id FROM b2b_business_crm
WHERE tenant_id = $1 AND advisor_id = $2 AND place_id = ANY($3)`

	rows, err := s.db.Query(ctx, q, tenantID, advisorID, keys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string

	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}

		out = append(out, k)
	}

	return out, rows.Err()
}
