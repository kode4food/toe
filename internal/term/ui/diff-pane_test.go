package ui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

func TestDiffPane(t *testing.T) {
	t.Run("renders and navigates", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "main.go")
		old := "package main\n\nfunc main() {\n\tprintln(\"old\")\n}\n"
		proposed := "package main\n\nfunc main() {\n\tprintln(\"new\")\n}\n"
		assert.NoError(t, os.WriteFile(path, []byte(old), 0o644))
		e := view.NewEditor(root)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		m := resize(ui.New(e, command.NewKeymaps()), 80, 8)

		m = updateAndFeed(m, ui.OpenDiffMsg{
			TabName:     "main",
			Path:        path,
			NewContents: proposed,
		})
		pane := diffPane(e)
		assert.NotNil(t, pane)
		assert.Equal(t, view.ModeDiff, pane.Mode())
		assert.Equal(t, path, pane.Path())
		assert.Equal(t, 2, e.Tree().Count())

		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "…")
		assert.Contains(t, out, `-     println("old")`)
		assert.Contains(t, out, `+     println("new")`)

		second := filepath.Join(root, "second.go")
		lines := strings.Repeat("line\n", 12)
		m = updateAndFeed(m, ui.OpenDiffMsg{
			TabName:     "second",
			Path:        second,
			NewContents: lines,
		})
		m = updateAndFeed(m, ui.OpenDiffMsg{
			TabName:     "second",
			Path:        second,
			NewContents: lines + "last\n",
		})

		cx := &ui.Context{Editor: e}
		for _, msg := range []tea.Msg{
			tea.KeyPressMsg{Code: tea.KeyDown},
			tea.KeyPressMsg{Code: tea.KeyUp},
			tea.KeyPressMsg{Code: tea.KeyPgDown},
			tea.KeyPressMsg{Code: tea.KeyKpPgDown},
			tea.KeyPressMsg{Code: tea.KeyPgUp},
			tea.KeyPressMsg{Code: tea.KeyKpPgUp},
			tea.KeyPressMsg{Code: tea.KeyTab},
			tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift},
			tea.MouseWheelMsg{Button: tea.MouseWheelDown},
			tea.MouseWheelMsg{Button: tea.MouseWheelUp},
		} {
			_, consumed := pane.HandleEvent(cx, msg)
			assert.True(t, consumed)
		}
		_, consumed := pane.HandleEvent(cx,
			tea.MouseWheelMsg{Button: tea.MouseWheelRight})
		assert.False(t, consumed)
		_, consumed = pane.HandleEvent(cx, tea.KeyPressMsg{
			Code: 'x',
			Text: "x",
		})
		assert.False(t, consumed)
		_, consumed = pane.HandleEvent(cx, struct{}{})
		assert.False(t, consumed)

		id := pane.ID()
		pane.SetID(id)
		assert.Equal(t, id, pane.ID())
		pane.SetArea(geom.Area{Width: 20, Height: 3})
		assert.Equal(t, geom.Area{Width: 20, Height: 3}, pane.Area())
		assert.True(t, pane.ConsumeDirty())
		assert.False(t, pane.ConsumeDirty())
		pane.MarkDirty()
		assert.True(t, pane.ConsumeDirty())
		_, ok := pane.Cursor(cx)
		assert.False(t, ok)
		_, err = pane.Split()
		assert.ErrorIs(t, err, view.ErrNoView)
		pane.SaveSession(nil)
		pane.Discard()
		pane.Shutdown()

		m = updateAndFeed(m, ui.CloseDiffMsg{TabName: "missing"})
		m = updateAndFeed(m, ui.CloseDiffMsg{TabName: "second"})
		assert.NotNil(t, diffPane(e))
		_, _ = m.Update(ui.CloseDiffMsg{})
		assert.Nil(t, diffPane(e))
		assert.Equal(t, 1, e.Tree().Count())
	})

	t.Run("preserves panes and focus", func(t *testing.T) {
		e := editorWithText(t, "old\n")
		m := ui.New(e, command.NewKeymaps())
		m = updateAndFeed(m, tea.WindowSizeMsg{Width: 100, Height: 12})
		assert.NotNil(t, e.VSplitNew())
		focus := e.Tree().Focus()
		shown := make(chan bool, 1)

		m = updateAndFeed(m, ui.OpenDiffMsg{
			TabName:     "change",
			Path:        "missing.txt",
			NewContents: "new\n",
			Shown:       shown,
		})
		assert.True(t, <-shown)
		assert.NotNil(t, diffPane(e))
		assert.Equal(t, 3, e.Tree().Count())
		assert.Equal(t, focus, e.Tree().Focus())

		_, _ = m.Update(ui.CloseDiffMsg{TabName: "change"})
		assert.Nil(t, diffPane(e))
		assert.Equal(t, 2, e.Tree().Count())
	})

	for _, tc := range []struct {
		name     string
		width    int
		maximize bool
	}{
		{name: "narrow", width: 20},
		{name: "maximized", width: 80, maximize: true},
	} {
		t.Run("declines "+tc.name, func(t *testing.T) {
			e := editorWithText(t, "old\n")
			m := ui.New(e, command.NewKeymaps())
			m = updateAndFeed(
				m, tea.WindowSizeMsg{Width: tc.width, Height: 12},
			)
			if tc.maximize {
				assert.NotNil(t, e.VSplitNew())
				e.TogglePaneMaximized()
			}
			count := e.Tree().Count()
			shown := make(chan bool, 1)

			updateAndFeed(m, ui.OpenDiffMsg{
				TabName:     "change",
				Path:        "missing.txt",
				NewContents: "new\n",
				Shown:       shown,
			})

			assert.False(t, <-shown)
			assert.Nil(t, diffPane(e))
			assert.Equal(t, count, e.Tree().Count())
		})
	}

	t.Run("close absent pane", func(t *testing.T) {
		e := editorWithText(t, "text")
		m := ui.New(e, command.NewKeymaps())

		_, _ = m.Update(ui.CloseDiffMsg{TabName: "missing"})

		assert.Nil(t, diffPane(e))
	})
}

func diffPane(e *view.Editor) *ui.DiffPane {
	var out *ui.DiffPane
	e.Tree().Range(func(p view.Pane) bool {
		if pane, ok := p.(*ui.DiffPane); ok {
			out = pane
			return false
		}
		return true
	})
	return out
}
