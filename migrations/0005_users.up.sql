-- =============================================================================
--  0005 — Users (source: res_users JOIN res_partner)
--  password_hash & password_generated_at adalah kolom milik BE
--  (worker tidak menulis kolom-kolom tersebut setelah user ada di tabel).
-- =============================================================================

CREATE TABLE IF NOT EXISTS auth.users (
    id                          BIGSERIAL   PRIMARY KEY,
    source_id                   INTEGER     NOT NULL UNIQUE,        -- res_users.id
    partner_source_id           INTEGER,                            -- res_users.partner_id
    default_company_source_id   INTEGER,                            -- res_users.company_id

    login                       TEXT        NOT NULL UNIQUE,        -- res_users.login (boleh non-email)
    email                       TEXT,                               -- res_partner.email
    name                        TEXT        NOT NULL,               -- res_partner.name
    phone                       TEXT,
    mobile                      TEXT,

    is_active                   BOOLEAN     NOT NULL DEFAULT TRUE,
    is_share                    BOOLEAN     NOT NULL DEFAULT FALSE, -- TRUE = portal/external
    last_login_at               TIMESTAMP,

    -- ---- App-managed (BE only writes these) -------------------------------
    password_hash               TEXT,                               -- bcrypt
    password_generated_at       TIMESTAMP,
    -- -----------------------------------------------------------------------

    source_created_at           TIMESTAMP,
    source_updated_at           TIMESTAMP,
    synced_at                   TIMESTAMP   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email   ON auth.users (LOWER(email));
CREATE INDEX IF NOT EXISTS idx_users_company ON auth.users (default_company_source_id);
CREATE INDEX IF NOT EXISTS idx_users_active  ON auth.users (is_active);
