-- =============================================================================
--  0001 — Bootstrap schemas + sync bookkeeping
--  Owner: worker-erp
--  Notes: dberphm hanya menerima data dari worker (insert/update). Kolom yang
--         dimanage BE (password_hash, password_generated_at) ada di auth.users.
-- =============================================================================

CREATE SCHEMA IF NOT EXISTS auth;
CREATE SCHEMA IF NOT EXISTS sync;

-- One row per synced table. Worker reads + writes here.
CREATE TABLE IF NOT EXISTS sync.sync_state (
    table_name           TEXT        PRIMARY KEY,
    last_watermark       TIMESTAMP,                -- max(write_date) processed
    last_run_at          TIMESTAMP,
    last_run_duration_ms INTEGER,
    rows_inserted        INTEGER     NOT NULL DEFAULT 0,
    rows_updated         INTEGER     NOT NULL DEFAULT 0,
    rows_failed          INTEGER     NOT NULL DEFAULT 0,
    status               TEXT        NOT NULL DEFAULT 'pending',
    last_error           TEXT,
    updated_at           TIMESTAMP   NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sync.sync_errors (
    id          BIGSERIAL PRIMARY KEY,
    table_name  TEXT        NOT NULL,
    source_id   BIGINT,
    error       TEXT        NOT NULL,
    created_at  TIMESTAMP   NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sync_errors_table
    ON sync.sync_errors (table_name, created_at DESC);
