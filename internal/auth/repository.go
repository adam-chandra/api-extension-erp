package auth

import (
	"context"
	"errors"
	"time"

	"github.com/extension-erp/be-extension-erp/internal/user"
	"gorm.io/gorm"
)

// Repository is the persistence boundary for auth/user operations.
// Backed by dberphm.auth.* (worker-erp owns the schema).
type Repository interface {
	// FindByLoginOrEmail looks up an active user whose `login` OR `email`
	// matches the identifier (case-insensitive on email).
	FindByLoginOrEmail(ctx context.Context, identifier string) (*user.User, error)
	// FindByID looks up a user by primary key.
	FindByID(ctx context.Context, id int64) (*user.User, error)
	// UpdatePassword sets password_hash + password_generated_at on auth.users.
	UpdatePassword(ctx context.Context, id int64, hash string) error
	// Accessible companies & modules for a user (via views).
	UserCompanies(ctx context.Context, userSourceID int64) ([]UserCompanyRow, error)
	UserModules(ctx context.Context, userSourceID int64) ([]UserModuleRow, error)
	// UserMenus returns the flat list of menus visible to the user.
	// FE assembles the tree by walking parent_source_id.
	UserMenus(ctx context.Context, userSourceID int64) ([]UserMenuRow, error)
}

type UserCompanyRow struct {
	CompanyID       int64  `json:"id"`
	CompanySourceID int64  `json:"sourceId"`
	CompanyName     string `json:"name"`
	IsDefault       bool   `json:"isDefault"`
}

type UserModuleRow struct {
	ModuleID       int64  `json:"id"`
	ModuleSourceID int64  `json:"sourceId"`
	ModuleName     string `json:"name"`
}

type UserMenuRow struct {
	MenuID         int64   `json:"id"`
	MenuSourceID   int64   `json:"sourceId"`
	ParentSourceID *int64  `json:"parentSourceId,omitempty"`
	MenuName       string  `json:"name"`
	Sequence       *int32  `json:"sequence,omitempty"`
	Action         *string `json:"action,omitempty"`
	WebIcon        *string `json:"webIcon,omitempty"`
}

var ErrNotFound = errors.New("auth: not found")

type gormRepo struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &gormRepo{db: db}
}

func (r *gormRepo) FindByLoginOrEmail(ctx context.Context, identifier string) (*user.User, error) {
	if identifier == "" {
		return nil, ErrNotFound
	}
	var u user.User
	err := r.db.WithContext(ctx).
		Where("is_active = TRUE AND (login = ? OR LOWER(email) = LOWER(?))", identifier, identifier).
		First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *gormRepo) FindByID(ctx context.Context, id int64) (*user.User, error) {
	var u user.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *gormRepo) UpdatePassword(ctx context.Context, id int64, hash string) error {
	now := time.Now().UTC()
	res := r.db.WithContext(ctx).
		Table("auth.users").
		Where("id = ?", id).
		Updates(map[string]any{
			"password_hash":         hash,
			"password_generated_at": now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *gormRepo) UserCompanies(ctx context.Context, userSourceID int64) ([]UserCompanyRow, error) {
	var rows []UserCompanyRow
	err := r.db.WithContext(ctx).
		Raw(`SELECT company_id, company_source_id, company_name, is_default
		     FROM auth.v_user_companies WHERE user_source_id = ?
		     ORDER BY is_default DESC, company_name`, userSourceID).
		Scan(&rows).Error
	return rows, err
}

func (r *gormRepo) UserModules(ctx context.Context, userSourceID int64) ([]UserModuleRow, error) {
	// Inline the recursive role-inheritance walk filtered by user_source_id
	// at the seed. The plain view auth.v_user_modules does the same logic but
	// without predicate pushdown, forcing PG to materialize effective roles
	// for ALL users on every query.
	var rows []UserModuleRow
	err := r.db.WithContext(ctx).
		Raw(`WITH RECURSIVE eff(role_source_id) AS (
		         SELECT role_source_id
		         FROM auth.user_roles
		         WHERE user_source_id = ?
		         UNION
		         SELECT ri.child_role_source_id
		         FROM eff e
		         JOIN auth.role_inheritance ri
		           ON ri.parent_role_source_id = e.role_source_id
		     )
		     SELECT DISTINCT
		         m.id        AS module_id,
		         m.source_id AS module_source_id,
		         m.name      AS module_name
		     FROM eff e
		     JOIN auth.roles   r ON r.source_id = e.role_source_id
		     JOIN auth.modules m ON m.source_id = r.module_source_id
		     ORDER BY module_name`, userSourceID).
		Scan(&rows).Error
	return rows, err
}

func (r *gormRepo) UserMenus(ctx context.Context, userSourceID int64) ([]UserMenuRow, error) {
	// See UserModules for why we inline the CTE instead of using v_user_menus.
	var rows []UserMenuRow
	err := r.db.WithContext(ctx).
		Raw(`WITH RECURSIVE eff(role_source_id) AS (
		         SELECT role_source_id
		         FROM auth.user_roles
		         WHERE user_source_id = ?
		         UNION
		         SELECT ri.child_role_source_id
		         FROM eff e
		         JOIN auth.role_inheritance ri
		           ON ri.parent_role_source_id = e.role_source_id
		     ),
		     accessible_menus AS (
		         -- Public menus: no menu_roles row exists.
		         SELECT m.source_id
		         FROM auth.menus m
		         WHERE m.is_active = TRUE
		           AND NOT EXISTS (
		               SELECT 1 FROM auth.menu_roles mr
		               WHERE mr.menu_source_id = m.source_id
		           )
		         UNION
		         -- Menus user has at least one effective role for.
		         SELECT mr.menu_source_id
		         FROM eff e
		         JOIN auth.menu_roles mr ON mr.role_source_id = e.role_source_id
		         JOIN auth.menus      m  ON m.source_id      = mr.menu_source_id
		         WHERE m.is_active = TRUE
		     ),
		     menu_rows AS (
		         SELECT DISTINCT
		             m.id               AS menu_id,
		             m.source_id        AS menu_source_id,
		             m.parent_source_id AS parent_source_id,
		             m.name             AS menu_name,
		             m.sequence,
		             m.action,
		             m.web_icon
		         FROM accessible_menus am
		         JOIN auth.menus m ON m.source_id = am.source_id
		     )
		     SELECT *
		     FROM menu_rows
		     ORDER BY COALESCE(parent_source_id, 0), sequence NULLS LAST, menu_name`, userSourceID).
		Scan(&rows).Error
	return rows, err
}
