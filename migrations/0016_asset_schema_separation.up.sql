-- =============================================================================
--  0016 — Separate asset dashboard cache from finance schema
--  Move worker/API asset cache table from finance.* to asset.* to reflect
--  independent menu/module ownership.
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

DO $$
BEGIN
    IF to_regclass('finance.asset_equipment') IS NOT NULL THEN
        INSERT INTO asset.asset_equipment (
            source_id, company_source_id, name, status_raw,
            status_normalized, employee_heldby, asset_value, synced_at
        )
        SELECT
            source_id, company_source_id, name, status_raw,
            status_normalized, employee_heldby, asset_value, synced_at
        FROM finance.asset_equipment
        ON CONFLICT (source_id) DO UPDATE SET
            company_source_id = EXCLUDED.company_source_id,
            name              = EXCLUDED.name,
            status_raw        = EXCLUDED.status_raw,
            status_normalized = EXCLUDED.status_normalized,
            employee_heldby   = EXCLUDED.employee_heldby,
            asset_value       = EXCLUDED.asset_value,
            synced_at         = EXCLUDED.synced_at;

        DROP TABLE finance.asset_equipment;
    END IF;

    UPDATE sync.sync_state
    SET table_name = 'asset.asset_equipment', updated_at = NOW()
    WHERE table_name = 'finance.asset_equipment';
END $$;
