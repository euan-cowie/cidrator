package cidr

import (
	"errors"
	"fmt"
)

// CIDRError represents a CIDR-specific error with operation context
type CIDRError struct {
	Op   string // Operation that failed (parse, expand, divide, etc.)
	CIDR string // CIDR string that caused the error
	Err  error  // Underlying error
}

func (e *CIDRError) Error() string {
	if e.CIDR != "" {
		return fmt.Sprintf("cidr %s failed for %s: %v", e.Op, e.CIDR, e.Err)
	}
	return fmt.Sprintf("cidr %s failed: %v", e.Op, e.Err)
}

func (e *CIDRError) Unwrap() error {
	return e.Err
}

// ValidationError represents input validation failures
type ValidationError struct {
	Field string // Field that failed validation
	Value string // Value that was invalid
	Err   error  // Underlying validation error
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for %s '%s': %v", e.Field, e.Value, e.Err)
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

// Common error variables
var (
	ErrInvalidCIDR   = errors.New("invalid CIDR format")
	ErrInvalidIP     = errors.New("invalid IP address")
	ErrTooLarge      = errors.New("CIDR range too large for expansion")
	ErrInvalidParts  = errors.New("invalid number of parts")
	ErrInsufficientBits = errors.New("insufficient host bits for division")
)

// Error creation helpers

// NewCIDRError creates a new CIDRError with the specified operation and CIDR
func NewCIDRError(op, cidr string, err error) *CIDRError {
	return &CIDRError{
		Op:   op,
		CIDR: cidr,
		Err:  err,
	}
}

// NewValidationError creates a new ValidationError for the specified field
func NewValidationError(field, value string, err error) *ValidationError {
	return &ValidationError{
		Field: field,
		Value: value,
		Err:   err,
	}
}

// IsInvalidCIDR checks if an error is due to invalid CIDR format
func IsInvalidCIDR(err error) bool {
	var cidrErr *CIDRError
	if errors.As(err, &cidrErr) {
		return errors.Is(cidrErr.Err, ErrInvalidCIDR)
	}
	return errors.Is(err, ErrInvalidCIDR)
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	var valErr *ValidationError
	return errors.As(err, &valErr)
} 