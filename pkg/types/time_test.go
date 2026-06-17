package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCSTTimeMarshalJSON(t *testing.T) {
	// 构造一个已知时间（UTC 时间 2026-06-15 10:00:00 UTC = 18:00:00 CST）
	utc := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	cst := NewCSTTime(utc)

	data, err := json.Marshal(cst)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	want := `"2026-06-15T18:00:00+08:00"`
	if string(data) != want {
		t.Errorf("MarshalJSON = %s, want %s", string(data), want)
	}
}

func TestCSTTimeMarshalJSONZeroValue(t *testing.T) {
	cst := CSTTime{}

	data, err := json.Marshal(cst)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	want := `"0001-01-01T00:00:00+08:00"`
	if string(data) != want {
		t.Errorf("MarshalJSON zero = %s, want %s", string(data), want)
	}
}

func TestCSTTimeUnmarshalJSON(t *testing.T) {
	input := `"2026-06-15T18:00:00+08:00"`
	var cst CSTTime

	if err := json.Unmarshal([]byte(input), &cst); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	// 解析后应等价于 UTC 2026-06-15 10:00:00
	wantUTC := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	if !cst.Time.Equal(wantUTC) {
		t.Errorf("UnmarshalJSON time = %v, want %v", cst.Time, wantUTC)
	}
}

func TestCSTTimeUnmarshalJSONNull(t *testing.T) {
	input := `null`
	var cst CSTTime

	if err := json.Unmarshal([]byte(input), &cst); err != nil {
		t.Fatalf("UnmarshalJSON null failed: %v", err)
	}

	if !cst.Time.IsZero() {
		t.Errorf("UnmarshalJSON null should be zero time, got %v", cst.Time)
	}
}

func TestCSTTimeRoundTrip(t *testing.T) {
	// 使用精确到秒的时间，避免纳秒精度在 JSON 序列化中丢失
	original := NewCSTTime(time.Date(2026, 6, 15, 18, 30, 0, 0, CSTLocation))

	type Wrapper struct {
		T CSTTime `json:"t"`
	}
	w := Wrapper{T: original}

	data, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Wrapper
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if !original.Time.Equal(decoded.T.Time) {
		t.Errorf("Round trip failed: %v != %v", original.Time, decoded.T.Time)
	}
}

// ---- NullableCSTTime ----

func TestNullableCSTTimeMarshalJSONNull(t *testing.T) {
	nt := NullableCSTTime{Valid: false}

	data, err := json.Marshal(nt)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	if string(data) != "null" {
		t.Errorf("MarshalJSON null = %s, want null", string(data))
	}
}

func TestNullableCSTTimeMarshalJSONValue(t *testing.T) {
	utc := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	nt := NewNullableCSTTime(&utc)

	data, err := json.Marshal(nt)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	want := `"2026-06-15T18:00:00+08:00"`
	if string(data) != want {
		t.Errorf("MarshalJSON = %s, want %s", string(data), want)
	}
}

func TestNullableCSTTimeMarshalJSONNilPointer(t *testing.T) {
	nt := NewNullableCSTTime(nil)

	data, err := json.Marshal(nt)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	if string(data) != "null" {
		t.Errorf("MarshalJSON nil ptr = %s, want null", string(data))
	}
}

func TestNullableCSTTimeUnmarshalJSONNull(t *testing.T) {
	var nt NullableCSTTime

	if err := json.Unmarshal([]byte("null"), &nt); err != nil {
		t.Fatalf("UnmarshalJSON null failed: %v", err)
	}

	if nt.Valid {
		t.Error("UnmarshalJSON null should set Valid=false")
	}
	if nt.Time != nil {
		t.Error("UnmarshalJSON null should set Time=nil")
	}
}

func TestNullableCSTTimeUnmarshalJSONValue(t *testing.T) {
	input := `"2026-06-15T18:00:00+08:00"`
	var nt NullableCSTTime

	if err := json.Unmarshal([]byte(input), &nt); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if !nt.Valid {
		t.Error("UnmarshalJSON value should set Valid=true")
	}
	wantUTC := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	if !nt.Time.Equal(wantUTC) {
		t.Errorf("UnmarshalJSON time = %v, want %v", nt.Time, wantUTC)
	}
}

func TestNullableCSTTimeIsNil(t *testing.T) {
	nt1 := NullableCSTTime{Valid: false}
	if !nt1.IsNil() {
		t.Error("IsNil should return true when Valid=false")
	}

	nt2 := NullableCSTTime{Time: nil, Valid: true}
	if !nt2.IsNil() {
		t.Error("IsNil should return true when Time=nil")
	}

	now := time.Now()
	nt3 := NullableCSTTime{Time: &now, Valid: true}
	if nt3.IsNil() {
		t.Error("IsNil should return false when Time is set")
	}
}

// ---- 结构体嵌套测试 ----

type sampleModel struct {
	ID        string         `json:"id"`
	CreatedAt CSTTime        `json:"created_at"`
	ExpiresAt NullableCSTTime `json:"expires_at"`
}

func TestStructJSONMarshal(t *testing.T) {
	utc := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	expires := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)

	m := sampleModel{
		ID:        "test_001",
		CreatedAt: NewCSTTime(utc),
		ExpiresAt: NewNullableCSTTime(&expires),
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// 验证输出格式
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal result failed: %v", err)
	}

	createdAt, ok := result["created_at"].(string)
	if !ok {
		t.Fatal("created_at should be string")
	}
	if createdAt != "2026-06-15T18:00:00+08:00" {
		t.Errorf("created_at = %s, want 2026-06-15T18:00:00+08:00", createdAt)
	}

	expiresAt, ok := result["expires_at"].(string)
	if !ok {
		t.Fatal("expires_at should be string")
	}
	// 2026-12-31 23:59:59 UTC = 2027-01-01 07:59:59 CST
	if expiresAt != "2027-01-01T07:59:59+08:00" {
		t.Errorf("expires_at = %s, want 2027-01-01T07:59:59+08:00", expiresAt)
	}
}

func TestStructJSONMarshalNullField(t *testing.T) {
	m := sampleModel{
		ID:        "test_002",
		CreatedAt: NowCST(),
		ExpiresAt: NullableCSTTime{Valid: false},
	}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal result failed: %v", err)
	}

	if result["expires_at"] != nil {
		t.Errorf("expires_at should be null, got %v", result["expires_at"])
	}
}

// ---- Scanner/Valuer ----

func TestCSTTimeScanTimeValue(t *testing.T) {
	utc := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	var cst CSTTime

	if err := cst.Scan(utc); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if cst.Time.In(time.UTC).Hour() != 10 {
		t.Errorf("Scan should preserve UTC hour, got %d", cst.Time.UTC().Hour())
	}
}

func TestCSTTimeScanNil(t *testing.T) {
	var cst CSTTime

	if err := cst.Scan(nil); err != nil {
		t.Fatalf("Scan nil failed: %v", err)
	}

	if !cst.Time.IsZero() {
		t.Errorf("Scan nil should be zero time, got %v", cst.Time)
	}
}

func TestCSTTimeValueZero(t *testing.T) {
	cst := CSTTime{}
	v, err := cst.Value()
	if err != nil {
		t.Fatalf("Value failed: %v", err)
	}
	if v != nil {
		t.Errorf("Value zero should be nil, got %v", v)
	}
}

func TestCSTTimeValueNonZero(t *testing.T) {
	cst := NowCST()
	v, err := cst.Value()
	if err != nil {
		t.Fatalf("Value failed: %v", err)
	}
	if v == nil {
		t.Error("Value non-zero should not be nil")
	}
}

func TestNullableCSTTimeScanNil(t *testing.T) {
	var nt NullableCSTTime
	if err := nt.Scan(nil); err != nil {
		t.Fatalf("Scan nil failed: %v", err)
	}
	if nt.Valid {
		t.Error("Scan nil should set Valid=false")
	}
}

func TestNullableCSTTimeValueNil(t *testing.T) {
	nt := NullableCSTTime{Valid: false}
	v, err := nt.Value()
	if err != nil {
		t.Fatalf("Value failed: %v", err)
	}
	if v != nil {
		t.Errorf("Value nil should be nil, got %v", v)
	}
}
