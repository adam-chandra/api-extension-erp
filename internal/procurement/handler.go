package procurement

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

func (h *Handler) Dashboard(c *gin.Context) {
	companyID, ok := parseCompanyID(c)
	if !ok {
		return
	}
	res, err := h.svc.Dashboard(c.Request.Context(), companyID,
		c.Query("period"), c.Query("start"), c.Query("end"))
	if err != nil {
		if errors.Is(err, ErrInvalidPeriod) {
			response.BadRequest(c, "invalid period (use all|year|month|custom)")
			return
		}
		if errors.Is(err, ErrInvalidRange) {
			response.BadRequest(c, "invalid date range")
			return
		}
		response.Internal(c, "could not load procurement dashboard")
		return
	}
	response.OK(c, res)
}

func (h *Handler) POCycleTimeTrend(c *gin.Context) {
	companyID, ok := parseCompanyID(c)
	if !ok {
		return
	}
	res, err := h.svc.POCycleTimeTrend(c.Request.Context(), companyID,
		c.Query("period"), c.Query("start"), c.Query("end"))
	if err != nil {
		if errors.Is(err, ErrInvalidPeriod) {
			response.BadRequest(c, "invalid period")
			return
		}
		response.Internal(c, "could not load po cycle time trend")
		return
	}
	response.OK(c, res)
}

func (h *Handler) ReceivingCycleTimeTrend(c *gin.Context) {
	companyID, ok := parseCompanyID(c)
	if !ok {
		return
	}
	res, err := h.svc.ReceivingCycleTimeTrend(c.Request.Context(), companyID,
		c.Query("period"), c.Query("start"), c.Query("end"))
	if err != nil {
		if errors.Is(err, ErrInvalidPeriod) {
			response.BadRequest(c, "invalid period")
			return
		}
		if errors.Is(err, ErrInvalidRange) {
			response.BadRequest(c, "invalid date range")
			return
		}
		response.Internal(c, "could not load receiving cycle time trend")
		return
	}
	response.OK(c, res)
}

func (h *Handler) ProcurementCycleTimeTrend(c *gin.Context) {
	companyID, ok := parseCompanyID(c)
	if !ok {
		return
	}
	res, err := h.svc.ProcurementCycleTimeTrend(c.Request.Context(), companyID,
		c.Query("period"), c.Query("start"), c.Query("end"))
	if err != nil {
		if errors.Is(err, ErrInvalidPeriod) {
			response.BadRequest(c, "invalid period")
			return
		}
		if errors.Is(err, ErrInvalidRange) {
			response.BadRequest(c, "invalid date range")
			return
		}
		response.Internal(c, "could not load procurement cycle time trend")
		return
	}
	response.OK(c, res)
}

func (h *Handler) PurchaseTrendYTD(c *gin.Context) {
	companyID, ok := parseCompanyID(c)
	if !ok {
		return
	}
	res, err := h.svc.PurchaseTrendYTD(c.Request.Context(), companyID)
	if err != nil {
		response.Internal(c, "could not load purchase trend ytd")
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

func (h *Handler) ListDocuments(c *gin.Context) {
	companyID, ok := parseCompanyID(c)
	if !ok {
		return
	}
	metricType := c.Query("metric")
	if metricType == "" {
		response.BadRequest(c, "metric is required")
		return
	}
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))

	res, err := h.svc.ListDocuments(c.Request.Context(), companyID, ListDocumentsQuery{
		PeriodCode:  c.Query("period"),
		CustomStart: c.Query("start"),
		CustomEnd:   c.Query("end"),
		MetricType:  metricType,
		Search:      c.Query("search"),
		Page:        page,
		Limit:       limit,
	})
	if err != nil {
		if errors.Is(err, ErrInvalidPeriod) {
			response.BadRequest(c, "invalid period")
			return
		}
		if errors.Is(err, ErrInvalidRange) {
			response.BadRequest(c, "invalid date range")
			return
		}
		response.Internal(c, "could not list documents")
		return
	}
	response.OK(c, res)
}
