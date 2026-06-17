package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPaginationMiddlewareDefaults(t *testing.T) {
	var captured Pagination

	handler := PaginationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = GetPagination(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if captured.PageNum != 1 {
		t.Errorf("PageNum = %d, want 1", captured.PageNum)
	}
	if captured.PageSize != 20 {
		t.Errorf("PageSize = %d, want 20", captured.PageSize)
	}
}

func TestPaginationMiddlewareCustomValues(t *testing.T) {
	var captured Pagination

	handler := PaginationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = GetPagination(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/test?pageNum=3&pageSize=10", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if captured.PageNum != 3 {
		t.Errorf("PageNum = %d, want 3", captured.PageNum)
	}
	if captured.PageSize != 10 {
		t.Errorf("PageSize = %d, want 10", captured.PageSize)
	}
}

func TestPaginationMiddlewarePageSizeTruncation(t *testing.T) {
	var captured Pagination

	handler := PaginationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = GetPagination(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	// 请求 pageSize=100，应自动截断为 50
	req := httptest.NewRequest("GET", "/api/v1/test?pageNum=1&pageSize=100", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if captured.PageSize != 50 {
		t.Errorf("PageSize = %d, want 50 (truncated from 100)", captured.PageSize)
	}
}

func TestPaginationMiddlewarePageSizeMaxBoundary(t *testing.T) {
	var captured Pagination

	handler := PaginationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = GetPagination(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	// 请求恰好 pageSize=50，不应截断
	req := httptest.NewRequest("GET", "/api/v1/test?pageSize=50", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if captured.PageSize != 50 {
		t.Errorf("PageSize = %d, want 50", captured.PageSize)
	}
}

func TestPaginationMiddlewarePageNumBelowOne(t *testing.T) {
	var captured Pagination

	handler := PaginationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = GetPagination(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/test?pageNum=0", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if captured.PageNum != 1 {
		t.Errorf("PageNum = %d, want 1 (clamped from 0)", captured.PageNum)
	}
}

func TestPaginationMiddlewarePageNumNegative(t *testing.T) {
	var captured Pagination

	handler := PaginationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = GetPagination(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/test?pageNum=-5", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if captured.PageNum != 1 {
		t.Errorf("PageNum = %d, want 1 (clamped from -5)", captured.PageNum)
	}
}

func TestPaginationMiddlewarePageSizeBelowOne(t *testing.T) {
	var captured Pagination

	handler := PaginationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = GetPagination(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/test?pageSize=0", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if captured.PageSize != 20 {
		t.Errorf("PageSize = %d, want 20 (default, clamped from 0)", captured.PageSize)
	}
}

func TestPaginationMiddlewareInvalidValues(t *testing.T) {
	var captured Pagination

	handler := PaginationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = GetPagination(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/test?pageNum=abc&pageSize=xyz", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if captured.PageNum != 1 {
		t.Errorf("PageNum = %d, want 1 (default for invalid)", captured.PageNum)
	}
	if captured.PageSize != 20 {
		t.Errorf("PageSize = %d, want 20 (default for invalid)", captured.PageSize)
	}
}

func TestPaginationOffset(t *testing.T) {
	tests := []struct {
		pageNum  int
		pageSize int
		want     int
	}{
		{1, 20, 0},
		{2, 20, 20},
		{3, 10, 20},
		{5, 50, 200},
	}
	for _, tt := range tests {
		pg := Pagination{PageNum: tt.pageNum, PageSize: tt.pageSize}
		if got := pg.Offset(); got != tt.want {
			t.Errorf("Offset(%d, %d) = %d, want %d", tt.pageNum, tt.pageSize, got, tt.want)
		}
	}
}

func TestGetPaginationDefaultWhenNoMiddleware(t *testing.T) {
	// 未经过中间件时（context 无分页值），GetPagination 应返回默认值
	pg := GetPagination(context.Background())

	if pg.PageNum != 1 {
		t.Errorf("default PageNum = %d, want 1", pg.PageNum)
	}
	if pg.PageSize != 20 {
		t.Errorf("default PageSize = %d, want 20", pg.PageSize)
	}
}
