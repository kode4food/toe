package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/defaults"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

func TestBufferNavigation(t *testing.T) {
	t.Run("next and previous cycle focus", func(t *testing.T) {
		// focus navigation walks the split tree, so a split is needed for the
		// focused view to actually change
		dir := t.TempDir()
		km := command.NewKeymaps()
		e := view.NewEditor(dir)
		e.ResizeTree(80, 24)
		_, _ = defaults.RegisterDefaults(ui.New(e, km), km)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		_, ok = e.VSplit(v.DocID())
		assert.True(t, ok)

		start := mustFocusedView(t, e).ID()
		runCmd(t, km, e, "buffer_next")
		assert.NotEqual(t, start, mustFocusedView(t, e).ID())
		runCmd(t, km, e, "buffer_previous")
		assert.Equal(t, start, mustFocusedView(t, e).ID())
	})
}

func TestBufferClose(t *testing.T) {
	t.Run("close clean buffer drops a view", func(t *testing.T) {
		e, km := twoBufferEnv(t)
		before := len(e.AllViews())
		res := runCmd(t, km, e, "buffer_close")
		assert.Contains(t, res.Message, "closed")
		assert.Less(t, len(e.AllViews()), before)
	})

	t.Run("close warns on unsaved changes", func(t *testing.T) {
		e, km := twoBufferEnv(t)
		setText(t, e, "dirty")
		res := runCmd(t, km, e, "buffer_close")
		assert.Contains(t, res.Message, "unsaved")
	})

	t.Run("close force ignores unsaved changes", func(t *testing.T) {
		e, km := twoBufferEnv(t)
		setText(t, e, "dirty")
		before := len(e.AllViews())
		runCmd(t, km, e, "buffer_close_force")
		assert.Less(t, len(e.AllViews()), before)
	})

	t.Run("close others leaves one view", func(t *testing.T) {
		e, km := twoBufferEnv(t)
		runCmd(t, km, e, "buffer_close_others")
		assert.Equal(t, 1, len(e.AllViews()))
	})

	t.Run("close all clean closes everything", func(t *testing.T) {
		e, km := twoBufferEnv(t)
		res := runCmd(t, km, e, "buffer_close_all")
		assert.Contains(t, res.Message, "all buffers closed")
		assert.Equal(t, 0, len(e.AllViews()))
	})

	t.Run("close all warns on unsaved changes", func(t *testing.T) {
		e, km := twoBufferEnv(t)
		setText(t, e, "dirty")
		res := runCmd(t, km, e, "buffer_close_all")
		assert.Contains(t, res.Message, "unsaved")
	})
}
