package ui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/defaults"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
)

type highlightRefreshController struct {
	locationController
}

func TestModelView(t *testing.T) {
	t.Run("returns empty before resize", func(t *testing.T) {
		e := editorWithText(t, "")
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)

		v := m.View()

		assert.Empty(t, v.Content)
	})
}

func TestInsertMode(t *testing.T) {
	t.Run("inserts space", func(t *testing.T) {
		e := editorWithText(t, "")
		e.SetMode(view.ModeInsert)
		km := command.NewKeymaps()
		m := resize(ui.New(e, km), 80, 24)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)

		m = sendKey(m, 'a')
		m = sendSpecial(m, tea.KeySpace)
		_ = sendKey(m, 'b')
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		assert.Equal(t, "a b", doc.Text().String())
	})
}

func TestEditorKeys(t *testing.T) {
	t.Run("accepts count in select mode", func(t *testing.T) {
		e := editorWithText(t, "abcdefgh")
		e.SetMode(view.ModeSelect)
		m := renderedModel(e)

		_ = sendKey(m, '3')

		assert.Equal(t, 3, e.Count())
	})
}

func TestMouseMiddlePaste(t *testing.T) {
	t.Run("pastes at clicked position", func(t *testing.T) {
		clipFile := filepath.Join(t.TempDir(), "clip.txt")
		testutil.WriteFakeClipboardTools(t, clipFile)
		assert.NoError(t, os.WriteFile(clipFile, []byte("XY"), 0o644))
		e := editorWithText(t, "abcd")
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseReleaseMsg{
			X: 9, Y: 0, Button: tea.MouseMiddle,
		})
		_ = m2

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abXYcd", doc.Text().String())
		assert.Equal(t, 2, testutil.CursorPos(t, e))
	})

	t.Run("disabled leaves document unchanged", func(t *testing.T) {
		clipFile := filepath.Join(t.TempDir(), "clip.txt")
		testutil.WriteFakeClipboardTools(t, clipFile)
		assert.NoError(t, os.WriteFile(clipFile, []byte("XY"), 0o644))
		e := editorWithText(t, "abcd")
		e.Options().MiddleClickPaste = false
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseReleaseMsg{
			X: 9, Y: 0, Button: tea.MouseMiddle,
		})
		_ = m2

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abcd", doc.Text().String())
	})

	t.Run("alt replaces selection", func(t *testing.T) {
		clipFile := filepath.Join(t.TempDir(), "clip.txt")
		testutil.WriteFakeClipboardTools(t, clipFile)
		assert.NoError(t, os.WriteFile(clipFile, []byte("XY"), 0o644))
		e := editorWithText(t, "abcd")
		testutil.SetSelection(t, e, []core.Range{core.NewRange(1, 3)}, 0)
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseReleaseMsg{
			X: 0, Y: 0, Button: tea.MouseMiddle, Mod: tea.ModAlt,
		})
		_ = m2

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "aXYd", doc.Text().String())
	})

	t.Run("outside content is ignored", func(t *testing.T) {
		clipFile := filepath.Join(t.TempDir(), "clip.txt")
		testutil.WriteFakeClipboardTools(t, clipFile)
		assert.NoError(t, os.WriteFile(clipFile, []byte("XY"), 0o644))
		e := editorWithText(t, "abcd")
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseReleaseMsg{
			X: 2, Y: 7, Button: tea.MouseMiddle,
		})
		_ = m2

		doc, _ := e.FocusedDocument()
		assert.Equal(t, "abcd", doc.Text().String())
	})
}

func TestMouseWheelScroll(t *testing.T) {
	// renderedModel gives a 40×8 window; row 7 is the status/command line,
	// which is outside all editor panes

	t.Run("wheel over pane scrolls that pane", func(t *testing.T) {
		e := editorWithText(t, "a\nb\nc\nd\ne\nf\ng\nh\ni\nj")
		e.SetViewHeight(6)
		m := renderedModel(e)

		v, ok := e.FocusedView()
		assert.True(t, ok)
		before := v.Offset().Anchor

		m2, _ := m.Update(tea.MouseWheelMsg{
			X: 5, Y: 0, Button: tea.MouseWheelDown,
		})
		_ = m2

		assert.Greater(t, v.Offset().Anchor, before)
	})

	t.Run("status bar wheel ignored", func(t *testing.T) {
		e := editorWithText(t, "a\nb\nc\nd\ne\nf\ng\nh\ni\nj")
		e.SetViewHeight(6)
		m := renderedModel(e)

		v, ok := e.FocusedView()
		assert.True(t, ok)
		before := v.Offset().Anchor

		// 40×8 window, no bufferline: ResizeTree gives the pane height=7,
		// so Y=6 is the pane's own status bar row (not content)
		m2, _ := m.Update(tea.MouseWheelMsg{
			X: 5, Y: 6, Button: tea.MouseWheelDown,
		})
		_ = m2

		assert.Equal(t, before, v.Offset().Anchor)
	})

	t.Run("wheel outside all panes does not scroll", func(t *testing.T) {
		e := editorWithText(t, "a\nb\nc\nd\ne\nf\ng\nh\ni\nj")
		e.SetViewHeight(6)
		m := renderedModel(e)

		v, ok := e.FocusedView()
		assert.True(t, ok)
		before := v.Offset().Anchor

		// Y=7 is the command line, outside any editor pane in this window
		m2, _ := m.Update(tea.MouseWheelMsg{
			X: 5, Y: 7, Button: tea.MouseWheelDown,
		})
		_ = m2

		assert.Equal(t, before, v.Offset().Anchor)
	})
}

func TestMouseClickPositioning(t *testing.T) {
	t.Run("positions cursor on content click", func(t *testing.T) {
		e := editorWithText(t, "abcdef")
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseClickMsg{
			X: 10, Y: 0, Button: tea.MouseLeft,
		})
		_ = m2

		assert.Equal(t, 3, testutil.CursorPos(t, e))
	})

	t.Run("clicks rendered character cell", func(t *testing.T) {
		e := editorWithText(t, "abcdef")
		m := renderedModel(e)
		x, y := renderedTextPoint(t, m, "abcdef", 3)

		m2, _ := m.Update(tea.MouseClickMsg{
			X: x, Y: y, Button: tea.MouseLeft,
		})
		_ = m2

		assert.Equal(t, 3, testutil.CursorPos(t, e))
	})

	t.Run("ignores click on status or command line", func(t *testing.T) {
		e := editorWithText(t, "abcdef")
		m := renderedModel(e)

		// place the cursor in the content area first
		m2, _ := m.Update(tea.MouseClickMsg{
			X: 10, Y: 0, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		assert.Equal(t, 3, testutil.CursorPos(t, e))

		// the bottom row of the 8-high window is the status/command line, never
		// editor content, so a click there must leave the cursor put
		m3, _ := m.Update(tea.MouseClickMsg{
			X: 1, Y: 7, Button: tea.MouseLeft,
		})
		_ = m3

		assert.Equal(t, 3, testutil.CursorPos(t, e))
	})

	t.Run("alt click adds secondary selection", func(t *testing.T) {
		e := editorWithText(t, "abcdef")
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseClickMsg{
			X: 9, Y: 0, Button: tea.MouseLeft, Mod: tea.ModAlt,
		})
		_ = m2

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, 2, len(sel.Ranges()))
		assert.Equal(t, 1, sel.PrimaryIndex())
	})

	t.Run("select mode click extends primary", func(t *testing.T) {
		e := editorWithText(t, "abcdef")
		e.SetMode(view.ModeSelect)
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseClickMsg{
			X: 10, Y: 0, Button: tea.MouseLeft,
		})
		_ = m2

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		sel := doc.SelectionFor(v.ID())
		assert.Equal(t, []core.Range{core.NewRange(0, 4)}, sel.Ranges())
	})

	t.Run("bufferline row is ignored", func(t *testing.T) {
		e := editorWithText(t, "abcdef")
		e.Options().BufferLine = view.BufferLineAlways
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseClickMsg{
			X: 10, Y: 0, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		assert.Equal(t, 0, testutil.CursorPos(t, e))

		m2, _ = m.Update(tea.MouseClickMsg{
			X: 10, Y: 1, Button: tea.MouseLeft,
		})
		_ = m2
		assert.Equal(t, 3, testutil.CursorPos(t, e))
	})

	t.Run("click below row map clamps", func(t *testing.T) {
		e := editorWithText(t, "a\nbc")
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseClickMsg{
			X: 10, Y: 5, Button: tea.MouseLeft,
		})
		_ = m2

		assert.Equal(t, 4, testutil.CursorPos(t, e))
	})

	t.Run("tab click uses expanded width", func(t *testing.T) {
		e := editorWithText(t, "\tab")
		m := renderedModel(e)

		m2, _ := m.Update(tea.MouseClickMsg{
			X: 11, Y: 0, Button: tea.MouseLeft,
		})
		_ = m2

		assert.Equal(t, 1, testutil.CursorPos(t, e))
	})
}

func TestMouseSeparatorDrag(t *testing.T) {
	t.Run("vertical separator resizes panes", func(t *testing.T) {
		e := editorWithText(t, "abcdef")
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
		_ = m.View()
		v, ok := e.FocusedView()
		assert.True(t, ok)
		_, ok = e.VSplit(v.DocID())
		assert.True(t, ok)
		_ = m.View()

		views := e.Tree().Views()
		before := views[0].View.Area().Width
		sepX := views[0].View.Area().X + before
		m2, _ := m.Update(tea.MouseClickMsg{
			X: sepX, Y: 0, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		m2, _ = m.Update(tea.MouseMotionMsg{
			X: sepX - 5, Y: 0, Button: tea.MouseLeft,
		})
		_ = m2

		after := e.Tree().Views()[0].View.Area().Width
		assert.Less(t, after, before)
	})

	t.Run("horizontal separator resizes panes", func(t *testing.T) {
		e := editorWithText(t, "abcdef")
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
		_ = m.View()
		v, ok := e.FocusedView()
		assert.True(t, ok)
		_, ok = e.HSplit(v.DocID())
		assert.True(t, ok)
		_ = m.View()

		views := e.Tree().Views()
		before := views[0].View.Area().Height
		sepY := views[0].View.Area().Y + before
		m2, _ := m.Update(tea.MouseClickMsg{
			X: 0, Y: sepY, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		m2, _ = m.Update(tea.MouseMotionMsg{
			X: 0, Y: sepY - 2, Button: tea.MouseLeft,
		})
		_ = m2

		after := e.Tree().Views()[0].View.Area().Height
		assert.Less(t, after, before)
	})
}

func TestMouseDragBounds(t *testing.T) {
	t.Run("negative row leaves selection unchanged", func(t *testing.T) {
		e := editorWithText(t, "abcdef")
		m := renderedModel(e)
		m2, _ := m.Update(tea.MouseClickMsg{
			X: 8, Y: 0, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)

		m2, _ = m.Update(tea.MouseMotionMsg{
			X: 8, Y: -1, Button: tea.MouseLeft,
		})
		_ = m2

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		assert.Equal(t,
			[]core.Range{core.PointRange(1)},
			doc.SelectionFor(v.ID()).Ranges(),
		)
	})

	t.Run("bufferline drag extends selection", func(t *testing.T) {
		e := editorWithText(t, "abcdef")
		e.Options().BufferLine = view.BufferLineAlways
		m := renderedModel(e)
		m2, _ := m.Update(tea.MouseClickMsg{
			X: 8, Y: 1, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)

		m2, _ = m.Update(tea.MouseMotionMsg{
			X: 13, Y: 1, Button: tea.MouseLeft,
		})
		_ = m2

		v, _ := e.FocusedView()
		doc, _ := e.FocusedDocument()
		assert.Equal(t,
			[]core.Range{core.NewRange(1, 6)},
			doc.SelectionFor(v.ID()).Ranges(),
		)
	})
}

func TestFreeScroll(t *testing.T) {
	t.Run("keypress keeps other scrolled view", func(t *testing.T) {
		e := editorWithText(t, strings.Repeat("0123456789abcdef\n", 20))
		e.ResizeTree(80, 24)
		v1, ok := e.FocusedView()
		assert.True(t, ok)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v2, ok := e.VSplit(doc.ID())
		assert.True(t, ok)
		assert.Equal(t, v2.ID(), e.Tree().Focus())
		anchor, err := doc.Text().LineToChar(10)
		assert.NoError(t, err)
		scrolled := view.Position{Anchor: anchor}
		v1.SetOffset(scrolled)
		v1.BeginFreeScroll(doc.Revision(), doc.SelectionFor(v1.ID()))
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		m = sendKey(m, 'l')
		_ = m.View().Content

		assert.True(t, v1.FreeScroll())
		assert.Equal(t, scrolled, v1.Offset())
		assert.False(t, v2.FreeScroll())
	})
}

func renderedModel(e *view.Editor) ui.Model {
	m := ui.New(e, command.NewKeymaps())
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 8})
	m = m2.(ui.Model)
	_ = m.View()
	return m
}

func editorWithText(t *testing.T, text string) *view.Editor {
	t.Helper()
	e := view.NewEditor("/tmp")
	e.Options().Theme = view.DefaultTheme
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	rope := doc.Text()
	cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
		core.TextChange(0, 0, text),
	})
	assert.NoError(t, err)
	tx := core.NewTransaction(rope).
		WithChanges(cs).
		WithSelection(core.PointSelection(0))
	assert.NoError(t, e.Apply(tx))
	return e
}

func TestFocusMessages(t *testing.T) {
	t.Run("focus message handled", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		m2, _ := m.Update(tea.FocusMsg{})
		m = m2.(ui.Model)

		assert.NotEmpty(t, m.View().Content)
	})

	t.Run("blur focus lost triggers autosave", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		e.Options().AutoSaveFocusLost = true
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		m2, _ := m.Update(tea.BlurMsg{})
		m = m2.(ui.Model)

		assert.NotEmpty(t, m.View().Content)
	})
}

func TestMouseDisabledEvents(t *testing.T) {
	t.Run("mouse release ignored disabled", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.Options().Mouse = false
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		m2, _ := m.Update(tea.MouseReleaseMsg{Button: tea.MouseLeft})
		m = m2.(ui.Model)

		assert.NotEmpty(t, m.View().Content)
	})

	t.Run("mouse wheel ignored when mouse disabled", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.Options().Mouse = false
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		m2, _ := m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
		m = m2.(ui.Model)

		assert.NotEmpty(t, m.View().Content)
	})

	t.Run("unknown message returns ignored", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		m2, _ := m.Update(struct{ unexpected string }{"msg"})
		m = m2.(ui.Model)

		assert.NotEmpty(t, m.View().Content)
	})
}

func TestSyncEditorMessages(t *testing.T) {
	t.Run("status shown from action", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		// No LSP sets "No configured language server"
		bindNormalTestAction(
			km, "goto_decl", m.GotoDeclarationAction(),
			[]command.KeyEvent{char('g')},
		)
		m = resize(m, 80, 24)

		m = sendKey(m, 'g')

		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "No configured language server")
	})
}

func TestDocumentHighlightRefresh(t *testing.T) {
	t.Run("refreshes after cursor move", func(t *testing.T) {
		e := editorWithText(t, "hello\n")
		ctl := &highlightRefreshController{
			locationController: locationController{
				highlights: []view.DocumentHighlight{{From: 1, To: 3}},
			},
		}
		e.SetLanguageServerController(ctl)
		km := command.NewKeymaps()
		m := resize(ui.New(e, km), 80, 24)
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)

		m2, cmd := m.Update(tea.KeyPressMsg{Code: 'l', Text: "l"})
		m = m2.(ui.Model)
		assert.NotNil(t, cmd)
		drainCmd(m, cmd)

		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		assert.Equal(t, ctl.highlights, doc.DocumentHighlights(v.ID()))
	})
}

func TestAutoSaveCmd(t *testing.T) {
	t.Run("autosave tick created on keypress", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.Options().AutoSaveAfterDelay = true
		km := command.NewKeymaps()
		m := resize(ui.New(e, km), 80, 24)

		m2, cmd := m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
		m = m2.(ui.Model)

		assert.NotEmpty(t, m.View().Content)
		assert.NotNil(t, cmd)
	})

	t.Run("autosave fires on gen match", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		e.Options().AutoSaveAfterDelay = true
		e.Options().AutoSaveDelayTimeout = 0
		km := command.NewKeymaps()
		m := resize(ui.New(e, km), 80, 24)

		// Execute the autosave command returned by the keypress
		m2, cmd := m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
		m = m2.(ui.Model)
		if cmd != nil {
			msg := cmd()
			if msg != nil {
				m2, _ = m.Update(msg)
				m = m2.(ui.Model)
			}
		}

		assert.NotEmpty(t, m.View().Content)
	})
}

func (c *highlightRefreshController) DocumentHighlights(
	doc *view.Document, id view.Id,
) ([]view.DocumentHighlight, error) {
	doc.SetDocumentHighlights(id, c.highlights)
	return c.highlights, nil
}

func renderedTextPoint(
	t *testing.T, m ui.Model, text string, off int,
) (int, int) {
	t.Helper()
	lines := strings.Split(stripANSI(m.View().Content), "\n")
	for y, line := range lines {
		if x := strings.Index(line, text); x >= 0 {
			return x + off, y
		}
	}
	t.Fatalf("rendered text %q not found", text)
	return 0, 0
}

func drainCmd(m ui.Model, cmd tea.Cmd) ui.Model {
	for cmd != nil {
		msg := cmd()
		if msg == nil {
			return m
		}
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, c := range batch {
				m = drainCmd(m, c)
			}
			return m
		}
		m2, next := m.Update(msg)
		m = m2.(ui.Model)
		cmd = next
	}
	return m
}
