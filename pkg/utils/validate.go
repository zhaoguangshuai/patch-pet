// Package utils 通用入参校验
// 长度 / 格式 / 范围校验工具
package utils

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// ValidationError 校验错误
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("字段 %s: %s", e.Field, e.Message)
}

// ValidationErrors 校验错误集合
type ValidationErrors []ValidationError

func (es ValidationErrors) Error() string {
	msgs := make([]string, len(es))
	for i, e := range es {
		msgs[i] = e.Error()
	}
	return strings.Join(msgs, "; ")
}

// Validator 校验器
type Validator struct {
	errors ValidationErrors
}

// NewValidator 创建校验器
func NewValidator() *Validator {
	return &Validator{}
}

// Validate 执行校验链，返回所有错误
func (v *Validator) Validate() ValidationErrors {
	if len(v.errors) == 0 {
		return nil
	}
	return v.errors
}

// addError 添加校验错误
func (v *Validator) addError(field, message string) {
	v.errors = append(v.errors, ValidationError{Field: field, Message: message})
}

// Required 必填校验
func (v *Validator) Required(field, value string) *Validator {
	if strings.TrimSpace(value) == "" {
		v.addError(field, "不能为空")
	}
	return v
}

// MaxLen 最大长度校验（按 rune 计算）
func (v *Validator) MaxLen(field, value string, max int) *Validator {
	if utf8.RuneCountInString(value) > max {
		v.addError(field, fmt.Sprintf("长度不能超过 %d", max))
	}
	return v
}

// MinLen 最小长度校验
func (v *Validator) MinLen(field, value string, min int) *Validator {
	if utf8.RuneCountInString(value) < min {
		v.addError(field, fmt.Sprintf("长度不能少于 %d", min))
	}
	return v
}

// Range 数值范围校验
func (v *Validator) Range(field string, value, min, max float64) *Validator {
	if value < min || value > max {
		v.addError(field, fmt.Sprintf("必须在 %.0f ~ %.0f 之间", min, max))
	}
	return v
}

// Pattern 正则校验
func (v *Validator) Pattern(field, value, pattern, message string) *Validator {
 matched, _ := regexp.MatchString(pattern, value)
	if !matched {
		v.addError(field, message)
	}
	return v
}

// OneOf 枚举校验
func (v *Validator) OneOf(field, value string, allowed []string) *Validator {
	for _, a := range allowed {
		if value == a {
			return v
		}
	}
	v.addError(field, fmt.Sprintf("必须是以下之一: %s", strings.Join(allowed, ", ")))
	return v
}

// ULIDFormat ULID 格式校验（带前缀）
func (v *Validator) ULIDFormat(field, value, prefix string) *Validator {
	expected := prefix + "_"
	if !strings.HasPrefix(value, expected) {
		v.addError(field, fmt.Sprintf("必须以 %s 开头", expected))
	}
	// 前缀后应为 26 字符
	suffix := strings.TrimPrefix(value, expected)
	if len(suffix) != 26 {
		v.addError(field, "ULID 部分长度必须为 26")
	}
	return v
}

// --- 常用校验快捷函数 ---

// ValidatePetID 校验宠物 ID 格式
func ValidatePetID(id string) error {
	v := NewValidator()
	v.Required("pet_id", id)
	if errs := v.Validate(); errs != nil {
		return errs
	}
	return nil
}

// ValidatePagination 校验分页参数
func ValidatePagination(pageNum, pageSize int) (int, int) {
	if pageNum < 1 {
		pageNum = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 50 {
		pageSize = 50
	}
	return pageNum, pageSize
}

// ValidateConfidence 校验置信度范围 0~1
func ValidateConfidence(field string, value float64) error {
	errs := NewValidator().Range(field, value, 0, 1).Validate()
	if len(errs) == 0 {
		return nil
	}
	return errs
}
