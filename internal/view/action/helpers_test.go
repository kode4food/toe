package action_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view"
)

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

func registeredValue(t *testing.T, e *view.Editor, reg rune) string {
	t.Helper()
	v, ok := e.Registers().First(reg)
	assert.True(t, ok)
	return v
}
