package ui_test

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/builtin"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

func TestSignatureHelpComponent(t *testing.T) {
	t.Run("prefers below cursor", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{editor: e}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 40)

		m = sendKeyAndFeed(m, '(')
		lines := strings.Split(stripANSI(m.View().Content), "\n")
		docY := signatureLineIndex(lines, "()")
		sigY := signatureLineIndex(lines, "Println(a ...any)")

		assert.Greater(t, sigY, docY)
	})

	t.Run("opens after trigger character", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{editor: e}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendKeyAndFeed(m, '(')
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
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendKeyAndFeed(m, '(')
		m = updateAndFeed(m, tea.KeyPressMsg{
			Code: 'n', Text: "n", Mod: tea.ModAlt,
		})
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
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendKeyAndFeed(m, '(')
		assert.Contains(t, stripANSI(m.View().Content), "format parameter")

		m = sendKeyAndFeed(m, ',')
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
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendKeyAndFeed(m, '(')
		assert.Contains(t, stripANSI(m.View().Content), "signature docs")

		m = sendSpecialAndFeed(m, tea.KeyLeft)
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
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendKeyAndFeed(m, '(')
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
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendKeyAndFeed(m, '(')
		assert.Contains(t, stripANSI(m.View().Content), "signature docs")

		m = sendSpecialAndFeed(m, tea.KeyEscape)
		m = sendKeyAndFeed(m, ',')
		out := stripANSI(m.View().Content)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		assert.Equal(t, "(,)", doc.Text().String())
		assert.NotContains(t, out, "signature docs")
	})

	t.Run("moves backward with alt-p", func(t *testing.T) {
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
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendKeyAndFeed(m, '(')
		// cycle forward then back
		m = updateAndFeed(m, tea.KeyPressMsg{
			Code: 'n', Text: "n", Mod: tea.ModAlt,
		})
		m = updateAndFeed(m, tea.KeyPressMsg{
			Code: 'p', Text: "p", Mod: tea.ModAlt,
		})
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "Println(a ...any)")
	})

	t.Run("empty help dismisses", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{editor: e}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendKeyAndFeed(m, '(')
		assert.Contains(t, stripANSI(m.View().Content), "signature docs")

		ctl.signatureEmpty = true
		m = sendKeyAndFeed(m, 'x')
		out := stripANSI(m.View().Content)

		assert.NotContains(t, out, "signature docs")
	})

	t.Run("dismisses on signature help error", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{editor: e}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendKeyAndFeed(m, '(')
		assert.Contains(t, stripANSI(m.View().Content), "signature docs")

		ctl.signatureErr = errors.New("sig error")
		m = sendKeyAndFeed(m, 'x')
		out := stripANSI(m.View().Content)

		assert.NotContains(t, out, "signature docs")
	})

	t.Run("renders with unfocused cursor position", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{editor: e}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendKeyAndFeed(m, '(')
		assert.Contains(t, stripANSI(m.View().Content), "signature docs")

		m = updateAndFeed(m, tea.BlurMsg{})
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "signature docs")
	})
}

func signatureLineIndex(lines []string, text string) int {
	for i, line := range lines {
		if strings.Contains(line, text) {
			return i
		}
	}
	return -1
}
