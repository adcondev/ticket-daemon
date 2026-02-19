package server

import (
	"sync"
	"time"
)

// JobRateLimiter restricts how frequently a single client
// can submit print jobs via WebSocket.
type JobRateLimiter struct {
	mu        sync.Mutex
	attempts  map[string][]time.Time
	maxPerMin int
}

// NewJobRateLimiter creates a limiter allowing maxPerMinute jobs per client.
func NewJobRateLimiter(maxPerMinute int) *JobRateLimiter {
	return &JobRateLimiter{
		attempts:  make(map[string][]time.Time),
		maxPerMin: maxPerMinute,
	}
}

// Allow returns true if the client has not exceeded the rate limit.
func (rl *JobRateLimiter) Allow(clientAddr string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-time.Minute)

	recent := make([]time.Time, 0, rl.maxPerMin)
	for _, t := range rl.attempts[clientAddr] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= rl.maxPerMin {
		return false
	}

	rl.attempts[clientAddr] = append(recent, now)
	return true
}
