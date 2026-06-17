package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/patch-pet/patch-pet/internal/auth"
)

func TestGatewayConfigDefaults(t *testing.T) {
	cfg := DefaultGatewayConfig()
	if !cfg.EnableTrace {
		t.Error("EnableTrace should default to true")
	}
	if cfg.RateLimit.MaxRequests != 100 {
		t.Errorf("RateLimit.MaxRequests = %d, want 100", cfg.RateLimit.MaxRequests)
	}
}

func TestGinTraceMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GinTraceMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if w.Header().Get("X-Trace-Id") == "" {
		t.Error("X-Trace-Id header should be set")
	}
}

func TestGinTraceMiddlewarePassThrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GinTraceMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	validTraceID := "trace_01HXYZ1234567890ABCDEFGHJK"
	req.Header.Set("X-Trace-Id", validTraceID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should pass through valid trace ID
	if w.Header().Get("X-Trace-Id") != validTraceID {
		t.Errorf("X-Trace-Id = %s, want %s", w.Header().Get("X-Trace-Id"), validTraceID)
	}
}

func TestGinHeaderValidationMissingAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GinHeaderValidation())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestGinHeaderValidationMissingSource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GinHeaderValidation())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer token123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestGinHeaderValidationSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GinHeaderValidation())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("X-Request-Source", "mobile")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestGinHeaderValidationHealthSkip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GinHeaderValidation())
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestGinPaginationDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GinPaginationMiddleware())
	r.GET("/test", func(c *gin.Context) {
		pn, _ := c.Get("pageNum")
		ps, _ := c.Get("pageSize")
		c.JSON(200, gin.H{"pageNum": pn, "pageSize": ps})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["pageNum"].(float64) != 1 {
		t.Errorf("pageNum = %v, want 1", resp["pageNum"])
	}
	if resp["pageSize"].(float64) != 20 {
		t.Errorf("pageSize = %v, want 20", resp["pageSize"])
	}
}

func TestGinPaginationCustom(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GinPaginationMiddleware())
	r.GET("/test", func(c *gin.Context) {
		pn, _ := c.Get("pageNum")
		ps, _ := c.Get("pageSize")
		c.JSON(200, gin.H{"pageNum": pn, "pageSize": ps})
	})

	req := httptest.NewRequest(http.MethodGet, "/test?pageNum=3&pageSize=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["pageNum"].(float64) != 3 {
		t.Errorf("pageNum = %v, want 3", resp["pageNum"])
	}
	if resp["pageSize"].(float64) != 10 {
		t.Errorf("pageSize = %v, want 10", resp["pageSize"])
	}
}

func TestGinPaginationMaxTruncation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GinPaginationMiddleware())
	r.GET("/test", func(c *gin.Context) {
		ps, _ := c.Get("pageSize")
		c.JSON(200, gin.H{"pageSize": ps})
	})

	req := httptest.NewRequest(http.MethodGet, "/test?pageSize=100", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["pageSize"].(float64) != 50 {
		t.Errorf("pageSize = %v, want 50 (max)", resp["pageSize"])
	}
}

func TestIPBlockerBlacklist(t *testing.T) {
	blocker := NewIPBlocker(IPBlockerConfig{
		Mode:    IPBlockModeBlacklist,
		IPList:  []string{"192.168.1.1", "10.0.0.0/8"},
		Enabled: true,
	})

	if !blocker.IsBlocked("192.168.1.1") {
		t.Error("192.168.1.1 should be blocked")
	}
	if !blocker.IsBlocked("10.0.0.5") {
		t.Error("10.0.0.5 should be blocked (in CIDR)")
	}
	if blocker.IsBlocked("172.16.0.1") {
		t.Error("172.16.0.1 should not be blocked")
	}
}

func TestIPBlockerWhitelist(t *testing.T) {
	blocker := NewIPBlocker(IPBlockerConfig{
		Mode:    IPBlockModeWhitelist,
		IPList:  []string{"192.168.1.1"},
		Enabled: true,
	})

	if blocker.IsBlocked("192.168.1.1") {
		t.Error("192.168.1.1 should not be blocked (in whitelist)")
	}
	if !blocker.IsBlocked("10.0.0.1") {
		t.Error("10.0.0.1 should be blocked (not in whitelist)")
	}
}

func TestIPBlockerDisabled(t *testing.T) {
	blocker := NewIPBlocker(IPBlockerConfig{
		Enabled: false,
	})

	if blocker.IsBlocked("192.168.1.1") {
		t.Error("Nothing should be blocked when disabled")
	}
}

func TestIPBlockerMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	blocker := NewIPBlocker(IPBlockerConfig{
		Mode:    IPBlockModeBlacklist,
		IPList:  []string{"192.168.1.1"},
		Enabled: true,
	})

	r := gin.New()
	r.Use(blocker.Middleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

// --- GinAuthMiddleware tests ---

func TestGinAuthMiddlewareMissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GinAuthMiddleware(auth.NewJWTAuthenticator()))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestGinAuthMiddlewareInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GinAuthMiddleware(auth.NewJWTAuthenticator()))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer bad")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestGinAuthMiddlewareValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GinAuthMiddleware(auth.NewJWTAuthenticator()))
	r.GET("/test", func(c *gin.Context) {
		rc := GetRequestContext(c)
		if rc == nil {
			t.Error("expected RequestContext in context")
			c.JSON(500, gin.H{"error": "no context"})
			return
		}
		c.JSON(200, gin.H{"user_id": rc.UserID, "role": rc.Role})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer user001|fam001|owner|app")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["user_id"] != "user001" {
		t.Errorf("user_id = %v, want user001", resp["user_id"])
	}
	if resp["role"] != "owner" {
		t.Errorf("role = %v, want owner", resp["role"])
	}
}

func TestGinAuthMiddlewareHealthSkip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GinAuthMiddleware(auth.NewJWTAuthenticator()))
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}
