package validation

import (
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
)

// NetworkValidator provides validation methods for network-related inputs
type NetworkValidator struct{}

// NewNetworkValidator creates a new network validator
func NewNetworkValidator() *NetworkValidator {
	return &NetworkValidator{}
}

// ValidateCIDR validates a CIDR string format
func (v *NetworkValidator) ValidateCIDR(cidr string) error {
	if strings.TrimSpace(cidr) == "" {
		return fmt.Errorf("CIDR cannot be empty")
	}

	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR format '%s': %v", cidr, err)
	}

	return nil
}

// ValidateIP validates an IP address string format
func (v *NetworkValidator) ValidateIP(ip string) error {
	if strings.TrimSpace(ip) == "" {
		return fmt.Errorf("IP address cannot be empty")
	}

	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP address format '%s'", ip)
	}

	return nil
}

// ValidatePositiveInteger validates that a string represents a positive integer
func (v *NetworkValidator) ValidatePositiveInteger(value, fieldName string) (int, error) {
	if strings.TrimSpace(value) == "" {
		return 0, fmt.Errorf("%s cannot be empty", fieldName)
	}

	num, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s '%s': must be a number", fieldName, value)
	}

	if num <= 0 {
		return 0, fmt.Errorf("%s must be positive, got %d", fieldName, num)
	}

	return num, nil
}

// ValidateNonNegativeInteger validates that a string represents a non-negative integer
func (v *NetworkValidator) ValidateNonNegativeInteger(value, fieldName string) (int, error) {
	if strings.TrimSpace(value) == "" {
		return 0, fmt.Errorf("%s cannot be empty", fieldName)
	}

	num, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s '%s': must be a number", fieldName, value)
	}

	if num < 0 {
		return 0, fmt.Errorf("%s must be non-negative, got %d", fieldName, num)
	}

	return num, nil
}

// ValidateSubnetDivision validates that a CIDR can be divided into the specified number of parts
func (v *NetworkValidator) ValidateSubnetDivision(cidr string, parts int) error {
	if err := v.ValidateCIDR(cidr); err != nil {
		return err
	}

	if parts <= 0 {
		return fmt.Errorf("number of parts must be positive, got %d", parts)
	}

	// Check if we can actually divide the network
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("failed to parse CIDR: %v", err)
	}

	prefixLen, totalBits := network.Mask.Size()
	bitsNeeded := int(math.Ceil(math.Log2(float64(parts))))
	newPrefixLen := prefixLen + bitsNeeded

	if newPrefixLen > totalBits {
		return fmt.Errorf("cannot divide %s into %d parts: insufficient host bits", cidr, parts)
	}

	return nil
}

// ValidateOutputFormat validates output format strings
func (v *NetworkValidator) ValidateOutputFormat(format string) error {
	validFormats := []string{"table", "json", "yaml"}

	for _, validFormat := range validFormats {
		if format == validFormat {
			return nil
		}
	}

	return fmt.Errorf("invalid output format '%s': supported formats are %v", format, validFormats)
}

// ValidationRules defines common validation rules
type ValidationRules struct {
	validator *NetworkValidator
}

// NewValidationRules creates a new validation rules instance
func NewValidationRules() *ValidationRules {
	return &ValidationRules{
		validator: NewNetworkValidator(),
	}
}

// ValidateExplainInputs validates inputs for the explain command
func (r *ValidationRules) ValidateExplainInputs(cidr, format string) error {
	if err := r.validator.ValidateCIDR(cidr); err != nil {
		return fmt.Errorf("CIDR validation failed: %v", err)
	}

	if err := r.validator.ValidateOutputFormat(format); err != nil {
		return fmt.Errorf("format validation failed: %v", err)
	}

	return nil
}

// ValidateExpandInputs validates inputs for the expand command
func (r *ValidationRules) ValidateExpandInputs(cidr string, limit int) error {
	if err := r.validator.ValidateCIDR(cidr); err != nil {
		return fmt.Errorf("CIDR validation failed: %v", err)
	}

	if limit < 0 {
		return fmt.Errorf("limit must be non-negative, got %d", limit)
	}

	return nil
}

// ValidateContainsInputs validates inputs for the contains command
func (r *ValidationRules) ValidateContainsInputs(cidr, ip string) error {
	if err := r.validator.ValidateCIDR(cidr); err != nil {
		return fmt.Errorf("CIDR validation failed: %v", err)
	}

	if err := r.validator.ValidateIP(ip); err != nil {
		return fmt.Errorf("IP validation failed: %v", err)
	}

	return nil
}

// ValidateDivideInputs validates inputs for the divide command
func (r *ValidationRules) ValidateDivideInputs(cidr string, parts int) error {
	return r.validator.ValidateSubnetDivision(cidr, parts)
}
