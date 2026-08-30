package auth_test

import (
	"encoding/gob"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"net/url"
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/alexedwards/scs/v2"
	"github.com/sunesimonsen/notes-viewer/auth"
	"github.com/sunesimonsen/notes-viewer/internal/testutil"
	"github.com/sunesimonsen/notes-viewer/users"
)

func init() {
	gob.Register(users.User{})
}

var defaultUser = mail.Address{Name: "Test User", Address: "test@example.com"}

func newSessionManager() *scs.SessionManager {
	return scs.New()
}

func newVerification(code string, emailer *testutil.Emailer) users.Verification {
	return testutil.NewVerification(code, emailer)
}

// withSession wraps a handler with the session manager's LoadAndSave middleware
// to ensure session data is properly loaded and saved.
func withSession(sm *scs.SessionManager, handler http.Handler) http.Handler {
	return sm.LoadAndSave(handler)
}

func TestMiddleware(t *testing.T) {
	t.Run("allows request when skipVerification is true", func(t *testing.T) {
		sm := newSessionManager()
		called := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		handler := withSession(sm, auth.Middleware(sm, true)(next))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.True(t, called, "next handler should be called")
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("redirects to login when no session user", func(t *testing.T) {
		sm := newSessionManager()
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler should not be called")
		})

		handler := withSession(sm, auth.Middleware(sm, false)(next))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusSeeOther, rr.Code)
		assert.Equal(t, "/login", rr.Header().Get("Location"))
	})

	t.Run("redirects to login when user is not verified", func(t *testing.T) {
		sm := newSessionManager()
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler should not be called")
		})

		// Set an unverified user in session via a setup request.
		setupHandler := withSession(sm, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sm.Put(r.Context(), auth.SessionKeyUser, users.User{IsVerified: false})
			w.WriteHeader(http.StatusOK)
		}))
		setupReq := httptest.NewRequest(http.MethodGet, "/setup", nil)
		setupRR := httptest.NewRecorder()
		setupHandler.ServeHTTP(setupRR, setupReq)
		cookie := setupRR.Header().Get("Set-Cookie")

		handler := withSession(sm, auth.Middleware(sm, false)(next))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Cookie", cookie)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusSeeOther, rr.Code)
		assert.Equal(t, "/login", rr.Header().Get("Location"))
	})

	t.Run("allows request when user is verified", func(t *testing.T) {
		sm := newSessionManager()
		called := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		// Set a verified user in session.
		setupHandler := withSession(sm, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sm.Put(r.Context(), auth.SessionKeyUser, users.User{IsVerified: true})
			w.WriteHeader(http.StatusOK)
		}))
		setupReq := httptest.NewRequest(http.MethodGet, "/setup", nil)
		setupRR := httptest.NewRecorder()
		setupHandler.ServeHTTP(setupRR, setupReq)
		cookie := setupRR.Header().Get("Set-Cookie")

		handler := withSession(sm, auth.Middleware(sm, false)(next))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Cookie", cookie)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.True(t, called, "next handler should be called")
		assert.Equal(t, http.StatusOK, rr.Code)
	})
}

func TestRequestTokenHandler(t *testing.T) {
	t.Run("redirects to / when skipVerification is true", func(t *testing.T) {
		sm := newSessionManager()
		emailer := &testutil.Emailer{}
		verification := newVerification("123456", emailer)

		handler := withSession(sm, auth.RequestTokenHandler(sm, verification, true, defaultUser, nil))
		req := httptest.NewRequest(http.MethodPost, "/request-token", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusSeeOther, rr.Code)
		assert.Equal(t, "/", rr.Header().Get("Location"))
	})

	t.Run("redirects to / when user is already verified", func(t *testing.T) {
		sm := newSessionManager()
		emailer := &testutil.Emailer{}
		verification := newVerification("123456", emailer)

		// Set a verified user.
		setupHandler := withSession(sm, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sm.Put(r.Context(), auth.SessionKeyUser, users.User{IsVerified: true})
			w.WriteHeader(http.StatusOK)
		}))
		setupReq := httptest.NewRequest(http.MethodGet, "/setup", nil)
		setupRR := httptest.NewRecorder()
		setupHandler.ServeHTTP(setupRR, setupReq)
		cookie := setupRR.Header().Get("Set-Cookie")

		handler := withSession(sm, auth.RequestTokenHandler(sm, verification, false, defaultUser, nil))
		req := httptest.NewRequest(http.MethodPost, "/request-token", nil)
		req.Header.Set("Cookie", cookie)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusSeeOther, rr.Code)
		assert.Equal(t, "/", rr.Header().Get("Location"))
	})

	t.Run("sends verification code and redirects to /login", func(t *testing.T) {
		sm := newSessionManager()
		emailer := &testutil.Emailer{}
		verification := newVerification("654321", emailer)

		handler := withSession(sm, auth.RequestTokenHandler(sm, verification, false, defaultUser, nil))
		req := httptest.NewRequest(http.MethodPost, "/request-token", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusSeeOther, rr.Code)
		assert.Equal(t, "/login", rr.Header().Get("Location"))
		assert.Equal(t, defaultUser, emailer.To)
		assert.Contains(t, emailer.Message, "654321")
	})

	t.Run("returns 500 when email sending fails", func(t *testing.T) {
		sm := newSessionManager()
		emailer := &testutil.Emailer{Err: errors.New("send failed")}
		verification := newVerification("654321", emailer)

		handler := withSession(sm, auth.RequestTokenHandler(sm, verification, false, defaultUser, nil))
		req := httptest.NewRequest(http.MethodPost, "/request-token", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestVerifyTokenHandler(t *testing.T) {
	t.Run("redirects to / when skipVerification is true", func(t *testing.T) {
		sm := newSessionManager()
		emailer := &testutil.Emailer{}
		verification := newVerification("123456", emailer)

		handler := withSession(sm, auth.VerifyTokenHandler(sm, verification, true))
		req := httptest.NewRequest(http.MethodPost, "/verify-token", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusSeeOther, rr.Code)
		assert.Equal(t, "/", rr.Header().Get("Location"))
	})

	t.Run("redirects to /login when token is wrong", func(t *testing.T) {
		sm := newSessionManager()
		emailer := &testutil.Emailer{}
		verification := newVerification("123456", emailer)

		// First, request a token to set up the user in session.
		reqTokenHandler := withSession(sm, auth.RequestTokenHandler(sm, verification, false, defaultUser, nil))
		reqTokenReq := httptest.NewRequest(http.MethodPost, "/request-token", nil)
		reqTokenRR := httptest.NewRecorder()
		reqTokenHandler.ServeHTTP(reqTokenRR, reqTokenReq)
		cookie := reqTokenRR.Header().Get("Set-Cookie")

		// Now try to verify with wrong token.
		handler := withSession(sm, auth.VerifyTokenHandler(sm, verification, false))
		form := url.Values{"token": {"wrong"}}
		req := httptest.NewRequest(http.MethodPost, "/verify-token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", cookie)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusSeeOther, rr.Code)
		assert.Equal(t, "/login", rr.Header().Get("Location"))
	})

	t.Run("redirects to / when token is correct", func(t *testing.T) {
		sm := newSessionManager()
		emailer := &testutil.Emailer{}
		code := "123456"
		verification := newVerification(code, emailer)

		// First, request a token to set up the user in session.
		reqTokenHandler := withSession(sm, auth.RequestTokenHandler(sm, verification, false, defaultUser, nil))
		reqTokenReq := httptest.NewRequest(http.MethodPost, "/request-token", nil)
		reqTokenRR := httptest.NewRecorder()
		reqTokenHandler.ServeHTTP(reqTokenRR, reqTokenReq)
		cookie := reqTokenRR.Header().Get("Set-Cookie")

		// Now verify with the correct token.
		handler := withSession(sm, auth.VerifyTokenHandler(sm, verification, false))
		form := url.Values{"token": {code}}
		req := httptest.NewRequest(http.MethodPost, "/verify-token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", cookie)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusSeeOther, rr.Code)
		assert.Equal(t, "/", rr.Header().Get("Location"))
	})

	t.Run("redirects to /login when no session exists", func(t *testing.T) {
		sm := newSessionManager()
		emailer := &testutil.Emailer{}
		verification := newVerification("123456", emailer)

		handler := withSession(sm, auth.VerifyTokenHandler(sm, verification, false))
		form := url.Values{"token": {"123456"}}
		req := httptest.NewRequest(http.MethodPost, "/verify-token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusSeeOther, rr.Code)
		assert.Equal(t, "/login", rr.Header().Get("Location"))
	})
}
