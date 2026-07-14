package response_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/extension-erp/be-extension-erp/pkg/response"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newContext(method, path string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, nil)
	return c, w
}

func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder) response.Envelope {
	t.Helper()
	var env response.Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	return env
}

func TestOK(t *testing.T) {
	c, w := newContext(http.MethodGet, "/test")
	response.OK(c, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("OK() status = %d, want %d", w.Code, http.StatusOK)
	}
	env := decodeEnvelope(t, w)
	if env.Message != "ok" {
		t.Errorf("OK() message = %q, want ok", env.Message)
	}
	if env.Code != http.StatusOK {
		t.Errorf("OK() envelope code = %d, want %d", env.Code, http.StatusOK)
	}
}

func TestCreated(t *testing.T) {
	c, w := newContext(http.MethodPost, "/resource")
	response.Created(c, map[string]string{"id": "42"})

	if w.Code != http.StatusCreated {
		t.Errorf("Created() status = %d, want %d", w.Code, http.StatusCreated)
	}
	env := decodeEnvelope(t, w)
	if env.Message != "created" {
		t.Errorf("Created() message = %q, want created", env.Message)
	}
}

func TestMessage(t *testing.T) {
	c, w := newContext(http.MethodGet, "/ping")
	response.Message(c, "pong")

	if w.Code != http.StatusOK {
		t.Errorf("Message() status = %d, want %d", w.Code, http.StatusOK)
	}
	env := decodeEnvelope(t, w)
	if env.Message != "pong" {
		t.Errorf("Message() message = %q, want pong", env.Message)
	}
}

func TestBadRequest_NoDetails(t *testing.T) {
	c, w := newContext(http.MethodPost, "/resource")
	response.BadRequest(c, "validation failed")

	if w.Code != http.StatusBadRequest {
		t.Errorf("BadRequest() status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	env := decodeEnvelope(t, w)
	if env.Message != "validation failed" {
		t.Errorf("BadRequest() message = %q, want validation failed", env.Message)
	}
	if env.Error != nil {
		t.Error("BadRequest() without details should have nil Error field")
	}
}

func TestBadRequest_WithDetails(t *testing.T) {
	c, w := newContext(http.MethodPost, "/resource")
	response.BadRequest(c, "invalid input", map[string]string{"field": "required"})

	if w.Code != http.StatusBadRequest {
		t.Errorf("BadRequest() with details status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	env := decodeEnvelope(t, w)
	if env.Error == nil {
		t.Error("BadRequest() with details should include Error field")
	}
}

func TestUnauthorized(t *testing.T) {
	c, w := newContext(http.MethodGet, "/secure")
	response.Unauthorized(c, "missing token")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Unauthorized() status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	env := decodeEnvelope(t, w)
	if env.Message != "missing token" {
		t.Errorf("Unauthorized() message = %q, want missing token", env.Message)
	}
}

func TestForbidden(t *testing.T) {
	c, w := newContext(http.MethodGet, "/admin")
	response.Forbidden(c, "insufficient permissions")

	if w.Code != http.StatusForbidden {
		t.Errorf("Forbidden() status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestNotFound(t *testing.T) {
	c, w := newContext(http.MethodGet, "/missing")
	response.NotFound(c, "resource not found")

	if w.Code != http.StatusNotFound {
		t.Errorf("NotFound() status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestInternal(t *testing.T) {
	c, w := newContext(http.MethodGet, "/crash")
	response.Internal(c, "internal server error")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Internal() status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestJSON_CodeMirrorsHTTPStatus(t *testing.T) {
	c, w := newContext(http.MethodGet, "/teapot")
	response.JSON(c, http.StatusTeapot, response.Envelope{Message: "I am a teapot"})

	if w.Code != http.StatusTeapot {
		t.Errorf("JSON() HTTP status = %d, want %d", w.Code, http.StatusTeapot)
	}
	env := decodeEnvelope(t, w)
	if env.Code != http.StatusTeapot {
		t.Errorf("JSON() envelope code = %d, want %d", env.Code, http.StatusTeapot)
	}
}
