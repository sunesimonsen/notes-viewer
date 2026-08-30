package auth

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/sunesimonsen/notes-viewer/users"
)

// SessionKeyUser is the session key used to store the authenticated user.
const SessionKeyUser = "user"
const SessionKeyTokenError = "token_error"

func RequestTokenHandler(
	sessionManager *scs.SessionManager,
	verification users.Verification,
	skipVerification bool,
	defaultUser mail.Address,
	limiter *TokenRateLimiter,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if skipVerification {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		user, ok := sessionManager.Get(r.Context(), SessionKeyUser).(users.User)

		if !ok {
			user = users.NewUser(defaultUser)
		}

		if user.IsVerified {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		now := time.Now()
		if limiter != nil {
			client := clientIP(r)
			allowed, wait := limiter.Allow(client, now)
			if !allowed {
				sessionManager.Put(r.Context(), SessionKeyTokenError, formatWait(wait))
				sessionManager.Put(r.Context(), SessionKeyUser, user)
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
		}

		allowed, wait := verification.CanSend(&user, now)
		if !allowed {
			sessionManager.Put(r.Context(), SessionKeyTokenError, formatWait(wait))
			sessionManager.Put(r.Context(), SessionKeyUser, user)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		err := verification.SendCode(&user)
		if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		sessionManager.Put(r.Context(), SessionKeyUser, user)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func VerifyTokenHandler(
	sessionManager *scs.SessionManager,
	verification users.Verification,
	skipVerification bool,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if skipVerification {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		r.ParseForm()
		token := r.Form.Get("token")

		user, ok := sessionManager.Get(r.Context(), SessionKeyUser).(users.User)

		if ok {
			verification.VerifyCode(&user, token)
		}

		sessionManager.Put(r.Context(), SessionKeyUser, user)

		if user.IsVerified {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func Middleware(sessionManager *scs.SessionManager, skipVerification bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if skipVerification {
				next.ServeHTTP(w, r)
				return
			}

			user, ok := sessionManager.Get(r.Context(), SessionKeyUser).(users.User)

			if !ok || !user.IsVerified {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

func formatWait(wait time.Duration) string {
	seconds := max(int(wait.Seconds()), 1)
	return fmt.Sprintf("Please wait %ds before requesting a new token.", seconds)
}

func clientIP(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
