-- +migrate Up

-- Public zone/group map link visits (IP + geo + local clock).
CREATE TABLE IF NOT EXISTS b2b_share_visits (
    id          BIGSERIAL PRIMARY KEY,
    tenant_id   BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    kind        TEXT NOT NULL,
    target_id   BIGINT NOT NULL,
    target_name TEXT NOT NULL DEFAULT '',
    ip          TEXT NOT NULL DEFAULT '',
    country     TEXT NOT NULL DEFAULT '',
    country_iso TEXT NOT NULL DEFAULT '',
    city        TEXT NOT NULL DEFAULT '',
    timezone    TEXT NOT NULL DEFAULT '',
    user_agent  TEXT NOT NULL DEFAULT '',
    visited_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_b2b_share_visits_tenant_visited
    ON b2b_share_visits (tenant_id, visited_at DESC);

-- +migrate Down

DROP TABLE IF EXISTS b2b_share_visits;
