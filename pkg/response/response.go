package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Envelope is the standard JSON response shape used across the API.
//
//	{ "code": <http status>, "message": "...", "data": {...} }
//
// `data` is omitted when empty. `code` mirrors the HTTP status so clients can
// switch on a single field without inspecting headers.
type Envelope struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   any    `json:"error,omitempty"`
}

// JSON writes a response with the given HTTP status as both the response
// status code and the envelope `code` field.
func JSON(c *gin.Context, status int, env Envelope) {
	env.Code = status
	c.JSON(status, env)
}

// OK writes a 200 response with data.
func OK(c *gin.Context, data any) {
	JSON(c, http.StatusOK, Envelope{Message: "ok", Data: data})
}

// Created writes a 201 response with data.
func Created(c *gin.Context, data any) {
	JSON(c, http.StatusCreated, Envelope{Message: "created", Data: data})
}

// Message writes a 200 response with just a message (no data).
func Message(c *gin.Context, msg string) {
	JSON(c, http.StatusOK, Envelope{Message: msg})
}

// Error aborts the request with the given status, message, and optional details.
func Error(c *gin.Context, status int, msg string, details ...any) {
	env := Envelope{Code: status, Message: msg}
	if len(details) > 0 {
		env.Error = details[0]
	}
	c.AbortWithStatusJSON(status, env)
}

// BadRequest is a shortcut for 400 errors (validation, etc.).
func BadRequest(c *gin.Context, msg string, details ...any) {
	Error(c, http.StatusBadRequest, msg, details...)
}

// Unauthorized is a shortcut for 401 errors.
func Unauthorized(c *gin.Context, msg string) {
	Error(c, http.StatusUnauthorized, msg)
}

// Forbidden is a shortcut for 403 errors.
func Forbidden(c *gin.Context, msg string) {
	Error(c, http.StatusForbidden, msg)
}

// NotFound is a shortcut for 404 errors.
func NotFound(c *gin.Context, msg string) {
	Error(c, http.StatusNotFound, msg)
}

// Internal is a shortcut for 500 errors.
func Internal(c *gin.Context, msg string) {
	Error(c, http.StatusInternalServerError, msg)
}
