package validation

import (
	"strings"
	"testing"
)

func TestNetworkValidator(t *testing.T) {
	validator := NewNetworkValidator()

	t.Run("ValidateCIDR", func(t *testing.T) {
		if err := validator.ValidateCIDR("192.168.1.0/24"); err != nil {
			t.Fatalf("expected valid CIDR, got %v", err)
		}
		if err := validator.ValidateCIDR(""); err == nil || !strings.Contains(err.Error(), "cannot be empty") {
			t.Fatalf("expected empty CIDR error, got %v", err)
		}
		if err := validator.ValidateCIDR("192.168.1.0"); err == nil || !strings.Contains(err.Error(), "invalid CIDR format") {
			t.Fatalf("expected invalid CIDR error, got %v", err)
		}
	})

	t.Run("ValidateIP", func(t *testing.T) {
		if err := validator.ValidateIP("2001:db8::1"); err != nil {
			t.Fatalf("expected valid IP, got %v", err)
		}
		if err := validator.ValidateIP(""); err == nil || !strings.Contains(err.Error(), "cannot be empty") {
			t.Fatalf("expected empty IP error, got %v", err)
		}
		if err := validator.ValidateIP("bad-ip"); err == nil || !strings.Contains(err.Error(), "invalid IP address format") {
			t.Fatalf("expected invalid IP error, got %v", err)
		}
	})

	t.Run("ValidatePositiveInteger", func(t *testing.T) {
		value, err := validator.ValidatePositiveInteger("8", "parts")
		if err != nil || value != 8 {
			t.Fatalf("expected valid positive integer, got value=%d err=%v", value, err)
		}
		if _, err := validator.ValidatePositiveInteger("", "parts"); err == nil || !strings.Contains(err.Error(), "cannot be empty") {
			t.Fatalf("expected empty value error, got %v", err)
		}
		if _, err := validator.ValidatePositiveInteger("abc", "parts"); err == nil || !strings.Contains(err.Error(), "must be a number") {
			t.Fatalf("expected numeric error, got %v", err)
		}
		if _, err := validator.ValidatePositiveInteger("0", "parts"); err == nil || !strings.Contains(err.Error(), "must be positive") {
			t.Fatalf("expected positive integer error, got %v", err)
		}
	})

	t.Run("ValidateNonNegativeInteger", func(t *testing.T) {
		value, err := validator.ValidateNonNegativeInteger("0", "limit")
		if err != nil || value != 0 {
			t.Fatalf("expected valid non-negative integer, got value=%d err=%v", value, err)
		}
		if _, err := validator.ValidateNonNegativeInteger("", "limit"); err == nil || !strings.Contains(err.Error(), "cannot be empty") {
			t.Fatalf("expected empty value error, got %v", err)
		}
		if _, err := validator.ValidateNonNegativeInteger("bad", "limit"); err == nil || !strings.Contains(err.Error(), "must be a number") {
			t.Fatalf("expected numeric error, got %v", err)
		}
		if _, err := validator.ValidateNonNegativeInteger("-1", "limit"); err == nil || !strings.Contains(err.Error(), "must be non-negative") {
			t.Fatalf("expected non-negative integer error, got %v", err)
		}
	})

	t.Run("ValidateSubnetDivision", func(t *testing.T) {
		if err := validator.ValidateSubnetDivision("192.168.1.0/24", 4); err != nil {
			t.Fatalf("expected valid subnet division, got %v", err)
		}
		if err := validator.ValidateSubnetDivision("192.168.1.0/24", 0); err == nil || !strings.Contains(err.Error(), "must be positive") {
			t.Fatalf("expected positive parts error, got %v", err)
		}
		if err := validator.ValidateSubnetDivision("192.168.1.0/30", 8); err == nil || !strings.Contains(err.Error(), "insufficient host bits") {
			t.Fatalf("expected insufficient host bits error, got %v", err)
		}
	})

	t.Run("ValidateOutputFormat", func(t *testing.T) {
		for _, format := range []string{"table", "json", "yaml"} {
			if err := validator.ValidateOutputFormat(format); err != nil {
				t.Fatalf("expected valid format %q, got %v", format, err)
			}
		}
		if err := validator.ValidateOutputFormat("xml"); err == nil || !strings.Contains(err.Error(), "invalid output format") {
			t.Fatalf("expected invalid format error, got %v", err)
		}
	})
}

func TestValidationRules(t *testing.T) {
	rules := NewValidationRules()

	if err := rules.ValidateExplainInputs("192.168.1.0/24", "json"); err != nil {
		t.Fatalf("expected valid explain inputs, got %v", err)
	}
	if err := rules.ValidateExplainInputs("bad", "json"); err == nil || !strings.Contains(err.Error(), "CIDR validation failed") {
		t.Fatalf("expected explain CIDR validation error, got %v", err)
	}
	if err := rules.ValidateExplainInputs("192.168.1.0/24", "xml"); err == nil || !strings.Contains(err.Error(), "format validation failed") {
		t.Fatalf("expected explain format validation error, got %v", err)
	}

	if err := rules.ValidateExpandInputs("10.0.0.0/8", 0); err != nil {
		t.Fatalf("expected valid expand inputs, got %v", err)
	}
	if err := rules.ValidateExpandInputs("10.0.0.0/8", -1); err == nil || !strings.Contains(err.Error(), "limit must be non-negative") {
		t.Fatalf("expected expand limit validation error, got %v", err)
	}

	if err := rules.ValidateContainsInputs("10.0.0.0/8", "10.0.0.1"); err != nil {
		t.Fatalf("expected valid contains inputs, got %v", err)
	}
	if err := rules.ValidateContainsInputs("bad", "10.0.0.1"); err == nil || !strings.Contains(err.Error(), "CIDR validation failed") {
		t.Fatalf("expected contains CIDR validation error, got %v", err)
	}
	if err := rules.ValidateContainsInputs("10.0.0.0/8", "bad"); err == nil || !strings.Contains(err.Error(), "IP validation failed") {
		t.Fatalf("expected contains IP validation error, got %v", err)
	}

	if err := rules.ValidateDivideInputs("192.168.1.0/24", 2); err != nil {
		t.Fatalf("expected valid divide inputs, got %v", err)
	}
	if err := rules.ValidateDivideInputs("192.168.1.0/30", 8); err == nil || !strings.Contains(err.Error(), "insufficient host bits") {
		t.Fatalf("expected divide validation error, got %v", err)
	}
}
