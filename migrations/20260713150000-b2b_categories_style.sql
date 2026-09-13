-- +migrate Up

-- Give categories a color and an icon so they can be styled on the map,
-- legend and lists.
ALTER TABLE b2b_categories ADD COLUMN IF NOT EXISTS color TEXT NOT NULL DEFAULT '#2563eb';
ALTER TABLE b2b_categories ADD COLUMN IF NOT EXISTS icon  TEXT NOT NULL DEFAULT '🏢';

-- Give the common seeded categories sensible defaults.
UPDATE b2b_categories SET icon = '🍽️', color = '#e11d48' WHERE name = 'Restaurantes' AND icon = '🏢';
UPDATE b2b_categories SET icon = '🏨', color = '#7c3aed' WHERE name = 'Hoteles'      AND icon = '🏢';
UPDATE b2b_categories SET icon = '🛒', color = '#059669' WHERE name = 'Supermercados' AND icon = '🏢';

-- +migrate Down
ALTER TABLE b2b_categories DROP COLUMN IF EXISTS icon;
ALTER TABLE b2b_categories DROP COLUMN IF EXISTS color;
