package files_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/builtin"
	"github.com/kode4food/toe/internal/term/builtin/test"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/testutil"
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
		_, _ = builtin.Register(ui.New(e, km), km)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		_, ok = e.VSplit(v.DocID())
		assert.True(t, ok)

		start := test.MustFocusedView(t, e).ID()
		test.RunCmd(t, km, e, "buffer_next")
		assert.NotEqual(t, start, test.MustFocusedView(t, e).ID())
		test.RunCmd(t, km, e, "buffer_previous")
		assert.Equal(t, start, test.MustFocusedView(t, e).ID())
	})
}

func TestBufferClose(t *testing.T) {
	t.Run("close clean buffer drops a view", func(t *testing.T) {
		e, km := test.TwoBufferEnv(t)
		before := len(e.AllViews())
		res := test.RunCmd(t, km, e, "buffer_close")
		assert.Contains(t, res.Message, "closed")
		assert.Less(t, len(e.AllViews()), before)
	})

	t.Run("close warns on unsaved changes", func(t *testing.T) {
		e, km := test.TwoBufferEnv(t)
		testutil.SetEditorText(t, e, "dirty")
		res := test.RunCmd(t, km, e, "buffer_close")
		assert.Contains(t, res.Message, "unsaved")
	})

	t.Run("close force ignores unsaved changes", func(t *testing.T) {
		e, km := test.TwoBufferEnv(t)
		testutil.SetEditorText(t, e, "dirty")
		before := len(e.AllViews())
		test.RunCmd(t, km, e, "buffer_close_force")
		assert.Less(t, len(e.AllViews()), before)
	})

	t.Run("close others leaves one view", func(t *testing.T) {
		e, km := test.TwoBufferEnv(t)
		test.RunCmd(t, km, e, "buffer_close_others")
		assert.Equal(t, 1, len(e.AllViews()))
	})

	t.Run("close all clean closes everything", func(t *testing.T) {
		e, km := test.TwoBufferEnv(t)
		res := test.RunCmd(t, km, e, "buffer_close_all")
		assert.Contains(t, res.Message, "all buffers closed")
		assert.Equal(t, 0, len(e.AllViews()))
	})

	t.Run("close all warns on unsaved changes", func(t *testing.T) {
		e, km := test.TwoBufferEnv(t)
		testutil.SetEditorText(t, e, "dirty")
		res := test.RunCmd(t, km, e, "buffer_close_all")
		assert.Contains(t, res.Message, "unsaved")
	})
}
