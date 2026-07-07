package hris

import (
	"strconv"
	"time"

	"github.com/extension-erp/be-extension-erp/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Dashboard handles GET /api/hris/dashboard?companyId=<n>&year=<YYYY>
func (h *Handler) Dashboard(c *gin.Context) {
	companyID, ok := parseCompanyID(c)
	if !ok {
		return
	}
	year := parseYear(c)
	res, err := h.svc.Dashboard(c.Request.Context(), companyID, year)
	if err != nil {
		response.Internal(c, "could not load hris dashboard")
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

func parseYear(c *gin.Context) int {
	raw := c.Query("year")
	if raw == "" {
		return time.Now().Year()
	}
	y, err := strconv.Atoi(raw)
	if err != nil || y < 2000 || y > 2100 {
		return time.Now().Year()
	}
	return y
}
