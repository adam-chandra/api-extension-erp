package middleware

import (
	"strings"

	"github.com/extension-erp/be-extension-erp/pkg/jwt"
	"github.com/extension-erp/be-extension-erp/pkg/response"
	"github.com/gin-gonic/gin"
)

const (
	ctxUserID = "auth.user_id"
	ctxEmail  = "auth.email"
)

// Auth requires a valid Bearer access token on the request.
func Auth(jwtMgr *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			response.Unauthorized(c, "missing or malformed authorization header")
			return
		}
		token := strings.TrimPrefix(header, "Bearer ")

		claims, err := jwtMgr.Parse(token)
		if err != nil {
			response.Unauthorized(c, "invalid or expired token")
			return
		}
		if claims.Type != "access" {
			response.Unauthorized(c, "invalid token type")
			return
		}

		c.Set(ctxUserID, claims.UserID)
		c.Set(ctxEmail, claims.Email)
		c.Next()
	}
}

// UserID returns the authenticated user id from context.
func UserID(c *gin.Context) string {
	v, _ := c.Get(ctxUserID)
	s, _ := v.(string)
	return s
}

// Email returns the authenticated user email from context.
func Email(c *gin.Context) string {
	v, _ := c.Get(ctxEmail)
	s, _ := v.(string)
	return s
}
