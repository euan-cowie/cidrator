package dns

import (
	"errors"
	"fmt"
)

// Sentinel errors for DNS operations
var (
	ErrEmptyDomain = errors.New("domain cannot be empty")
	ErrEmptyIP     = errors.New("IP address cannot be empty")
	ErrInvalidIP   = errors.New("invalid IP address format")
	ErrNXDomain    = errors.New("domain does not exist (NXDOMAIN)")
	ErrTimeout     = errors.New("DNS query timed out")
)

// DNSError represents a DNS operation error with context
type DNSError struct {
	Operation string
	Target    string
	Err       error
}

func (e *DNSError) Error() string {
	return fmt.Sprintf("dns %s error for %q: %v", e.Operation, e.Target, e.Err)
}

func (e *DNSError) Unwrap() error {
	return e.Err
}

// NewDNSError creates a new DNSError
func NewDNSError(operation, target string, err error) *DNSError {
	return &DNSError{
		Operation: operation,
		Target:    target,
		Err:       err,
	}
}
