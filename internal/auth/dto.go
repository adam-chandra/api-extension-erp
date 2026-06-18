package auth

// DTOs validated by gin's binding/validator tags.

// LoginRequest accepts either the user's login (e.g. "admin", "lisa") OR email
// in the `email` field. The name is kept as `email` to match the FE form;
// the server still resolves both shapes via FindByLoginOrEmail.
type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// GeneratePasswordRequest is the admin-side flow: provide login + email, the
// server looks the user up in dberphm.auth.users and (if found) generates a
// fresh password, stores its bcrypt hash, and returns the plaintext once.
type GeneratePasswordRequest struct {
	Login string `json:"login" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

type GeneratedPasswordResponse struct {
	UserID   int64  `json:"userId"`
	Login    string `json:"login"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

// UserDTO is the authenticated user payload returned by /auth/me and login flows.
type UserDTO struct {
	ID        int64            `json:"id"`
	SourceID  int64            `json:"sourceId"`
	Login     string           `json:"login"`
	Email     string           `json:"email,omitempty"`
	Name      string           `json:"name"`
	IsActive  bool             `json:"isActive"`
	Companies []UserCompanyRow `json:"companies,omitempty"`
	Modules   []UserModuleRow  `json:"modules,omitempty"`
}

type AuthResponse struct {
	AccessToken  string  `json:"accessToken"`
	RefreshToken string  `json:"refreshToken,omitempty"`
	User         UserDTO `json:"user"`
}
