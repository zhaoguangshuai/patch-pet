// Package constants 定义全局常量、错误码、配置键名
package constants

// 全局错误码分段
// 0: 成功
// 1000-1999: 认证/权限
// 2000-2999: 医疗
// 3000-3999: 代遛
// 4000-4999: 支付
// 5000-5999: 工具/网关
// 6000-6999: LLM/AI

const (
	CodeSuccess = 0

	// 认证/权限 1000-1999
	CodeAuthUnauthorized   = 1001 // 未登录/Token 无效
	CodeAuthForbidden      = 1002 // 无权限
	CodeAuthHeaderMissing  = 1003 // 缺少必要 Header
	CodeAuthTokenExpired   = 1004 // Token 过期
	CodeAuthReplayDetected = 1005 // 防重放拦截

	// 医疗 2000-2999
	CodeMedicalEpisodeNotFound  = 2001 // 疗程不存在
	CodeMedicalInvalidTransition = 2002 // 非法状态转换
	CodeMedicalNoAuthorization   = 2003 // 无医疗数据授权
	CodeMedicalDisclaimerRequired = 2004 // 缺少免责声明
	CodeMedicalDiagnosisBlocked  = 2005 // AI 诊断拦截

	// 代遛 3000-3999
	CodeDogwalkOppNotFound      = 3001 // 需求不存在
	CodeDogwalkInvalidTransition = 3002 // 非法状态转换
	CodeDogwalkAutoOrderBlocked  = 3003 // 禁止自动下单
	CodeDogwalkVendorNotFound    = 3004 // 服务商不存在
	CodeDogwalkRouteFailed       = 3005 // 路线规划失败

	// 支付 4000-4999
	CodePaymentFailed     = 4001 // 支付失败
	CodePaymentTimeout    = 4002 // 支付超时
	CodePaymentRefundFailed = 4003 // 退款失败
	CodePaymentIdempotent = 4004 // 幂等键冲突

	// 工具/网关 5000-5999
	CodeToolNotFound        = 5001 // 工具未注册
	CodeToolPermissionDenied = 5002 // 工具权限被拒
	CodeToolExecutionFailed  = 5003 // 工具执行失败
	CodeToolThresholdExceeded = 5004 // 超过调用阈值
	CodeToolCircuitOpen      = 5005 // 熔断器打开

	// LLM/AI 6000-6999
	CodeLLMOutputInvalid   = 6001 // LLM 输出格式不合规
	CodeLLMTokenExceeded   = 6002 // Token 超限
	CodeLLMBudgetExhausted = 6003 // Token 预算耗尽
	CodeLLMAllDegraded     = 6004 // 全链路降级（使用静态模板）
	CodeLLMPromptInjection = 6005 // Prompt 注入拦截
)
