package postgres

import (
	"context"
	"fmt"

	"github.com/gosom/google-maps-scraper/admin"
)

// ListCategories returns the tenant's categories with how many businesses are
// filed under each.
func (s *store) ListCategories(ctx context.Context, tenantID int64) ([]admin.Category, error) {
	const q = `
SELECT c.id, c.name, c.color, c.icon, c.created_at, COUNT(crm.place_id)
FROM b2b_categories c
LEFT JOIN b2b_business_crm crm ON crm.category_id = c.id AND crm.tenant_id = c.tenant_id
WHERE c.tenant_id = $1
GROUP BY c.id, c.name, c.color, c.icon, c.created_at
ORDER BY c.name`

	rows, err := s.db.Query(ctx, q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []admin.Category

	for rows.Next() {
		var c admin.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Color, &c.Icon, &c.CreatedAt, &c.Count); err != nil {
			return nil, err
		}

		out = append(out, c)
	}

	return out, rows.Err()
}

// categoryColor normalizes a category color, falling back to the default blue.
func categoryColor(color string) string {
	if color == "" {
		return "#2563eb"
	}

	return color
}

// categoryIcon normalizes a category icon, falling back to a generic building.
func categoryIcon(icon string) string {
	if icon == "" {
		return "🏢"
	}

	return icon
}

// CreateCategory inserts a new category for the tenant.
func (s *store) CreateCategory(ctx context.Context, tenantID int64, name, color, icon string) (*admin.Category, error) {
	const q = `INSERT INTO b2b_categories (tenant_id, name, color, icon) VALUES ($1, $2, $3, $4)
RETURNING id, name, color, icon, created_at`

	var c admin.Category
	if err := s.db.QueryRow(ctx, q, tenantID, name, categoryColor(color), categoryIcon(icon)).Scan(
		&c.ID, &c.Name, &c.Color, &c.Icon, &c.CreatedAt,
	); err != nil {
		return nil, err
	}

	return &c, nil
}

// UpdateCategory edits one of the tenant's categories (name, color, icon).
func (s *store) UpdateCategory(ctx context.Context, tenantID, id int64, name, color, icon string) error {
	ct, err := s.db.Exec(ctx,
		`UPDATE b2b_categories SET name = $1, color = $2, icon = $3 WHERE id = $4 AND tenant_id = $5`,
		name, categoryColor(color), categoryIcon(icon), id, tenantID)
	if err != nil {
		return err
	}

	if ct.RowsAffected() == 0 {
		return admin.ErrResourceNotFound
	}

	return nil
}

// DeleteCategory removes one of the tenant's categories (businesses keep their
// CRM row; their category_id is set to NULL).
func (s *store) DeleteCategory(ctx context.Context, tenantID, id int64) error {
	_, err := s.db.Exec(ctx, `DELETE FROM b2b_categories WHERE id = $1 AND tenant_id = $2`, id, tenantID)

	return err
}

// ReassignCategory moves every CRM row from one category to another.
func (s *store) ReassignCategory(ctx context.Context, tenantID, fromID, toID int64) error {
	if fromID == 0 || toID == 0 || fromID == toID {
		return nil
	}

	_, err := s.db.Exec(ctx, `
UPDATE b2b_business_crm
SET category_id = $1, updated_at = NOW()
WHERE tenant_id = $2 AND category_id = $3`, toID, tenantID, fromID)

	return err
}

// SetBusinessCategory files a single business under a category (or clears it
// when categoryID is nil).
func (s *store) SetBusinessCategory(ctx context.Context, tenantID int64, key string, categoryID *int64) error {
	if key == "" {
		return fmt.Errorf("empty business key")
	}

	const q = `
INSERT INTO b2b_business_crm (place_id, tenant_id, category_id)
VALUES ($1, $2, $3)
ON CONFLICT (place_id, tenant_id) DO UPDATE SET category_id = EXCLUDED.category_id, updated_at = NOW()`

	_, err := s.db.Exec(ctx, q, key, tenantID, categoryID)

	return err
}

// BulkSetStatus sets the CRM status for many businesses at once (upsert).
func (s *store) BulkSetStatus(ctx context.Context, tenantID int64, keys []string, status string) error {
	if !admin.ValidBusinessStatus(status) {
		return fmt.Errorf("invalid status: %s", status)
	}

	if len(keys) == 0 {
		return nil
	}

	const q = `
INSERT INTO b2b_business_crm (place_id, tenant_id, status)
SELECT k, $2, $3 FROM unnest($1::text[]) AS k
ON CONFLICT (place_id, tenant_id) DO UPDATE SET status = EXCLUDED.status, updated_at = NOW()`

	_, err := s.db.Exec(ctx, q, keys, tenantID, status)

	return err
}

// BulkSetAdvisor assigns many businesses to an advisor at once (upsert).
func (s *store) BulkSetAdvisor(ctx context.Context, tenantID int64, keys []string, advisorID int64) error {
	if len(keys) == 0 {
		return nil
	}

	const q = `
INSERT INTO b2b_business_crm (place_id, tenant_id, advisor_id)
SELECT k, $2, $3 FROM unnest($1::text[]) AS k
ON CONFLICT (place_id, tenant_id) DO UPDATE SET advisor_id = EXCLUDED.advisor_id, updated_at = NOW()`

	_, err := s.db.Exec(ctx, q, keys, tenantID, advisorID)

	return err
}

// BulkSetZone assigns many businesses to a zone at once (upsert).
func (s *store) BulkSetZone(ctx context.Context, tenantID int64, keys []string, zoneID int64) error {
	if len(keys) == 0 {
		return nil
	}

	const q = `
INSERT INTO b2b_business_crm (place_id, tenant_id, zone_id)
SELECT k, $2, $3 FROM unnest($1::text[]) AS k
ON CONFLICT (place_id, tenant_id) DO UPDATE SET zone_id = EXCLUDED.zone_id, updated_at = NOW()`

	_, err := s.db.Exec(ctx, q, keys, tenantID, zoneID)

	return err
}

// BulkSetHidden "deletes" (hides) or restores many businesses for the tenant.
func (s *store) BulkSetHidden(ctx context.Context, tenantID int64, keys []string, hidden bool) error {
	if len(keys) == 0 {
		return nil
	}

	const q = `
INSERT INTO b2b_business_crm (place_id, tenant_id, hidden)
SELECT k, $2, $3 FROM unnest($1::text[]) AS k
ON CONFLICT (place_id, tenant_id) DO UPDATE SET hidden = EXCLUDED.hidden, updated_at = NOW()`

	_, err := s.db.Exec(ctx, q, keys, tenantID, hidden)

	return err
}

// BulkSetCategory files many businesses under a category at once (upsert).
func (s *store) BulkSetCategory(ctx context.Context, tenantID int64, keys []string, categoryID int64) error {
	if len(keys) == 0 {
		return nil
	}

	const q = `
INSERT INTO b2b_business_crm (place_id, tenant_id, category_id)
SELECT k, $2, $3 FROM unnest($1::text[]) AS k
ON CONFLICT (place_id, tenant_id) DO UPDATE SET category_id = EXCLUDED.category_id, updated_at = NOW()`

	_, err := s.db.Exec(ctx, q, keys, tenantID, categoryID)

	return err
}
