package server

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sunesimonsen/notes-viewer/notes"
	"github.com/sunesimonsen/notes-viewer/templates"
	"github.com/sunesimonsen/notes-viewer/users"
)

func (s *Server) noteHandler(w http.ResponseWriter, r *http.Request) {
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
			s.renderNote(w, r, user, entry, "/note/"+id)
			return
		}
	}

	http.NotFound(w, r)
}

func (s *Server) renderNote(w http.ResponseWriter, r *http.Request, user users.User, entry notes.Entry, currentPath string) {
	document, err := s.store.ReadDocument(user, entry.ID)
	if err != nil {
		log.Println(err)
		http.NotFound(w, r)
		return
	}

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
