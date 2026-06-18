-- =============================================================================
--  0008 — Menus (source: ir_ui_menu + ir_ui_menu_group_rel)
--  Menu hirarkis Odoo + relasi menu↔role. Menu yang dilihat user adalah
--  hasil join dengan v_user_effective_roles via menu_roles.
-- =============================================================================

CREATE TABLE IF NOT EXISTS auth.menus (
    id                  BIGSERIAL   PRIMARY KEY,
    source_id           INTEGER     NOT NULL UNIQUE,
    parent_source_id    INTEGER,
    name                TEXT        NOT NULL,
    sequence            INTEGER,
    action              TEXT,       -- e.g. 'ir.actions.act_window,320' (raw Odoo ref)
    web_icon            TEXT,
    is_active           BOOLEAN     NOT NULL DEFAULT TRUE,
    source_created_at   TIMESTAMP,
    source_updated_at   TIMESTAMP,
    synced_at           TIMESTAMP   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_menus_parent ON auth.menus (parent_source_id);
CREATE INDEX IF NOT EXISTS idx_menus_active ON auth.menus (is_active);

-- Menu yang TIDAK punya baris di sini = public (Odoo behaviour: menu tanpa
-- group_ids dapat diakses semua user). Lihat v_user_menus untuk filter final.
CREATE TABLE IF NOT EXISTS auth.menu_roles (
    menu_source_id  INTEGER NOT NULL,
    role_source_id  INTEGER NOT NULL,
    PRIMARY KEY (menu_source_id, role_source_id)
);

CREATE INDEX IF NOT EXISTS idx_menu_roles_role ON auth.menu_roles (role_source_id);
