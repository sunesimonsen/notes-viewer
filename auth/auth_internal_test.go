package auth

import (
	"encoding/gob"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
	"github.com/alexedwards/scs/v2"
	"github.com/sunesimonsen/notes-viewer/internal/testutil"
	"github.com/sunesimonsen/notes-viewer/users"
)

func init() {
	gob.Register(users.User{})
}

func TestFormatWait(t *testing.T) {
	assert.Equal(t, "Please wait 1s before requesting a new token.", formatWait(200*time.Millisecond))
	assert.Equal(t, "Please wait 12s before requesting a new token.", formatWait(12*time.Second))
}

func TestClientIP(t *testing.T) {
	t.Run("uses first X-Forwarded-For entry", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.1, 70.0.0.1")

		assert.Equal(t, "203.0.113.1", clientIP(req))
	})

	t.Run("uses X-Real-IP when forwarded header is empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Real-IP", "198.51.100.2")

		assert.Equal(t, "198.51.100.2", clientIP(req))
	})

	t.Run("falls back to RemoteAddr host", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.0.2.10:1234"

		assert.Equal(t, "192.0.2.10", clientIP(req))
	})

	t.Run("returns RemoteAddr when not host:port", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "bad-addr"

		assert.Equal(t, "bad-addr", clientIP(req))
	})
}

func TestRequestTokenHandlerRateLimit(t *testing.T) {
	sm := scs.New()
	defaultUser := mail.Address{Name: "Test", Address: "test@test.com"}
	verification := users.Verification{
		Tokens:  testutil.TokenGenerator{Code: "123456"},
		Emailer: &testutil.Emailer{},
	}

	limiter := NewTokenRateLimiter(1 * time.Minute)
	limiter.lastByIP["203.0.113.1"] = time.Now()

	handler := sm.LoadAndSave(RequestTokenHandler(sm, verification, false, defaultUser, limiter))
	req := httptest.NewRequest(http.MethodPost, "/request-token", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusSeeOther, rr.Code)
	assert.Equal(t, "/login", rr.Header().Get("Location"))

	cookie := rr.Header().Get("Set-Cookie")
	assert.NotEqual(t, "", cookie)

	readHandler := sm.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenError, _ := sm.Get(r.Context(), SessionKeyTokenError).(string)
		assert.Contains(t, tokenError, "Please wait")
		user, ok := sm.Get(r.Context(), SessionKeyUser).(users.User)
		assert.True(t, ok)
		assert.Equal(t, defaultUser, user.Email)
	}))
	readReq := httptest.NewRequest(http.MethodGet, "/", nil)
	readReq.Header.Set("Cookie", cookie)
	readRR := httptest.NewRecorder()

	readHandler.ServeHTTP(readRR, readReq)
}

func TestRequestTokenHandlerCooldown(t *testing.T) {
	sm := scs.New()
	defaultUser := mail.Address{Name: "Test", Address: "test@test.com"}
	verification := users.Verification{
		Tokens:  testutil.TokenGenerator{Code: "123456"},
		Emailer: &testutil.Emailer{},
	}

	setupHandler := sm.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := users.NewUser(defaultUser)
		user.VerificationCodeSentAt = time.Now()
		sm.Put(r.Context(), SessionKeyUser, user)
		w.WriteHeader(http.StatusOK)
	}))
	setupReq := httptest.NewRequest(http.MethodGet, "/setup", nil)
	setupRR := httptest.NewRecorder()
	setupHandler.ServeHTTP(setupRR, setupReq)
	cookie := setupRR.Header().Get("Set-Cookie")

	handler := sm.LoadAndSave(RequestTokenHandler(sm, verification, false, defaultUser, nil))
	req := httptest.NewRequest(http.MethodPost, "/request-token", nil)
	req.Header.Set("Cookie", cookie)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusSeeOther, rr.Code)
	assert.Equal(t, "/login", rr.Header().Get("Location"))

	cookie = rr.Header().Get("Set-Cookie")
	readHandler := sm.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenError, _ := sm.Get(r.Context(), SessionKeyTokenError).(string)
		assert.Contains(t, tokenError, "Please wait")
	}))
	readReq := httptest.NewRequest(http.MethodGet, "/", nil)
	readReq.Header.Set("Cookie", cookie)
	readRR := httptest.NewRecorder()

	readHandler.ServeHTTP(readRR, readReq)
}
