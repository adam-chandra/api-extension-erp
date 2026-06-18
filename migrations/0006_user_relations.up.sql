-- =============================================================================
--  0006 — User M2M (companies + roles)
--  Source:
--    res_company_users_rel (cid, user_id)  -> auth.user_companies
--    res_groups_users_rel  (gid, uid)      -> auth.user_roles
--  Strategi sync: full-refresh per tick (kedua tabel tidak punya write_date).
-- =============================================================================

CREATE TABLE IF NOT EXISTS auth.user_companies (
    user_source_id     INTEGER NOT NULL,
    company_source_id  INTEGER NOT NULL,
    synced_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_source_id, company_source_id)
);
CREATE INDEX IF NOT EXISTS idx_user_companies_company ON auth.user_companies (company_source_id);

CREATE TABLE IF NOT EXISTS auth.user_roles (
    user_source_id  INTEGER NOT NULL,
    role_source_id  INTEGER NOT NULL,
    synced_at       TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_source_id, role_source_id)
);
CREATE INDEX IF NOT EXISTS idx_user_roles_role ON auth.user_roles (role_source_id);
