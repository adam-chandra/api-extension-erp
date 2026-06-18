-- =============================================================================
--  0003 — Modules (source: ir_module_category)
--  Hirarki Odoo dipertahankan via parent_source_id (234 kategori).
-- =============================================================================

CREATE TABLE IF NOT EXISTS auth.modules (
    id                  BIGSERIAL   PRIMARY KEY,
    source_id           INTEGER     NOT NULL UNIQUE,
    parent_source_id    INTEGER,
    name                TEXT        NOT NULL,
    description         TEXT,
    sequence            INTEGER,
    is_visible          BOOLEAN     NOT NULL DEFAULT TRUE,
    source_created_at   TIMESTAMP,
    source_updated_at   TIMESTAMP,
    synced_at           TIMESTAMP   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_modules_parent ON auth.modules (parent_source_id);
