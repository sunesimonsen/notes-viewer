package notes

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestMarkdownDocumentIncludesHeadingTree(t *testing.T) {
	document := mdToDocument([]byte("# Introduction\n\n## Getting started\n\n### Install `notes-viewer`\n\n## Getting started\n"))

	if got, want := string(document.HTML), "<h1 id=\"introduction\">Introduction</h1>\n\n<h2 id=\"getting-started\">Getting started</h2>\n\n<h3 id=\"install-notes-viewer\">Install <code>notes-viewer</code></h3>\n\n<h2 id=\"getting-started-1\">Getting started</h2>\n"; got != want {
		t.Fatalf("rendered HTML = %q, want %q", got, want)
	}

	want := []Heading{
		{
			Level: 1,
			ID:    "introduction",
			Text:  "Introduction",
			Children: []Heading{
				{
					Level: 2,
					ID:    "getting-started",
					Text:  "Getting started",
					Children: []Heading{
						{Level: 3, ID: "install-notes-viewer", Text: "Install notes-viewer"},
					},
				},
				{Level: 2, ID: "getting-started-1", Text: "Getting started"},
			},
		},
	}

	if !reflect.DeepEqual(document.Headings, want) {
		t.Fatalf("headings = %#v, want %#v", document.Headings, want)
	}
}

func TestEntryIDLinksOpenInCurrentTarget(t *testing.T) {
	html := string(mdToHTML([]byte("[note](/entry/20230504T162825.id) [external](https://example.com)")))

	if want := `<a href="/entry/20230504T162825.id">note</a>`; !strings.Contains(html, want) {
		t.Fatalf("entry ID link = %q, want it to contain %q", html, want)
	}
	if want := `<a href="https://example.com" target="_blank">external</a>`; !strings.Contains(html, want) {
		t.Fatalf("external link = %q, want it to contain %q", html, want)
	}
}

func TestNewEntry(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want Entry
	}{
		{
			name: "title and tags",
			id:   "20210218T082747--ckeditor__zendesk.md",
			want: Entry{
				ID:        "20210218T082747--ckeditor__zendesk.md",
				Timestamp: time.Date(2021, time.February, 18, 8, 27, 47, 0, time.UTC),
				Title:     "ckeditor",
				Tags:      []string{"zendesk"},
			},
		},
		{
			name: "multiple title words and tags",
			id:   "20240229T123456--my-note-title__work_personal.md",
			want: Entry{
				ID:        "20240229T123456--my-note-title__work_personal.md",
				Timestamp: time.Date(2024, time.February, 29, 12, 34, 56, 0, time.UTC),
				Title:     "my note title",
				Tags:      []string{"work", "personal"},
			},
		},
		{
			name: "without tags",
			id:   "20210218T082747--ckeditor.md",
			want: Entry{
				ID:        "20210218T082747--ckeditor.md",
				Timestamp: time.Date(2021, time.February, 18, 8, 27, 47, 0, time.UTC),
				Title:     "ckeditor",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewEntry(tt.id)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewEntry(%q) = %#v, want %#v", tt.id, got, tt.want)
			}
		})
	}
}
