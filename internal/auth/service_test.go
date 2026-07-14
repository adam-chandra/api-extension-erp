package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/extension-erp/be-extension-erp/internal/auth"
	"github.com/extension-erp/be-extension-erp/internal/config"
	"github.com/extension-erp/be-extension-erp/internal/user"
	"github.com/extension-erp/be-extension-erp/pkg/hash"
	jwtpkg "github.com/extension-erp/be-extension-erp/pkg/jwt"
)

// ---------------------------------------------------------------------------
// Mock repository
// ---------------------------------------------------------------------------

type mockAuthRepo struct {
	findByLoginOrEmailFn func(ctx context.Context, identifier string) (*user.User, error)
	findByIDFn           func(ctx context.Context, id int64) (*user.User, error)
	updatePasswordFn     func(ctx context.Context, id int64, h string) error
	userCompaniesFn      func(ctx context.Context, sourceID int64) ([]auth.UserCompanyRow, error)
	userModulesFn        func(ctx context.Context, sourceID int64) ([]auth.UserModuleRow, error)
	userMenusFn          func(ctx context.Context, sourceID int64) ([]auth.UserMenuRow, error)
}

func (m *mockAuthRepo) FindByLoginOrEmail(ctx context.Context, id string) (*user.User, error) {
	if m.findByLoginOrEmailFn != nil {
		return m.findByLoginOrEmailFn(ctx, id)
	}
	return nil, auth.ErrNotFound
}

func (m *mockAuthRepo) FindByID(ctx context.Context, id int64) (*user.User, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, auth.ErrNotFound
}

func (m *mockAuthRepo) UpdatePassword(ctx context.Context, id int64, h string) error {
	if m.updatePasswordFn != nil {
		return m.updatePasswordFn(ctx, id, h)
	}
	return nil
}

func (m *mockAuthRepo) UserCompanies(ctx context.Context, sourceID int64) ([]auth.UserCompanyRow, error) {
	if m.userCompaniesFn != nil {
		return m.userCompaniesFn(ctx, sourceID)
	}
	return []auth.UserCompanyRow{}, nil
}

func (m *mockAuthRepo) UserModules(ctx context.Context, sourceID int64) ([]auth.UserModuleRow, error) {
	if m.userModulesFn != nil {
		return m.userModulesFn(ctx, sourceID)
	}
	return []auth.UserModuleRow{}, nil
}

func (m *mockAuthRepo) UserMenus(ctx context.Context, sourceID int64) ([]auth.UserMenuRow, error) {
	if m.userMenusFn != nil {
		return m.userMenusFn(ctx, sourceID)
	}
	return []auth.UserMenuRow{}, nil
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

// newTestService creates a Service with the given repo and no cache.
// Cache-dependent methods (Me, Logout) are NOT tested here to avoid needing Redis.
func newTestService(repo auth.Repository) *auth.Service {
	jm := jwtpkg.New(config.JWTConfig{
		Secret:     "auth-test-secret",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
	})
	return auth.NewService(repo, jm, nil)
}

func makeUser(id int64, login, email, passwordHash string) *user.User {
	e := email
	h := passwordHash
	return &user.User{
		ID:           id,
		SourceID:     id * 10,
		Login:        login,
		Email:        &e,
		Name:         "Test User",
		IsActive:     true,
		PasswordHash: &h,
	}
}

// ---------------------------------------------------------------------------
// Service.Login
// ---------------------------------------------------------------------------

func TestLogin_Success(t *testing.T) {
	plain := "correct-password"
	hashed, _ := hash.Password(plain)
	u := makeUser(1, "testuser", "user@example.com", hashed)

	repo := &mockAuthRepo{
		findByLoginOrEmailFn: func(_ context.Context, _ string) (*user.User, error) { return u, nil },
	}
	svc := newTestService(repo)

	resp, err := svc.Login(context.Background(), auth.LoginRequest{
		Email: "testuser", Password: plain,
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("Login() AccessToken should not be empty")
	}
	if resp.RefreshToken == "" {
		t.Error("Login() RefreshToken should not be empty")
	}
	if resp.User.ID != 1 {
		t.Errorf("Login() User.ID = %d, want 1", resp.User.ID)
	}
	if resp.User.Email != "user@example.com" {
		t.Errorf("Login() User.Email = %q, want user@example.com", resp.User.Email)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	repo := &mockAuthRepo{
		findByLoginOrEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return nil, auth.ErrNotFound
		},
	}
	svc := newTestService(repo)

	_, err := svc.Login(context.Background(), auth.LoginRequest{Email: "noone", Password: "pass"})
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("Login() not found: error = %v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_RepoUnexpectedError(t *testing.T) {
	repo := &mockAuthRepo{
		findByLoginOrEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return nil, errors.New("db timeout")
		},
	}
	svc := newTestService(repo)

	_, err := svc.Login(context.Background(), auth.LoginRequest{Email: "user", Password: "pass"})
	if err == nil {
		t.Error("Login() unexpected repo error should be propagated")
	}
}

func TestLogin_NilPasswordHash(t *testing.T) {
	u := &user.User{ID: 1, Login: "user", IsActive: true, PasswordHash: nil}
	repo := &mockAuthRepo{
		findByLoginOrEmailFn: func(_ context.Context, _ string) (*user.User, error) { return u, nil },
	}
	svc := newTestService(repo)

	_, err := svc.Login(context.Background(), auth.LoginRequest{Email: "user", Password: "pass"})
	if !errors.Is(err, auth.ErrPasswordNotSet) {
		t.Errorf("Login() nil hash: error = %v, want ErrPasswordNotSet", err)
	}
}

func TestLogin_EmptyPasswordHash(t *testing.T) {
	empty := ""
	u := &user.User{ID: 1, Login: "user", IsActive: true, PasswordHash: &empty}
	repo := &mockAuthRepo{
		findByLoginOrEmailFn: func(_ context.Context, _ string) (*user.User, error) { return u, nil },
	}
	svc := newTestService(repo)

	_, err := svc.Login(context.Background(), auth.LoginRequest{Email: "user", Password: "pass"})
	if !errors.Is(err, auth.ErrPasswordNotSet) {
		t.Errorf("Login() empty hash: error = %v, want ErrPasswordNotSet", err)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	hashed, _ := hash.Password("correct-pass")
	u := makeUser(1, "user", "user@example.com", hashed)
	repo := &mockAuthRepo{
		findByLoginOrEmailFn: func(_ context.Context, _ string) (*user.User, error) { return u, nil },
	}
	svc := newTestService(repo)

	_, err := svc.Login(context.Background(), auth.LoginRequest{Email: "user", Password: "wrong-pass"})
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("Login() wrong password: error = %v, want ErrInvalidCredentials", err)
	}
}

// ---------------------------------------------------------------------------
// Service.GeneratePassword
// ---------------------------------------------------------------------------

func TestGeneratePassword_Success_ByLogin(t *testing.T) {
	u := makeUser(5, "adminuser", "admin@example.com", "")
	repo := &mockAuthRepo{
		findByLoginOrEmailFn: func(_ context.Context, _ string) (*user.User, error) { return u, nil },
		updatePasswordFn:     func(_ context.Context, _ int64, _ string) error { return nil },
	}
	svc := newTestService(repo)

	resp, err := svc.GeneratePassword(context.Background(), auth.GeneratePasswordRequest{
		Login: "adminuser",
		Email: "admin@example.com",
	})
	if err != nil {
		t.Fatalf("GeneratePassword() error = %v", err)
	}
	if resp.Password == "" {
		t.Error("GeneratePassword() should return a non-empty plaintext password")
	}
	if resp.UserID != 5 {
		t.Errorf("GeneratePassword() UserID = %d, want 5", resp.UserID)
	}
	if len(resp.Password) < 12 {
		t.Errorf("GeneratePassword() password length = %d, want >= 12", len(resp.Password))
	}
}

func TestGeneratePassword_FallbackToEmail(t *testing.T) {
	u := makeUser(6, "someuser", "fallback@example.com", "")
	callCount := 0
	repo := &mockAuthRepo{
		findByLoginOrEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			callCount++
			if callCount == 1 {
				return nil, auth.ErrNotFound // login not found
			}
			return u, nil // email found
		},
		updatePasswordFn: func(_ context.Context, _ int64, _ string) error { return nil },
	}
	svc := newTestService(repo)

	resp, err := svc.GeneratePassword(context.Background(), auth.GeneratePasswordRequest{
		Login: "notexist",
		Email: "fallback@example.com",
	})
	if err != nil {
		t.Fatalf("GeneratePassword() fallback to email: error = %v", err)
	}
	if resp.Password == "" {
		t.Error("GeneratePassword() fallback: Password should not be empty")
	}
}

func TestGeneratePassword_UserNotFound(t *testing.T) {
	repo := &mockAuthRepo{
		findByLoginOrEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return nil, auth.ErrNotFound
		},
	}
	svc := newTestService(repo)

	_, err := svc.GeneratePassword(context.Background(), auth.GeneratePasswordRequest{
		Login: "ghost",
		Email: "ghost@example.com",
	})
	if !errors.Is(err, auth.ErrUserNotFound) {
		t.Errorf("GeneratePassword() not found: error = %v, want ErrUserNotFound", err)
	}
}

func TestGeneratePassword_UpdateError(t *testing.T) {
	u := makeUser(7, "user", "user@example.com", "")
	repo := &mockAuthRepo{
		findByLoginOrEmailFn: func(_ context.Context, _ string) (*user.User, error) { return u, nil },
		updatePasswordFn: func(_ context.Context, _ int64, _ string) error {
			return errors.New("db write error")
		},
	}
	svc := newTestService(repo)

	_, err := svc.GeneratePassword(context.Background(), auth.GeneratePasswordRequest{
		Login: "user",
		Email: "user@example.com",
	})
	if err == nil {
		t.Error("GeneratePassword() update error should be propagated")
	}
}

// ---------------------------------------------------------------------------
// Service.Refresh
// ---------------------------------------------------------------------------

func TestRefresh_Success(t *testing.T) {
	jm := jwtpkg.New(config.JWTConfig{
		Secret:     "auth-test-secret",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
	})
	refreshToken, _, _ := jm.IssueRefresh("10", "user@example.com")

	u := makeUser(10, "user10", "user@example.com", "hash")
	repo := &mockAuthRepo{
		findByIDFn: func(_ context.Context, _ int64) (*user.User, error) { return u, nil },
	}
	svc := newTestService(repo)

	resp, err := svc.Refresh(context.Background(), refreshToken)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("Refresh() AccessToken should not be empty")
	}
}

func TestRefresh_InvalidToken(t *testing.T) {
	svc := newTestService(&mockAuthRepo{})
	_, err := svc.Refresh(context.Background(), "not-a-token")
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("Refresh() invalid token: error = %v, want ErrInvalidCredentials", err)
	}
}

func TestRefresh_AccessTokenRejected(t *testing.T) {
	jm := jwtpkg.New(config.JWTConfig{
		Secret:    "auth-test-secret",
		AccessTTL: 15 * time.Minute,
	})
	accessToken, _, _ := jm.IssueAccess("1", "user@example.com")
	svc := newTestService(&mockAuthRepo{})

	_, err := svc.Refresh(context.Background(), accessToken)
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("Refresh() access token: error = %v, want ErrInvalidCredentials", err)
	}
}

func TestRefresh_UserNotFound(t *testing.T) {
	jm := jwtpkg.New(config.JWTConfig{
		Secret:     "auth-test-secret",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
	})
	token, _, _ := jm.IssueRefresh("99", "ghost@example.com")

	repo := &mockAuthRepo{
		findByIDFn: func(_ context.Context, _ int64) (*user.User, error) {
			return nil, auth.ErrNotFound
		},
	}
	svc := newTestService(repo)

	_, err := svc.Refresh(context.Background(), token)
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("Refresh() user not found: error = %v, want ErrInvalidCredentials", err)
	}
}

func TestRefresh_InactiveUser(t *testing.T) {
	jm := jwtpkg.New(config.JWTConfig{
		Secret:     "auth-test-secret",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
	})
	token, _, _ := jm.IssueRefresh("20", "inactive@example.com")

	u := &user.User{ID: 20, Login: "inactive", IsActive: false}
	repo := &mockAuthRepo{
		findByIDFn: func(_ context.Context, _ int64) (*user.User, error) { return u, nil },
	}
	svc := newTestService(repo)

	_, err := svc.Refresh(context.Background(), token)
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("Refresh() inactive user: error = %v, want ErrInvalidCredentials", err)
	}
}

// ---------------------------------------------------------------------------
// Service.Menus
// ---------------------------------------------------------------------------

func TestMenus_InvalidUserID(t *testing.T) {
	svc := newTestService(&mockAuthRepo{})
	_, err := svc.Menus(context.Background(), "not-a-number")
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("Menus() invalid id: error = %v, want ErrInvalidCredentials", err)
	}
}

func TestMenus_UserNotFound(t *testing.T) {
	repo := &mockAuthRepo{
		findByIDFn: func(_ context.Context, _ int64) (*user.User, error) {
			return nil, auth.ErrNotFound
		},
	}
	svc := newTestService(repo)
	_, err := svc.Menus(context.Background(), "42")
	if err == nil {
		t.Error("Menus() should return error when user not found")
	}
}

func TestMenus_Success(t *testing.T) {
	u := makeUser(42, "user42", "u@example.com", "hash")
	repo := &mockAuthRepo{
		findByIDFn: func(_ context.Context, _ int64) (*user.User, error) { return u, nil },
		userMenusFn: func(_ context.Context, _ int64) ([]auth.UserMenuRow, error) {
			return []auth.UserMenuRow{{MenuID: 1, MenuName: "Dashboard"}}, nil
		},
	}
	svc := newTestService(repo)

	menus, err := svc.Menus(context.Background(), "42")
	if err != nil {
		t.Fatalf("Menus() error = %v", err)
	}
	if len(menus) != 1 {
		t.Errorf("Menus() len = %d, want 1", len(menus))
	}
}
