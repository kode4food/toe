package action

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// GotoTarget is the resolved target under the primary cursor
type GotoTarget struct {
	Path string
	URL  string
}

var (
	// ErrNoFilePath is returned when no file path is found under the cursor
	ErrNoFilePath = errors.New("no file path under cursor")
	// ErrDocumentLinkTarget is returned when a caller needs a local path but
	// the document link points elsewhere
	ErrDocumentLinkTarget = errors.New("document link target unsupported")
	// ErrExternalURLOpener is returned when no external URL opener is available
	ErrExternalURLOpener = errors.New("external URL opener unavailable")
)

// GotoFileTarget resolves the file or URL target under the primary cursor
func GotoFileTarget(e *view.Editor) (GotoTarget, error) {
	doc, ok := e.FocusedDocument()
	if !ok {
		return GotoTarget{}, view.ErrNoDocument
	}
	v, ok := e.FocusedView()
	if !ok {
		return GotoTarget{}, view.ErrNoView
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	pos := sel.Primary().Cursor(text)
	target, ok, err := documentLinkTarget(e, doc, sel.Primary())
	if ok || err != nil {
		return target, err
	}

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
		return GotoTarget{}, ErrNoFilePath
	}
	slice, err := text.Slice(from, to)
	if err != nil {
		return GotoTarget{}, err
	}
	path := slice.String()

	// Resolve relative paths against the document's directory
	if !filepath.IsAbs(path) {
		base := doc.Path()
		if base != "" {
			base = filepath.Dir(base)
		} else {
			base = e.Cwd()
		}
		path = filepath.Join(base, path)
	}

	if _, err := os.Stat(path); err != nil {
		return GotoTarget{}, fmt.Errorf("%w: '%s'", err, path)
	}
	return GotoTarget{Path: path}, nil
}

func documentLinkTarget(
	e *view.Editor, doc *view.Document, sel core.Range,
) (GotoTarget, bool, error) {
	for _, link := range doc.DocumentLinks() {
		if !selectionOverlapsDocumentLink(sel, link) {
			continue
		}
		if link.Target == "" {
			resolved, err := resolveDocumentLink(e, doc, link)
			if err != nil {
				return GotoTarget{}, true, err
			}
			link = resolved
		}
		target, err := parseDocumentLinkTarget(link.Target)
		if err != nil {
			return GotoTarget{}, true, err
		}
		if target.URL != "" {
			return target, true, nil
		}
		path := target.Path
		if _, err := os.Stat(path); err != nil {
			return GotoTarget{}, true, fmt.Errorf("%w: '%s'", err, path)
		}
		return target, true, nil
	}
	return GotoTarget{}, false, nil
}

func resolveDocumentLink(
	e *view.Editor, doc *view.Document, link view.DocumentLink,
) (view.DocumentLink, error) {
	ctl := e.LanguageServerController()
	if ctl == nil {
		return link, fmt.Errorf("%w: unresolved", ErrDocumentLinkTarget)
	}
	resolved, err := ctl.ResolveDocumentLink(doc, link)
	if err != nil {
		return link, err
	}
	if resolved.Target == "" {
		return resolved, fmt.Errorf("%w: unresolved", ErrDocumentLinkTarget)
	}
	return resolved, nil
}

func selectionOverlapsDocumentLink(
	sel core.Range, link view.DocumentLink,
) bool {
	if sel.Empty() {
		pos := sel.From()
		return link.From <= pos && pos < link.To
	}
	return sel.From() < link.To && sel.To() > link.From
}

func parseDocumentLinkTarget(target string) (GotoTarget, error) {
	if target == "" {
		return GotoTarget{}, fmt.Errorf(
			"%w: empty target", ErrDocumentLinkTarget,
		)
	}
	u, err := url.Parse(target)
	if err != nil {
		return GotoTarget{}, fmt.Errorf("%w: %s", ErrDocumentLinkTarget, target)
	}
	if u.Scheme == "" {
		return GotoTarget{Path: filepath.Clean(target)}, nil
	}
	if u.Scheme != "file" {
		return GotoTarget{URL: target}, nil
	}
	if u.Host != "" && u.Host != "localhost" {
		return GotoTarget{}, fmt.Errorf("%w: %s", ErrDocumentLinkTarget, target)
	}
	return GotoTarget{Path: filepath.FromSlash(u.Path)}, nil
}

func isPathDelim(ch rune) bool {
	switch ch {
	case ' ', '\t', '\n', '\r', '"', '\'', '(', ')', '[', ']', '{', '}':
		return true
	default:
		return false
	}
}
