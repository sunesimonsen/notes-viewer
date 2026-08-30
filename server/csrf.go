package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
)

const csrfSessionKey = "csrf_token"

func (s *Server) csrfMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isSafeMethod(r.Method) {
			s.csrfToken(r)
			next.ServeHTTP(w, r)
			return
		}

		expected, _ := s.sessionManager.Get(r.Context(), csrfSessionKey).(string)
		token := r.Header.Get("X-CSRF-Token")
		if token == "" {
			if err := r.ParseForm(); err == nil {
				token = r.Form.Get(csrfSessionKey)
			}
		}

		if !tokensMatch(expected, token) {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) csrfToken(r *http.Request) string {
	if token, ok := s.sessionManager.Get(r.Context(), csrfSessionKey).(string); ok && token != "" {
		return token
	}

	token, err := generateCSRFToken()
	if err != nil {
		return ""
	}
	if token != "" {
		s.sessionManager.Put(r.Context(), csrfSessionKey, token)
	}
	return token
}

func generateCSRFToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(buf), nil
}

func tokensMatch(expected string, actual string) bool {
	if expected == "" || actual == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}
