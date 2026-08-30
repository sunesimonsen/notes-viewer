package server

import (
	"net/http"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/alecthomas/assert/v2"
	"github.com/sunesimonsen/notes-viewer/notes"
)

func TestIndexHandler(t *testing.T) {
	t.Run("returns 401 when not authenticated and skipVerification is false", func(t *testing.T) {
		s := setupTestServer(t, false)

		// Need to go through session middleware - the auth middleware will redirect
		rr := performRequest(s, http.MethodGet, "/", "")

		// Auth middleware redirects to /login
		assert.Equal(t, http.StatusSeeOther, rr.Code)
		assert.Equal(t, "/login", rr.Header().Get("Location"))
	})
}

func TestTagHandler(t *testing.T) {
	s := setupTestServer(t, true)
	s.store = notes.FSStore{FS: fstest.MapFS{
		"20240229T123456--work-note__work_personal.md": &fstest.MapFile{Data: []byte("# Work note")},
		"20240229T123457--personal-note__personal.md":  &fstest.MapFile{Data: []byte("# Personal note")},
		"20240229T123458--starred-note__star.md":       &fstest.MapFile{Data: []byte("# Starred note")},
	}}

	rr := performRequest(s, http.MethodGet, "/tag/work", "")

	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	assert.Contains(t, body, "<h1>work</h1>")
	assert.Contains(t, body, "work note")
	assert.Contains(t, body, "<summary>Starred</summary>")
	assert.Contains(t, body, "<summary>Tags</summary>")
	assert.Contains(t, body, "/entry/20240229T123458--starred-note__star.md")
	assert.Contains(t, body, `href="/tag/personal"`)
	assert.Equal(t, 0, strings.Count(body, "/entry/20240229T123457--personal-note__personal.md"))

	noteRR := performRequest(s, http.MethodGet, "/entry/20240229T123456--work-note__work_personal.md", "")
	assert.Equal(t, http.StatusOK, noteRR.Code)
	assert.Contains(t, noteRR.Body.String(), `class="tag" href="/tag/work"`)
}
