package ui_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mattn/go-runewidth"
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
	refreshItems        []view.CompletionItem
	item                view.CompletionItem
	docs                string
	signature           view.SignatureHelp
	signatureAfterComma view.SignatureHelp
	signatureErr        error
	signatureEmpty      bool
	incomplete          bool
	refreshIncomplete   bool
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

	t.Run("accepts through keymap action", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{Label: "Println", Insert: "Println"},
			},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		km.Bind(
			ui.CompletionMode, ui.CompletionAcceptAction,
			[]command.KeyEvent{{
				Code: command.KeyCode{Char: 'j'}, Mods: command.ModCtrl,
			}},
		)
		m = resize(m, 80, 24)

		m = sendModifiedAndFeed(m, 'x', tea.ModCtrl)
		_ = sendModifiedAndFeed(m, 'j', tea.ModCtrl)

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

		assert.Contains(t, out, " Name")
		assert.NotContains(t, out, "detail")
	})

	t.Run("renders compact detail", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{
					Label:  "Printf",
					Insert: "Printf",
					Kind:   "function",
					Detail: "func(format string, args ...any)",
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

		assert.Contains(t, out, " Printf")
		assert.Contains(t, out, "func(format string, args ...any)")
	})

	t.Run("only selected row shows detail", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{
					Label:  "Printf",
					Insert: "Printf",
					Kind:   "function",
					Detail: "func(format string, args ...any)",
				},
				{
					Label:  "Println",
					Insert: "Println",
					Kind:   "function",
					Detail: "func(a ...any)",
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

		assert.Contains(t, out, "func(format string, args ...any)")
		assert.NotContains(t, out, "func(a ...any)")
	})

	t.Run("clips selected row detail preview", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{
					Label:  "Printf",
					Insert: "Printf",
					Kind:   "function",
					Detail: "func(format string, args ...any) " +
						"with a very long explanatory suffix",
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

		assert.Contains(t, out, "func(format string, args ...any) with...")
		assert.NotContains(t, out, "very long explanatory suffix")
	})

	t.Run("styles row segments separately", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{
					Label:       "Printf",
					LabelDetail: "(format)",
					Insert:      "Printf",
					Kind:        "function",
					Detail:      "func(format string)",
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
		m = sendKey(m, 'P')
		line := rawCompletionLine(t, m, "Printf")

		assert.Contains(t, stripANSI(line), " Printf(format)")
		assert.Contains(t, stripANSI(line), "func(format string)")
		assert.GreaterOrEqual(t, strings.Count(line, "\x1b["), 4)
	})

	t.Run("popup width stays stable across selection", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{
					Label:  "A",
					Insert: "A",
					Kind:   "function",
					Detail: "func(a int, b string, c bool)",
				},
				{
					Label:  "VeryLongCompletionName",
					Insert: "VeryLongCompletionName",
					Kind:   "function",
					Detail: "func()",
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
		before := completionPopupWidth(t, stripANSI(m.View().Content))
		m = sendSpecial(m, tea.KeyDown)
		after := completionPopupWidth(t, stripANSI(m.View().Content))

		assert.Equal(t, before, after)
	})

	t.Run("popup renders border rows", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{Label: "Printf", Insert: "Printf", Kind: "function"},
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

		assert.Contains(t, out, "╭")
		assert.Contains(t, out, "╰")
	})

	t.Run("renders ascii icons", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{Label: "Printf", Insert: "Printf", Kind: "function"},
			},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		m.SetCompletionOptions(ui.CompletionOptions{
			Icons: ui.CompletionIconsASCII,
		})
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendModifiedAndFeed(m, 'x', tea.ModCtrl)
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "fn Printf")
	})

	t.Run("renders no icons", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{Label: "Printf", Insert: "Printf", Kind: "function"},
			},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		m.SetCompletionOptions(ui.CompletionOptions{
			Icons: ui.CompletionIconsNone,
		})
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendModifiedAndFeed(m, 'x', tea.ModCtrl)
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "Printf")
		assert.NotContains(t, out, " Printf")
		assert.NotContains(t, out, "fn Printf")
	})

	t.Run("renders richer metadata", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{
					Label:            "Printf",
					LabelDetail:      "(format string)",
					LabelDescription: "fmt",
					Kind:             "function",
					Deprecated:       true,
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

		assert.Contains(t, out, " Printf(format string)")
		assert.Contains(t, out, "fmt")
		assert.Contains(t, out, "deprecated")
		assert.Contains(t, out, "")
	})

	t.Run("does not show completion docs", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			docs:   "completion docs should stay hidden",
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
		assert.NotContains(t, out, "completion docs should stay hidden")
	})

	t.Run("markdown docs hidden", func(t *testing.T) {
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
		assert.NotContains(t, out, "hello")
	})

	t.Run("does not render markdown docs", func(t *testing.T) {
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
		assert.NotContains(t, out, "func main() {}")
		assert.NotContains(t, out, "```")
	})

	t.Run("keeps narrow popup docs hidden", func(t *testing.T) {
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
		assert.NotContains(t, out, "docs for narrow screen")
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

	t.Run("page down moves selection", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		items := make([]view.CompletionItem, 0, 12)
		for i := range 12 {
			label := "item_" + string(rune('a'+i))
			items = append(items, view.CompletionItem{
				Label:  label,
				Insert: label,
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
		m = sendSpecialText(m, tea.KeyPgDown, "pgdown")
		_ = sendSpecial(m, tea.KeyEnter)

		assert.Equal(t, "item_j", ctl.item.Label)
	})

	t.Run("home and end move selection", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{Label: "Alpha", Insert: "Alpha"},
				{Label: "Beta", Insert: "Beta"},
				{Label: "Gamma", Insert: "Gamma"},
			},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendModifiedAndFeed(m, 'x', tea.ModCtrl)
		m = sendSpecialText(m, tea.KeyEnd, "end")
		m = sendSpecialText(m, tea.KeyHome, "home")
		_ = sendSpecial(m, tea.KeyEnter)

		assert.Equal(t, "Alpha", ctl.item.Label)
	})

	t.Run("ctrl navigation moves selection", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{Label: "Alpha", Insert: "Alpha"},
				{Label: "Beta", Insert: "Beta"},
				{Label: "Gamma", Insert: "Gamma"},
			},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendModifiedAndFeed(m, 'x', tea.ModCtrl)
		m = sendModifiedAndFeed(m, 'n', tea.ModCtrl)
		_ = sendSpecial(m, tea.KeyEnter)

		assert.Equal(t, "Beta", ctl.item.Label)
	})

	t.Run("ctrl previous wraps selection", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{Label: "Alpha", Insert: "Alpha"},
				{Label: "Beta", Insert: "Beta"},
				{Label: "Gamma", Insert: "Gamma"},
			},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendModifiedAndFeed(m, 'x', tea.ModCtrl)
		m = sendModifiedAndFeed(m, 'p', tea.ModCtrl)
		_ = sendSpecial(m, tea.KeyEnter)

		assert.Equal(t, "Gamma", ctl.item.Label)
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

		m = sendSpecialText(m, tea.KeyLeft, "left")
		out = stripANSI(m.View().Content)

		assert.NotContains(t, out, "Println")
	})

	t.Run("outside click dismisses and reaches editor", func(t *testing.T) {
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
			X: 10, Y: 0, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		out = stripANSI(m.View().Content)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)

		assert.NotContains(t, out, "Println")
		assert.Equal(t, 3, doc.SelectionFor(v.ID()).Primary().Cursor(doc.Text()))
	})

	t.Run("inside click selects without accepting", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		e.Options().Mouse = true
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{Label: "Alpha", Insert: "Alpha"},
				{Label: "Beta", Insert: "Beta"},
			},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendModifiedAndFeed(m, 'x', tea.ModCtrl)
		x, y := completionTextPoint(t, m, "Beta")
		m2, _ := m.Update(tea.MouseClickMsg{
			X: x, Y: y, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		assert.Equal(t, "", doc.Text().String())
		assert.Empty(t, ctl.item.Label)

		_ = sendSpecial(m, tea.KeyEnter)

		assert.Equal(t, "Beta", doc.Text().String())
		assert.Equal(t, "Beta", ctl.item.Label)
	})

	t.Run("wheel list scrolls without selecting", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		e.Options().Mouse = true
		items := make([]view.CompletionItem, 15)
		for i := range items {
			label := "item" + string(rune('A'+i))
			items[i] = view.CompletionItem{Label: label, Insert: label}
		}
		ctl := &completionController{editor: e, items: items}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendModifiedAndFeed(m, 'x', tea.ModCtrl)
		x, y := completionTextPoint(t, m, "itemF")
		m2, _ := m.Update(tea.MouseWheelMsg{
			X: x, Y: y, Button: tea.MouseWheelDown,
		})
		m = m2.(ui.Model)
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "itemM")

		_ = sendSpecial(m, tea.KeyEnter)
		assert.Equal(t, "itemA", ctl.item.Label)
	})

	t.Run("outside wheel dismisses and reaches editor", func(t *testing.T) {
		e := editorWithText(t, "a\nb\nc\nd\ne\nf\ng\nh\ni\nj")
		e.SetMode(view.ModeInsert)
		e.Options().Mouse = true
		e.SetViewHeight(6)
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
		m = resize(m, 40, 8)

		m = sendModifiedAndFeed(m, 'x', tea.ModCtrl)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		before := v.Offset().Anchor
		m2, _ := m.Update(tea.MouseWheelMsg{
			X: 30, Y: 0, Button: tea.MouseWheelDown,
		})
		m = m2.(ui.Model)
		out := stripANSI(m.View().Content)

		assert.NotContains(t, out, "Println")
		assert.Greater(t, v.Offset().Anchor, before)
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

	t.Run("filters by query", func(t *testing.T) {
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
		m = sendKey(m, 'o')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "Close")
		assert.NotContains(t, out, "Clear")
		assert.NotContains(t, out, "Cancel")
		assert.NotContains(t, out, "Scanln")
	})

	t.Run("fuzzy filters subsequence", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{Label: "Panic", Insert: "Panic", Kind: "function"},
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
		m = sendKey(m, 'P')
		m = sendKey(m, 'l')
		m = sendKey(m, 'n')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "Println")
		assert.NotContains(t, out, "Panic")
		assert.NotContains(t, out, "Scanln")
	})

	t.Run("requeries incomplete list", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{Label: "Alpha", Insert: "Alpha"},
			},
			refreshItems: []view.CompletionItem{
				{Label: "Println", Insert: "Println"},
			},
			incomplete: true,
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendModifiedAndFeed(m, 'x', tea.ModCtrl)
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "Alpha")

		m = sendKeyAndFeed(m, 'P')
		out = stripANSI(m.View().Content)

		assert.Contains(t, out, "Println")
		assert.NotContains(t, out, "Alpha")
	})

	t.Run("ranks prefix before fuzzy", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		ctl := &completionController{
			editor: e,
			items: []view.CompletionItem{
				{Label: "SomethingPrint", Insert: "SomethingPrint"},
				{Label: "Println", Insert: "Println"},
			},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 80, 24)

		m = sendModifiedAndFeed(m, 'x', tea.ModCtrl)
		m = sendKey(m, 'P')
		_ = sendSpecial(m, tea.KeyEnter)

		assert.Equal(t, "Println", ctl.item.Label)
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
	doc *view.Document, _ view.Id,
) (view.CompletionResult, error) {
	items := c.items
	incomplete := c.incomplete
	if doc != nil && len(c.refreshItems) > 0 &&
		strings.Contains(doc.Text().String(), "P") {
		items = c.refreshItems
		incomplete = c.refreshIncomplete
	}
	return view.CompletionResult{
		Items:      items,
		Incomplete: incomplete,
	}, nil
}

func (c *completionController) TriggerCompletions(
	doc *view.Document, viewID view.Id,
) (view.CompletionResult, error) {
	return c.Completions(doc, viewID)
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

func rawCompletionLine(t *testing.T, m ui.Model, text string) string {
	t.Helper()
	for line := range strings.SplitSeq(m.View().Content, "\n") {
		if strings.Contains(stripANSI(line), text) {
			return line
		}
	}
	t.Fatalf("completion text %q not found", text)
	return ""
}

func completionTextPoint(t *testing.T, m ui.Model, text string) (int, int) {
	t.Helper()
	lines := strings.Split(stripANSI(m.View().Content), "\n")
	for y, line := range lines {
		if x := strings.Index(line, text); x >= 0 {
			return x, y
		}
	}
	t.Fatalf("completion text %q not found", text)
	return 0, 0
}

func completionPopupWidth(t *testing.T, out string) int {
	t.Helper()
	for line := range strings.SplitSeq(out, "\n") {
		if strings.Contains(line, "╭") {
			return runewidth.StringWidth(strings.TrimRight(line, " "))
		}
	}
	t.Fatal("completion popup border not found")
	return 0
}
