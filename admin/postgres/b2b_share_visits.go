package postgres

import (
	"context"
	"fmt"

	"github.com/gosom/google-maps-scraper/admin"
)

// CreateShareVisit stores one public map-link view for the tenant.
func (s *store) CreateShareVisit(ctx context.Context, tenantID int64, v admin.ShareVisit) error {
	const q = `
INSERT INTO b2b_share_visits (
    tenant_id, kind, target_id, target_name, ip, country, country_iso, city, timezone, user_agent, visited_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	if _, err := s.db.Exec(ctx, q,
		tenantID, v.Kind, v.TargetID, v.TargetName, v.IP, v.Country, v.CountryISO,
		v.City, v.Timezone, v.UserAgent, v.VisitedAt,
	); err != nil {
		return fmt.Errorf("create share visit: %w", err)
	}

	return nil
}

// ListShareVisits returns the newest public map-link visits for the tenant.
func (s *store) ListShareVisits(ctx context.Context, tenantID int64, limit int) ([]admin.ShareVisit, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}

	const q = `
SELECT id, kind, target_id, target_name, ip, country, country_iso, city, timezone, user_agent, visited_at
FROM b2b_share_visits
WHERE tenant_id = $1
ORDER BY visited_at DESC, id DESC
LIMIT $2`

	rows, err := s.db.Query(ctx, q, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("list share visits: %w", err)
	}
	defer rows.Close()

	out := make([]admin.ShareVisit, 0, limit)

	for rows.Next() {
		var v admin.ShareVisit
		if err := rows.Scan(
			&v.ID, &v.Kind, &v.TargetID, &v.TargetName, &v.IP, &v.Country, &v.CountryISO,
			&v.City, &v.Timezone, &v.UserAgent, &v.VisitedAt,
		); err != nil {
			return nil, fmt.Errorf("scan share visit: %w", err)
		}

		out = append(out, v)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list share visits: %w", err)
	}

	return out, nil
}
