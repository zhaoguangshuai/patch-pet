package types

// ApiResponse 统一 API 响应格式
// 所有接口必须通过此结构返回，禁止直接返回裸数据
type ApiResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// PageResponse 分页响应结构
type PageResponse struct {
	Total    int64 `json:"total"`
	PageNum  int   `json:"pageNum"`
	PageSize int   `json:"pageSize"`
	List     any   `json:"list"`
}

// Success 返回成功响应
func Success(data any) ApiResponse {
	return ApiResponse{Code: 0, Message: "ok", Data: data}
}

// Fail 返回业务错误响应
func Fail(code int, message string) ApiResponse {
	return ApiResponse{Code: code, Message: message, Data: nil}
}

// Page 返回分页响应
func Page(total int64, pageNum, pageSize int, list any) ApiResponse {
	return Success(PageResponse{
		Total:    total,
		PageNum:  pageNum,
		PageSize: pageSize,
		List:     list,
	})
}
