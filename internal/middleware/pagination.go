package middleware

import (
	"context"
	"net/http"
	"strconv"

	"github.com/patch-pet/patch-pet/pkg/constants"
)

// PaginationKey 分页参数在 context 中的键
const PaginationKey ContextKey = "pagination"

// Pagination 分页参数
type Pagination struct {
	PageNum  int `json:"pageNum"`
	PageSize int `json:"pageSize"`
}

// Offset 计算数据库偏移量（用于 LIMIT/OFFSET 查询）
func (p Pagination) Offset() int {
	return (p.PageNum - 1) * p.PageSize
}

// PaginationMiddleware 通用分页中间件
// 从查询参数解析 pageNum/pageSize，自动截断 pageSize 上限
// 默认值：pageNum=1，pageSize=20，maxPageSize=50
func PaginationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pg := parsePagination(r)

		ctx := context.WithValue(r.Context(), PaginationKey, pg)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// GetPagination 从 context 获取分页参数
func GetPagination(ctx context.Context) Pagination {
	if v, ok := ctx.Value(PaginationKey).(Pagination); ok {
		return v
	}
	// 兜底默认值
	return Pagination{
		PageNum:  1,
		PageSize: constants.DefaultPageSize,
	}
}

// parsePagination 从查询参数解析分页参数
func parsePagination(r *http.Request) Pagination {
	pageNum := parseIntParam(r, "pageNum", 1)
	pageSize := parseIntParam(r, "pageSize", constants.DefaultPageSize)

	// pageNum 最小值为 1
	if pageNum < 1 {
		pageNum = 1
	}

	// pageSize 上限自动截断
	if pageSize > constants.MaxPageSize {
		pageSize = constants.MaxPageSize
	}
	// pageSize 最小值为 1
	if pageSize < 1 {
		pageSize = constants.DefaultPageSize
	}

	return Pagination{
		PageNum:  pageNum,
		PageSize: pageSize,
	}
}

// parseIntParam 从查询参数解析整数，失败返回默认值
func parseIntParam(r *http.Request, key string, defaultVal int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}
