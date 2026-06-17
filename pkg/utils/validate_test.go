package utils

import "testing"

func TestValidatorRequired(t *testing.T) {
	v := NewValidator()
	v.Required("name", "")
	errs := v.Validate()
	if errs == nil {
		t.Error("expected error for empty required field")
	}
	if errs[0].Field != "name" {
		t.Errorf("field = %s, want name", errs[0].Field)
	}
}

func TestValidatorRequiredPass(t *testing.T) {
	v := NewValidator()
	v.Required("name", "hello")
	errs := v.Validate()
	if errs != nil {
		t.Errorf("unexpected errors: %v", errs)
	}
}

func TestValidatorMaxLen(t *testing.T) {
	v := NewValidator()
	v.MaxLen("desc", "hello world", 5)
	errs := v.Validate()
	if errs == nil {
		t.Error("expected error for exceeding max length")
	}
}

func TestValidatorMaxLenPass(t *testing.T) {
	v := NewValidator()
	v.MaxLen("desc", "hi", 5)
	errs := v.Validate()
	if errs != nil {
		t.Errorf("unexpected errors: %v", errs)
	}
}

func TestValidatorMinLen(t *testing.T) {
	v := NewValidator()
	v.MinLen("password", "ab", 8)
	errs := v.Validate()
	if errs == nil {
		t.Error("expected error for below min length")
	}
}

func TestValidatorRange(t *testing.T) {
	v := NewValidator()
	v.Range("confidence", 1.5, 0, 1)
	errs := v.Validate()
	if errs == nil {
		t.Error("expected error for out of range")
	}
}

func TestValidatorRangePass(t *testing.T) {
	v := NewValidator()
	v.Range("confidence", 0.8, 0, 1)
	errs := v.Validate()
	if errs != nil {
		t.Errorf("unexpected errors: %v", errs)
	}
}

func TestValidatorPattern(t *testing.T) {
	v := NewValidator()
	v.Pattern("email", "not-an-email", `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`, "邮箱格式无效")
	errs := v.Validate()
	if errs == nil {
		t.Error("expected error for invalid pattern")
	}
}

func TestValidatorOneOf(t *testing.T) {
	v := NewValidator()
	v.OneOf("role", "admin", []string{"owner", "member", "guest"})
	errs := v.Validate()
	if errs == nil {
		t.Error("expected error for invalid enum value")
	}
}

func TestValidatorOneOfPass(t *testing.T) {
	v := NewValidator()
	v.OneOf("role", "owner", []string{"owner", "member", "guest"})
	errs := v.Validate()
	if errs != nil {
		t.Errorf("unexpected errors: %v", errs)
	}
}

func TestValidatorChain(t *testing.T) {
	v := NewValidator()
	v.Required("name", "").
		MinLen("name", "", 3).
		MaxLen("name", "very long name that exceeds limit", 10)
	errs := v.Validate()
	if len(errs) != 3 {
		t.Errorf("errors count = %d, want 3", len(errs))
	}
}

func TestValidatePagination(t *testing.T) {
	tests := []struct {
	 pageNum, pageSize, wantPN, wantPS int
	}{
		{0, 0, 1, 20},
		{-1, 100, 1, 50},
		{3, 10, 3, 10},
		{1, 50, 1, 50},
	}
	for _, tt := range tests {
		pn, ps := ValidatePagination(tt.pageNum, tt.pageSize)
		if pn != tt.wantPN || ps != tt.wantPS {
			t.Errorf("ValidatePagination(%d, %d) = (%d, %d), want (%d, %d)",
				tt.pageNum, tt.pageSize, pn, ps, tt.wantPN, tt.wantPS)
		}
	}
}

func TestValidateConfidence(t *testing.T) {
	if err := ValidateConfidence("confidence", 0.8); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := ValidateConfidence("confidence", 1.5); err == nil {
		t.Error("expected error for confidence > 1")
	}
	if err := ValidateConfidence("confidence", -0.1); err == nil {
		t.Error("expected error for confidence < 0")
	}
}

func TestValidationErrorMessages(t *testing.T) {
	v := NewValidator()
	v.Required("name", "")
	errs := v.Validate()
	if errs.Error() == "" {
		t.Error("error message should not be empty")
	}
}
