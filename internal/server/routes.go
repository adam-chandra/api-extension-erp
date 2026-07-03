package server

import (
	"github.com/extension-erp/be-extension-erp/internal/asset"
	"github.com/extension-erp/be-extension-erp/internal/auth"
	"github.com/extension-erp/be-extension-erp/internal/finance"
	"github.com/extension-erp/be-extension-erp/internal/hris"
	"github.com/extension-erp/be-extension-erp/internal/middleware"
	"github.com/extension-erp/be-extension-erp/pkg/jwt"
	"github.com/extension-erp/be-extension-erp/pkg/response"
	"github.com/gin-gonic/gin"
)

// registerRoutes mounts all HTTP routes. Add new modules here.
func registerRoutes(r *gin.Engine, jwtMgr *jwt.Manager, authH *auth.Handler, financeH *finance.Handler, assetH *asset.Handler, hrisH *hris.Handler) {
	r.GET("/health", func(c *gin.Context) {
		response.Message(c, "ok")
	})

	api := r.Group("/api")
	{
		a := api.Group("/auth")
		{
			a.POST("/generate-password", authH.GeneratePassword)
			a.POST("/login", authH.Login)
			a.POST("/refresh", authH.Refresh)
		}

		// Protected
		protected := api.Group("")
		protected.Use(middleware.Auth(jwtMgr))
		{
			protected.GET("/auth/me", authH.Me)
			protected.GET("/auth/menus", authH.Menus)
			protected.POST("/auth/logout", authH.Logout)

			fin := protected.Group("/finance")
			{
				fin.GET("/dashboard", financeH.Dashboard)
				fin.GET("/returns", financeH.Returns)
				fin.GET("/returns/by-account", financeH.ReturnsByAccount)
			}

			as := protected.Group("/asset")
			{
				as.GET("/dashboard", assetH.Dashboard)
			}

			hr := protected.Group("/hris")
			{
				hr.GET("/dashboard", hrisH.Dashboard)
			}
		}
	}
}
