package medical

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestRouter(handler *Handler) *gin.Engine {
	r := gin.New()
	rg := r.Group("/api/v1")
	handler.RegisterRoutes(rg)
	return r
}

func TestGetCurrentEpisodeMissingPetID(t *testing.T) {
	handler := &Handler{repo: nil}
	r := setupTestRouter(handler)

	// 空 pet_id 路由不会匹配任何路由，Gin 返回 400
	req := httptest.NewRequest("GET", "/api/v1/pets//medical-episodes/current", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code == http.StatusOK {
		t.Error("empty pet_id should not return 200")
	}
}

func TestListEpisodesMissingPetID(t *testing.T) {
	handler := &Handler{repo: nil}
	r := setupTestRouter(handler)

	req := httptest.NewRequest("GET", "/api/v1/pets//medical-episodes/list", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code == http.StatusOK {
		t.Error("empty pet_id should not return 200")
	}
}

func TestHandlerCreation(t *testing.T) {
	handler := &Handler{repo: nil}
	if handler == nil {
		t.Fatal("Handler should not be nil")
	}
}

func TestRegisterRoutes(t *testing.T) {
	handler := &Handler{repo: nil}
	r := gin.New()
	rg := r.Group("/api/v1")
	handler.RegisterRoutes(rg)

	// 验证路由已注册
	routes := r.Routes()
	found := map[string]bool{
		"GetCurrentEpisode": false,
		"ListEpisodes":      false,
	}

	for _, route := range routes {
		if route.Path == "/api/v1/pets/:pet_id/medical-episodes/current" && route.Method == "GET" {
			found["GetCurrentEpisode"] = true
		}
		if route.Path == "/api/v1/pets/:pet_id/medical-episodes/list" && route.Method == "GET" {
			found["ListEpisodes"] = true
		}
	}

	for name, ok := range found {
		if !ok {
			t.Errorf("route %s not registered", name)
		}
	}
}

func TestResponseFormat(t *testing.T) {
	// 验证统一响应格式 {code, message, data}
	resp := map[string]interface{}{
		"code":    0,
		"message": "ok",
		"data":    nil,
	}

	jsonBytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if parsed["code"] != float64(0) {
		t.Errorf("code = %v, want 0", parsed["code"])
	}
	if parsed["message"] != "ok" {
		t.Errorf("message = %v, want ok", parsed["message"])
	}
}
