package ui_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/defaults"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

func TestHoverComponent(t *testing.T) {
	t.Run("opens hover popup", func(t *testing.T) {
		e := editorWithText(t, "Println")
		e.SetMode(view.ModeNormal)
		e.SetLanguageServerController(&completionController{editor: e})
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
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
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendKey(m, ' ')
		m = sendKey(m, 'k')
		assert.Contains(t, stripANSI(m.View().Content), "hover docs")

		m = sendSpecial(m, tea.KeyRight)
		out := stripANSI(m.View().Content)

		assert.NotContains(t, out, "hover docs")
		assert.Equal(t, 1, cursorPos(t, e))
	})

	t.Run("does not follow moved cursor", func(t *testing.T) {
		e := editorWithText(t, "Println")
		e.SetMode(view.ModeNormal)
		e.SetLanguageServerController(&completionController{editor: e})
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
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
}
