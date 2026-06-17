package utils

import (
	"regexp"
	"strings"
)

// MaskPhone 手机号脱敏：138****8888
func MaskPhone(phone string) string {
	if len(phone) < 7 {
		return phone
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

// MaskIDCard 身份证号脱敏：110101********0011
func MaskIDCard(idCard string) string {
	if len(idCard) < 8 {
		return idCard
	}
	return idCard[:6] + strings.Repeat("*", len(idCard)-8) + idCard[len(idCard)-2:]
}

// MaskBankCard 银行卡号脱敏：****1234
func MaskBankCard(cardNo string) string {
	if len(cardNo) < 4 {
		return cardNo
	}
	return "****" + cardNo[len(cardNo)-4:]
}

// MaskToken Token 脱敏：仅保留前 8 位
func MaskToken(token string) string {
	if len(token) <= 8 {
		return strings.Repeat("*", len(token))
	}
	return token[:8] + strings.Repeat("*", len(token)-8)
}

// RemovePassword 移除密码字段，返回空字符串
func RemovePassword(_ string) string {
	return ""
}

// MaskEmail 邮箱脱敏：t***@example.com
func MaskEmail(email string) string {
	idx := strings.Index(email, "@")
	if idx <= 1 {
		return "***"
	}
	return email[:1] + "***" + email[idx:]
}

// MaskAddress 地址脱敏：保留前 6 字符，详细地址替换
func MaskAddress(addr string) string {
	if len(addr) <= 6 {
		return "***"
	}
	return addr[:6] + "***"
}

// MaskName 姓名脱敏：张* / 张*明
func MaskName(name string) string {
	runes := []rune(name)
	if len(runes) <= 1 {
		return "*"
	}
	if len(runes) == 2 {
		return string(runes[0]) + "*"
	}
	return string(runes[0]) + strings.Repeat("*", len(runes)-2) + string(runes[len(runes)-1])
}

// MaskMedicalData 医疗数据脱敏：保留类型标识，隐藏具体值
func MaskMedicalData(data string) string {
	if len(data) <= 4 {
		return "***"
	}
	return data[:2] + "***" + data[len(data)-2:]
}

// MaskSensitiveText 自动检测并脱敏文本中的敏感数据
func MaskSensitiveText(text string) string {
	result := text

	// 手机号
	result = regexp.MustCompile(`1[3-9]\d{9}`).ReplaceAllStringFunc(result, func(s string) string {
		return MaskPhone(s)
	})

	// 身份证号
	result = regexp.MustCompile(`\d{17}[\dXx]`).ReplaceAllStringFunc(result, func(s string) string {
		return MaskIDCard(s)
	})

	// 邮箱
	result = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`).ReplaceAllStringFunc(result, func(s string) string {
		return MaskEmail(s)
	})

	return result
}
