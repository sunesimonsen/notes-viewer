package server

import (
	"net/http"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/alecthomas/assert/v2"
	"github.com/sunesimonsen/notes-viewer/notes"
)

func TestInboxHandler(t *testing.T) {
	t.Run("redirects unauthenticated users to the login page", func(t *testing.T) {
		s := setupTestServer(t, false)

		rr := performRequest(s, http.MethodGet, "/", "")

		// Auth middleware redirects to /login
		assert.Equal(t, http.StatusSeeOther, rr.Code)
		assert.Equal(t, "/login", rr.Header().Get("Location"))
	})

	t.Run("redirects to the first note titled Inbox", func(t *testing.T) {
		s := setupTestServer(t, true)
		s.store = notes.FSStore{FS: fstest.MapFS{
			"20240229T123456--work-note__work.md": &fstest.MapFile{Data: []byte("# Work note")},
			"20240229T123457--inbox__inbox.md":    &fstest.MapFile{Data: []byte("# Inbox")},
		}}

		rr := performRequest(s, http.MethodGet, "/", "")

		assert.Equal(t, http.StatusFound, rr.Code)
		assert.Equal(t, "/note/20240229T123457", rr.Header().Get("Location"))
	})

	t.Run("returns 404 when no note is titled Inbox", func(t *testing.T) {
		s := setupTestServer(t, true)
		s.store = notes.FSStore{FS: fstest.MapFS{
			"20240229T123456--work-note__work.md": &fstest.MapFile{Data: []byte("# Work note")},
		}}

		rr := performRequest(s, http.MethodGet, "/", "")

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}

func TestTagHandler(t *testing.T) {
	s := setupTestServer(t, true)
	s.store = notes.FSStore{FS: fstest.MapFS{
		"20240229T123456--work-note__work_personal.md": &fstest.MapFile{Data: []byte("# Work note")},
		"20240229T123457--personal-note__personal.md":  &fstest.MapFile{Data: []byte("# Personal note")},
	}}

	rr := performRequest(s, http.MethodGet, "/tag/work", "")

	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	assert.Contains(t, body, "<h1>work</h1>")
	assert.Contains(t, body, "work note")
	assert.Equal(t, 0, strings.Count(body, "/note/20240229T123457"))

	noteRR := performRequest(s, http.MethodGet, "/note/20240229T123456", "")
	assert.Equal(t, http.StatusOK, noteRR.Code)
	assert.Contains(t, noteRR.Body.String(), `class="tag" href="/tag/work"`)
}
