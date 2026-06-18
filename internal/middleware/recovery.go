package middleware

import (
	"log"
	"runtime/debug"

	"github.com/extension-erp/be-extension-erp/pkg/response"
	"github.com/gin-gonic/gin"
)

// Recovery catches panics, logs them, and returns a 500 JSON error.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic recovered: %v\n%s", r, debug.Stack())
				response.Internal(c, "internal server error")
			}
		}()
		c.Next()
	}
}
