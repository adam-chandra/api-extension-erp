-- =============================================================================
--  0013 — Performance indexes for auth join paths
--  Login flow scans v_user_effective_roles which seeds from user_roles and
--  walks role_inheritance recursively. Without these indexes the seed and
--  the recursion both fall back to seq-scans (97k+ rows), causing 14s+ login
--  latency. All indexes are idempotent.
-- =============================================================================

CREATE INDEX IF NOT EXISTS idx_user_roles_user
    ON auth.user_roles (user_source_id);

CREATE INDEX IF NOT EXISTS idx_role_inh_parent
    ON auth.role_inheritance (parent_role_source_id);

CREATE INDEX IF NOT EXISTS idx_user_companies_user
    ON auth.user_companies (user_source_id);

CREATE INDEX IF NOT EXISTS idx_menu_roles_menu
    ON auth.menu_roles (menu_source_id);
