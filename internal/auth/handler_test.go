package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/extension-erp/be-extension-erp/internal/auth"
	"github.com/extension-erp/be-extension-erp/internal/config"
	"github.com/extension-erp/be-extension-erp/internal/user"
	"github.com/extension-erp/be-extension-erp/pkg/hash"
	jwtpkg "github.com/extension-erp/be-extension-erp/pkg/jwt"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// newHandlerWithRepo creates a real Handler backed by a mock repo and no cache.
// Suitable for handlers that do not call Me or Logout (which require Redis).
func newHandlerWithRepo(repo auth.Repository) *auth.Handler {
	jm := jwtpkg.New(config.JWTConfig{
		Secret:     "handler-test-secret",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
	})
	svc := auth.NewService(repo, jm, nil)
	return auth.NewHandler(svc)
}

func jsonBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return bytes.NewBuffer(b)
}

func performHandlerRequest(r http.Handler, method, path string, body *bytes.Buffer) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, body)
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ---------------------------------------------------------------------------
// GeneratePassword handler
// ---------------------------------------------------------------------------

func TestGeneratePasswordHandler_InvalidBody(t *testing.T) {
	h := newHandlerWithRepo(&mockAuthRepo{})
	r := gin.New()
	r.POST("/generate-password", h.GeneratePassword)

	w := performHandlerRequest(r, http.MethodPost, "/generate-password", bytes.NewBufferString("{bad-json}"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("GeneratePassword bad body: status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestGeneratePasswordHandler_UserNotFound(t *testing.T) {
	repo := &mockAuthRepo{
		findByLoginOrEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return nil, auth.ErrNotFound
		},
	}
	h := newHandlerWithRepo(repo)
	r := gin.New()
	r.POST("/generate-password", h.GeneratePassword)

	body := jsonBody(t, auth.GeneratePasswordRequest{Login: "ghost", Email: "ghost@example.com"})
	w := performHandlerRequest(r, http.MethodPost, "/generate-password", body)
	if w.Code != http.StatusNotFound {
		t.Errorf("GeneratePassword not found: status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGeneratePasswordHandler_Success(t *testing.T) {
	u := &user.User{ID: 1, SourceID: 10, Login: "admin", Name: "Admin"}
	emailVal := "admin@example.com"
	u.Email = &emailVal
	repo := &mockAuthRepo{
		findByLoginOrEmailFn: func(_ context.Context, _ string) (*user.User, error) { return u, nil },
		updatePasswordFn:     func(_ context.Context, _ int64, _ string) error { return nil },
	}
	h := newHandlerWithRepo(repo)
	r := gin.New()
	r.POST("/generate-password", h.GeneratePassword)

	body := jsonBody(t, auth.GeneratePasswordRequest{Login: "admin", Email: "admin@example.com"})
	w := performHandlerRequest(r, http.MethodPost, "/generate-password", body)
	if w.Code != http.StatusCreated {
		t.Errorf("GeneratePassword success: status = %d, want %d", w.Code, http.StatusCreated)
	}
}

// ---------------------------------------------------------------------------
// Login handler
// ---------------------------------------------------------------------------

func TestLoginHandler_InvalidBody(t *testing.T) {
	h := newHandlerWithRepo(&mockAuthRepo{})
	r := gin.New()
	r.POST("/login", h.Login)

	w := performHandlerRequest(r, http.MethodPost, "/login", bytes.NewBufferString("not-json"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("Login bad body: status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestLoginHandler_InvalidCredentials(t *testing.T) {
	repo := &mockAuthRepo{
		findByLoginOrEmailFn: func(_ context.Context, _ string) (*user.User, error) {
			return nil, auth.ErrNotFound
		},
	}
	h := newHandlerWithRepo(repo)
	r := gin.New()
	r.POST("/login", h.Login)

	body := jsonBody(t, auth.LoginRequest{Email: "ghost", Password: "pass"})
	w := performHandlerRequest(r, http.MethodPost, "/login", body)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Login invalid creds: status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestLoginHandler_PasswordNotSet(t *testing.T) {
	nilHash := ""
	u := &user.User{ID: 1, Login: "user", IsActive: true, PasswordHash: &nilHash}
	repo := &mockAuthRepo{
		findByLoginOrEmailFn: func(_ context.Context, _ string) (*user.User, error) { return u, nil },
	}
	h := newHandlerWithRepo(repo)
	r := gin.New()
	r.POST("/login", h.Login)

	body := jsonBody(t, auth.LoginRequest{Email: "user", Password: "pass"})
	w := performHandlerRequest(r, http.MethodPost, "/login", body)
	if w.Code != http.StatusForbidden {
		t.Errorf("Login password not set: status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestLoginHandler_Success(t *testing.T) {
	plain := "secure-pass"
	hashed, _ := hash.Password(plain)
	emailVal := "user@example.com"
	u := &user.User{ID: 5, SourceID: 50, Login: "user", Email: &emailVal, IsActive: true, PasswordHash: &hashed}
	repo := &mockAuthRepo{
		findByLoginOrEmailFn: func(_ context.Context, _ string) (*user.User, error) { return u, nil },
		userCompaniesFn: func(_ context.Context, _ int64) ([]auth.UserCompanyRow, error) {
			return []auth.UserCompanyRow{}, nil
		},
		userModulesFn: func(_ context.Context, _ int64) ([]auth.UserModuleRow, error) {
			return []auth.UserModuleRow{}, nil
		},
	}
	h := newHandlerWithRepo(repo)
	r := gin.New()
	r.POST("/login", h.Login)

	body := jsonBody(t, auth.LoginRequest{Email: "user", Password: plain})
	w := performHandlerRequest(r, http.MethodPost, "/login", body)
	if w.Code != http.StatusOK {
		t.Errorf("Login success: status = %d, want %d", w.Code, http.StatusOK)
	}
}

// ---------------------------------------------------------------------------
// Refresh handler
// ---------------------------------------------------------------------------

func TestRefreshHandler_InvalidBody(t *testing.T) {
	h := newHandlerWithRepo(&mockAuthRepo{})
	r := gin.New()
	r.POST("/refresh", h.Refresh)

	w := performHandlerRequest(r, http.MethodPost, "/refresh", bytes.NewBufferString("!!!"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("Refresh bad body: status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRefreshHandler_InvalidToken(t *testing.T) {
	h := newHandlerWithRepo(&mockAuthRepo{})
	r := gin.New()
	r.POST("/refresh", h.Refresh)

	body := jsonBody(t, auth.RefreshRequest{RefreshToken: "not-a-valid-jwt"})
	w := performHandlerRequest(r, http.MethodPost, "/refresh", body)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Refresh invalid token: status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// ---------------------------------------------------------------------------
// Me handler
// ---------------------------------------------------------------------------

func TestMeHandler_NoUserID(t *testing.T) {
	h := newHandlerWithRepo(&mockAuthRepo{})
	r := gin.New()
	r.GET("/me", h.Me)

	// No Auth middleware → UserID context key is absent
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Me no uid: status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// ---------------------------------------------------------------------------
// Menus handler
// ---------------------------------------------------------------------------

func TestMenusHandler_NoUserID(t *testing.T) {
	h := newHandlerWithRepo(&mockAuthRepo{})
	r := gin.New()
	r.GET("/menus", h.Menus)

	req := httptest.NewRequest(http.MethodGet, "/menus", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Menus no uid: status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}
