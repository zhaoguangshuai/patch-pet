package types

// IDPrefix ULID 前缀枚举，全局统一
// DB 字段类型 varchar(64)，禁止 UUID/雪花/自增
type IDPrefix string

const (
	PrefixEpisode     IDPrefix = "ep"     // medical_episode
	PrefixTask        IDPrefix = "task"   // care_task
	PrefixAction      IDPrefix = "act"    // care_task_action
	PrefixSummary     IDPrefix = "sum"    // medical_summary
	PrefixAuth        IDPrefix = "auth"   // medical_authorization
	PrefixOpportunity IDPrefix = "opp"    // dog_walk_opportunity
	PrefixPlan        IDPrefix = "plan"   // dog_walk_plan
	PrefixOrder       IDPrefix = "order"  // dog_walk_order
	PrefixEvent       IDPrefix = "event"  // dog_walk_live_event
	PrefixReport      IDPrefix = "report" // dog_walk_report
	PrefixKafkaEvent  IDPrefix = "evt"    // Kafka 事件 ID
)
