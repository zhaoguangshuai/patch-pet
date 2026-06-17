package constants

// 全局运行阈值（固定不可改，由 Nacos 配置中心托管，代码中仅定义默认值）
const (
	// MaxToolCallsPerSession 单次会话最大工具调用数
	MaxToolCallsPerSession = 5
	// MaxToolNestingDepth 工具嵌套最大递归深度
	MaxToolNestingDepth = 3
	// MaxLLMInputTokens LLM 输入 Token 上限
	MaxLLMInputTokens = 8000
	// MaxLLMOutputTokens LLM 输出 Token 上限
	MaxLLMOutputTokens = 1500
	// ThirdPartyTimeoutMs 第三方接口超时（毫秒）
	ThirdPartyTimeoutMs = 3000
	// ThirdPartyCircuitBreakerThreshold 连续失败次数触发熔断
	ThirdPartyCircuitBreakerThreshold = 10
	// ThirdPartyCircuitBreakerDurationSec 熔断持续时间（秒）
	ThirdPartyCircuitBreakerDurationSec = 300
	// ScheduledTaskTimeoutSec 定时任务超时阈值（秒）
	ScheduledTaskTimeoutSec = 10
	// GracefulShutdownTimeoutSec 优雅停机最长等待（秒）
	GracefulShutdownTimeoutSec = 30

	// 分页
	DefaultPageSize = 20
	MaxPageSize     = 50

	// 防重放
	ReplayTimestampToleranceSec = 300 // 5 分钟

	// Token 预算（Nacos 配置，此处为默认值）
	DailyTokenBudget   = 100000
	MonthlyTokenBudget = 2000000
	MaxCostPerUser     = 5000

	// LLM 降级重试
	LLMRetryCount = 1

	// 支付回调重试
	PaymentCallbackRetryIntervalSec = 30
	PaymentCallbackMaxRetries       = 3

	// Kafka 消费重试
	KafkaMaxRetries = 3

	// Redis 缓存 TTL
	SessionMemoryTTLHours = 24
)
