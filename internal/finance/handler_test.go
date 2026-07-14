// Finance handler tests — same package (white-box) to reach internal types.
package finance

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newHandlerWithMockRepo(repo Repository) *Handler {
	return NewHandler(NewService(repo))
}

func doHandlerRequest(r http.Handler, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ---------------------------------------------------------------------------
// parseCompanyID (exercised via handlers)
// ---------------------------------------------------------------------------

func TestDashboardHandler_MissingCompanyID(t *testing.T) {
	h := newHandlerWithMockRepo(&mockRepo{})
	r := gin.New()
	r.GET("/dashboard", h.Dashboard)

	w := doHandlerRequest(r, http.MethodGet, "/dashboard")
	if w.Code != http.StatusBadRequest {
		t.Errorf("Dashboard no companyId: status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestDashboardHandler_InvalidCompanyID(t *testing.T) {
	h := newHandlerWithMockRepo(&mockRepo{})
	r := gin.New()
	r.GET("/dashboard", h.Dashboard)

	w := doHandlerRequest(r, http.MethodGet, "/dashboard?companyId=abc")
	if w.Code != http.StatusBadRequest {
		t.Errorf("Dashboard bad companyId: status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestDashboardHandler_ZeroCompanyID(t *testing.T) {
	h := newHandlerWithMockRepo(&mockRepo{})
	r := gin.New()
	r.GET("/dashboard", h.Dashboard)

	w := doHandlerRequest(r, http.MethodGet, "/dashboard?companyId=0")
	if w.Code != http.StatusBadRequest {
		t.Errorf("Dashboard companyId=0: status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestDashboardHandler_InvalidPeriod(t *testing.T) {
	h := newHandlerWithMockRepo(&mockRepo{dailyRows: []dailyRow{}})
	r := gin.New()
	r.GET("/dashboard", h.Dashboard)

	w := doHandlerRequest(r, http.MethodGet, "/dashboard?companyId=1&period=quarterly")
	if w.Code != http.StatusBadRequest {
		t.Errorf("Dashboard invalid period: status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestDashboardHandler_InvalidRange(t *testing.T) {
	h := newHandlerWithMockRepo(&mockRepo{dailyRows: []dailyRow{}})
	r := gin.New()
	r.GET("/dashboard", h.Dashboard)

	// custom period with end before start
	w := doHandlerRequest(r, http.MethodGet, "/dashboard?companyId=1&period=custom&start=2025-06-01&end=2025-01-01")
	if w.Code != http.StatusBadRequest {
		t.Errorf("Dashboard invalid range: status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestDashboardHandler_RepoError(t *testing.T) {
	h := newHandlerWithMockRepo(&mockRepo{dailyErr: errors.New("db error")})
	r := gin.New()
	r.GET("/dashboard", h.Dashboard)

	w := doHandlerRequest(r, http.MethodGet, "/dashboard?companyId=1&period=year")
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Dashboard repo error: status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestDashboardHandler_Success(t *testing.T) {
	date := time.Date(2025, 5, 10, 0, 0, 0, 0, time.UTC)
	h := newHandlerWithMockRepo(&mockRepo{
		dailyRows: []dailyRow{
			{Date: date, Category: "revenue_operating", Net: 5000000},
		},
	})
	r := gin.New()
	r.GET("/dashboard", h.Dashboard)

	w := doHandlerRequest(r, http.MethodGet, "/dashboard?companyId=1&period=year")
	if w.Code != http.StatusOK {
		t.Errorf("Dashboard success: status = %d, want %d", w.Code, http.StatusOK)
	}
}

// ---------------------------------------------------------------------------
// Returns handler
// ---------------------------------------------------------------------------

func TestReturnsHandler_MissingCompanyID(t *testing.T) {
	h := newHandlerWithMockRepo(&mockRepo{})
	r := gin.New()
	r.GET("/returns", h.Returns)

	w := doHandlerRequest(r, http.MethodGet, "/returns")
	if w.Code != http.StatusBadRequest {
		t.Errorf("Returns no companyId: status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestReturnsHandler_RepoError(t *testing.T) {
	h := newHandlerWithMockRepo(&mockRepo{returErr: errors.New("db error")})
	r := gin.New()
	r.GET("/returns", h.Returns)

	w := doHandlerRequest(r, http.MethodGet, "/returns?companyId=1")
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Returns repo error: status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestReturnsHandler_Success(t *testing.T) {
	date := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	h := newHandlerWithMockRepo(&mockRepo{
		returRows: []returRow{{Description: "Item A", Date: date, Amount: 1000000}},
	})
	r := gin.New()
	r.GET("/returns", h.Returns)

	w := doHandlerRequest(r, http.MethodGet, "/returns?companyId=1&limit=5")
	if w.Code != http.StatusOK {
		t.Errorf("Returns success: status = %d, want %d", w.Code, http.StatusOK)
	}
}

// ---------------------------------------------------------------------------
// ReturnsByAccount handler
// ---------------------------------------------------------------------------

func TestReturnsByAccountHandler_MissingCompanyID(t *testing.T) {
	h := newHandlerWithMockRepo(&mockRepo{})
	r := gin.New()
	r.GET("/returns/by-account", h.ReturnsByAccount)

	w := doHandlerRequest(r, http.MethodGet, "/returns/by-account")
	if w.Code != http.StatusBadRequest {
		t.Errorf("ReturnsByAccount no companyId: status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestReturnsByAccountHandler_InvalidPeriod(t *testing.T) {
	h := newHandlerWithMockRepo(&mockRepo{})
	r := gin.New()
	r.GET("/returns/by-account", h.ReturnsByAccount)

	w := doHandlerRequest(r, http.MethodGet, "/returns/by-account?companyId=1&period=bad")
	if w.Code != http.StatusBadRequest {
		t.Errorf("ReturnsByAccount invalid period: status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestReturnsByAccountHandler_InvalidRange(t *testing.T) {
	h := newHandlerWithMockRepo(&mockRepo{})
	r := gin.New()
	r.GET("/returns/by-account", h.ReturnsByAccount)

	w := doHandlerRequest(r, http.MethodGet, "/returns/by-account?companyId=1&period=custom&start=2025-06-01&end=2025-01-01")
	if w.Code != http.StatusBadRequest {
		t.Errorf("ReturnsByAccount invalid range: status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestReturnsByAccountHandler_RepoError(t *testing.T) {
	h := newHandlerWithMockRepo(&mockRepo{returAccountErr: errors.New("db error")})
	r := gin.New()
	r.GET("/returns/by-account", h.ReturnsByAccount)

	w := doHandlerRequest(r, http.MethodGet, "/returns/by-account?companyId=1&period=year")
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ReturnsByAccount repo error: status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestReturnsByAccountHandler_Success(t *testing.T) {
	h := newHandlerWithMockRepo(&mockRepo{
		returAccountRows: []returAccountRow{
			{Code: "403", Name: "Retur Penjualan", Balance: 500000},
		},
	})
	r := gin.New()
	r.GET("/returns/by-account", h.ReturnsByAccount)

	w := doHandlerRequest(r, http.MethodGet, "/returns/by-account?companyId=1&period=year&limit=5")
	if w.Code != http.StatusOK {
		t.Errorf("ReturnsByAccount success: status = %d, want %d", w.Code, http.StatusOK)
	}
}
