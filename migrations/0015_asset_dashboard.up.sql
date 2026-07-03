-- =============================================================================
--  0015 — Asset dashboard cache table
--  Worker mirrors Odoo maintenance_equipment into this table (full refresh).
--  API reads this table to serve asset dashboard tiles.
-- =============================================================================

CREATE SCHEMA IF NOT EXISTS asset;

CREATE TABLE IF NOT EXISTS asset.asset_equipment (
    source_id           INTEGER         PRIMARY KEY,
    company_source_id   INTEGER         NOT NULL,
    name                TEXT            NOT NULL,
    status_raw          TEXT,
    status_normalized   TEXT            NOT NULL,
    employee_heldby     TEXT,
    asset_value         NUMERIC(20,2)   NOT NULL DEFAULT 0,
    synced_at           TIMESTAMP       NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_asset_eq_company ON asset.asset_equipment (company_source_id);
CREATE INDEX IF NOT EXISTS idx_asset_eq_status  ON asset.asset_equipment (status_normalized);
CREATE INDEX IF NOT EXISTS idx_asset_eq_sync    ON asset.asset_equipment (synced_at DESC);
