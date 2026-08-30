package auth_test

import (
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
	"github.com/sunesimonsen/notes-viewer/auth"
)

func TestTokenRateLimiterAllow(t *testing.T) {
	t.Run("allows when limiter is nil", func(t *testing.T) {
		var limiter *auth.TokenRateLimiter
		now := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)

		allowed, remaining := limiter.Allow("127.0.0.1", now)

		assert.True(t, allowed)
		assert.Equal(t, time.Duration(0), remaining)
	})

	t.Run("allows when key is empty", func(t *testing.T) {
		limiter := auth.NewTokenRateLimiter(2 * time.Minute)
		now := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)

		allowed, remaining := limiter.Allow("", now)

		assert.True(t, allowed)
		assert.Equal(t, time.Duration(0), remaining)
	})

	t.Run("blocks during cooldown window", func(t *testing.T) {
		limiter := auth.NewTokenRateLimiter(2 * time.Minute)
		start := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)

		allowed, remaining := limiter.Allow("127.0.0.1", start)

		assert.True(t, allowed)
		assert.Equal(t, time.Duration(0), remaining)

		allowed, remaining = limiter.Allow("127.0.0.1", start.Add(time.Minute))

		assert.False(t, allowed)
		assert.Equal(t, time.Minute, remaining)
	})

	t.Run("allows after cooldown window", func(t *testing.T) {
		limiter := auth.NewTokenRateLimiter(2 * time.Minute)
		start := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)

		allowed, remaining := limiter.Allow("127.0.0.1", start)

		assert.True(t, allowed)
		assert.Equal(t, time.Duration(0), remaining)

		allowed, remaining = limiter.Allow("127.0.0.1", start.Add(2*time.Minute))

		assert.True(t, allowed)
		assert.Equal(t, time.Duration(0), remaining)
	})
}
