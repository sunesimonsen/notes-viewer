package auth

import (
	"sync"
	"time"
)

type TokenRateLimiter struct {
	mu       sync.Mutex
	cooldown time.Duration
	lastByIP map[string]time.Time
}

func NewTokenRateLimiter(cooldown time.Duration) *TokenRateLimiter {
	return &TokenRateLimiter{
		cooldown: cooldown,
		lastByIP: make(map[string]time.Time),
	}
}

func (l *TokenRateLimiter) Allow(key string, now time.Time) (bool, time.Duration) {
	if l == nil || key == "" {
		return true, 0
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if last, ok := l.lastByIP[key]; ok {
		remaining := l.cooldown - now.Sub(last)
		if remaining > 0 {
			return false, remaining
		}
	}

	l.lastByIP[key] = now
	return true, 0
}
