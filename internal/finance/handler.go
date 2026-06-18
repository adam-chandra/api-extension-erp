package finance

import (
	"errors"
	"strconv"

	"github.com/extension-erp/be-extension-erp/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Dashboard handles GET /api/finance/dashboard
//
//	?companyId=<n>&period=<all|year|month|custom>
//	[&start=YYYY-MM-DD&end=YYYY-MM-DD]  -- required when period=custom
func (h *Handler) Dashboard(c *gin.Context) {
	companyID, ok := parseCompanyID(c)
	if !ok {
		return
	}
	period := c.Query("period")
	start := c.Query("start")
	end := c.Query("end")

	res, err := h.svc.Dashboard(c.Request.Context(), companyID, period, start, end)
	if err != nil {
		if errors.Is(err, ErrInvalidPeriod) {
			response.BadRequest(c, "invalid period (use all|year|month|custom)")
			return
		}
		if errors.Is(err, ErrInvalidRange) {
			response.BadRequest(c, "invalid date range (start, end must be YYYY-MM-DD and end >= start)")
			return
		}
		response.Internal(c, "could not load dashboard")
		return
	}
	response.OK(c, res)
}

// Returns handles GET /api/finance/returns?companyId=<n>&limit=<n>
func (h *Handler) Returns(c *gin.Context) {
	companyID, ok := parseCompanyID(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	res, err := h.svc.Returns(c.Request.Context(), companyID, limit)
	if err != nil {
		response.Internal(c, "could not load returns")
		return
	}
	response.OK(c, res)
}

// ReturnsByAccount handles GET /api/finance/returns/by-account
//
//	?companyId=<n>&period=<all|year|month|custom>&start=&end=&limit=<n>
func (h *Handler) ReturnsByAccount(c *gin.Context) {
	companyID, ok := parseCompanyID(c)
	if !ok {
		return
	}
	period := c.Query("period")
	start := c.Query("start")
	end := c.Query("end")
	limit, _ := strconv.Atoi(c.Query("limit"))

	res, err := h.svc.ReturnsByAccount(c.Request.Context(), companyID, period, start, end, limit)
	if err != nil {
		if errors.Is(err, ErrInvalidPeriod) {
			response.BadRequest(c, "invalid period (use all|year|month|custom)")
			return
		}
		if errors.Is(err, ErrInvalidRange) {
			response.BadRequest(c, "invalid date range")
			return
		}
		response.Internal(c, "could not load returns by account")
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
