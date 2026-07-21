package ui_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

func TestImagePane(t *testing.T) {
	t.Run("loads image", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		path := writeRenderImage(t, t.TempDir(), 20, 10, nil)
		pane, err := ui.NewImagePane(e, path)
		assert.NoError(t, err)
		size := pane.Image().Size()
		assert.Equal(t, 20, size.Width)
		assert.Equal(t, 10, size.Height)
		assert.Equal(t, view.ModeImage, pane.Mode())
	})

	t.Run("reloads after external change", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		dir := t.TempDir()
		path := writeRenderImage(t, dir, 20, 10, nil)
		pane, err := ui.NewImagePane(e, path)
		assert.NoError(t, err)
		before := pane.Image().ContentID()

		// overwrite pic.png at the same path with different dimensions
		writeRenderImage(t, dir, 30, 12, nil)
		assert.NoError(t, pane.Reload())

		assert.NotEqual(t, before, pane.Image().ContentID())
		assert.Equal(t, geom.Size{Width: 30, Height: 12}, pane.Image().Size())
	})

	t.Run("document rejects image", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		_, err := e.OpenFile(writeRenderImage(t, t.TempDir(), 20, 10, nil))
		assert.Error(t, err)
	})

	t.Run("survives session restore", func(t *testing.T) {
		dir := t.TempDir()
		e := view.NewEditor(dir)
		_ = ui.New(e, command.NewKeymaps()) // registers the image pane factory
		pane, err := ui.NewImagePane(e, writeRenderImage(t, dir, 20, 10, nil))
		assert.NoError(t, err)
		pane.ZoomIn()
		pane.ZoomIn()
		e.ReplacePane(e.Tree().Focus(), pane)
		session := filepath.Join(dir, "session.toml")
		assert.NoError(t, e.SaveSession(session, nil))

		next := view.NewEditor(dir)
		_ = ui.New(next, command.NewKeymaps())
		_, restored, err := next.RestoreSession(session)
		assert.NoError(t, err)
		assert.True(t, restored)
		restoredPane, ok := next.Tree().Get(next.Tree().Focus()).(*ui.ImagePane)
		assert.True(t, ok)
		assert.Equal(t, 150, restoredPane.Zoom())
	})

	t.Run("supports pane commands", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		path := writeRenderImage(t, t.TempDir(), 20, 10, nil)
		pane, err := ui.NewImagePane(e, path)
		assert.NoError(t, err)
		old := e.ReplacePane(e.Tree().Focus(), pane)
		e.DiscardPane(old)

		action.ImageZoomIn(e)
		assert.Equal(t, 125, pane.Zoom())

		action.VSplit(e)
		assert.Equal(t, 2, e.Tree().Count())
		split, ok := e.FocusedPane().(*ui.ImagePane)
		assert.True(t, ok)
		assert.Equal(t, pane.Zoom(), split.Zoom())
		first := e.Tree().Focus()

		action.RotateView(e)
		assert.NotEqual(t, first, e.Tree().Focus())

		action.CloseOtherViews(e)
		assert.Equal(t, 1, e.Tree().Count())

		action.CloseCurrentView(e)
		_, ok = e.Tree().Get(e.Tree().Focus()).(*view.View)
		assert.True(t, ok)
	})
}
