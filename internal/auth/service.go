package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/extension-erp/be-extension-erp/internal/user"
	"github.com/extension-erp/be-extension-erp/pkg/cache"
	"github.com/extension-erp/be-extension-erp/pkg/hash"
	"github.com/extension-erp/be-extension-erp/pkg/jwt"
)

// Domain errors. Handler maps these to HTTP status codes.
var (
	// ErrUserNotFound: identifier did not match any active user in dberphm.auth.users.
	// (BE has no way to create users — they come from the ERP sync.)
	ErrUserNotFound       = errors.New("auth: user not found")
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
	// ErrPasswordNotSet: user exists but admin has not generated a password yet.
	ErrPasswordNotSet = errors.New("auth: password not set, ask admin to generate one")
)

type Service struct {
	repo   Repository
	jwt    *jwt.Manager
	cache  *cache.Cache
	userTT time.Duration
}

func NewService(repo Repository, jwtMgr *jwt.Manager, c *cache.Cache) *Service {
	return &Service{
		repo:   repo,
		jwt:    jwtMgr,
		cache:  c,
		userTT: 5 * time.Minute,
	}
}

// GeneratePassword is the admin-side flow. The caller provides `login` and
// `email`; if EITHER matches an active row in dberphm.auth.users, a fresh
// password is generated, stored as bcrypt hash, and returned in plaintext
// (the only time it is ever visible). The response always echoes the actual
// login + email from the database — so a typo in one of the inputs is still
// resolvable as long as the other is correct. Calling it again rotates the
// password.
func (s *Service) GeneratePassword(ctx context.Context, req GeneratePasswordRequest) (*GeneratedPasswordResponse, error) {
	// Try login first (exact match), then email (case-insensitive).
	u, err := s.repo.FindByLoginOrEmail(ctx, req.Login)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return nil, err
		}
		u, err = s.repo.FindByLoginOrEmail(ctx, req.Email)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, ErrUserNotFound
			}
			return nil, err
		}
	}

	pass, err := generatePassword(12)
	if err != nil {
		return nil, err
	}
	hashed, err := hash.Password(pass)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdatePassword(ctx, u.ID, hashed); err != nil {
		return nil, err
	}

	email := ""
	if u.Email != nil {
		email = *u.Email
	}
	return &GeneratedPasswordResponse{
		UserID:   u.ID,
		Login:    u.Login,
		Email:    email,
		Name:     u.Name,
		Password: pass,
	}, nil
}

func generatePassword(n int) (string, error) {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	max := big.NewInt(int64(len(letters)))
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = letters[num.Int64()]
	}
	return string(b), nil
}

// Login verifies credentials. The `email` field may carry either the user's
// login or their email — both are tried via FindByLoginOrEmail.
func (s *Service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	u, err := s.repo.FindByLoginOrEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if u.PasswordHash == nil || *u.PasswordHash == "" {
		return nil, ErrPasswordNotSet
	}
	if !hash.Compare(*u.PasswordHash, req.Password) {
		return nil, ErrInvalidCredentials
	}
	return s.issue(ctx, u)
}

// Refresh validates a refresh token and issues new tokens.
func (s *Service) Refresh(ctx context.Context, refresh string) (*AuthResponse, error) {
	claims, err := s.jwt.Parse(refresh)
	if err != nil || claims.Type != "refresh" {
		return nil, ErrInvalidCredentials
	}
	id, err := strconv.ParseInt(claims.UserID, 10, 64)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if !u.IsActive {
		return nil, ErrInvalidCredentials
	}
	return s.issue(ctx, u)
}

// Me returns the cached or freshly-loaded user (with companies + modules).
func (s *Service) Me(ctx context.Context, userID string) (*UserDTO, error) {
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	key := fmt.Sprintf("user:%d", id)
	dto, err := cache.Remember(ctx, s.cache, key, s.userTT, func(ctx context.Context) (UserDTO, error) {
		u, err := s.repo.FindByID(ctx, id)
		if err != nil {
			return UserDTO{}, err
		}
		return s.buildDTO(ctx, u)
	})
	if err != nil {
		return nil, err
	}
	return &dto, nil
}

// Logout invalidates the user cache. Token revocation requires a denylist
// (e.g. Redis SET with TTL = remaining expiry) — add when needed.
func (s *Service) Logout(ctx context.Context, userID string) error {
	return s.cache.Delete(ctx, fmt.Sprintf("user:%s", userID))
}

// Menus returns the flat list of menus visible to the user.
func (s *Service) Menus(ctx context.Context, userID string) ([]UserMenuRow, error) {
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.repo.UserMenus(ctx, u.SourceID)
}

func (s *Service) issue(ctx context.Context, u *user.User) (*AuthResponse, error) {
	idStr := strconv.FormatInt(u.ID, 10)
	loginOrEmail := u.Login
	if u.Email != nil {
		loginOrEmail = *u.Email
	}
	access, _, err := s.jwt.IssueAccess(idStr, loginOrEmail)
	if err != nil {
		return nil, err
	}
	refresh, _, err := s.jwt.IssueRefresh(idStr, loginOrEmail)
	if err != nil {
		return nil, err
	}
	dto, err := s.buildDTO(ctx, u)
	if err != nil {
		return nil, err
	}
	return &AuthResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		User:         dto,
	}, nil
}

func (s *Service) buildDTO(ctx context.Context, u *user.User) (UserDTO, error) {
	email := ""
	if u.Email != nil {
		email = *u.Email
	}
	dto := UserDTO{
		ID:       u.ID,
		SourceID: u.SourceID,
		Login:    u.Login,
		Email:    email,
		Name:     u.Name,
		IsActive: u.IsActive,
	}
	companies, err := s.repo.UserCompanies(ctx, u.SourceID)
	if err != nil {
		return dto, fmt.Errorf("load companies: %w", err)
	}
	dto.Companies = companies
	modules, err := s.repo.UserModules(ctx, u.SourceID)
	if err != nil {
		return dto, fmt.Errorf("load modules: %w", err)
	}
	dto.Modules = modules
	return dto, nil
}
