//go:build darwin

package mtu

import (
	"errors"
	"testing"

	"golang.org/x/net/route"
)

func TestGetInterfaceTypeFromOSDarwinErrorPaths(t *testing.T) {
	originalFetch := fetchDarwinRouteRIB
	originalParse := parseDarwinRouteRIB
	t.Cleanup(func() {
		fetchDarwinRouteRIB = originalFetch
		parseDarwinRouteRIB = originalParse
	})

	t.Run("returns false when rib fetch fails", func(t *testing.T) {
		fetchDarwinRouteRIB = func() ([]byte, error) {
			return nil, errors.New("fetch failed")
		}
		if typ, ok := getInterfaceTypeFromOS("en0"); ok || typ != "" {
			t.Fatalf("expected fetch failure result, got type=%q ok=%v", typ, ok)
		}
	})

	t.Run("returns false when rib parse fails", func(t *testing.T) {
		fetchDarwinRouteRIB = func() ([]byte, error) {
			return []byte("rib"), nil
		}
		parseDarwinRouteRIB = func(rib []byte) ([]route.Message, error) {
			return nil, errors.New("parse failed")
		}
		if typ, ok := getInterfaceTypeFromOS("en0"); ok || typ != "" {
			t.Fatalf("expected parse failure result, got type=%q ok=%v", typ, ok)
		}
	})

	t.Run("returns false when interface is not present", func(t *testing.T) {
		fetchDarwinRouteRIB = func() ([]byte, error) {
			return []byte("rib"), nil
		}
		parseDarwinRouteRIB = func(rib []byte) ([]route.Message, error) {
			return []route.Message{}, nil
		}
		if typ, ok := getInterfaceTypeFromOS("en0"); ok || typ != "" {
			t.Fatalf("expected missing interface result, got type=%q ok=%v", typ, ok)
		}
	})
}
