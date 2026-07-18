package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

func TestOpenPathImage(t *testing.T) {
	openImage := func(t *testing.T) (*view.Editor, bool, error) {
		t.Helper()
		e := view.NewEditor(t.TempDir())
		e.ResizeTree(80, 24)
		path := writeRenderImage(t, t.TempDir(), 20, 10, nil)
		_, ok, err := ui.OpenPath(e, path, ui.PickerAcceptReplace)
		return e, ok, err
	}

	// The pane is created regardless of terminal support; the pane itself
	// presents the unsupported message and skips transmission
	t.Run("unsupported terminal opens pane", func(t *testing.T) {
		for _, k := range []string{
			"KITTY_WINDOW_ID", "TERM", "TERM_PROGRAM", "KONSOLE_VERSION",
		} {
			t.Setenv(k, "")
		}
		e, ok, err := openImage(t)
		assert.NoError(t, err)
		assert.True(t, ok)
		_, isImage := e.Tree().Get(e.Tree().Focus()).(*ui.ImagePane)
		assert.True(t, isImage)
	})

	t.Run("supported terminal opens pane", func(t *testing.T) {
		t.Setenv("KITTY_WINDOW_ID", "1")
		e, ok, err := openImage(t)
		assert.NoError(t, err)
		assert.True(t, ok)
		_, isImage := e.Tree().Get(e.Tree().Focus()).(*ui.ImagePane)
		assert.True(t, isImage)
	})
}
