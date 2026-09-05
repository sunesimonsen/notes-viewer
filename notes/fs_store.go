package notes

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// User represents an authenticated user for authorization checks.
type User interface {
	Verified() bool
}

type FSStore struct {
	FS fs.FS
}

type Entry struct {
	ID        string
	Timestamp time.Time
	Title     string
	Tags      []string
}

// Heading is an entry in a note's table of contents.
// Children contains headings nested beneath this heading in the document.
type Heading struct {
	Level    int
	ID       string
	Text     string
	Children []Heading
}

// RenderedDocument contains the rendered note and the headings used to build
// its table of contents. The headings are extracted from the same AST that is
// rendered into HTML, so their IDs always match the anchors in HTML.
type RenderedDocument struct {
	HTML     []byte
	Headings []Heading
}

func NewEntry(id string) Entry {
	entry := Entry{ID: id}

	filename := filepath.Base(id)
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))

	timestampAndTitle, tags, _ := strings.Cut(filename, "__")
	timestampText, title, ok := strings.Cut(timestampAndTitle, "--")
	if !ok {
		return entry
	}

	timestamp, err := time.Parse("20060102T150405", timestampText)
	if err != nil {
		return entry
	}

	entry.Timestamp = timestamp
	entry.Title = strings.ReplaceAll(title, "-", " ")
	if tags != "" {
		entry.Tags = strings.Split(tags, "_")
	}

	return entry
}

type SearchResult struct {
	Entries   []Entry
	HasMore   bool
	EndCursor string
}

func matches(text string, pattern string) bool {
	terms := strings.SplitSeq(strings.ToLower(pattern), " ")

	lowerText := strings.ToLower(text)
	for term := range terms {
		if !strings.Contains(lowerText, term) {
			return false
		}
	}

	return true
}

func (q FSStore) Search(user User, text string, cursor string, limit int) (SearchResult, error) {
	result := SearchResult{}
	if limit <= 0 {
		return result, nil
	}

	if !user.Verified() {
		return result, fmt.Errorf("user not verified: %w", ErrForbidden)
	}

	err := fs.WalkDir(q.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}

		if !matches(path, text) {
			return nil
		}
		if cursor != "" && path <= cursor {
			return nil
		}
		if len(result.Entries) >= limit {
			result.HasMore = true
			return fs.SkipAll
		}
		result.Entries = append(result.Entries, NewEntry(path))

		return nil
	})

	if err != nil {
		return result, fmt.Errorf("listing notes: %w", err)
	}
	if len(result.Entries) > 0 {
		result.EndCursor = result.Entries[len(result.Entries)-1].ID
	}

	return result, nil
}

func closeFile(f fs.File) {
	err := f.Close()
	if err != nil {
		log.Printf("error closing file: %v", err)
	}
}

// mdToDocument parses the note once and uses that AST for both rendering and
// heading extraction. Rendering first is intentional: the HTML renderer makes
// explicitly repeated heading IDs unique, and the extracted IDs must match the
// IDs that ended up in the HTML.
func mdToDocument(md []byte) RenderedDocument {
	// create markdown parser with extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	// create HTML renderer with extensions
	htmlFlags := html.CommonFlags
	renderer := html.NewRenderer(html.RendererOptions{Flags: htmlFlags})
	renderer.Opts.RenderNodeHook = func(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
		link, ok := node.(*ast.Link)
		if !ok {
			return ast.GoToNext, false
		}

		if id, ok := entryIDFromLink(link.Destination); ok {
			link.Destination = []byte("/note/" + url.PathEscape(id))
		} else {
			link.AdditionalAttributes = append(link.AdditionalAttributes, `target="_blank"`)
		}

		renderer.Link(w, link, entering)
		return ast.GoToNext, true
	}
	output := markdown.Render(doc, renderer)

	return RenderedDocument{
		HTML:     output,
		Headings: documentHeadings(doc),
	}
}

// documentHeadings collects headings in document order and turns the flat AST
// traversal into a nested tree. A skipped level (for example h2 -> h4) is
// treated as a child of the preceding heading, which is how most TOCs handle
// imperfect but valid Markdown outlines.
func documentHeadings(doc ast.Node) []Heading {
	flat := make([]Heading, 0)
	ast.WalkFunc(doc, func(node ast.Node, entering bool) ast.WalkStatus {
		if !entering {
			return ast.GoToNext
		}

		heading, ok := node.(*ast.Heading)
		if !ok || heading.IsTitleblock || heading.HeadingID == "" {
			return ast.GoToNext
		}

		flat = append(flat, Heading{
			Level: heading.Level,
			ID:    heading.HeadingID,
			Text:  astText(heading),
		})
		return ast.SkipChildren
	})

	roots := make([]Heading, 0, len(flat))
	stack := make([]*Heading, 0, len(flat))
	for _, heading := range flat {
		for len(stack) > 0 && heading.Level <= stack[len(stack)-1].Level {
			stack = stack[:len(stack)-1]
		}

		if len(stack) == 0 {
			roots = append(roots, heading)
			stack = append(stack, &roots[len(roots)-1])
			continue
		}

		parent := stack[len(stack)-1]
		parent.Children = append(parent.Children, heading)
		stack = append(stack, &parent.Children[len(parent.Children)-1])
	}

	return roots
}

// astText returns the visible text of an inline heading without carrying
// inline markup into the TOC. Text and code literals are leaves; links,
// emphasis, and images are containers whose visible content is in Children.
func astText(node ast.Node) string {
	var text strings.Builder
	var visit func(ast.Node)
	visit = func(node ast.Node) {
		switch n := node.(type) {
		case *ast.Text:
			text.Write(n.Literal)
		case *ast.Code:
			text.Write(n.Literal)
		case *ast.Softbreak, *ast.Hardbreak, *ast.NonBlockingSpace:
			text.WriteByte(' ')
		default:
			for _, child := range node.GetChildren() {
				visit(child)
			}
		}
	}
	visit(node)
	return strings.Join(strings.Fields(text.String()), " ")
}

func entryIDFromLink(destination []byte) (string, bool) {
	const suffix = ".id"

	path := string(destination)
	id := strings.TrimSuffix(path, suffix)
	return id, id != "" && id != path
}

// mdToHTML is kept as a small compatibility wrapper for callers that only
// need the rendered body.
func mdToHTML(md []byte) []byte {
	return mdToDocument(md).HTML
}

func (q FSStore) ReadDocument(user User, path string) (RenderedDocument, error) {
	if !user.Verified() {
		return RenderedDocument{}, fmt.Errorf("user not verified: %w", ErrForbidden)
	}

	file, err := q.FS.Open(path)
	if err != nil {
		return RenderedDocument{}, fmt.Errorf("opening note file: %w", err)
	}
	defer closeFile(file)

	md, err := io.ReadAll(file)
	if err != nil {
		return RenderedDocument{}, fmt.Errorf("reading note file: %w", err)
	}

	return mdToDocument(md), nil
}

func (q FSStore) Read(user User, path string) ([]byte, error) {
	document, err := q.ReadDocument(user, path)
	if err != nil {
		return nil, err
	}
	return document.HTML, nil
}
