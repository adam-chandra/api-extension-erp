-- Reverse 0016.

CREATE TABLE IF NOT EXISTS finance.asset_equipment (
    source_id           INTEGER         PRIMARY KEY,
    company_source_id   INTEGER         NOT NULL,
    name                TEXT            NOT NULL,
    status_raw          TEXT,
    status_normalized   TEXT            NOT NULL,
    employee_heldby     TEXT,
    asset_value         NUMERIC(20,2)   NOT NULL DEFAULT 0,
    synced_at           TIMESTAMP       NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_asset_eq_company ON finance.asset_equipment (company_source_id);
CREATE INDEX IF NOT EXISTS idx_asset_eq_status  ON finance.asset_equipment (status_normalized);
CREATE INDEX IF NOT EXISTS idx_asset_eq_sync    ON finance.asset_equipment (synced_at DESC);

DO $$
BEGIN
    IF to_regclass('asset.asset_equipment') IS NOT NULL THEN
        INSERT INTO finance.asset_equipment (
            source_id, company_source_id, name, status_raw,
            status_normalized, employee_heldby, asset_value, synced_at
        )
        SELECT
            source_id, company_source_id, name, status_raw,
            status_normalized, employee_heldby, asset_value, synced_at
        FROM asset.asset_equipment
        ON CONFLICT (source_id) DO UPDATE SET
            company_source_id = EXCLUDED.company_source_id,
            name              = EXCLUDED.name,
            status_raw        = EXCLUDED.status_raw,
            status_normalized = EXCLUDED.status_normalized,
            employee_heldby   = EXCLUDED.employee_heldby,
            asset_value       = EXCLUDED.asset_value,
            synced_at         = EXCLUDED.synced_at;
    END IF;

    DROP TABLE IF EXISTS asset.asset_equipment;
    DROP SCHEMA IF EXISTS asset;

    UPDATE sync.sync_state
    SET table_name = 'finance.asset_equipment', updated_at = NOW()
    WHERE table_name = 'asset.asset_equipment';
END $$;
