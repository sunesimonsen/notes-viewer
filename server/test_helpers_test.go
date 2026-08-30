package server

import (
	"context"
	"encoding/gob"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/sunesimonsen/notes-viewer/users"
)

func setupTestServer(t *testing.T, skipVerification bool) *Server {
	t.Helper()
	gob.Register(users.User{})

	sm := scs.New()

	router := chi.NewRouter()
	s := &Server{
		router:           router,
		sessionManager:   sm,
		skipVerification: skipVerification,
		defaultUser:      mail.Address{Name: "Test", Address: "test@test.com"},
	}

	s.verification = users.Verification{
		Tokens:  users.RandomTokenGenerator{},
		Emailer: &stubEmailerForHandlers{},
	}

	s.setupRoutes()

	return s
}

type stubEmailerForHandlers struct{}

func (e *stubEmailerForHandlers) SendMail(to mail.Address, subject string, body string) error {
	return nil
}

// runWithSession executes fn within a session context, returning the session cookie.
func runWithSession(sm *scs.SessionManager, cookie string, fn func(ctx context.Context)) string {
	var resultCookie string
	handler := sm.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fn(r.Context())
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if c := rr.Header().Get("Set-Cookie"); c != "" {
		resultCookie = c
	} else {
		resultCookie = cookie
	}
	return resultCookie
}

// performRequest executes a request against the server and returns the response
// and any session cookie for follow-up requests.
func performRequest(s *Server, method, path string, cookie string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)
	return rr
}

func postForm(s *Server, path string, cookie string, form url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)
	return rr
}

func extractCookie(rr *httptest.ResponseRecorder) string {
	return rr.Header().Get("Set-Cookie")
}

func extractCSRFToken(body string) string {
	re := regexp.MustCompile(`name="csrf_token" value="([^"]+)"`)
	matches := re.FindStringSubmatch(body)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func getCSRFTokenAndCookie(s *Server, path string, cookie string) (string, string) {
	rr := performRequest(s, http.MethodGet, path, cookie)
	return extractCSRFToken(rr.Body.String()), extractCookie(rr)
}
