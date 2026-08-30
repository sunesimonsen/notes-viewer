package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestLoginHandler(t *testing.T) {
	t.Run("redirects to / when skipVerification is true", func(t *testing.T) {
		s := setupTestServer(t, true)

		rr := performRequest(s, http.MethodGet, "/login", "")

		assert.Equal(t, http.StatusSeeOther, rr.Code)
		assert.Equal(t, "/", rr.Header().Get("Location"))
	})

	t.Run("shows login page when verification is required", func(t *testing.T) {
		s := setupTestServer(t, false)

		rr := performRequest(s, http.MethodGet, "/login", "")

		assert.Equal(t, http.StatusOK, rr.Code)
		body := rr.Body.String()
		assert.Contains(t, body, "token")
	})

	t.Run("shows login page for already-requested verification", func(t *testing.T) {
		s := setupTestServer(t, false)

		// First visit renders login page
		rr1 := performRequest(s, http.MethodGet, "/login", "")
		csrfToken := extractCSRFToken(rr1.Body.String())
		cookie := extractCookie(rr1)

		assert.Equal(t, http.StatusOK, rr1.Code)
		body1 := rr1.Body.String()
		assert.Contains(t, body1, "Request a verification token")

		// Request token to mark verification as requested
		form := url.Values{"csrf_token": {csrfToken}}
		postRR := postForm(s, "/request-token", cookie, form)
		cookie = extractCookie(postRR)

		// Second visit should show the requested state
		req := httptest.NewRequest(http.MethodGet, "/login", nil)
		if cookie != "" {
			req.Header.Set("Cookie", cookie)
		}
		rr2 := httptest.NewRecorder()
		s.ServeHTTP(rr2, req)

		assert.Equal(t, http.StatusOK, rr2.Code)
		body2 := rr2.Body.String()
		assert.Contains(t, body2, "Check your email")
	})
}
