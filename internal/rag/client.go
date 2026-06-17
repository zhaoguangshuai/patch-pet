// Package rag Go 侧 RAG 调用封装
// RAG 向量化/检索实现归属 Python ai-service，此包封装 Go 调用接口
package rag

import "context"

// QueryRequest RAG 查询请求
type QueryRequest struct {
	Query   string `json:"query"`    // 查询文本
	TopK    int    `json:"top_k"`    // 召回数量（默认 5）
	RerankN int    `json:"rerank_n"` // 重排后保留数量（默认 3）
}

// QueryResult RAG 查询结果
type QueryResult struct {
	Content  string  `json:"content"`   // 文本片段
	Score    float64 `json:"score"`     // 相似度分数
	Source   string  `json:"source"`    // 来源标识
	Metadata map[string]any `json:"metadata"` // 元数据
}

// QueryResponse RAG 查询响应
type QueryResponse struct {
	Results []QueryResult `json:"results"`
}

// Client RAG 客户端接口（Go 侧调用 Python ai-service）
type Client interface {
	// Query 执行 RAG 检索（gRPC 调用 Python 服务）
	Query(ctx context.Context, req QueryRequest) (*QueryResponse, error)
}
