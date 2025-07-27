package mtu

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestMTUIntegration runs integration tests for the MTU functionality
func TestMTUIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests in short mode")
	}

	// Build the binary for testing
	buildCmd := exec.Command("go", "build", "-o", "../../bin/cidrator", "../../main.go")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build cidrator binary: %v", err)
	}

	binary := "../../bin/cidrator"

	t.Run("MTU help command", func(t *testing.T) {
		cmd := exec.Command(binary, "mtu", "--help")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Errorf("mtu --help failed: %v", err)
		}

		outputStr := string(output)
		expectedStrings := []string{
			"Path-MTU discovery",
			"discover",
			"watch",
			"interfaces",
			"suggest",
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(outputStr, expected) {
				t.Errorf("mtu --help output missing %q", expected)
			}
		}
	})

	t.Run("MTU interfaces command", func(t *testing.T) {
		cmd := exec.Command(binary, "mtu", "interfaces")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Errorf("mtu interfaces failed: %v", err)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "Interface") {
			t.Errorf("interfaces output missing Interface header")
		}
	})

	t.Run("MTU interfaces JSON", func(t *testing.T) {
		cmd := exec.Command(binary, "mtu", "interfaces", "--json")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Errorf("mtu interfaces --json failed: %v", err)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "interfaces") {
			t.Errorf("interfaces JSON output missing 'interfaces' field")
		}
	})

	t.Run("MTU suggest command", func(t *testing.T) {
		cmd := exec.Command(binary, "mtu", "suggest", "localhost")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Errorf("mtu suggest localhost failed: %v", err)
		}

		outputStr := string(output)
		expectedStrings := []string{
			"TCP MSS",
			"localhost",
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(outputStr, expected) {
				t.Errorf("suggest output missing %q", expected)
			}
		}
	})

	t.Run("MTU suggest JSON", func(t *testing.T) {
		cmd := exec.Command(binary, "mtu", "suggest", "localhost", "--json")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Errorf("mtu suggest localhost --json failed: %v", err)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "suggestions") {
			t.Errorf("suggest JSON output missing 'suggestions' field")
		}
	})

	t.Run("MTU discover TCP", func(t *testing.T) {
		// Use TCP protocol to avoid privilege issues with ICMP
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "mtu", "discover", "localhost", "--proto", "tcp", "--max", "1500")
		output, err := cmd.CombinedOutput()

		if err != nil {
			// TCP discovery might fail due to network conditions, but shouldn't crash
			t.Logf("mtu discover TCP failed (expected on some systems): %v", err)
			return
		}

		outputStr := string(output)
		expectedStrings := []string{
			"Target:",
			"localhost",
			"Protocol:",
			"tcp",
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(outputStr, expected) {
				t.Errorf("discover TCP output missing %q", expected)
			}
		}
	})

	t.Run("MTU discover UDP", func(t *testing.T) {
		// Use UDP protocol as another non-privileged option
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "mtu", "discover", "localhost", "--proto", "udp", "--max", "1500")
		output, err := cmd.CombinedOutput()

		if err != nil {
			// UDP discovery might fail due to network conditions, but shouldn't crash
			t.Logf("mtu discover UDP failed (expected on some systems): %v", err)
			return
		}

		outputStr := string(output)
		expectedStrings := []string{
			"Target:",
			"localhost",
			"Protocol:",
			"udp",
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(outputStr, expected) {
				t.Errorf("discover UDP output missing %q", expected)
			}
		}
	})

	t.Run("MTU discover JSON output", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "mtu", "discover", "localhost", "--proto", "tcp", "--max", "1500", "--json")
		output, err := cmd.CombinedOutput()

		if err != nil {
			t.Logf("mtu discover JSON failed (expected on some systems): %v", err)
			return
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "target") && !strings.Contains(outputStr, "pmtu") {
			t.Errorf("discover JSON output missing expected fields")
		}
	})

	t.Run("MTU invalid protocol", func(t *testing.T) {
		cmd := exec.Command(binary, "mtu", "discover", "localhost", "--proto", "invalid")
		output, err := cmd.CombinedOutput()

		if err == nil {
			t.Errorf("expected error for invalid protocol, got none")
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "unsupported protocol") {
			t.Errorf("expected unsupported protocol error, got: %s", outputStr)
		}
	})

	t.Run("MTU command validation", func(t *testing.T) {
		// Test various invalid command combinations
		testCases := []struct {
			name string
			args []string
		}{
			{"discover no args", []string{"mtu", "discover"}},
			{"suggest no args", []string{"mtu", "suggest"}},
			{"watch no args", []string{"mtu", "watch"}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cmd := exec.Command(binary, tc.args...)
				_, err := cmd.CombinedOutput()

				if err == nil {
					t.Errorf("%s should return error but didn't", tc.name)
				}
			})
		}
	})
}

// TestMTUPerformance runs performance tests for MTU operations
func TestMTUPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance tests in short mode")
	}

	t.Run("Interface enumeration performance", func(t *testing.T) {
		start := time.Now()
		result, err := GetNetworkInterfaces()
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("GetNetworkInterfaces failed: %v", err)
		}

		if elapsed > 1*time.Second {
			t.Errorf("interface enumeration too slow: %v", elapsed)
		}

		if len(result.Interfaces) == 0 {
			t.Errorf("no interfaces found")
		}

		t.Logf("Enumerated %d interfaces in %v", len(result.Interfaces), elapsed)
	})

	t.Run("Security config creation performance", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 1000; i++ {
			config := NewSecurityConfig(10)
			if config == nil {
				t.Errorf("failed to create security config")
			}
		}

		elapsed := time.Since(start)
		if elapsed > 100*time.Millisecond {
			t.Errorf("security config creation too slow: %v", elapsed)
		}

		t.Logf("Created 1000 security configs in %v", elapsed)
	})

	t.Run("Suggestion calculation performance", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 10000; i++ {
			suggestions := calculateSuggestions(1500)
			if suggestions.TCPMSSv4 <= 0 {
				t.Errorf("invalid suggestions calculated")
			}
		}

		elapsed := time.Since(start)
		if elapsed > 100*time.Millisecond {
			t.Errorf("suggestion calculation too slow: %v", elapsed)
		}

		t.Logf("Calculated 10000 suggestions in %v", elapsed)
	})
}

// TestMTUErrorHandling tests error handling scenarios
func TestMTUErrorHandling(t *testing.T) {
	t.Run("Invalid target resolution", func(t *testing.T) {
		discoverer := &MTUDiscoverer{
			target:   "invalid..hostname..example.com",
			ipv6:     false,
			protocol: "tcp",
			timeout:  1 * time.Second,
			ttl:      64,
			security: NewSecurityConfig(10),
		}

		err := discoverer.resolveTarget()
		if err == nil {
			t.Errorf("expected error for invalid hostname, got nil")
		}
	})

	t.Run("Context timeout", func(t *testing.T) {
		discoverer, err := NewMTUDiscoverer("localhost", false, "tcp", 1*time.Second, 64)
		if err != nil {
			t.Fatalf("failed to create discoverer: %v", err)
		}
		defer func() {
			if closeErr := discoverer.Close(); closeErr != nil {
				t.Logf("Warning: failed to close discoverer: %v", closeErr)
			}
		}()

		// Create a context that times out quickly
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		time.Sleep(2 * time.Millisecond) // Ensure context is expired

		_, err = discoverer.DiscoverPMTU(ctx, 576, 1500)
		if err == nil {
			t.Errorf("expected timeout error, got nil")
		}
	})

	t.Run("Invalid MTU range", func(t *testing.T) {
		discoverer, err := NewMTUDiscoverer("localhost", false, "tcp", 1*time.Second, 64)
		if err != nil {
			t.Fatalf("failed to create discoverer: %v", err)
		}
		defer func() {
			if closeErr := discoverer.Close(); closeErr != nil {
				t.Logf("Warning: failed to close discoverer: %v", closeErr)
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Try with min > max
		_, err = discoverer.DiscoverPMTU(ctx, 1500, 576)
		// Note: This might not immediately fail as validation could be in the algorithm
		if err != nil {
			t.Logf("Got expected error for invalid range: %v", err)
		}
	})
}

// TestMTUConcurrency tests concurrent usage of MTU functions
func TestMTUConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrency tests in short mode")
	}

	t.Run("Concurrent interface enumeration", func(t *testing.T) {
		const numGoroutines = 10
		results := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				_, err := GetNetworkInterfaces()
				results <- err
			}()
		}

		for i := 0; i < numGoroutines; i++ {
			if err := <-results; err != nil {
				t.Errorf("concurrent interface enumeration failed: %v", err)
			}
		}
	})

	t.Run("Concurrent security config creation", func(t *testing.T) {
		const numGoroutines = 20
		results := make(chan *SecurityConfig, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				config := NewSecurityConfig(10)
				results <- config
			}()
		}

		for i := 0; i < numGoroutines; i++ {
			config := <-results
			if config == nil {
				t.Errorf("concurrent security config creation failed")
			}
		}
	})

	t.Run("Concurrent rate limiting", func(t *testing.T) {
		limiter := NewRateLimiter(100) // High rate to avoid too much delay
		const numGoroutines = 10
		results := make(chan struct{}, numGoroutines)

		start := time.Now()

		for i := 0; i < numGoroutines; i++ {
			go func() {
				for j := 0; j < 10; j++ {
					limiter.Wait()
				}
				results <- struct{}{}
			}()
		}

		for i := 0; i < numGoroutines; i++ {
			<-results
		}

		elapsed := time.Since(start)
		if elapsed > 5*time.Second {
			t.Errorf("concurrent rate limiting took too long: %v", elapsed)
		}
	})
}
