-- =============================================================================
--  0009 — v_user_menus
--  Menu efektif per user. Aturan Odoo:
--   - Menu tanpa baris di menu_roles → public (siapa pun yang aktif boleh)
--   - Menu dengan baris di menu_roles → user harus punya minimal satu role
--     (direct atau inherited via role_inheritance) yang cocok.
-- =============================================================================

CREATE OR REPLACE VIEW auth.v_user_menus AS
WITH user_menu AS (
    -- Menu publik (tidak ada role-restriction)
    SELECT u.source_id AS user_source_id, m.source_id AS menu_source_id
    FROM auth.users u
    CROSS JOIN auth.menus m
    WHERE u.is_active = TRUE
      AND m.is_active = TRUE
      AND NOT EXISTS (
          SELECT 1 FROM auth.menu_roles mr WHERE mr.menu_source_id = m.source_id
      )
    UNION
    -- Menu yang user punya role-nya (langsung / inherited)
    SELECT er.user_source_id, mr.menu_source_id
    FROM auth.v_user_effective_roles er
    JOIN auth.menu_roles mr ON mr.role_source_id = er.role_source_id
    JOIN auth.menus      m  ON m.source_id       = mr.menu_source_id
    WHERE m.is_active = TRUE
)
SELECT DISTINCT
    um.user_source_id,
    m.id                    AS menu_id,
    m.source_id             AS menu_source_id,
    m.parent_source_id      AS parent_source_id,
    m.name                  AS menu_name,
    m.sequence,
    m.action,
    m.web_icon
FROM user_menu um
JOIN auth.menus m ON m.source_id = um.menu_source_id;
