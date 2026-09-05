package server

import (
	"encoding/gob"
	"fmt"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/sunesimonsen/notes-viewer/emails"
	"github.com/sunesimonsen/notes-viewer/notes"
	"github.com/sunesimonsen/notes-viewer/users"
)

type Config struct {
	Port             string
	NotesStorePath   string
	SkipVerification bool
	SMTPHost         string
	SMTPPort         string
	SMTPFromAddress  string
	SMTPPassword     string
	UserName         string
	UserEmail        string
}

type Server struct {
	router           chi.Router
	sessionManager   *scs.SessionManager
	verification     users.Verification
	store            notes.FSStore
	skipVerification bool
	defaultUser      mail.Address
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func ConfigFromEnv() (Config, error) {
	config := Config{
		Port:            os.Getenv("PORT"),
		SMTPHost:        os.Getenv("NOTES_VIEWER_SMTP_HOST"),
		SMTPPort:        os.Getenv("NOTES_VIEWER_SMTP_PORT"),
		SMTPFromAddress: os.Getenv("NOTES_VIEWER_SMTP_FROM_ADDRESS"),
		SMTPPassword:    os.Getenv("NOTES_VIEWER_SMTP_PASSWORD"),
		UserName:        os.Getenv("NOTES_VIEWER_USER_NAME"),
		UserEmail:       os.Getenv("NOTES_VIEWER_USER_EMAIL"),
		SkipVerification: strings.EqualFold(
			os.Getenv("NOTES_VIEWER_SKIP_VERIFICATION"), "true",
		),
	}

	if config.Port == "" {
		config.Port = "8080"
	}

	storePath := os.Getenv("NOTES_VIEWER_STORE_PATH")
	if storePath == "" {
		return config, fmt.Errorf("NOTES_VIEWER_STORE_PATH is not set")
	}

	if storePath == "~" {
		return config, fmt.Errorf("NOTES_VIEWER_STORE_PATH is set to ~")
	}

	if strings.HasPrefix(storePath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return config, fmt.Errorf("resolving home directory: %w", err)
		}
		storePath = filepath.Join(home, strings.TrimPrefix(storePath, "~/"))
	}

	config.NotesStorePath = filepath.Clean(storePath)

	return config, nil
}

func NewServer(config Config) (*Server, error) {
	gob.Register(users.User{})
	router := chi.NewRouter()
	sessionManager := scs.New()
	sessionManager.Cookie.HttpOnly = true
	sessionManager.Cookie.SameSite = http.SameSiteLaxMode
	sessionManager.Cookie.Secure = true

	s := &Server{
		router:           router,
		sessionManager:   sessionManager,
		store:            notes.FSStore{FS: os.DirFS(config.NotesStorePath)},
		skipVerification: config.SkipVerification,
		defaultUser: mail.Address{
			Name:    config.UserName,
			Address: config.UserEmail,
		},
	}

	emailer, err := emails.NewSmtpEmailer(
		config.SMTPHost,
		config.SMTPPort,
		config.SMTPFromAddress,
		config.SMTPPassword,
	)
	if err != nil {
		return nil, fmt.Errorf("creating SMTP emailer: %w", err)
	}

	s.verification = users.Verification{
		Emailer: emailer,
		Tokens:  users.RandomTokenGenerator{},
	}

	s.setupRoutes()

	return s, nil
}
