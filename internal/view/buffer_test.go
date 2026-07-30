package view_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view"
)

func TestEditorFocusView(t *testing.T) {
	e := view.NewEditor("/tmp")
	v := e.FocusedView()
	assert.NotNil(t, v)
	e.FocusView(v.ID())
	v2 := e.FocusedView()
	assert.NotNil(t, v2)
	assert.Equal(t, v.ID(), v2.ID())
}

func TestEditorView(t *testing.T) {
	e := view.NewEditor("/tmp")
	v := e.FocusedView()
	got := e.View(v.ID())
	assert.NotNil(t, got)
	assert.Equal(t, v.ID(), got.ID())
}

func TestEditorCwd(t *testing.T) {
	e := view.NewEditor("/tmp")
	assert.Equal(t, "/tmp", e.Cwd())
}
