package utils

import "testing"

func TestEscapeXSS(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{`<script>alert("xss")</script>`, `&lt;script&gt;alert(&#34;xss&#34;)&lt;/script&gt;`},
		{"hello", "hello"},
		{`<img onerror="alert(1)">`, "&lt;img onerror=&#34;alert(1)&#34;&gt;"},
	}
	for _, tt := range tests {
		got := EscapeXSS(tt.input)
		if got != tt.want {
			t.Errorf("EscapeXSS(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSanitizeHTML(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"<b>hello</b>", "hello"},
		{`<script>alert("xss")</script>`, "alert(&#34;xss&#34;)"},
		{"plain text", "plain text"},
	}
	for _, tt := range tests {
		got := SanitizeHTML(tt.input)
		if got != tt.want {
			t.Errorf("SanitizeHTML(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"  hello  ", 0, "hello"},
		{"hello\x00world", 0, "helloworld"},
		{"very long text", 5, "very "},
		{"", 0, ""},
	}
	for _, tt := range tests {
		got := SanitizeInput(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("SanitizeInput(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

func TestDetectSQLInjection(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"normal input", false},
		{"' OR 1=1 --", true},
		{"UNION SELECT * FROM users", true},
		{"admin'--", true},
		{"hello world", false},
		{"1; DROP TABLE users", true},
	}
	for _, tt := range tests {
		got := DetectSQLInjection(tt.input)
		if got != tt.want {
			t.Errorf("DetectSQLInjection(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestSanitizeFieldName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"user_name", "user_name"},
		{"user-name", "username"},
		{"user; DROP TABLE", "userDROPTABLE"},
		{"validField123", "validField123"},
	}
	for _, tt := range tests {
		got := SanitizeFieldName(tt.input)
		if got != tt.want {
			t.Errorf("SanitizeFieldName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDetectXSS(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"normal text", false},
		{`<script>alert("xss")</script>`, true},
		{`<img onerror="alert(1)">`, true},
		{`javascript:alert(1)`, true},
		{`<iframe src="evil">`, true},
		{"hello world", false},
	}
	for _, tt := range tests {
		got := DetectXSS(tt.input)
		if got != tt.want {
			t.Errorf("DetectXSS(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
