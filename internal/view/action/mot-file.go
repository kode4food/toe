package action

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/kode4food/toe/internal/view"
)

// ErrNoFilePath is returned by GotoFile when no file path is found under
// the cursor
var ErrNoFilePath = errors.New("no file path under cursor")

// GotoFile opens the file whose path the primary cursor sits on. Returns the
// resolved path, or an error if no valid path can be found
func GotoFile(e *view.Editor) (string, error) {
	doc, ok := e.FocusedDocument()
	if !ok {
		return "", view.ErrNoDocument
	}
	v, ok := e.FocusedView()
	if !ok {
		return "", view.ErrNoView
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	pos := sel.Primary().Cursor(text)

	// Expand outward from the cursor to capture a file-path token
	n := text.LenChars()
	from := pos
	for from > 0 {
		ch, err := text.CharAt(from - 1)
		if err != nil || isPathDelim(ch) {
			break
		}
		from--
	}
	to := pos
	for to < n {
		ch, err := text.CharAt(to)
		if err != nil || isPathDelim(ch) {
			break
		}
		to++
	}
	if from >= to {
		return "", ErrNoFilePath
	}
	slice, err := text.Slice(from, to)
	if err != nil {
		return "", err
	}
	path := slice.String()

	// Resolve relative paths against the document's directory
	if !strings.HasPrefix(path, "/") {
		base := doc.Path()
		if base != "" {
			base = base[:strings.LastIndex(base, "/")+1]
		} else {
			base = e.Cwd() + "/"
		}
		path = base + path
	}

	// Check the file exists
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("%w: '%s'", err, path)
	}
	return path, nil
}

func isPathDelim(ch rune) bool {
	switch ch {
	case ' ', '\t', '\n', '\r', '"', '\'', '(', ')', '[', ']', '{', '}':
		return true
	default:
		return false
	}
}
