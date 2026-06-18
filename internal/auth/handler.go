package auth

import (
	"errors"
	"net/http"

	"github.com/extension-erp/be-extension-erp/internal/middleware"
	"github.com/extension-erp/be-extension-erp/pkg/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GeneratePassword is the admin-side endpoint: input { login, email } — both
// must match the same active ERP user. On success the freshly generated
// plaintext password is returned exactly once.
func (h *Handler) GeneratePassword(c *gin.Context) {
	var req GeneratePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	res, err := h.svc.GeneratePassword(c.Request.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			response.Error(c, http.StatusNotFound, "user with matching login & email not found")
		default:
			response.Internal(c, "could not generate password")
		}
		return
	}
	response.Created(c, res)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	res, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			response.Unauthorized(c, "invalid credentials")
		case errors.Is(err, ErrPasswordNotSet):
			response.Error(c, http.StatusForbidden, "password not set; ask admin to generate one")
		default:
			response.Internal(c, "login failed")
		}
		return
	}
	response.OK(c, res)
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	res, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Unauthorized(c, "invalid refresh token")
		return
	}
	response.OK(c, res)
}

func (h *Handler) Me(c *gin.Context) {
	uid := middleware.UserID(c)
	if uid == "" {
		response.Unauthorized(c, "unauthenticated")
		return
	}
	u, err := h.svc.Me(c.Request.Context(), uid)
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}
	response.OK(c, u)
}

func (h *Handler) Logout(c *gin.Context) {
	uid := middleware.UserID(c)
	if err := h.svc.Logout(c.Request.Context(), uid); err != nil {
		response.Internal(c, "logout failed")
		return
	}
	response.Message(c, "logged out")
}

func (h *Handler) Menus(c *gin.Context) {
	uid := middleware.UserID(c)
	if uid == "" {
		response.Unauthorized(c, "unauthenticated")
		return
	}
	menus, err := h.svc.Menus(c.Request.Context(), uid)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.NotFound(c, "user not found")
			return
		}
		response.Internal(c, "could not load menus")
		return
	}
	response.OK(c, menus)
}
