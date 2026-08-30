package server

import (
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sunesimonsen/notes-viewer/notes"
	"github.com/sunesimonsen/notes-viewer/templates"
)

func (s *Server) entryRedirectHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := s.getSessionUser(r)
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	entries, err := s.searchEntries(user, id)
	if err != nil {
		log.Println(err)
		http.NotFound(w, r)
		return
	}

	for _, entry := range entries {
		if !entry.Timestamp.IsZero() && entry.Timestamp.Format("20060102T150405") == id {
			http.Redirect(w, r, "/entry/"+entry.ID, http.StatusSeeOther)
			return
		}
	}

	http.NotFound(w, r)
}

func (s *Server) entryHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := s.getSessionUser(r)
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	entryPath := strings.TrimPrefix(chi.URLParam(r, "*"), "/")
	if entryPath == "" || entryPath == "." || !fs.ValidPath(entryPath) || !strings.HasSuffix(entryPath, ".md") {
		http.NotFound(w, r)
		return
	}

	document, err := s.store.ReadDocument(user, entryPath)
	if err != nil {
		log.Println(err)
		http.NotFound(w, r)
		return
	}

	entry := notes.NewEntry(entryPath)
	currentPath := "/entry/" + entryPath
	if r.Header.Get("HX-Request") == "true" {
		renderComponent(w, r, templates.NoteBody(entry, document.HTML, document.Headings))
		return
	}

	entries, err := s.searchEntries(user, "")
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	component := templates.MainLayout(entries, currentPath, templates.NoteDocument(entry, document.HTML, document.Headings))
	renderComponent(w, r, component)
}
