package files_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/geom"
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
		e.ResizeTree(geom.Size{Width: 80, Height: 24})
		_, _ = builtin.Register(ui.New(e, km), km)
		v := e.FocusedView()
		assert.NotNil(t, v)
		split := e.VSplit(v.DocID())
		assert.NotNil(t, split)

		start := test.MustFocusedView(t, e).ID()
		test.RunCmd(t, km, e, "buffer_next")
		assert.NotEqual(t, start, test.MustFocusedView(t, e).ID())
		test.RunCmd(t, km, e, "buffer_previous")
		assert.Equal(t, start, test.MustFocusedView(t, e).ID())
	})
}

func TestBufferClose(t *testing.T) {
	t.Run("close clean shows scratch", func(t *testing.T) {
		e, km := test.TwoBufferEnv(t)
		doc := e.FocusedDocument()
		res := test.RunCmd(t, km, e, "buffer_close")
		assert.Contains(t, res.Message, "closed")
		assert.Equal(t, 1, len(e.AllViews()))
		assert.NotSame(t, doc, e.FocusedDocument())
	})

	t.Run("close warns on unsaved changes", func(t *testing.T) {
		e, km := test.TwoBufferEnv(t)
		testutil.SetEditorText(t, e, "dirty")
		res := test.RunCmd(t, km, e, "buffer_close")
		assert.Contains(t, res.Message, "unsaved")
	})

	t.Run("close force shows scratch", func(t *testing.T) {
		e, km := test.TwoBufferEnv(t)
		doc := e.FocusedDocument()
		testutil.SetEditorText(t, e, "dirty")
		test.RunCmd(t, km, e, "buffer_close_force")
		assert.Equal(t, 1, len(e.AllViews()))
		assert.NotSame(t, doc, e.FocusedDocument())
	})

	t.Run("close others leaves one view", func(t *testing.T) {
		e, km := test.TwoBufferEnv(t)
		test.RunCmd(t, km, e, "buffer_close_others")
		assert.Equal(t, 1, len(e.AllViews()))
	})

	t.Run("close all clean leaves one pane", func(t *testing.T) {
		e, km := test.TwoBufferEnv(t)
		res := test.RunCmd(t, km, e, "buffer_close_all")
		assert.Contains(t, res.Message, "all buffers closed")
		assert.Equal(t, 1, len(e.AllViews()))
	})

	t.Run("close all warns on unsaved changes", func(t *testing.T) {
		e, km := test.TwoBufferEnv(t)
		testutil.SetEditorText(t, e, "dirty")
		res := test.RunCmd(t, km, e, "buffer_close_all")
		assert.Contains(t, res.Message, "unsaved")
	})

	t.Run("closes the named buffer", func(t *testing.T) {
		e, km := test.TwoBufferEnv(t)
		_ = e.SplitFocused(view.LayoutVertical)
		target := e.FocusedDocument().RelativeName(e.Cwd())
		before := len(e.AllViews())

		res := test.RunCmdArgs(t, km, e, "buffer_close", target)

		assert.Contains(t, res.Message, "buffer closed")
		assert.Less(t, len(e.AllViews()), before)
	})

	t.Run("unknown buffer name errors", func(t *testing.T) {
		e, km := test.TwoBufferEnv(t)
		before := len(e.AllViews())

		res := test.RunCmdArgs(t, km, e, "buffer_close", "nope.txt")

		assert.Contains(t, res.Message, "no such buffer")
		assert.Equal(t, before, len(e.AllViews()))
	})
}
