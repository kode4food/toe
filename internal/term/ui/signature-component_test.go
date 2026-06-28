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

func TestSignatureHelpComponent(t *testing.T) {
	t.Run("opens after trigger character", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{editor: e}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m2, _ := m.Update(tea.KeyPressMsg{Code: '(', Text: "("})
		m = m2.(ui.Model)
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "Println(a ...any)")
		assert.Contains(t, out, "signature docs")
		assert.Contains(t, out, "├")
		assert.Contains(t, out, "┤")
	})

	t.Run("cycles signatures", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			signature: view.SignatureHelp{
				Signatures: []view.SignatureInformation{
					{Label: "Println(a ...any)"},
					{Label: "Printf(format string, a ...any)"},
				},
			},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m2, _ := m.Update(tea.KeyPressMsg{Code: '(', Text: "("})
		m = m2.(ui.Model)
		m2, _ = m.Update(tea.KeyPressMsg{
			Code: 'n', Text: "n", Mod: tea.ModAlt,
		})
		m = m2.(ui.Model)
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "Printf(format string, a ...any)")
		assert.Contains(t, out, "(2/2)")
	})

	t.Run("updates on parameter typing", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			signature: view.SignatureHelp{
				Signatures: []view.SignatureInformation{
					{
						Label:       "Printf(format string, a ...any)",
						Docs:        "signature docs",
						ParamDocs:   "format parameter",
						ActiveStart: 7,
						ActiveEnd:   20,
					},
				},
			},
			signatureAfterComma: view.SignatureHelp{
				Signatures: []view.SignatureInformation{
					{
						Label:       "Printf(format string, a ...any)",
						Docs:        "signature docs",
						ParamDocs:   "args parameter",
						ActiveStart: 22,
						ActiveEnd:   30,
					},
				},
			},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m2, _ := m.Update(tea.KeyPressMsg{Code: '(', Text: "("})
		m = m2.(ui.Model)
		assert.Contains(t, stripANSI(m.View().Content), "format parameter")

		m = sendKey(m, ',')
		out := stripANSI(m.View().Content)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		assert.Equal(t, "(,)", doc.Text().String())
		assert.Contains(t, out, "args parameter")
		assert.NotContains(t, out, "format parameter")
	})

	t.Run("dismisses on navigation", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{editor: e}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m2, _ := m.Update(tea.KeyPressMsg{Code: '(', Text: "("})
		m = m2.(ui.Model)
		assert.Contains(t, stripANSI(m.View().Content), "signature docs")

		m = sendSpecial(m, tea.KeyLeft)
		out := stripANSI(m.View().Content)

		assert.NotContains(t, out, "signature docs")
	})

	t.Run("does not follow outside call", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{editor: e}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m2, _ := m.Update(tea.KeyPressMsg{Code: '(', Text: "("})
		m = m2.(ui.Model)
		assert.Contains(t, stripANSI(m.View().Content), "signature docs")

		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(2))
		out := stripANSI(m.View().Content)

		assert.NotContains(t, out, "signature docs")
	})

	t.Run("stays dismissed for same call", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{editor: e}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m2, _ := m.Update(tea.KeyPressMsg{Code: '(', Text: "("})
		m = m2.(ui.Model)
		assert.Contains(t, stripANSI(m.View().Content), "signature docs")

		m = sendSpecial(m, tea.KeyEscape)
		m = sendKey(m, ',')
		out := stripANSI(m.View().Content)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		assert.Equal(t, "(,)", doc.Text().String())
		assert.NotContains(t, out, "signature docs")
	})
}
