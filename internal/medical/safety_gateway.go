// Package medical 医疗安全网关
// 医疗输出强制附加免责声明，拦截 LLM 违规输出
// 禁止 AI 执行医疗诊断
package medical

import (
	"regexp"
	"strings"
)

// DisclaimerCN 中文免责声明（所有医疗输出必须附加）
const DisclaimerCN = "以下内容仅基于设备数据观察，不构成医疗建议，请咨询执业宠物医师"

// DisclaimerEN 英文免责声明
const DisclaimerEN = "The following is based on device data observation only and does not constitute medical advice. Please consult a licensed veterinarian."

// blockedPatterns 医疗违规输出模式（正则）
// 拦截：诊断、改剂量、处方、用药建议等
var blockedPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(诊断|确诊|诊为|诊断为|diagnos)`),
	regexp.MustCompile(`(?i)(剂量|用量|加量|减量|调整剂量|dosage|dose)`),
	regexp.MustCompile(`(?i)(处方|开药|用药|服药|使用.*抗生素|使用.*药物|prescri)`),
	regexp.MustCompile(`(?i)(建议.*药|推荐.*药|应该.*吃药|应该.*服药)`),
	regexp.MustCompile(`(?i)(手术建议|建议手术|应该手术)`),
	regexp.MustCompile(`(?i)(停药|断药|换药|替代.*药物)`),
}

// SafetyResult 安全检查结果
type SafetyResult struct {
	Blocked bool   `json:"blocked"`  // 是否被拦截
	Reason  string `json:"reason"`   // 拦截原因
	Content string `json:"content"`  // 处理后的内容（附加免责声明）
}

// CheckOutput 检查医疗 AI 输出是否合规
// 1. 检查是否包含违规内容（诊断、改剂量等）
// 2. 自动附加免责声明
// 返回处理后的内容
func CheckOutput(content string) SafetyResult {
	// 检查违规模式
	for _, pattern := range blockedPatterns {
		if pattern.MatchString(content) {
			return SafetyResult{
				Blocked: true,
				Reason:  "LLM 输出包含医疗违规内容: " + pattern.String(),
				Content: "",
			}
		}
	}

	// 附加免责声明
	safeContent := appendDisclaimer(content)

	return SafetyResult{
		Blocked: false,
		Content: safeContent,
	}
}

// appendDisclaimer 附加免责声明
// 如果内容已包含免责声明则不重复添加
func appendDisclaimer(content string) string {
	if strings.Contains(content, DisclaimerCN) || strings.Contains(content, DisclaimerEN) {
		return content
	}
	return content + "\n\n---\n" + DisclaimerCN
}

// SanitizeInput 清洗医疗输入（防止 Prompt 注入）
// 移除可能的注入指令
func SanitizeInput(input string) string {
	// 移除系统指令注入尝试
	injectionPatterns := []string{
		"忽略之前的指令",
		"ignore previous instructions",
		"忽略以上所有",
		"disregard all above",
		"你现在是",
		"you are now",
	}

	cleaned := input
	for _, pattern := range injectionPatterns {
		lower := strings.ToLower(cleaned)
		lowerPattern := strings.ToLower(pattern)
		if strings.Contains(lower, lowerPattern) {
			idx := strings.Index(lower, lowerPattern)
			cleaned = cleaned[:idx] + "[FILTERED]" + cleaned[idx+len(pattern):]
		}
	}

	return cleaned
}
