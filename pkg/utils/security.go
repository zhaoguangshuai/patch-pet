// Package utils 安全工具
// 防 SQL 注入（GORM 参数化）+ XSS 输出转义
package utils

import (
	"html"
	"regexp"
	"strings"
)

// EscapeXSS XSS 输出转义
// 对 HTML 特殊字符进行转义，防止 XSS 攻击
func EscapeXSS(input string) string {
	return html.EscapeString(input)
}

// UnescapeXSS 反转义（仅用于可信内容展示）
func UnescapeXSS(input string) string {
	return html.UnescapeString(input)
}

// SanitizeHTML 清理 HTML 标签（白名单模式）
// 移除所有 HTML 标签，仅保留纯文本
func SanitizeHTML(input string) string {
	// 移除 HTML 标签
	re := regexp.MustCompile(`<[^>]*>`)
	cleaned := re.ReplaceAllString(input, "")
	// 转义残留特殊字符
	return html.EscapeString(cleaned)
}

// SanitizeInput 清理用户输入
// 1. 去除首尾空白
// 2. 移除危险字符
// 3. 限制长度
func SanitizeInput(input string, maxLen int) string {
	result := strings.TrimSpace(input)
	// 移除 null 字节
	result = strings.ReplaceAll(result, "\x00", "")
	if maxLen > 0 && len(result) > maxLen {
		result = result[:maxLen]
	}
	return result
}

// SQLInjectionPatterns 常见 SQL 注入模式
var sqlInjectionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(union\s+select)`),
	regexp.MustCompile(`(?i)(or\s+1\s*=\s*1)`),
	regexp.MustCompile(`(?i)(;\s*drop\s+table)`),
	regexp.MustCompile(`(?i)(;\s*delete\s+from)`),
	regexp.MustCompile(`(?i)(;\s*update\s+\w+\s+set)`),
	regexp.MustCompile(`(?i)(--)`),
	regexp.MustCompile(`(?i)(/\*.*\*/)`),
	regexp.MustCompile(`(?i)('\s*or\s+')`),
}

// DetectSQLInjection 检测疑似 SQL 注入
// 仅用于日志告警，实际防注入靠 GORM 参数化查询
func DetectSQLInjection(input string) bool {
	lower := strings.ToLower(input)
	for _, pattern := range sqlInjectionPatterns {
		if pattern.MatchString(lower) {
			return true
		}
	}
	return false
}

// SanitizeFieldName 清理字段名（仅允许字母数字下划线）
// 用于动态排序/过滤场景，防止字段名注入
func SanitizeFieldName(field string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	return re.ReplaceAllString(field, "")
}

// XSSPatterns 常见 XSS 攻击模式
var xssPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)<script[^>]*>`),
	regexp.MustCompile(`(?i)</script>`),
	regexp.MustCompile(`(?i)javascript:`),
	regexp.MustCompile(`(?i)on\w+\s*=`),
	regexp.MustCompile(`(?i)<iframe[^>]*>`),
	regexp.MustCompile(`(?i)<object[^>]*>`),
	regexp.MustCompile(`(?i)<embed[^>]*>`),
}

// DetectXSS 检测疑似 XSS 攻击
func DetectXSS(input string) bool {
	lower := strings.ToLower(input)
	for _, pattern := range xssPatterns {
		if pattern.MatchString(lower) {
			return true
		}
	}
	return false
}
