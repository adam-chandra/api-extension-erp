package user

import "time"

// User is the authn/authz subject. The row lives in dberphm.auth.users and is
// kept up-to-date by worker-erp from the source Odoo ERP. The BE only writes
// PasswordHash and PasswordGeneratedAt; every other column is worker-owned.
type User struct {
	ID                     int64  `gorm:"primaryKey;column:id" json:"id"`
	SourceID               int64  `gorm:"column:source_id;uniqueIndex" json:"sourceId"`
	PartnerSourceID        *int64 `gorm:"column:partner_source_id" json:"partnerSourceId,omitempty"`
	DefaultCompanySourceID *int64 `gorm:"column:default_company_source_id" json:"defaultCompanySourceId,omitempty"`

	Login  string  `gorm:"column:login;uniqueIndex" json:"login"`
	Email  *string `gorm:"column:email" json:"email,omitempty"`
	Name   string  `gorm:"column:name" json:"name"`
	Phone  *string `gorm:"column:phone" json:"phone,omitempty"`
	Mobile *string `gorm:"column:mobile" json:"mobile,omitempty"`

	IsActive    bool       `gorm:"column:is_active" json:"isActive"`
	IsShare     bool       `gorm:"column:is_share" json:"isShare"`
	LastLoginAt *time.Time `gorm:"column:last_login_at" json:"lastLoginAt,omitempty"`

	PasswordHash        *string    `gorm:"column:password_hash" json:"-"`
	PasswordGeneratedAt *time.Time `gorm:"column:password_generated_at" json:"passwordGeneratedAt,omitempty"`

	SourceCreatedAt *time.Time `gorm:"column:source_created_at" json:"sourceCreatedAt,omitempty"`
	SourceUpdatedAt *time.Time `gorm:"column:source_updated_at" json:"sourceUpdatedAt,omitempty"`
	SyncedAt        time.Time  `gorm:"column:synced_at" json:"syncedAt"`
}

// TableName binds the model to the worker-managed schema.
func (User) TableName() string { return "auth.users" }
