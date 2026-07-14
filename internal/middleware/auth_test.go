package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/extension-erp/be-extension-erp/internal/config"
	"github.com/extension-erp/be-extension-erp/internal/middleware"
	jwtpkg "github.com/extension-erp/be-extension-erp/pkg/jwt"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newJWTManager() *jwtpkg.Manager {
	return jwtpkg.New(config.JWTConfig{
		Secret:     "middleware-test-secret",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 24 * time.Hour,
	})
}

func doRequest(r http.Handler, method, path, authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAuth_NoAuthorizationHeader(t *testing.T) {
	jm := newJWTManager()
	r := gin.New()
	r.GET("/", middleware.Auth(jm), func(c *gin.Context) { c.Status(http.StatusOK) })

	w := doRequest(r, http.MethodGet, "/", "")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Auth() no header: status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuth_WrongPrefix(t *testing.T) {
	jm := newJWTManager()
	r := gin.New()
	r.GET("/", middleware.Auth(jm), func(c *gin.Context) { c.Status(http.StatusOK) })

	w := doRequest(r, http.MethodGet, "/", "Token abc123")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Auth() wrong prefix: status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	jm := newJWTManager()
	r := gin.New()
	r.GET("/", middleware.Auth(jm), func(c *gin.Context) { c.Status(http.StatusOK) })

	w := doRequest(r, http.MethodGet, "/", "Bearer not.a.valid.jwt")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Auth() invalid token: status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuth_RefreshTokenRejected(t *testing.T) {
	jm := newJWTManager()
	refreshToken, _, _ := jm.IssueRefresh("1", "user@example.com")
	r := gin.New()
	r.GET("/", middleware.Auth(jm), func(c *gin.Context) { c.Status(http.StatusOK) })

	w := doRequest(r, http.MethodGet, "/", "Bearer "+refreshToken)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Auth() refresh token: status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuth_ValidAccessToken_SetsContext(t *testing.T) {
	jm := newJWTManager()
	accessToken, _, _ := jm.IssueAccess("42", "user@example.com")

	var gotUserID, gotEmail string
	r := gin.New()
	r.GET("/", middleware.Auth(jm), func(c *gin.Context) {
		gotUserID = middleware.UserID(c)
		gotEmail = middleware.Email(c)
		c.Status(http.StatusOK)
	})

	w := doRequest(r, http.MethodGet, "/", "Bearer "+accessToken)
	if w.Code != http.StatusOK {
		t.Errorf("Auth() valid token: status = %d, want %d", w.Code, http.StatusOK)
	}
	if gotUserID != "42" {
		t.Errorf("UserID() = %q, want 42", gotUserID)
	}
	if gotEmail != "user@example.com" {
		t.Errorf("Email() = %q, want user@example.com", gotEmail)
	}
}

func TestUserID_EmptyWhenNotSet(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	if id := middleware.UserID(c); id != "" {
		t.Errorf("UserID() without middleware = %q, want empty", id)
	}
}

func TestEmail_EmptyWhenNotSet(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	if email := middleware.Email(c); email != "" {
		t.Errorf("Email() without middleware = %q, want empty", email)
	}
}
