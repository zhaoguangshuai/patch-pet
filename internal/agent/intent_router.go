// Package agent LAKI 意图路由器
// 医疗/代遛 Agent 优先级管控 + 冲突处理
// 医疗（P0）> 代遛（P1），冲突时优先医疗
package agent

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/patch-pet/patch-pet/internal/audit"
	"github.com/patch-pet/patch-pet/pkg/logger"
	"github.com/patch-pet/patch-pet/pkg/types"
)

// IntentCategory 意图分类
type IntentCategory string

const (
	IntentMedical IntentCategory = "medical" // 医疗意图
	IntentDogwalk IntentCategory = "dogwalk" // 代遛意图
	IntentGeneral IntentCategory = "general" // 通用意图
)

// intentPriority 意图优先级（数值越小优先级越高）
var intentPriority = map[IntentCategory]int{
	IntentMedical: 0, // P0 最高优先
	IntentDogwalk: 1, // P1
	IntentGeneral: 2, // 最低
}

// medicalKeywords 医疗意图关键词
var medicalKeywords = []string{
	"医疗", "看病", "吃药", "用药", "剂量", "症状", "诊断",
	"复查", "复诊", "疗程", "医嘱", "护理", "治疗", "疫苗",
	"驱虫", "体检", "手术", "住院", "急诊", "呕吐", "拉稀",
	"发烧", "咳嗽", "皮肤病", "骨折", "伤口", "药物",
	"medical", "medication", "dose", "symptom", "diagnosis",
	"vet", "vaccine", "treatment", "care_task", "medical_episode",
}

// dogwalkKeywords 代遛意图关键词
var dogwalkKeywords = []string{
	"代遛", "遛狗", "散步", "遛弯", "出门", "外出", "路线",
	"服务商", "预约", "下单", "支付", "订单", "宠物店",
	"dogwalk", "walk", "route", "vendor", "booking", "payment",
	"dog_walk", "opportunity", "dog_walk_order", "dog_walk_plan",
}

// IntentRouter LAKI 意图路由器
type IntentRouter struct {
	agents    map[IntentCategory]Agent
	auditLog  audit.Recorder
}

// NewIntentRouter 创建意图路由器
func NewIntentRouter(auditLog audit.Recorder) *IntentRouter {
	return &IntentRouter{
		agents:   make(map[IntentCategory]Agent),
		auditLog: auditLog,
	}
}

// RegisterAgent 注册 Agent
func (r *IntentRouter) RegisterAgent(category IntentCategory, agent Agent) {
	r.agents[category] = agent
}

// RouteResult 路由结果
type RouteResult struct {
	Category IntentCategory `json:"category"`
	Agent    Agent          `json:"-"`
	Intent   *types.ToolIntent `json:"intent"`
	Conflict bool           `json:"conflict"` // 是否存在意图冲突
	Conflicts []IntentCategory `json:"conflicts,omitempty"` // 冲突的意图列表
}

// Route 路由用户输入到对应 Agent
// 1. 分析输入文本，识别意图类别
// 2. 多意图冲突时，按优先级选择（医疗 > 代遛 > 通用）
// 3. 记录冲突到审计日志
func (r *IntentRouter) Route(ctx context.Context, input string) (*RouteResult, error) {
	categories := r.classifyIntent(input)

	if len(categories) == 0 {
		categories = []IntentCategory{IntentGeneral}
	}

	// 按优先级排序
	r.sortByPriority(categories)

	selected := categories[0]
	conflict := len(categories) > 1

	// 记录冲突
	if conflict && r.auditLog != nil {
		r.auditLog.Record(ctx, audit.Entry{
			Action:  audit.ActionIntentConflict,
			Level:   audit.LevelWarn,
			Module:  "laki_router",
			Detail:  map[string]any{
				"categories": categoriesToStrings(categories),
				"selected":   string(selected),
			},
		})
		logger.Warn("LAKI 意图冲突",
			zap.String("selected", string(selected)),
			zap.Strings("all", categoriesToStrings(categories)),
		)
	}

	agent, exists := r.agents[selected]
	if !exists {
		// 降级到已有 Agent（优先医疗，其次代遛）
		agent = r.fallbackAgent()
		if agent == nil {
			return nil, fmt.Errorf("无可用 Agent 处理意图: %s", selected)
		}
		selected = agentCategory(agent)
	}

	result := &RouteResult{
		Category:  selected,
		Agent:     agent,
		Conflict:  conflict,
		Conflicts: categories,
	}

	return result, nil
}

// classifyIntent 基于关键词分类意图
func (r *IntentRouter) classifyIntent(input string) []IntentCategory {
	lower := strings.ToLower(input)
	var categories []IntentCategory

	if matchesKeywords(lower, medicalKeywords) {
		categories = append(categories, IntentMedical)
	}
	if matchesKeywords(lower, dogwalkKeywords) {
		categories = append(categories, IntentDogwalk)
	}

	return categories
}

// sortByPriority 按优先级排序（医疗 > 代遛 > 通用）
func (r *IntentRouter) sortByPriority(categories []IntentCategory) {
	for i := 1; i < len(categories); i++ {
		for j := i; j > 0; j-- {
			if intentPriority[categories[j]] < intentPriority[categories[j-1]] {
				categories[j], categories[j-1] = categories[j-1], categories[j]
			}
		}
	}
}

// fallbackAgent 降级 Agent 选择
func (r *IntentRouter) fallbackAgent() Agent {
	for _, cat := range []IntentCategory{IntentMedical, IntentDogwalk, IntentGeneral} {
		if agent, ok := r.agents[cat]; ok {
			return agent
		}
	}
	return nil
}

func matchesKeywords(input string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(input, kw) {
			return true
		}
	}
	return false
}

func categoriesToStrings(cats []IntentCategory) []string {
	s := make([]string, len(cats))
	for i, c := range cats {
		s[i] = string(c)
	}
	return s
}

func agentCategory(a Agent) IntentCategory {
	switch a.Type() {
	case AgentTypeMedical:
		return IntentMedical
	case AgentTypeDogwalk:
		return IntentDogwalk
	default:
		return IntentGeneral
	}
}
