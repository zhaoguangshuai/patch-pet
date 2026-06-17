// Package dogwalk 代遛服务风险拦截网关
// 禁止自动下单、强制营销、付费置顶、暗箱加权
// 服务商排序完全透明
package dogwalk

import (
	"fmt"
)

// RiskAction 风险动作类型
type RiskAction string

const (
	RiskActionAutoOrder   RiskAction = "auto_order"    // 自动下单
	RiskActionForceMarket RiskAction = "force_market"  // 强制营销
	RiskActionPaidTop     RiskAction = "paid_top"      // 付费置顶
	RiskActionHiddenWeight RiskAction = "hidden_weight" // 暗箱加权
)

// RiskDecision 风险决策结果
type RiskDecision struct {
	Blocked bool       `json:"blocked"` // 是否被拦截
	Action  RiskAction `json:"action"`  // 风险动作
	Reason  string     `json:"reason"`  // 拦截原因
}

// RiskGateway 服务风险拦截网关
type RiskGateway struct {
	blockedActions map[RiskAction]bool
}

// NewRiskGateway 创建风险拦截网关
// 默认拦截所有风险动作
func NewRiskGateway() *RiskGateway {
	return &RiskGateway{
		blockedActions: map[RiskAction]bool{
			RiskActionAutoOrder:    true,
			RiskActionForceMarket:  true,
			RiskActionPaidTop:      true,
			RiskActionHiddenWeight: true,
		},
	}
}

// CheckAutoOrder 检查是否为自动下单
// 所有订单创建必须经过用户明确确认
func (g *RiskGateway) CheckAutoOrder(userConfirmed bool) RiskDecision {
	if !userConfirmed {
		return RiskDecision{
			Blocked: true,
			Action:  RiskActionAutoOrder,
			Reason:  "禁止自动下单：订单创建必须经过用户明确确认",
		}
	}
	return RiskDecision{Blocked: false}
}

// CheckMarketing 检查是否为强制营销
// 禁止利用宠物异常状态诱导消费、制造焦虑、使用恐吓话术
func (g *RiskGateway) CheckMarketing(content string, userTriggered bool) RiskDecision {
	if !userTriggered {
		return RiskDecision{
			Blocked: true,
			Action:  RiskActionForceMarket,
			Reason:  "禁止强制营销：营销内容必须由用户主动触发",
		}
	}
	return RiskDecision{Blocked: false}
}

// CheckVendorSort 检查服务商排序是否透明
// 禁止付费置顶、暗箱加权
func (g *RiskGateway) CheckVendorSort(sortBy string) RiskDecision {
	allowedSortFields := map[string]bool{
		"rating":       true, // 用户评分
		"distance":     true, // 距离
		"price":        true, // 价格
		"completion":   true, // 完成率
		"responseTime": true, // 响应时间
	}

	if !allowedSortFields[sortBy] {
		return RiskDecision{
			Blocked: true,
			Action:  RiskActionHiddenWeight,
			Reason:  fmt.Sprintf("禁止暗箱加权：排序字段 %q 不在允许列表中", sortBy),
		}
	}
	return RiskDecision{Blocked: false}
}

// CheckPaidPlacement 检查是否为付费置顶
func (g *RiskGateway) CheckPaidPlacement(vendorID string, isPaid bool) RiskDecision {
	if isPaid {
		return RiskDecision{
			Blocked: true,
			Action:  RiskActionPaidTop,
			Reason:  "禁止付费置顶：服务商排序必须基于客观指标",
		}
	}
	return RiskDecision{Blocked: false}
}

// ValidateOrderCreation 验证订单创建合规性
// 综合检查：用户确认 + 非自动 + 非营销诱导
func (g *RiskGateway) ValidateOrderCreation(userConfirmed, isAutoOrder, isMarketingDriven bool) RiskDecision {
	if isAutoOrder {
		return RiskDecision{
			Blocked: true,
			Action:  RiskActionAutoOrder,
			Reason:  "禁止自动下单",
		}
	}
	if isMarketingDriven {
		return RiskDecision{
			Blocked: true,
			Action:  RiskActionForceMarket,
			Reason:  "禁止营销诱导下单",
		}
	}
	if !userConfirmed {
		return RiskDecision{
			Blocked: true,
			Action:  RiskActionAutoOrder,
			Reason:  "订单创建必须经过用户明确确认",
		}
	}
	return RiskDecision{Blocked: false}
}
