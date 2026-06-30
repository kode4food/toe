package ui_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/defaults"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
	action "github.com/kode4food/toe/internal/view/action"
)

type completionController struct {
	editor              *view.Editor
	items               []view.CompletionItem
	item                view.CompletionItem
	docs                string
	signature           view.SignatureHelp
	signatureAfterComma view.SignatureHelp
	signatureErr        error
	signatureEmpty      bool
}

func TestCompletionComponent(t *testing.T) {
	t.Run("opens and accepts completion", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{Label: "Printf", Insert: "Printf", Kind: "function"},
				{
					Label:     "Println",
					Insert:    "Println",
					Kind:      "function",
					Preselect: true,
				},
			},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendModifiedAndFeed(m, 'x', tea.ModCtrl)
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "Println")

		_ = sendSpecial(m, tea.KeyEnter)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		assert.Equal(t, "Println", doc.Text().String())
		assert.Equal(t, "Println", ctl.item.Label)
	})

	t.Run("opens after trigger character", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{Label: "Name", Insert: "Name", Kind: "field"},
			},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendKeyAndFeed(m, '.')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "Name")
		assert.Contains(t, out, "field")
		assert.NotContains(t, out, "detail")
	})

	t.Run("shows completion docs", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{
					ID:     "one",
					Label:  "Println",
					Insert: "Println",
					Kind:   "function",
				},
			},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendModifiedAndFeed(m, 'x', tea.ModCtrl)
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "resolved docs")
	})

	t.Run("markdown code block no syntax spans", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			docs:   "# Println\n\n```unknownlang\nhello\n```",
			items: []view.CompletionItem{
				{
					ID:     "one",
					Label:  "Println",
					Insert: "Println",
					Kind:   "function",
				},
			},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendModifiedAndFeed(m, 'x', tea.ModCtrl)
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "Println")
	})

	t.Run("renders markdown docs", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			docs:   "# Println\n\n```go\nfunc main() {}\n```",
			items: []view.CompletionItem{
				{
					ID:     "one",
					Label:  "Println",
					Insert: "Println",
					Kind:   "function",
				},
			},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendModifiedAndFeed(m, 'x', tea.ModCtrl)
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "Println")
		assert.Contains(t, out, "func main() {}")
		assert.NotContains(t, out, "```")
	})

	t.Run("shows docs above popup on narrow screen", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			docs:   "docs for narrow screen",
			items: []view.CompletionItem{
				{
					ID:     "one",
					Label:  "Println",
					Insert: "Println",
					Kind:   "function",
				},
			},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		// Narrow screen forces docs above/below the popup
		m = resize(m, 40, 24)

		m = sendModifiedAndFeed(m, 'x', tea.ModCtrl)
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "Println")
	})

	t.Run("shows scroll thumb", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		items := make([]view.CompletionItem, 0, 12)
		for i := range 12 {
			items = append(items, view.CompletionItem{
				Label:     "long_completion_" + string(rune('a'+i)),
				Insert:    string(rune('a' + i)),
				Kind:      "method",
				Preselect: i == 11,
			})
		}
		ctl := &completionController{editor: e, items: items}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendModifiedAndFeed(m, 'x', tea.ModCtrl)
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "▌")
		assert.NotContains(t, out, "▲")
		assert.NotContains(t, out, "▼")
		assert.True(t, hasRightEdgeThumb(out))
	})

	t.Run("dismisses on editor key", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{Label: "Println", Insert: "Println", Kind: "function"},
			},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendModifiedAndFeed(m, 'x', tea.ModCtrl)
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "Println")

		m = sendSpecialText(m, tea.KeyPgDown, "pgdown")
		out = stripANSI(m.View().Content)

		assert.NotContains(t, out, "Println")
	})

	t.Run("dismisses on mouse click", func(t *testing.T) {
		e := editorWithText(t, "alpha\nbeta\n")
		e.SetMode(view.ModeInsert)
		e.Options().Mouse = true
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{Label: "Println", Insert: "Println", Kind: "function"},
			},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendModifiedAndFeed(m, 'x', tea.ModCtrl)
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "Println")

		m2, _ := m.Update(tea.MouseClickMsg{
			X: 6, Y: 1, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		out = stripANSI(m.View().Content)

		assert.NotContains(t, out, "Println")
	})

	t.Run("filters while typing", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{Label: "Println", Insert: "Println", Kind: "function"},
				{Label: "Scanln", Insert: "Scanln", Kind: "function"},
			},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendModifiedAndFeed(m, 'x', tea.ModCtrl)
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "Println")
		assert.Contains(t, out, "Scanln")

		m = sendKey(m, 'P')
		out = stripANSI(m.View().Content)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		assert.Equal(t, "P", doc.Text().String())
		assert.Contains(t, out, "Println")
		assert.NotContains(t, out, "Scanln")
	})

	t.Run("filters by prefix", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{Label: "Close", Insert: "Close", Kind: "method"},
				{Label: "Clear", Insert: "Clear", Kind: "method"},
				{Label: "Cancel", Insert: "Cancel", Kind: "method"},
				{Label: "Scanln", Insert: "Scanln", Kind: "function"},
			},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendModifiedAndFeed(m, 'x', tea.ModCtrl)
		m = sendKey(m, 'C')
		m = sendKey(m, 'l')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "Close")
		assert.Contains(t, out, "Clear")
		assert.NotContains(t, out, "Cancel")
		assert.NotContains(t, out, "Scanln")
	})

	t.Run("keeps selection while filtering", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{Label: "Close", Insert: "Close", Kind: "method"},
				{Label: "Clear", Insert: "Clear", Kind: "method"},
				{Label: "Clone", Insert: "Clone", Kind: "method"},
			},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendModifiedAndFeed(m, 'x', tea.ModCtrl)
		m = sendSpecial(m, tea.KeyDown)
		m = sendKey(m, 'C')
		_ = sendSpecial(m, tea.KeyEnter)

		assert.Equal(t, "Clear", ctl.item.Label)
	})

	t.Run("typing punctuation does not accept", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{Label: "Println", Insert: "Println", Kind: "function"},
			},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendModifiedAndFeed(m, 'x', tea.ModCtrl)
		_ = sendKey(m, '.')
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		assert.Equal(t, ".", doc.Text().String())
		assert.Empty(t, ctl.item.Label)
	})

	t.Run("does not render before anchor", func(t *testing.T) {
		e := editorWithText(t, "alpha")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{Label: "Println", Insert: "Println", Kind: "function"},
			},
		}
		e.SetLanguageServerController(ctl)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(3))
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendModifiedAndFeed(m, 'x', tea.ModCtrl)
		assert.Contains(t, stripANSI(m.View().Content), "Println")

		doc.SetSelectionFor(v.ID(), core.PointSelection(1))
		out := stripANSI(m.View().Content)

		assert.NotContains(t, out, "Println")
	})

	t.Run("drops stale response", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{Label: "Println", Insert: "Println", Kind: "function"},
			},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m2, cmd := m.Update(tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl})
		m = m2.(ui.Model)
		m = sendKey(m, 'p')
		m = feedCmds(m, cmd)
		out := stripANSI(m.View().Content)

		assert.NotContains(t, out, "Println")
	})

}

func (c *completionController) RestartLanguageServers(
	*view.Document, []string,
) ([]string, error) {
	return nil, nil
}

func (c *completionController) StopLanguageServers(
	*view.Document, []string,
) ([]string, error) {
	return nil, nil
}

func (c *completionController) ExecuteWorkspaceCommand(
	*view.Document, string, []string,
) error {
	return nil
}

func (c *completionController) WorkspaceCommands(*view.Document) []string {
	return nil
}

func (c *completionController) Completions(
	*view.Document, view.Id,
) ([]view.CompletionItem, error) {
	return c.items, nil
}

func (c *completionController) TriggerCompletions(
	*view.Document, view.Id,
) ([]view.CompletionItem, error) {
	return c.items, nil
}

func (c *completionController) ResolveCompletion(
	_ *view.Document, _ view.Id, item view.CompletionItem,
) (view.CompletionItem, error) {
	item.Docs = c.docs
	if item.Docs == "" {
		item.Docs = "resolved docs"
	}
	return item, nil
}

func (c *completionController) ApplyCompletion(
	_ *view.Document, _ view.Id, item view.CompletionItem,
) error {
	c.item = item
	text := item.Insert
	if text == "" {
		text = item.Label
	}
	for _, ch := range text {
		action.InsertChar(c.editor, ch)
	}
	return nil
}

func (c *completionController) Hover(*view.Document, view.Id) (string, error) {
	return "hover docs", nil
}

func (c *completionController) SignatureHelp(
	doc *view.Document, viewID view.Id,
) (view.SignatureHelp, error) {
	if c.signatureErr != nil {
		return view.SignatureHelp{}, c.signatureErr
	}
	if c.signatureEmpty {
		return view.SignatureHelp{}, nil
	}
	if doc != nil && len(c.signatureAfterComma.Signatures) > 0 {
		sel := doc.SelectionFor(viewID)
		pos := sel.Primary().Cursor(doc.Text())
		before, err := doc.Text().SliceString(0, pos)
		if err == nil && strings.Contains(before, ",") {
			return c.signatureAfterComma, nil
		}
	}
	if len(c.signature.Signatures) > 0 {
		return c.signature, nil
	}
	return view.SignatureHelp{
		Signatures: []view.SignatureInformation{
			{
				Label:       "Println(a ...any)",
				Docs:        "signature docs",
				ActiveStart: 8,
				ActiveEnd:   9,
			},
		},
	}, nil
}

func (c *completionController) TriggerSignatureHelp(
	doc *view.Document, viewID view.Id,
) (view.SignatureHelp, error) {
	if doc == nil {
		return c.SignatureHelp(nil, 0)
	}
	sel := doc.SelectionFor(viewID)
	pos := sel.Primary().Cursor(doc.Text())
	before, err := doc.Text().SliceString(0, pos)
	if err != nil || !strings.HasSuffix(before, "(") {
		return view.SignatureHelp{}, nil
	}
	return c.SignatureHelp(doc, viewID)
}

func (c *completionController) GotoDeclaration(
	*view.Document, view.Id,
) ([]view.Location, error) {
	return nil, nil
}

func (c *completionController) GotoDefinition(
	*view.Document, view.Id,
) ([]view.Location, error) {
	return nil, nil
}

func (c *completionController) GotoTypeDefinition(
	*view.Document, view.Id,
) ([]view.Location, error) {
	return nil, nil
}

func (c *completionController) GotoImplementation(
	*view.Document, view.Id,
) ([]view.Location, error) {
	return nil, nil
}

func (c *completionController) GotoReference(
	*view.Document, view.Id,
) ([]view.Location, error) {
	return nil, nil
}

func (c *completionController) RenameSymbolPrefill(
	*view.Document, view.Id,
) (string, error) {
	return "", nil
}

func (c *completionController) RenameSymbol(
	*view.Document, view.Id, string,
) error {
	return nil
}

func (c *completionController) CodeActions(
	*view.Document, view.Id,
) ([]view.CodeAction, error) {
	return nil, nil
}

func (c *completionController) ApplyCodeAction(
	*view.Document, view.Id, view.CodeAction,
) error {
	return nil
}

func (c *completionController) DocumentHighlights(
	*view.Document, view.Id,
) ([]view.DocumentHighlight, error) {
	return nil, nil
}

func (c *completionController) DocumentLinks(
	*view.Document,
) ([]view.DocumentLink, error) {
	return nil, nil
}

func (c *completionController) ResolveDocumentLink(
	_ *view.Document, link view.DocumentLink,
) (view.DocumentLink, error) {
	return link, nil
}

func (c *completionController) FormatDocument(
	*view.Document, view.Id,
) error {
	return nil
}

func (c *completionController) FormatSelection(
	*view.Document, view.Id,
) error {
	return nil
}

func (c *completionController) DocumentSymbols(
	*view.Document,
) ([]view.Symbol, error) {
	return nil, nil
}

func (c *completionController) WorkspaceSymbols(
	*view.Document, string,
) ([]view.Symbol, error) {
	return nil, nil
}

func hasRightEdgeThumb(out string) bool {
	for line := range strings.SplitSeq(out, "\n") {
		trimmed := strings.TrimRight(line, " ")
		if strings.HasSuffix(trimmed, "▌") {
			return true
		}
	}
	return false
}
