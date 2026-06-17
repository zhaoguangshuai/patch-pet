// Package utils 通用工具函数
// 时间处理工具，统一 UTC+8 时区
package utils

import (
	"time"

	"github.com/patch-pet/patch-pet/pkg/types"
)

// NowCST 返回当前 UTC+8 时间（CSTTime 类型）
func NowCST() types.CSTTime {
	return types.NowCST()
}

// FormatCST 将时间格式化为 UTC+8 标准格式 yyyy-MM-ddTHH:mm:ss+08:00
func FormatCST(t time.Time) string {
	return types.NewCSTTime(t).Time.Format("2006-01-02T15:04:05+08:00")
}

// ParseCST 解析 UTC+8 格式时间字符串
func ParseCST(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02T15:04:05+08:00", s, types.CSTLocation)
}
