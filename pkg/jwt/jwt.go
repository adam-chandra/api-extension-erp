package jwt

import (
	"errors"
	"time"

	"github.com/extension-erp/be-extension-erp/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

// Claims is the JWT payload we issue.
type Claims struct {
	UserID string `json:"uid"`
	Email  string `json:"email"`
	Type   string `json:"typ"` // "access" or "refresh"
	jwt.RegisteredClaims
}

// Manager creates and validates tokens. Reuse a single instance.
type Manager struct {
	cfg config.JWTConfig
}

func New(cfg config.JWTConfig) *Manager {
	return &Manager{cfg: cfg}
}

// IssueAccess returns a signed access token for the given user.
func (m *Manager) IssueAccess(userID, email string) (string, time.Time, error) {
	return m.sign(userID, email, "access", m.cfg.AccessTTL)
}

// IssueRefresh returns a signed refresh token for the given user.
func (m *Manager) IssueRefresh(userID, email string) (string, time.Time, error) {
	return m.sign(userID, email, "refresh", m.cfg.RefreshTTL)
}

func (m *Manager) sign(userID, email, typ string, ttl time.Duration) (string, time.Time, error) {
	exp := time.Now().Add(ttl)
	claims := Claims{
		UserID: userID,
		Email:  email,
		Type:   typ,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Subject:   userID,
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(m.cfg.Secret))
	return signed, exp, err
}

// Parse verifies the token signature/expiry and returns the claims.
func (m *Manager) Parse(token string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(m.cfg.Secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
