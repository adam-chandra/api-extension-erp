package asset

import (
	"strconv"

	"github.com/extension-erp/be-extension-erp/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Dashboard handles GET /api/asset/dashboard?companyId=<n>
func (h *Handler) Dashboard(c *gin.Context) {
	companyID, ok := parseCompanyID(c)
	if !ok {
		return
	}

	res, err := h.svc.Dashboard(c.Request.Context(), companyID)
	if err != nil {
		response.Internal(c, "could not load asset dashboard")
		return
	}
	response.OK(c, res)
}

func parseCompanyID(c *gin.Context) (int64, bool) {
	raw := c.Query("companyId")
	if raw == "" {
		response.BadRequest(c, "companyId is required")
		return 0, false
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "companyId must be a positive integer")
		return 0, false
	}
	return id, true
}
