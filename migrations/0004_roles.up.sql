-- =============================================================================
--  0004 — Roles (source: res_groups)
--  module_source_id = res_groups.category_id (mengarah ke auth.modules.source_id).
-- =============================================================================

CREATE TABLE IF NOT EXISTS auth.roles (
    id                  BIGSERIAL   PRIMARY KEY,
    source_id           INTEGER     NOT NULL UNIQUE,
    module_source_id    INTEGER,
    name                TEXT        NOT NULL,
    description         TEXT,
    is_share            BOOLEAN     NOT NULL DEFAULT FALSE,
    source_created_at   TIMESTAMP,
    source_updated_at   TIMESTAMP,
    synced_at           TIMESTAMP   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_roles_module ON auth.roles (module_source_id);

-- Hirarki role: parent role "implies" child (mewarisi permission child).
-- Source: res_groups_implied_rel(gid, hid) — gid=parent, hid=child.
CREATE TABLE IF NOT EXISTS auth.role_inheritance (
    parent_role_source_id INTEGER NOT NULL,
    child_role_source_id  INTEGER NOT NULL,
    synced_at             TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (parent_role_source_id, child_role_source_id)
);
CREATE INDEX IF NOT EXISTS idx_role_inh_child ON auth.role_inheritance (child_role_source_id);
