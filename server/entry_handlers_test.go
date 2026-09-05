package server

import (
	"net/http"
	"testing"
	"testing/fstest"

	"github.com/alecthomas/assert/v2"
	"github.com/sunesimonsen/notes-viewer/notes"
)

func TestNoteHandler(t *testing.T) {
	s := setupTestServer(t, true)
	s.store = notes.FSStore{FS: fstest.MapFS{
		"20230504T162825--linked-note.md": &fstest.MapFile{Data: []byte("# Linked note")},
	}}

	rr := performRequest(s, http.MethodGet, "/note/20230504T162825", "")

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Linked note")

	notFoundRR := performRequest(s, http.MethodGet, "/note/20230504T162826", "")
	assert.Equal(t, http.StatusNotFound, notFoundRR.Code)

	legacyRouteRR := performRequest(s, http.MethodGet, "/entry/20230504T162825.id", "")
	assert.Equal(t, http.StatusNotFound, legacyRouteRR.Code)
}
