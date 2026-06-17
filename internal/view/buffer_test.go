package view_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view"
)

func TestEditorFocusView(t *testing.T) {
	e := view.NewEditor("/tmp")
	v, ok := e.FocusedView()
	assert.True(t, ok)
	e.FocusView(v.ID())
	v2, ok := e.FocusedView()
	assert.True(t, ok)
	assert.Equal(t, v.ID(), v2.ID())
}

func TestEditorView(t *testing.T) {
	e := view.NewEditor("/tmp")
	v, _ := e.FocusedView()
	got, ok := e.View(v.ID())
	assert.True(t, ok)
	assert.Equal(t, v.ID(), got.ID())
}

func TestEditorCwd(t *testing.T) {
	e := view.NewEditor("/tmp")
	assert.Equal(t, "/tmp", e.Cwd())
}
