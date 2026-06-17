package utils

import "testing"

func TestMaskPhone(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"13812345678", "138****5678"},
		{"15000001111", "150****1111"},
		{"123", "123"},
		{"", ""},
	}
	for _, tt := range tests {
		got := MaskPhone(tt.input)
		if got != tt.want {
			t.Errorf("MaskPhone(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMaskIDCard(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"110101199001011234", "110101**********34"},
		{"123", "123"},
	}
	for _, tt := range tests {
		got := MaskIDCard(tt.input)
		if got != tt.want {
			t.Errorf("MaskIDCard(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMaskBankCard(t *testing.T) {
	got := MaskBankCard("6222021234567890123")
	want := "****0123"
	if got != want {
		t.Errorf("MaskBankCard = %q, want %q", got, want)
	}
}

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"test@example.com", "t***@example.com"},
		{"a@b.cn", "***"},
		{"@", "***"},
	}
	for _, tt := range tests {
		got := MaskEmail(tt.input)
		if got != tt.want {
			t.Errorf("MaskEmail(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMaskAddress(t *testing.T) {
	got := MaskAddress("北京市朝阳区某某街道123号")
	if got != "北京***" {
		t.Errorf("MaskAddress = %q", got)
	}
}

func TestMaskName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"张三", "张*"},
		{"张三丰", "张*丰"},
		{"李", "*"},
	}
	for _, tt := range tests {
		got := MaskName(tt.input)
		if got != tt.want {
			t.Errorf("MaskName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMaskMedicalData(t *testing.T) {
	// Uses byte indexing: first 2 bytes + *** + last 2 bytes
	got := MaskMedicalData("血糖值偏高需要关注")
	if len(got) == 0 {
		t.Error("MaskMedicalData returned empty")
	}
	// Verify masking happened
	if got == "血糖值偏高需要关注" {
		t.Error("should be masked")
	}
}

func TestMaskSensitiveText(t *testing.T) {
	text := "用户手机号13812345678，邮箱test@example.com，身份证110101199001011234"
	result := MaskSensitiveText(text)

	if contains(result, "13812345678") {
		t.Error("phone should be masked")
	}
	if contains(result, "test@example.com") {
		t.Error("email should be masked")
	}
	if contains(result, "110101199001011234") {
		t.Error("ID card should be masked")
	}
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestMaskToken(t *testing.T) {
	got := MaskToken("abcdefghijklmnop")
	if got != "abcdefgh********" {
		t.Errorf("MaskToken = %q", got)
	}
}

func TestRemovePassword(t *testing.T) {
	got := RemovePassword("secret123")
	if got != "" {
		t.Errorf("RemovePassword = %q, want empty", got)
	}
}
