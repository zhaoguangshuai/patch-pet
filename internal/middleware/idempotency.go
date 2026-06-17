package middleware

import (
	"bytes"
	"net/http"
	"sync"
	"time"

	"github.com/patch-pet/patch-pet/pkg/constants"
)

// IdempotencyStore 幂等键存储接口
// 生产环境使用 Redis 实现，开发/测试使用内存实现
type IdempotencyStore interface {
	// Get 获取已缓存的响应
	Get(key string) (*CachedResponse, bool)
	// Set 缓存响应（含过期时间）
	Set(key string, resp *CachedResponse, ttl time.Duration)
	// Delete 删除缓存
	Delete(key string)
}

// CachedResponse 缓存的响应
type CachedResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	CreatedAt  time.Time
}

// MemoryIdempotencyStore 内存幂等键存储（开发/测试用）
// 生产环境应替换为 Redis 实现
type MemoryIdempotencyStore struct {
	mu      sync.RWMutex
	entries map[string]*CachedResponse
}

// NewMemoryIdempotencyStore 创建内存幂等存储
func NewMemoryIdempotencyStore() *MemoryIdempotencyStore {
	return &MemoryIdempotencyStore{
		entries: make(map[string]*CachedResponse),
	}
}

func (s *MemoryIdempotencyStore) Get(key string) (*CachedResponse, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resp, ok := s.entries[key]
	if !ok {
		return nil, false
	}

	// 检查是否过期
	if time.Since(resp.CreatedAt) > 24*time.Hour {
		return nil, false
	}
	return resp, true
}

func (s *MemoryIdempotencyStore) Set(key string, resp *CachedResponse, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[key] = resp
}

func (s *MemoryIdempotencyStore) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, key)
}

// idempotencyRecorder 用于捕获下游 handler 的响应
type idempotencyRecorder struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
	wroteHeader bool
}

func (r *idempotencyRecorder) WriteHeader(code int) {
	if !r.wroteHeader {
		r.statusCode = code
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *idempotencyRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.statusCode = http.StatusOK
		r.wroteHeader = true
	}
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

// IdempotencyMiddleware 幂等中间件
// 所有 POST/PUT/PATCH 写操作强制携带 Idempotency-Key
// 相同 key 的重复请求返回首次缓存的响应
func IdempotencyMiddleware(store IdempotencyStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 仅对写操作启用幂等校验
			if r.Method != http.MethodPost &&
				r.Method != http.MethodPut &&
				r.Method != http.MethodPatch {
				next.ServeHTTP(w, r)
				return
			}

			// 健康检查跳过
			if r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			// 读取幂等键
			key := r.Header.Get("Idempotency-Key")
			if key == "" {
				writeErrorResponse(w, http.StatusBadRequest, constants.CodePaymentIdempotent, "缺少 Idempotency-Key Header")
				return
			}

			// 检查是否已有缓存响应
			if cached, ok := store.Get(key); ok {
				// 返回缓存的响应
				for k, v := range cached.Headers {
					for _, val := range v {
						w.Header().Add(k, val)
					}
				}
				w.Header().Set("X-Idempotent-Replay", "true")
				w.WriteHeader(cached.StatusCode)
				w.Write(cached.Body)
				return
			}

			// 使用 recorder 捕获响应
			rec := &idempotencyRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(rec, r)

			// 缓存成功的响应（2xx）
			if rec.statusCode >= 200 && rec.statusCode < 300 {
				store.Set(key, &CachedResponse{
					StatusCode: rec.statusCode,
					Headers:    w.Header().Clone(),
					Body:       rec.body.Bytes(),
					CreatedAt:  time.Now(),
				}, 24*time.Hour)
			}
		})
	}
}
