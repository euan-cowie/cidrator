package mtu

import (
	"sync"
	"testing"
	"time"
)

// TestRateLimiter tests the rate limiting functionality
func TestRateLimiter(t *testing.T) {
	tests := []struct {
		name string
		pps  int
	}{
		{"5 PPS", 5},
		{"10 PPS", 10},
		{"100 PPS", 100},
		{"unlimited", 0}, // 0 means no rate limiting
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := NewRateLimiter(tt.pps)

			if limiter == nil {
				t.Fatalf("expected rate limiter, got nil")
			}

			if limiter.packetsPerSecond != tt.pps {
				t.Errorf("PPS mismatch: got %d, want %d", limiter.packetsPerSecond, tt.pps)
			}

			// Test that Wait() doesn't panic
			limiter.Wait()
			limiter.Wait()
		})
	}
}

// TestRateLimiterTiming tests rate limiter timing accuracy
func TestRateLimiterTiming(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timing test in short mode")
	}

	limiter := NewRateLimiter(2) // 2 packets per second

	start := time.Now()

	// Send 3 packets - should take at least 1 second due to rate limiting
	limiter.Wait() // First packet - should be immediate
	limiter.Wait() // Second packet - should wait 0.5s
	limiter.Wait() // Third packet - should wait another 0.5s

	elapsed := time.Since(start)

	// Should take at least 1 second for 3 packets at 2 PPS
	expectedMin := 1 * time.Second
	if elapsed < expectedMin {
		t.Errorf("rate limiting too fast: took %v, expected at least %v", elapsed, expectedMin)
	}

	// But shouldn't take too much longer (allow some tolerance for system load)
	expectedMax := 2000 * time.Millisecond // Increased tolerance
	if elapsed > expectedMax {
		t.Errorf("rate limiting too slow: took %v, expected at most %v", elapsed, expectedMax)
	}
}

// TestRateLimiterConcurrency tests rate limiter thread safety
func TestRateLimiterConcurrency(t *testing.T) {
	limiter := NewRateLimiter(10)

	var wg sync.WaitGroup
	numGoroutines := 5

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				limiter.Wait()
			}
		}()
	}

	// Should not deadlock or panic
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(10 * time.Second):
		t.Errorf("rate limiter concurrency test timed out")
	}
}

// TestPacketRandomizer tests packet randomization functionality
func TestPacketRandomizer(t *testing.T) {
	randomizer := NewPacketRandomizer()

	if randomizer == nil {
		t.Fatalf("expected packet randomizer, got nil")
	}

	// Test that randomization is enabled by default
	if !randomizer.useRandomID {
		t.Errorf("expected random ID to be enabled")
	}

	if !randomizer.useRandomSeq {
		t.Errorf("expected random sequence to be enabled")
	}

	if !randomizer.useRandomData {
		t.Errorf("expected random data to be enabled")
	}
}

// TestGenerateRandomID tests random ID generation
func TestGenerateRandomID(t *testing.T) {
	randomizer := NewPacketRandomizer()

	// Generate multiple IDs and check they're different
	ids := make(map[int]bool)
	numIds := 100

	for i := 0; i < numIds; i++ {
		id := randomizer.GenerateRandomID()

		// ID should be in valid range
		if id < 0 || id >= 65536 {
			t.Errorf("ID out of range: %d", id)
		}

		ids[id] = true
	}

	// Should have generated some variety (at least 80% unique)
	uniqueCount := len(ids)
	minUnique := int(float64(numIds) * 0.8)
	if uniqueCount < minUnique {
		t.Errorf("insufficient randomness in IDs: got %d unique out of %d", uniqueCount, numIds)
	}
}

// TestGenerateRandomSeq tests random sequence generation
func TestGenerateRandomSeq(t *testing.T) {
	randomizer := NewPacketRandomizer()

	// Generate multiple sequences and check they're different
	seqs := make(map[int]bool)
	numSeqs := 100

	for i := 0; i < numSeqs; i++ {
		seq := randomizer.GenerateRandomSeq()

		// Sequence should be in valid range
		if seq < 0 || seq >= 65536 {
			t.Errorf("sequence out of range: %d", seq)
		}

		seqs[seq] = true
	}

	// Should have generated some variety (at least 80% unique)
	uniqueCount := len(seqs)
	minUnique := int(float64(numSeqs) * 0.8)
	if uniqueCount < minUnique {
		t.Errorf("insufficient randomness in sequences: got %d unique out of %d", uniqueCount, numSeqs)
	}
}

// TestGenerateRandomPayload tests random payload generation
func TestGenerateRandomPayload(t *testing.T) {
	randomizer := NewPacketRandomizer()

	tests := []struct {
		name string
		size int
	}{
		{"small payload", 64},
		{"medium payload", 512},
		{"large payload", 1400},
		{"zero payload", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := randomizer.GenerateRandomPayload(tt.size)

			if len(payload) != tt.size {
				t.Errorf("payload size mismatch: got %d, want %d", len(payload), tt.size)
			}

			if tt.size > 0 {
				// Check that two payloads are different (very high probability)
				payload2 := randomizer.GenerateRandomPayload(tt.size)
				if string(payload) == string(payload2) {
					t.Errorf("two random payloads are identical (very unlikely)")
				}
			}
		})
	}
}

// TestNonRandomPayload tests predictable payload generation
func TestNonRandomPayload(t *testing.T) {
	randomizer := &PacketRandomizer{
		useRandomData: false,
	}

	size := 100
	payload1 := randomizer.GenerateRandomPayload(size)
	payload2 := randomizer.GenerateRandomPayload(size)

	// Non-random payloads should be identical
	if string(payload1) != string(payload2) {
		t.Errorf("non-random payloads should be identical")
	}

	// Should follow predictable pattern
	for i, b := range payload1 {
		expected := byte(i % 256)
		if b != expected {
			t.Errorf("byte %d: got %d, want %d", i, b, expected)
		}
	}
}

// TestRetryThrottler tests retry throttling functionality
func TestRetryThrottler(t *testing.T) {
	throttler := NewRetryThrottler(3, 100*time.Millisecond)

	if throttler == nil {
		t.Fatalf("expected retry throttler, got nil")
	}

	if throttler.maxRetries != 3 {
		t.Errorf("max retries mismatch: got %d, want %d", throttler.maxRetries, 3)
	}

	if throttler.baseDelay != 100*time.Millisecond {
		t.Errorf("base delay mismatch: got %v, want %v", throttler.baseDelay, 100*time.Millisecond)
	}
}

// TestRetryThrottlerLogic tests retry logic
func TestRetryThrottlerLogic(t *testing.T) {
	throttler := NewRetryThrottler(3, 10*time.Millisecond)

	// Should allow initial retries
	for i := 0; i < 3; i++ {
		if !throttler.ShouldRetry() {
			t.Errorf("should allow retry %d", i)
		}
		throttler.WaitForRetry()
	}

	// Should not allow more retries after limit
	if throttler.ShouldRetry() {
		t.Errorf("should not allow retry after limit")
	}

	// Reset should allow retries again
	throttler.Reset()
	if !throttler.ShouldRetry() {
		t.Errorf("should allow retry after reset")
	}
}

// TestRetryThrottlerBackoff tests exponential backoff
func TestRetryThrottlerBackoff(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping backoff timing test in short mode")
	}

	throttler := NewRetryThrottler(3, 50*time.Millisecond)

	// First call should be fast
	start := time.Now()
	throttler.WaitForRetry()
	elapsed1 := time.Since(start)

	// Second call should take at least the base delay
	start = time.Now()
	throttler.WaitForRetry()
	elapsed2 := time.Since(start)

	// Third call should take longer (exponential backoff)
	start = time.Now()
	throttler.WaitForRetry()
	elapsed3 := time.Since(start)

	// First call should be nearly instant
	if elapsed1 > 10*time.Millisecond {
		t.Errorf("first retry too slow: %v", elapsed1)
	}

	// Subsequent calls should show increasing delays
	if elapsed2 < 40*time.Millisecond {
		t.Errorf("second retry too fast: %v", elapsed2)
	}

	if elapsed3 <= elapsed2 {
		t.Errorf("exponential backoff not working: %v <= %v", elapsed3, elapsed2)
	}
}

// TestRetryThrottlerConcurrency tests retry throttler thread safety
func TestRetryThrottlerConcurrency(t *testing.T) {
	// Use a very short delay and reasonable retry count for faster test
	throttler := NewRetryThrottler(3, 1*time.Millisecond)

	var wg sync.WaitGroup
	numGoroutines := 3
	completed := make(chan int, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			retryCount := 0
			// Limit the number of retries to prevent infinite loops
			for retryCount < 5 && throttler.ShouldRetry() {
				throttler.WaitForRetry()
				retryCount++
			}
			completed <- id
		}(i)
	}

	// Should not deadlock or panic - use shorter timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Verify all goroutines completed
		close(completed)
		completedCount := 0
		for range completed {
			completedCount++
		}
		if completedCount != numGoroutines {
			t.Errorf("expected %d goroutines to complete, got %d", numGoroutines, completedCount)
		}
	case <-time.After(2 * time.Second):
		t.Errorf("retry throttler concurrency test timed out")
	}
}

// TestSecurityConfigCreation tests security configuration creation
func TestSecurityConfigCreation(t *testing.T) {
	config := NewSecurityConfig(15)

	if config == nil {
		t.Fatalf("expected security config, got nil")
	}

	if config.RateLimiter == nil {
		t.Errorf("expected rate limiter, got nil")
	}

	if config.Randomizer == nil {
		t.Errorf("expected randomizer, got nil")
	}

	if config.RetryThrottler == nil {
		t.Errorf("expected retry throttler, got nil")
	}

	// Check rate limiter configuration
	if config.RateLimiter.packetsPerSecond != 15 {
		t.Errorf("rate limiter PPS: got %d, want %d", config.RateLimiter.packetsPerSecond, 15)
	}

	// Check default settings
	if config.EnableThreatLogs {
		t.Errorf("threat logs should be disabled by default")
	}
}

// TestSecurityConfigLogEvent tests security event logging
func TestSecurityConfigLogEvent(t *testing.T) {
	// Test with logging disabled (default)
	config := NewSecurityConfig(10)
	config.LogSecurityEvent("test event") // Should not panic

	// Test with logging enabled
	config.EnableThreatLogs = true
	config.LogSecurityEvent("test event") // Should not panic
}

// TestZeroRateLimit tests behavior with zero rate limit (unlimited)
func TestZeroRateLimit(t *testing.T) {
	limiter := NewRateLimiter(0)

	start := time.Now()

	// Multiple rapid calls should be fast with no rate limiting
	for i := 0; i < 10; i++ {
		limiter.Wait()
	}

	elapsed := time.Since(start)

	// Should be very fast (under 10ms)
	if elapsed > 10*time.Millisecond {
		t.Errorf("zero rate limit too slow: %v", elapsed)
	}
}

// TestMaxDelayCap tests that retry delays are capped
func TestMaxDelayCap(t *testing.T) {
	// Use very high retry count to test max delay cap
	throttler := NewRetryThrottler(20, 1*time.Second)
	throttler.maxDelay = 100 * time.Millisecond // Set low max for testing

	// Skip to high retry count
	for i := 0; i < 10; i++ {
		throttler.WaitForRetry()
	}

	// Next retry should be capped at maxDelay
	start := time.Now()
	throttler.WaitForRetry()
	elapsed := time.Since(start)

	// Should be close to maxDelay, not exponentially larger
	if elapsed > 200*time.Millisecond {
		t.Errorf("delay not properly capped: got %v, max should be ~100ms", elapsed)
	}
}

// Benchmark tests for performance validation
func BenchmarkRateLimiterWait(b *testing.B) {
	limiter := NewRateLimiter(1000) // High rate to minimize waiting

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Wait()
	}
}

func BenchmarkGenerateRandomID(b *testing.B) {
	randomizer := NewPacketRandomizer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		randomizer.GenerateRandomID()
	}
}

func BenchmarkGenerateRandomPayload(b *testing.B) {
	randomizer := NewPacketRandomizer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		randomizer.GenerateRandomPayload(1400)
	}
}

func BenchmarkRetryThrottlerShouldRetry(b *testing.B) {
	throttler := NewRetryThrottler(1000, 1*time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		throttler.ShouldRetry()
	}
}
