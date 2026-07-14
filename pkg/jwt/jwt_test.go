package jwt_test

import (
	"testing"
	"time"

	"github.com/extension-erp/be-extension-erp/internal/config"
	"github.com/extension-erp/be-extension-erp/pkg/jwt"
)

func newTestManager() *jwt.Manager {
	return jwt.New(config.JWTConfig{
		Secret:     "test-secret-key-for-unit-tests",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
	})
}

func TestIssueAccess_ReturnsToken(t *testing.T) {
	m := newTestManager()
	token, exp, err := m.IssueAccess("123", "user@example.com")
	if err != nil {
		t.Fatalf("IssueAccess() error = %v", err)
	}
	if token == "" {
		t.Error("IssueAccess() returned empty token")
	}
	if !exp.After(time.Now()) {
		t.Error("IssueAccess() expiry should be in the future")
	}
}

func TestIssueRefresh_ReturnsToken(t *testing.T) {
	m := newTestManager()
	token, exp, err := m.IssueRefresh("456", "user@example.com")
	if err != nil {
		t.Fatalf("IssueRefresh() error = %v", err)
	}
	if token == "" {
		t.Error("IssueRefresh() returned empty token")
	}
	if !exp.After(time.Now()) {
		t.Error("IssueRefresh() expiry should be in the future")
	}
}

func TestParse_ValidAccessToken(t *testing.T) {
	m := newTestManager()
	token, _, _ := m.IssueAccess("789", "test@example.com")

	claims, err := m.Parse(token)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if claims.UserID != "789" {
		t.Errorf("Parse() UserID = %q, want 789", claims.UserID)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("Parse() Email = %q, want test@example.com", claims.Email)
	}
	if claims.Type != "access" {
		t.Errorf("Parse() Type = %q, want access", claims.Type)
	}
}

func TestParse_ValidRefreshToken(t *testing.T) {
	m := newTestManager()
	token, _, _ := m.IssueRefresh("101", "refresh@example.com")

	claims, err := m.Parse(token)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if claims.Type != "refresh" {
		t.Errorf("Parse() Type = %q, want refresh", claims.Type)
	}
}

func TestParse_InvalidToken(t *testing.T) {
	m := newTestManager()
	_, err := m.Parse("this.is.not.valid")
	if err == nil {
		t.Error("Parse() should return error for invalid token string")
	}
}

func TestParse_EmptyToken(t *testing.T) {
	m := newTestManager()
	_, err := m.Parse("")
	if err == nil {
		t.Error("Parse() should return error for empty token")
	}
}

func TestParse_WrongSecret(t *testing.T) {
	m1 := newTestManager()
	m2 := jwt.New(config.JWTConfig{
		Secret:    "completely-different-secret",
		AccessTTL: 15 * time.Minute,
	})

	token, _, _ := m1.IssueAccess("1", "a@b.com")
	_, err := m2.Parse(token)
	if err == nil {
		t.Error("Parse() should reject token signed with a different secret")
	}
}

func TestParse_ExpiredToken(t *testing.T) {
	m := jwt.New(config.JWTConfig{
		Secret:    "test-secret",
		AccessTTL: -time.Minute, // already expired when issued
	})
	token, _, _ := m.IssueAccess("1", "a@b.com")
	_, err := m.Parse(token)
	if err == nil {
		t.Error("Parse() should return error for expired token")
	}
}
