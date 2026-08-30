package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sunesimonsen/notes-viewer/auth"
)

func (s *Server) setupRoutes() {
	s.router.Use(middleware.RedirectSlashes)
	s.router.Use(middleware.Logger)
	s.router.Use(s.sessionManager.LoadAndSave)
	s.router.Use(s.csrfMiddleware)
	tokenLimiter := auth.NewTokenRateLimiter(1 * time.Minute)

	s.router.Group(func(r chi.Router) {
		r.Use(auth.Middleware(s.sessionManager, s.skipVerification))
		r.Get("/", s.indexHandler)
		r.Get("/search", s.searchHandler)
		r.Get("/tag/{tag}", s.tagHandler)
		r.Get("/entry/{id}.id", s.entryRedirectHandler)
		r.Get("/entry/*", s.entryHandler)
	})

	s.router.Get("/login", s.loginHandler)
	s.router.Post("/request-token", auth.RequestTokenHandler(s.sessionManager, s.verification, s.skipVerification, s.defaultUser, tokenLimiter))
	s.router.Post("/verify-token", auth.VerifyTokenHandler(s.sessionManager, s.verification, s.skipVerification))

	s.router.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))
}
