package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/extension-erp/be-extension-erp/internal/middleware"
	"github.com/gin-gonic/gin"
)

func TestRecovery_NoPanic(t *testing.T) {
	r := gin.New()
	r.Use(middleware.Recovery())
	r.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Recovery() no-panic: status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRecovery_Panic_Returns500(t *testing.T) {
	r := gin.New()
	r.Use(middleware.Recovery())
	r.GET("/crash", func(c *gin.Context) { panic("simulated panic") })

	req := httptest.NewRequest(http.MethodGet, "/crash", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Recovery() panic: status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestRecovery_PanicWithError_Returns500(t *testing.T) {
	r := gin.New()
	r.Use(middleware.Recovery())
	r.GET("/crash", func(c *gin.Context) { panic(http.ErrAbortHandler) })

	req := httptest.NewRequest(http.MethodGet, "/crash", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Recovery() panic error: status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}
