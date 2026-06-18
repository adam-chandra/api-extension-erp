-- =============================================================================
--  0002 — Companies (source: res_company)
-- =============================================================================

CREATE TABLE IF NOT EXISTS auth.companies (
    id                  BIGSERIAL   PRIMARY KEY,
    source_id           INTEGER     NOT NULL UNIQUE,
    parent_source_id    INTEGER,
    name                TEXT        NOT NULL,
    is_active           BOOLEAN     NOT NULL DEFAULT TRUE,
    source_created_at   TIMESTAMP,
    source_updated_at   TIMESTAMP,
    synced_at           TIMESTAMP   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_companies_parent ON auth.companies (parent_source_id);
CREATE INDEX IF NOT EXISTS idx_companies_active ON auth.companies (is_active);
