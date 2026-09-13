-- +migrate Up

-- Drawn territory for a CRM zone (GeoJSON Polygon / MultiPolygon).
ALTER TABLE b2b_zones ADD COLUMN IF NOT EXISTS geometry JSONB;

-- +migrate Down
ALTER TABLE b2b_zones DROP COLUMN IF EXISTS geometry;
