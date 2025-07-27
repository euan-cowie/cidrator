package mtu

import (
	"crypto/rand"
	"math/big"
	"sync"
	"time"
)

// RateLimiter controls the rate of packet sending
type RateLimiter struct {
	packetsPerSecond int
	lastSent         time.Time
	mutex            sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(pps int) *RateLimiter {
	return &RateLimiter{
		packetsPerSecond: pps,
		lastSent:         time.Now(),
	}
}

// Wait blocks until it's safe to send the next packet
func (rl *RateLimiter) Wait() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	if rl.packetsPerSecond <= 0 {
		return // No rate limiting
	}

	minInterval := time.Second / time.Duration(rl.packetsPerSecond)
	elapsed := time.Since(rl.lastSent)

	if elapsed < minInterval {
		time.Sleep(minInterval - elapsed)
	}

	rl.lastSent = time.Now()
}

// PacketRandomizer provides security through randomization
type PacketRandomizer struct {
	useRandomID   bool
	useRandomSeq  bool
	useRandomData bool
}

// NewPacketRandomizer creates a new packet randomizer
func NewPacketRandomizer() *PacketRandomizer {
	return &PacketRandomizer{
		useRandomID:   true,
		useRandomSeq:  true,
		useRandomData: true,
	}
}

// GenerateRandomID returns a random packet ID
func (pr *PacketRandomizer) GenerateRandomID() int {
	if !pr.useRandomID {
		return 1 // Static ID
	}

	id, _ := rand.Int(rand.Reader, big.NewInt(65536))
	return int(id.Int64())
}

// GenerateRandomSeq returns a random sequence number
func (pr *PacketRandomizer) GenerateRandomSeq() int {
	if !pr.useRandomSeq {
		return 1 // Static sequence
	}

	seq, _ := rand.Int(rand.Reader, big.NewInt(65536))
	return int(seq.Int64())
}

// GenerateRandomPayload creates randomized payload data
func (pr *PacketRandomizer) GenerateRandomPayload(size int) []byte {
	if !pr.useRandomData {
		// Use predictable pattern for debugging
		payload := make([]byte, size)
		for i := range payload {
			payload[i] = byte(i % 256)
		}
		return payload
	}

	// Generate cryptographically random payload
	payload := make([]byte, size)
	if _, err := rand.Read(payload); err != nil {
		// Fallback to a simple pattern if crypto/rand fails
		for i := range payload {
			payload[i] = byte(i % 256)
		}
	}
	return payload
}

// RetryThrottler manages retry attempts to avoid overwhelming networks
type RetryThrottler struct {
	maxRetries      int
	baseDelay       time.Duration
	maxDelay        time.Duration
	backoffFactor   float64
	currentAttempt  int
	lastAttemptTime time.Time
	mutex           sync.Mutex
}

// NewRetryThrottler creates a new retry throttler
func NewRetryThrottler(maxRetries int, baseDelay time.Duration) *RetryThrottler {
	return &RetryThrottler{
		maxRetries:    maxRetries,
		baseDelay:     baseDelay,
		maxDelay:      time.Second * 10, // Cap at 10 seconds
		backoffFactor: 2.0,              // Exponential backoff
	}
}

// ShouldRetry determines if another retry attempt is allowed
func (rt *RetryThrottler) ShouldRetry() bool {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()

	return rt.currentAttempt < rt.maxRetries
}

// WaitForRetry implements exponential backoff with jitter
func (rt *RetryThrottler) WaitForRetry() {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()

	if rt.currentAttempt == 0 {
		rt.currentAttempt++
		rt.lastAttemptTime = time.Now()
		return
	}

	// Calculate delay with exponential backoff
	delay := time.Duration(float64(rt.baseDelay) *
		func(base float64, exp int) float64 {
			result := 1.0
			for i := 0; i < exp; i++ {
				result *= base
			}
			return result
		}(rt.backoffFactor, rt.currentAttempt-1))

	if delay > rt.maxDelay {
		delay = rt.maxDelay
	}

	// Add jitter (±25%)
	jitter, _ := rand.Int(rand.Reader, big.NewInt(int64(delay/2)))
	delay = delay + time.Duration(jitter.Int64()) - delay/4

	time.Sleep(delay)
	rt.currentAttempt++
	rt.lastAttemptTime = time.Now()
}

// Reset resets the retry counter
func (rt *RetryThrottler) Reset() {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()

	rt.currentAttempt = 0
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	RateLimiter      *RateLimiter
	Randomizer       *PacketRandomizer
	RetryThrottler   *RetryThrottler
	EnableThreatLogs bool
}

// NewSecurityConfig creates a new security configuration
func NewSecurityConfig(pps int) *SecurityConfig {
	return &SecurityConfig{
		RateLimiter:      NewRateLimiter(pps),
		Randomizer:       NewPacketRandomizer(),
		RetryThrottler:   NewRetryThrottler(3, time.Millisecond*500),
		EnableThreatLogs: false, // Disable by default to avoid log spam
	}
}

// LogSecurityEvent logs security-related events if enabled
func (sc *SecurityConfig) LogSecurityEvent(event string) {
	if sc.EnableThreatLogs {
		// In a real implementation, this would log to syslog or structured logger
		// For now, we'll just track it internally
		_ = event
	}
}
