package ui_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/builtin"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
)

func TestHoverComponent(t *testing.T) {
	t.Run("opens hover popup", func(t *testing.T) {
		e := editorWithText(t, "Println")
		e.SetMode(view.ModeNormal)
		e.SetLanguageServerController(&completionController{editor: e})
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendKey(m, ' ')
		m = sendKey(m, 'k')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "hover docs")
	})

	t.Run("dismisses on editor key", func(t *testing.T) {
		e := editorWithText(t, "Println")
		e.SetMode(view.ModeNormal)
		e.SetLanguageServerController(&completionController{editor: e})
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendKey(m, ' ')
		m = sendKey(m, 'k')
		assert.Contains(t, stripANSI(m.View().Content), "hover docs")

		m = sendSpecial(m, tea.KeyRight)
		out := stripANSI(m.View().Content)

		assert.NotContains(t, out, "hover docs")
		assert.Equal(t, 1, testutil.CursorPos(t, e))
	})

	t.Run("does not follow moved cursor", func(t *testing.T) {
		e := editorWithText(t, "Println")
		e.SetMode(view.ModeNormal)
		e.SetLanguageServerController(&completionController{editor: e})
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendKey(m, ' ')
		m = sendKey(m, 'k')
		assert.Contains(t, stripANSI(m.View().Content), "hover docs")

		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(3))
		out := stripANSI(m.View().Content)

		assert.NotContains(t, out, "hover docs")
	})

	t.Run("renders markdown heading", func(t *testing.T) {
		e := editorWithText(t, "Println")
		e.SetMode(view.ModeNormal)
		e.SetLanguageServerController(&completionController{
			editor:    e,
			hoverText: "# Println\n\nPrints to standard output.\n\n",
		})
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendKey(m, ' ')
		m = sendKey(m, 'k')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "Println")
		assert.Contains(t, out, "Prints to standard output.")
	})

	for _, tc := range []struct {
		name string
		text string
	}{
		{"hides leading separator", "---\ncontent"},
		{"hides trailing separator", "content\n---"},
		{"hides separator before empty code", "content\n---\n```go\n \n```"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := editorWithText(t, "Println")
			e.SetMode(view.ModeNormal)
			e.SetLanguageServerController(&completionController{
				editor:    e,
				hoverText: tc.text,
			})
			km := command.NewKeymaps()
			m := ui.New(e, km)
			_, err := builtin.Register(m, km)
			assert.NoError(t, err)
			m = resize(m, 80, 24)
			m = sendKey(m, ' ')
			m = sendKey(m, 'k')
			out := stripANSI(m.View().Content)
			assert.Contains(t, out, "content")
			assert.NotContains(t, out, "├─")
		})
	}

	t.Run("renders thematic break as full-width rule", func(t *testing.T) {
		e := editorWithText(t, "Println")
		e.SetMode(view.ModeNormal)
		e.SetLanguageServerController(&completionController{
			editor:    e,
			hoverText: "above\n\n---\n\nbelow",
		})
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendKey(m, ' ')
		m = sendKey(m, 'k')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "above")
		assert.Contains(t, out, "below")
		// the literal --- becomes a border-tied horizontal rule
		assert.NotContains(t, out, "---")
		assert.Contains(t, out, "├─")
	})

	t.Run("renders code block in hover popup", func(t *testing.T) {
		e := editorWithText(t, "Println")
		e.SetMode(view.ModeNormal)
		e.SetLanguageServerController(&completionController{
			editor: e,
			hoverText: "# Println\n\n```go\n" +
				"func Println(a ...any) (n int, err error)\n```\n",
		})
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendKey(m, ' ')
		m = sendKey(m, 'k')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "Println")
		assert.Contains(t, out, "func Println")
	})
}
