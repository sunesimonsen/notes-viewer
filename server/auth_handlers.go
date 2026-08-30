package server

import (
	"net/http"

	"github.com/sunesimonsen/notes-viewer/auth"
	"github.com/sunesimonsen/notes-viewer/templates"
	"github.com/sunesimonsen/notes-viewer/users"
)

// getSessionUser retrieves the authenticated user from the session.
// When skipVerification is enabled, it returns a synthetic verified user.
// Returns the user and true on success, or a zero user and false if unauthorized.
func (s *Server) getSessionUser(r *http.Request) (users.User, bool) {
	user, ok := s.sessionManager.Get(r.Context(), auth.SessionKeyUser).(users.User)
	if !ok {
		if s.skipVerification {
			user = users.User{IsVerified: true}
			s.sessionManager.Put(r.Context(), auth.SessionKeyUser, user)
			return user, true
		}
		return user, false
	}
	return user, true
}

func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	if s.skipVerification {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user, ok := s.sessionManager.Get(r.Context(), auth.SessionKeyUser).(users.User)
	if !ok {
		user = users.NewUser(s.defaultUser)
	}

	s.sessionManager.Put(r.Context(), auth.SessionKeyUser, user)
	tokenError, _ := s.sessionManager.Get(r.Context(), auth.SessionKeyTokenError).(string)
	if tokenError != "" {
		s.sessionManager.Remove(r.Context(), auth.SessionKeyTokenError)
	}
	component := templates.AuthLayout(templates.Login(user.IsVerificationRequested(), tokenError, s.csrfToken(r)))
	renderComponent(w, r, component)
}
