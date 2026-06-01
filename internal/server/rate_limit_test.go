package server

import (
	"testing"
	"time"
)

func TestJobRateLimiter(t *testing.T) {
	// We set max to 3 requests per minute
	maxPerMin := 3
	rl := NewJobRateLimiter(maxPerMin)

	// Mock time
	currentTime := time.Now()
	rl.now = func() time.Time {
		return currentTime
	}

	clientAddr := "192.168.1.1:1234"

	// 1st request - should be allowed
	if !rl.Allow(clientAddr) {
		t.Errorf("Expected 1st request to be allowed")
	}

	// 2nd request - should be allowed
	if !rl.Allow(clientAddr) {
		t.Errorf("Expected 2nd request to be allowed")
	}

	// 3rd request - should be allowed
	if !rl.Allow(clientAddr) {
		t.Errorf("Expected 3rd request to be allowed")
	}

	// 4th request - should be denied (limit is 3)
	if rl.Allow(clientAddr) {
		t.Errorf("Expected 4th request to be denied")
	}

	// Fast forward time by 30 seconds
	currentTime = currentTime.Add(30 * time.Second)

	// Still denied
	if rl.Allow(clientAddr) {
		t.Errorf("Expected request after 30s to be denied")
	}

	// Fast forward time to clear the first request (1 minute and 1 second total)
	currentTime = currentTime.Add(31 * time.Second)

	// Now it should be allowed again
	if !rl.Allow(clientAddr) {
		t.Errorf("Expected request after 1m1s to be allowed")
	}

	// Another client should not be affected
	if !rl.Allow("192.168.1.2:5678") {
		t.Errorf("Expected request from different client to be allowed")
	}
}

func TestJobRateLimiter_FallbackTime(t *testing.T) {
	rl := NewJobRateLimiter(3)
	rl.now = nil // Force the nil branch in Allow

	if !rl.Allow("127.0.0.1") {
		t.Errorf("Expected request to be allowed with fallback time")
	}
}
