-- =============================================================================
--  0007 — Convenience views (read-only API surface untuk BE)
--  BE cukup query view ini, tidak perlu tahu skema sumber Odoo.
-- =============================================================================

-- Direct (langsung) roles per user
CREATE OR REPLACE VIEW auth.v_user_roles AS
SELECT
    u.id              AS user_id,
    u.source_id       AS user_source_id,
    u.login,
    r.id              AS role_id,
    r.source_id       AS role_source_id,
    r.name            AS role_name,
    r.module_source_id
FROM auth.users u
JOIN auth.user_roles ur ON ur.user_source_id = u.source_id
JOIN auth.roles      r  ON r.source_id      = ur.role_source_id
WHERE u.is_active = TRUE;

-- Effective roles per user (termasuk inherited roles via role_inheritance).
-- Catatan: di Odoo, group "parent" mewarisi semua permission group "child"
-- (implied_ids). Jadi role efektif user = union(direct, inherited).
CREATE OR REPLACE VIEW auth.v_user_effective_roles AS
WITH RECURSIVE direct AS (
    SELECT user_source_id, role_source_id
    FROM auth.user_roles
),
effective AS (
    SELECT user_source_id, role_source_id FROM direct
    UNION
    SELECT e.user_source_id, ri.child_role_source_id
    FROM effective e
    JOIN auth.role_inheritance ri
      ON ri.parent_role_source_id = e.role_source_id
)
SELECT DISTINCT
    u.id        AS user_id,
    u.source_id AS user_source_id,
    u.login,
    r.id        AS role_id,
    r.source_id AS role_source_id,
    r.name      AS role_name,
    r.module_source_id
FROM effective e
JOIN auth.users u ON u.source_id = e.user_source_id
JOIN auth.roles r ON r.source_id = e.role_source_id
WHERE u.is_active = TRUE;

-- Modul yang accessible oleh user (= modul yang punya minimal 1 role yg dipegang user)
CREATE OR REPLACE VIEW auth.v_user_modules AS
SELECT DISTINCT
    er.user_id,
    er.user_source_id,
    er.login,
    m.id        AS module_id,
    m.source_id AS module_source_id,
    m.name      AS module_name,
    m.parent_source_id AS module_parent_source_id
FROM auth.v_user_effective_roles er
JOIN auth.modules m ON m.source_id = er.module_source_id;

-- Companies yang accessible oleh user
CREATE OR REPLACE VIEW auth.v_user_companies AS
SELECT
    u.id        AS user_id,
    u.source_id AS user_source_id,
    u.login,
    c.id        AS company_id,
    c.source_id AS company_source_id,
    c.name      AS company_name,
    (c.source_id = u.default_company_source_id) AS is_default
FROM auth.users u
JOIN auth.user_companies uc ON uc.user_source_id = u.source_id
JOIN auth.companies c       ON c.source_id      = uc.company_source_id
WHERE u.is_active = TRUE
  AND c.is_active = TRUE;
