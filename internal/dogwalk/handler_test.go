package dogwalk

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupTestRouter(handler *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	rg := r.Group("/api/v1")
	handler.RegisterRoutes(rg)
	return r
}

func TestDetectOpportunityInvalidBody(t *testing.T) {
	handler := NewHandler(nil)
	r := setupTestRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/dog-walk/opportunities/detect", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("DetectOpportunity with empty body status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestDetectOpportunityInvalidConfidence(t *testing.T) {
	handler := NewHandler(nil)
	r := setupTestRouter(handler)

	body := DetectOpportunityRequest{
		PetID:       "pet_123",
		FamilyID:    "fam_456",
		TriggerReason: json.RawMessage(`{"type":"schedule"}`),
		Confidence:  1.5, // 超出范围
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/dog-walk/opportunities/detect", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("DetectOpportunity with invalid confidence status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestDetectOpportunityMissingPetID(t *testing.T) {
	handler := NewHandler(nil)
	r := setupTestRouter(handler)

	body := DetectOpportunityRequest{
		FamilyID:    "fam_456",
		TriggerReason: json.RawMessage(`{"type":"schedule"}`),
		Confidence:  0.5,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/dog-walk/opportunities/detect", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("DetectOpportunity with missing pet_id status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAllowPlanEmptyID(t *testing.T) {
	handler := NewHandler(nil)
	r := setupTestRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/dog-walk/opportunities//allow-plan", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// Empty ID path won't match the route → 404 or method not allowed
	if w.Code == http.StatusOK {
		t.Error("AllowPlan with empty ID should not return 200")
	}
}

func TestListOrdersMissingUserID(t *testing.T) {
	handler := NewHandler(nil)
	r := setupTestRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dog-walk/orders/list", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ListOrders without user_id status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestListVendorsSortBlocked(t *testing.T) {
	handler := NewHandler(nil)
	r := setupTestRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dog-walk/vendors?sort_by=paid_score", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("ListVendors with blocked sort status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestListVendorsSortAllowed(t *testing.T) {
	handler := NewHandler(nil)
	r := setupTestRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dog-walk/vendors?sort_by=rating", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListVendors with allowed sort status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRegisterRoutes(t *testing.T) {
	handler := NewHandler(nil)
	r := gin.New()
	rg := r.Group("/api/v1")
	handler.RegisterRoutes(rg)

	routes := r.Routes()
	expectedPaths := map[string]bool{
		"/api/v1/dog-walk/opportunities/detect":          false,
		"/api/v1/dog-walk/opportunities/:id/allow-plan":  false,
		"/api/v1/dog-walk/vendors":                       false,
		"/api/v1/dog-walk/routes/plan":                   false,
		"/api/v1/dog-walk/plans/:plan_id/payment-orders": false,
		"/api/v1/dog-walk/payment-callback":              false,
		"/api/v1/dog-walk/orders/:order_id/book":         false,
		"/api/v1/dog-walk/orders/:order_id":              false,
		"/api/v1/dog-walk/orders/:order_id/reports":      false,
		"/api/v1/dog-walk/orders/list":                   false,
	}

	for _, route := range routes {
		if _, ok := expectedPaths[route.Path]; ok {
			expectedPaths[route.Path] = true
		}
	}

	for path, found := range expectedPaths {
		if !found {
			t.Errorf("Route %s not registered", path)
		}
	}
}
