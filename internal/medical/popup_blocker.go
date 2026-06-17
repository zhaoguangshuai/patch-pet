// Package medical 医疗状态商业弹窗自动屏蔽
// 医疗状态下关闭所有商业弹窗，禁止利用宠物异常状态诱导消费
package medical

import (
	"context"
)

// PopupType 弹窗类型
type PopupType string

const (
	PopupCommercial PopupType = "commercial" // 商业营销弹窗
	PopupPromotion  PopupType = "promotion"  // 促销活动弹窗
	PopupUpsell     PopupType = "upsell"     // 增值服务弹窗
	PopupAd         PopupType = "ad"         // 广告弹窗
	PopupSystem     PopupType = "system"     // 系统通知（允许）
	PopupMedical    PopupType = "medical"    // 医疗提醒（允许）
)

// PopupCheckResult 弹窗检查结果
type PopupCheckResult struct {
	Blocked bool     `json:"blocked"` // 是否被屏蔽
	Reason  string   `json:"reason"`  // 屏蔽原因
}

// PopupBlocker 医疗状态弹窗屏蔽器
type PopupBlocker struct {
	repo *Repository
}

// NewPopupBlocker 创建弹窗屏蔽器
func NewPopupBlocker(repo *Repository) *PopupBlocker {
	return &PopupBlocker{repo: repo}
}

// CheckPopup 检查弹窗是否应被屏蔽
// 规则：宠物处于医疗疗程中时，屏蔽所有商业/营销/促销/广告弹窗
// 允许：系统通知、医疗提醒
func (p *PopupBlocker) CheckPopup(ctx context.Context, petID string, popupType PopupType) PopupCheckResult {
	// 系统通知和医疗提醒始终允许
	if popupType == PopupSystem || popupType == PopupMedical {
		return PopupCheckResult{Blocked: false}
	}

	// 查询宠物当前疗程
	episode, err := p.repo.GetCurrentEpisode(ctx, petID)
	if err != nil {
		// 查询失败时保守处理：不屏蔽（避免影响正常业务）
		return PopupCheckResult{Blocked: false}
	}

	// 无当前疗程 → 不屏蔽
	if episode == nil {
		return PopupCheckResult{Blocked: false}
	}

	// 有当前疗程 → 屏蔽所有商业弹窗
	return PopupCheckResult{
		Blocked: true,
		Reason:  "宠物处于医疗疗程中（状态: " + string(episode.Status) + "），已自动屏蔽商业弹窗",
	}
}

// IsCommercialPopup 判断是否为商业弹窗
func IsCommercialPopup(popupType PopupType) bool {
	switch popupType {
	case PopupCommercial, PopupPromotion, PopupUpsell, PopupAd:
		return true
	default:
		return false
	}
}
