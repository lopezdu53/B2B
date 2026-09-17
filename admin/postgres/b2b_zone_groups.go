package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/gosom/google-maps-scraper/admin"
)

func uniqPositiveIDs(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))

	for _, id := range ids {
		if id <= 0 {
			continue
		}

		if _, ok := seen[id]; ok {
			continue
		}

		seen[id] = struct{}{}
		out = append(out, id)
	}

	return out
}

func (s *store) tenantZoneIDs(ctx context.Context, tenantID int64, ids []int64) ([]int64, error) {
	ids = uniqPositiveIDs(ids)
	if len(ids) == 0 {
		return nil, nil
	}

	const q = `SELECT id FROM b2b_zones WHERE tenant_id = $1 AND id = ANY($2) ORDER BY city, name`

	rows, err := s.db.Query(ctx, q, tenantID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]int64, 0, len(ids))

	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}

		out = append(out, id)
	}

	return out, rows.Err()
}

func scanZoneGroupRow(row pgx.Row) (*admin.ZoneGroup, error) {
	var g admin.ZoneGroup
	if err := row.Scan(&g.ID, &g.Name, &g.CreatedAt); err != nil {
		return nil, err
	}

	return &g, nil
}

func (s *store) loadGroupZones(ctx context.Context, tenantID, groupID int64) ([]admin.Zone, []int64, error) {
	const q = `
SELECT z.id, z.name, z.city, z.advisor_id, z.color, z.geometry, z.created_at
FROM b2b_zone_group_members m
JOIN b2b_zones z ON z.id = m.zone_id AND z.tenant_id = $1
WHERE m.group_id = $2
ORDER BY z.city, z.name`

	rows, err := s.db.Query(ctx, q, tenantID, groupID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var zones []admin.Zone
	var ids []int64

	for rows.Next() {
		var z admin.Zone
		var geom []byte

		if err := rows.Scan(&z.ID, &z.Name, &z.City, &z.AdvisorID, &z.Color, &geom, &z.CreatedAt); err != nil {
			return nil, nil, err
		}

		if len(geom) > 0 {
			z.Geometry = geom
		}

		zones = append(zones, z)
		ids = append(ids, z.ID)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return zones, ids, nil
}

func (s *store) replaceGroupMembers(ctx context.Context, tx pgx.Tx, groupID int64, zoneIDs []int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM b2b_zone_group_members WHERE group_id = $1`, groupID); err != nil {
		return fmt.Errorf("clear zone group members: %w", err)
	}

	if len(zoneIDs) == 0 {
		return nil
	}

	const q = `
INSERT INTO b2b_zone_group_members (group_id, zone_id)
SELECT $1, z FROM unnest($2::bigint[]) AS z`

	if _, err := tx.Exec(ctx, q, groupID, zoneIDs); err != nil {
		return fmt.Errorf("insert zone group members: %w", err)
	}

	return nil
}

// ListZoneGroups returns the tenant's zone groups with their member zones.
func (s *store) ListZoneGroups(ctx context.Context, tenantID int64) ([]admin.ZoneGroup, error) {
	const q = `SELECT id, name, created_at FROM b2b_zone_groups WHERE tenant_id = $1 ORDER BY name`

	rows, err := s.db.Query(ctx, q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []admin.ZoneGroup

	for rows.Next() {
		g, err := scanZoneGroupRow(rows)
		if err != nil {
			return nil, err
		}

		out = append(out, *g)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range out {
		zones, ids, err := s.loadGroupZones(ctx, tenantID, out[i].ID)
		if err != nil {
			return nil, err
		}

		out[i].Zones = zones
		out[i].ZoneIDs = ids
	}

	return out, nil
}

// GetZoneGroup returns one of the tenant's zone groups with member zones.
func (s *store) GetZoneGroup(ctx context.Context, tenantID, id int64) (*admin.ZoneGroup, error) {
	const q = `SELECT id, name, created_at FROM b2b_zone_groups WHERE id = $1 AND tenant_id = $2`

	g, err := scanZoneGroupRow(s.db.QueryRow(ctx, q, id, tenantID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, admin.ErrResourceNotFound
		}

		return nil, err
	}

	zones, ids, err := s.loadGroupZones(ctx, tenantID, g.ID)
	if err != nil {
		return nil, err
	}

	g.Zones = zones
	g.ZoneIDs = ids

	return g, nil
}

// CreateZoneGroup inserts a named group of tenant zones.
func (s *store) CreateZoneGroup(ctx context.Context, tenantID int64, name string, zoneIDs []int64) (*admin.ZoneGroup, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("el nombre del grupo es obligatorio")
	}

	valid, err := s.tenantZoneIDs(ctx, tenantID, zoneIDs)
	if err != nil {
		return nil, err
	}

	if len(valid) == 0 {
		return nil, fmt.Errorf("elige al menos una zona")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const q = `INSERT INTO b2b_zone_groups (tenant_id, name) VALUES ($1, $2) RETURNING id, name, created_at`

	g, err := scanZoneGroupRow(tx.QueryRow(ctx, q, tenantID, name))
	if err != nil {
		return nil, fmt.Errorf("create zone group: %w", err)
	}

	if err := s.replaceGroupMembers(ctx, tx, g.ID, valid); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	zones, ids, err := s.loadGroupZones(ctx, tenantID, g.ID)
	if err != nil {
		return nil, err
	}

	g.Zones = zones
	g.ZoneIDs = ids

	return g, nil
}

// UpdateZoneGroup edits a group's name and member zones.
func (s *store) UpdateZoneGroup(ctx context.Context, tenantID, id int64, name string, zoneIDs []int64) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("el nombre del grupo es obligatorio")
	}

	valid, err := s.tenantZoneIDs(ctx, tenantID, zoneIDs)
	if err != nil {
		return err
	}

	if len(valid) == 0 {
		return fmt.Errorf("elige al menos una zona")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ct, err := tx.Exec(ctx, `UPDATE b2b_zone_groups SET name = $1 WHERE id = $2 AND tenant_id = $3`, name, id, tenantID)
	if err != nil {
		return fmt.Errorf("update zone group: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return admin.ErrResourceNotFound
	}

	if err := s.replaceGroupMembers(ctx, tx, id, valid); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// DeleteZoneGroup removes a group and its memberships.
func (s *store) DeleteZoneGroup(ctx context.Context, tenantID, id int64) error {
	ct, err := s.db.Exec(ctx, `DELETE FROM b2b_zone_groups WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return err
	}

	if ct.RowsAffected() == 0 {
		return admin.ErrResourceNotFound
	}

	return nil
}
