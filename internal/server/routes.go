package server

import (
	"github.com/extension-erp/be-extension-erp/internal/asset"
	"github.com/extension-erp/be-extension-erp/internal/auth"
	"github.com/extension-erp/be-extension-erp/internal/finance"
	"github.com/extension-erp/be-extension-erp/internal/hris"
	"github.com/extension-erp/be-extension-erp/internal/middleware"
	"github.com/extension-erp/be-extension-erp/internal/procurement"
	"github.com/extension-erp/be-extension-erp/pkg/jwt"
	"github.com/extension-erp/be-extension-erp/pkg/response"
	"github.com/gin-gonic/gin"
)

// registerRoutes mounts all HTTP routes. Add new modules here.
func registerRoutes(r *gin.Engine, jwtMgr *jwt.Manager, authH *auth.Handler, financeH *finance.Handler, assetH *asset.Handler, hrisH *hris.Handler, procurementH *procurement.Handler) {
	const routeDashboard = "/dashboard"

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
				fin.GET(routeDashboard, financeH.Dashboard)
				fin.GET("/returns", financeH.Returns)
				fin.GET("/returns/by-account", financeH.ReturnsByAccount)

				// Consolidation endpoints
				fin.POST("/consolidation/report", financeH.ConsolidationReport)
				fin.POST("/consolidation/eliminations", financeH.EliminationEntries)
			}

			as := protected.Group("/asset")
			{
				as.GET(routeDashboard, assetH.Dashboard)
			}

			hr := protected.Group("/hris")
			{
				hr.GET(routeDashboard, hrisH.Dashboard)
			}

			proc := protected.Group("/procurement")
			{
				proc.GET(routeDashboard, procurementH.Dashboard)
				proc.GET("/po-cycle-time-trend", procurementH.POCycleTimeTrend)
				proc.GET("/receiving-cycle-time-trend", procurementH.ReceivingCycleTimeTrend)
				proc.GET("/procurement-cycle-time-trend", procurementH.ProcurementCycleTimeTrend)
				proc.GET("/purchase-trend-ytd", procurementH.PurchaseTrendYTD)
				proc.GET("/documents", procurementH.ListDocuments)
			}
		}
	}
}
