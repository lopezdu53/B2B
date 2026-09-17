-- +migrate Up

-- Named collections of zones that share one public map link.
CREATE TABLE IF NOT EXISTS b2b_zone_groups (
    id         BIGSERIAL PRIMARY KEY,
    tenant_id  BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_b2b_zone_groups_tenant ON b2b_zone_groups(tenant_id);

CREATE TABLE IF NOT EXISTS b2b_zone_group_members (
    group_id BIGINT NOT NULL REFERENCES b2b_zone_groups(id) ON DELETE CASCADE,
    zone_id  BIGINT NOT NULL REFERENCES b2b_zones(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, zone_id)
);

CREATE INDEX IF NOT EXISTS idx_b2b_zone_group_members_zone ON b2b_zone_group_members(zone_id);

-- +migrate Down

DROP TABLE IF EXISTS b2b_zone_group_members;
DROP TABLE IF EXISTS b2b_zone_groups;
