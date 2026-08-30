package server

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/alexedwards/scs/v2"
)

func TestGenerateCSRFToken(t *testing.T) {
	token, err := generateCSRFToken()

	assert.NoError(t, err)
	assert.NotEqual(t, "", token)

	decoded, err := base64.RawStdEncoding.DecodeString(token)

	assert.NoError(t, err)
	assert.Equal(t, 32, len(decoded))
}

func TestTokensMatch(t *testing.T) {
	assert.False(t, tokensMatch("", ""))
	assert.False(t, tokensMatch("expected", ""))
	assert.False(t, tokensMatch("", "actual"))
	assert.False(t, tokensMatch("expected", "actual"))
	assert.True(t, tokensMatch("expected", "expected"))
}

func TestIsSafeMethod(t *testing.T) {
	assert.True(t, isSafeMethod(http.MethodGet))
	assert.True(t, isSafeMethod(http.MethodHead))
	assert.True(t, isSafeMethod(http.MethodOptions))
	assert.True(t, isSafeMethod(http.MethodTrace))
	assert.False(t, isSafeMethod(http.MethodPost))
}

func TestCSRFMiddleware(t *testing.T) {
	t.Run("allows safe methods and sets session token", func(t *testing.T) {
		s := &Server{sessionManager: scs.New()}
		var gotToken string

		handler := s.sessionManager.LoadAndSave(s.csrfMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotToken, _ = s.sessionManager.Get(r.Context(), csrfSessionKey).(string)
			w.WriteHeader(http.StatusOK)
		})))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.NotEqual(t, "", gotToken)
	})

	t.Run("blocks unsafe methods when token is missing", func(t *testing.T) {
		s := &Server{sessionManager: scs.New()}

		handler := s.sessionManager.LoadAndSave(s.csrfMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})))

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("allows unsafe methods with matching header token", func(t *testing.T) {
		s := &Server{sessionManager: scs.New()}
		token := "token-123"

		cookie := runWithSession(s.sessionManager, "", func(ctx context.Context) {
			s.sessionManager.Put(ctx, csrfSessionKey, token)
		})

		handler := s.sessionManager.LoadAndSave(s.csrfMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})))

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("X-CSRF-Token", token)
		req.Header.Set("Cookie", cookie)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("allows unsafe methods with matching form token", func(t *testing.T) {
		s := &Server{sessionManager: scs.New()}
		token := "token-456"

		cookie := runWithSession(s.sessionManager, "", func(ctx context.Context) {
			s.sessionManager.Put(ctx, csrfSessionKey, token)
		})

		handler := s.sessionManager.LoadAndSave(s.csrfMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})))

		form := url.Values{csrfSessionKey: {token}}
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", cookie)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})
}
