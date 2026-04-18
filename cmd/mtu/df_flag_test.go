package mtu

import (
	"errors"
	"testing"
)

func TestDFError(t *testing.T) {
	rootErr := errors.New("socket option failed")
	err := &DFError{
		Protocol: "tcp",
		IPv6:     true,
		Err:      rootErr,
	}

	if err.Error() != "failed to set DF flag for tcp IPv6: socket option failed" {
		t.Fatalf("unexpected DFError string: %q", err.Error())
	}
	if !errors.Is(err, rootErr) {
		t.Fatalf("expected DFError to unwrap to the root cause")
	}
}
