// Package types 全局通用类型定义
// CSTTime 自定义时间类型，强制 UTC+8 序列化
// 所有对外 JSON 输出统一 yyyy-MM-ddTHH:mm:ss+08:00 格式
package types

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// CSTLocation UTC+8 时区（进程级复用，避免重复加载）
var CSTLocation = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		// 降级为固定偏移量
		loc = time.FixedZone("CST", 8*3600)
	}
	return loc
}()

// cstFormat 统一时间输出格式
const cstFormat = "2006-01-02T15:04:05+08:00"

// CSTTime 自定义时间类型
// JSON 序列化强制输出 UTC+8 格式：yyyy-MM-ddTHH:mm:ss+08:00
// 实现 json.Marshaler / json.Unmarshaler / database/sql.Scanner / driver.Valuer
type CSTTime struct {
	time.Time
}

// NewCSTTime 从 time.Time 构造 CSTTime
func NewCSTTime(t time.Time) CSTTime {
	return CSTTime{Time: t.In(CSTLocation)}
}

// NowCST 返回当前 UTC+8 时间
func NowCST() CSTTime {
	return CSTTime{Time: time.Now().In(CSTLocation)}
}

// MarshalJSON 强制输出 UTC+8 格式
func (t CSTTime) MarshalJSON() ([]byte, error) {
	if t.Time.IsZero() {
		return []byte(`"0001-01-01T00:00:00+08:00"`), nil
	}
	return []byte(`"` + t.Time.In(CSTLocation).Format(cstFormat) + `"`), nil
}

// UnmarshalJSON 解析 UTC+8 格式时间
func (t *CSTTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" || string(data) == `""` {
		t.Time = time.Time{}
		return nil
	}
	// 去掉引号
	s := string(data)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	parsed, err := time.ParseInLocation(cstFormat, s, CSTLocation)
	if err != nil {
		return fmt.Errorf("解析时间失败: %w", err)
	}
	t.Time = parsed
	return nil
}

// Scan 实现 database/sql.Scanner，从数据库读取时间
func (t *CSTTime) Scan(value interface{}) error {
	if value == nil {
		t.Time = time.Time{}
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		t.Time = v.In(CSTLocation)
	case []byte:
		parsed, err := time.ParseInLocation(cstFormat, string(v), CSTLocation)
		if err != nil {
			parsed, err = time.Parse(time.RFC3339, string(v))
			if err != nil {
				return fmt.Errorf("解析数据库时间失败: %w", err)
			}
		}
		t.Time = parsed.In(CSTLocation)
	case string:
		parsed, err := time.ParseInLocation(cstFormat, v, CSTLocation)
		if err != nil {
			parsed, err = time.Parse(time.RFC3339, v)
			if err != nil {
				return fmt.Errorf("解析数据库时间失败: %w", err)
			}
		}
		t.Time = parsed.In(CSTLocation)
	default:
		return fmt.Errorf("不支持的时间类型: %T", value)
	}
	return nil
}

// Value 实现 driver.Valuer，写入数据库
func (t CSTTime) Value() (driver.Value, error) {
	if t.Time.IsZero() {
		return nil, nil
	}
	return t.Time.In(CSTLocation), nil
}

// NullableCSTTime 可空时间类型
// JSON 序列化为 null 或 UTC+8 格式时间字符串
type NullableCSTTime struct {
	Time  *time.Time
	Valid bool
}

// NewNullableCSTTime 从 *time.Time 构造 NullableCSTTime
func NewNullableCSTTime(t *time.Time) NullableCSTTime {
	if t == nil {
		return NullableCSTTime{Valid: false}
	}
	cst := t.In(CSTLocation)
	return NullableCSTTime{Time: &cst, Valid: true}
}

// MarshalJSON 输出 null 或 UTC+8 格式时间
func (t NullableCSTTime) MarshalJSON() ([]byte, error) {
	if !t.Valid || t.Time == nil {
		return []byte("null"), nil
	}
	return []byte(`"` + t.Time.In(CSTLocation).Format(cstFormat) + `"`), nil
}

// UnmarshalJSON 解析 null 或 UTC+8 格式时间
func (t *NullableCSTTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" || string(data) == `""` {
		t.Time = nil
		t.Valid = false
		return nil
	}
	s := string(data)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	parsed, err := time.ParseInLocation(cstFormat, s, CSTLocation)
	if err != nil {
		return fmt.Errorf("解析时间失败: %w", err)
	}
	t.Time = &parsed
	t.Valid = true
	return nil
}

// Scan 实现 database/sql.Scanner
func (t *NullableCSTTime) Scan(value interface{}) error {
	if value == nil {
		t.Time = nil
		t.Valid = false
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		cst := v.In(CSTLocation)
		t.Time = &cst
		t.Valid = true
	case []byte:
		parsed, err := time.Parse(time.RFC3339, string(v))
		if err != nil {
			parsed, err = time.ParseInLocation(cstFormat, string(v), CSTLocation)
			if err != nil {
				return fmt.Errorf("解析数据库时间失败: %w", err)
			}
		}
		cst := parsed.In(CSTLocation)
		t.Time = &cst
		t.Valid = true
	case string:
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			parsed, err = time.ParseInLocation(cstFormat, v, CSTLocation)
			if err != nil {
				return fmt.Errorf("解析数据库时间失败: %w", err)
			}
		}
		cst := parsed.In(CSTLocation)
		t.Time = &cst
		t.Valid = true
	default:
		return fmt.Errorf("不支持的时间类型: %T", value)
	}
	return nil
}

// Value 实现 driver.Valuer
func (t NullableCSTTime) Value() (driver.Value, error) {
	if !t.Valid || t.Time == nil {
		return nil, nil
	}
	return t.Time.In(CSTLocation), nil
}

// IsNil 是否为空值
func (t NullableCSTTime) IsNil() bool {
	return !t.Valid || t.Time == nil
}
