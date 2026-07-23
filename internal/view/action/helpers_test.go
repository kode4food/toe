package action_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view"
)

type stubLSP struct {
	view.LanguageServerController
	resolveLink func(
		*view.Document, view.DocumentLink,
	) (view.DocumentLink, error)
}

var _ view.LanguageServerController = (*stubLSP)(nil)

func (s *stubLSP) ResolveDocumentLink(
	doc *view.Document, link view.DocumentLink,
) (view.DocumentLink, error) {
	if s.resolveLink != nil {
		return s.resolveLink(doc, link)
	}
	return link, nil
}

func editorWithNoView(t *testing.T) *view.Editor {
	t.Helper()
	e := view.NewEditor("/tmp")
	v, ok := e.FocusedView()
	assert.True(t, ok)
	e.CloseView(v.ID())
	return e
}

func viewCount(t *testing.T, e *view.Editor) int {
	t.Helper()
	return len(e.AllViews())
}
