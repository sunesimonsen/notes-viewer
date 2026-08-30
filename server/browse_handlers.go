package server

import (
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sunesimonsen/notes-viewer/notes"
	"github.com/sunesimonsen/notes-viewer/templates"
	"github.com/sunesimonsen/notes-viewer/users"
)

const searchPageSize = 30

// searchEntries consumes the store's cursor pagination and returns the complete
// set of matching entries. Results should not silently omit notes just because
// there are more than one page of results.
func (s *Server) searchEntries(user users.User, query string) ([]notes.Entry, error) {
	entries := make([]notes.Entry, 0)
	cursor := ""

	for {
		result, err := s.store.Search(user, query, cursor, searchPageSize)
		if err != nil {
			return nil, err
		}
		entries = append(entries, result.Entries...)
		if !result.HasMore || result.EndCursor == "" || result.EndCursor == cursor {
			return entries, nil
		}
		cursor = result.EndCursor
	}
}

func requestQuery(r *http.Request) string {
	return strings.TrimSpace(r.URL.Query().Get("query"))
}

func entriesForTag(entries []notes.Entry, tag string) []notes.Entry {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil
	}

	matched := make([]notes.Entry, 0)
	for _, entry := range entries {
		for _, entryTag := range entry.Tags {
			if strings.TrimSpace(entryTag) == tag {
				matched = append(matched, entry)
				break
			}
		}
	}
	return matched
}

func refererPath(r *http.Request) string {
	referer, err := url.Parse(r.Referer())
	if err != nil {
		return ""
	}
	return referer.Path
}

func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := s.getSessionUser(r)
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	entries, err := s.searchEntries(user, "")
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		renderComponent(w, r, templates.SearchMenu(entries, refererPath(r)))
		return
	}

	component := templates.MainLayout(entries, "/", templates.Introduction())
	renderComponent(w, r, component)
}

// searchHandler is a partial endpoint for clients that want to request the
// matching note list without requesting the introduction page as well.
func (s *Server) searchHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := s.getSessionUser(r)
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	query := requestQuery(r)
	entries, err := s.searchEntries(user, query)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	renderComponent(w, r, templates.SearchMenu(entries, refererPath(r)))
}

func (s *Server) tagHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := s.getSessionUser(r)
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	tag := strings.TrimSpace(chi.URLParam(r, "tag"))
	if tag == "" {
		http.NotFound(w, r)
		return
	}

	entries, err := s.searchEntries(user, "")
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	component := templates.MainLayout(
		entries,
		r.URL.Path,
		templates.TagDocument(tag, entriesForTag(entries, tag), r.URL.Path),
	)
	renderComponent(w, r, component)
}
