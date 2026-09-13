-- +migrate Up

ALTER TABLE b2b_search_jobs ADD COLUMN IF NOT EXISTS rating_band TEXT NOT NULL DEFAULT '';

-- +migrate Down

ALTER TABLE b2b_search_jobs DROP COLUMN IF EXISTS rating_band;
