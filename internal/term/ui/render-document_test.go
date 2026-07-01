package ui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

func TestBufferlineRender(t *testing.T) {
	t.Run("bufferline always visible", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.Options().BufferLine = view.BufferLineAlways
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)

		assert.NotEmpty(t, out)
	})

	t.Run("modified doc shows marker", func(t *testing.T) {
		e := editorWithText(t, "hello")
		e.Options().BufferLine = view.BufferLineAlways
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "[+]")
	})

	t.Run("multiple docs sorted by ID", func(t *testing.T) {
		root := t.TempDir()
		path1 := filepath.Join(root, "a.txt")
		path2 := filepath.Join(root, "b.txt")
		assert.NoError(t, os.WriteFile(path1, []byte("a\n"), 0o644))
		assert.NoError(t, os.WriteFile(path2, []byte("b\n"), 0o644))
		e := view.NewEditor(root)
		_, err := e.OpenFile(path1)
		assert.NoError(t, err)
		_, err = e.OpenFile(path2)
		assert.NoError(t, err)
		e.Options().BufferLine = view.BufferLineAlways
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "a.txt")
		assert.Contains(t, out, "b.txt")
	})
}

func TestPromptAccept(t *testing.T) {
	t.Run("enter executes command from prompt", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_ = km.Register("command_mode", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				return command.Result{Continuation: m.CmdModeAction()(e)}
			},
			Modes: []string{"NOR"},
			Keys: map[string][]command.KeyBinding{
				"*": {[][]command.KeyEvent{{char(':')}}},
			},
		})
		m = resize(m, 80, 24)
		m = sendKey(m, ':')
		m = sendKey(m, 'n')
		m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		m = m2.(ui.Model)

		_ = m.View().Content
	})
}

func TestRenderCrash(t *testing.T) {
	e := view.NewEditor("/tmp")
	km := command.NewKeymaps()

	// Wire just enough bindings to reproduce the scenario
	act := func(fn func(*view.Editor)) command.KeyAction {
		return func(e *view.Editor) command.Continuation {
			fn(e)
			return nil
		}
	}
	bindTestAction(bindTestActionArgs{
		km: km, mode: "NOR", name: "insert_mode",
		fn:   act(action.InsertMode),
		seqs: [][]command.KeyEvent{{char('i')}},
	})
	bindTestAction(bindTestActionArgs{
		km: km, mode: "INS", name: "insert_newline",
		fn:   act(action.InsertNewline),
		seqs: [][]command.KeyEvent{{special("ret")}},
	})
	bindTestAction(bindTestActionArgs{
		km: km, mode: "INS", name: "insert_newline",
		fn: act(action.InsertNewline),
		seqs: [][]command.KeyEvent{{
			char('j').WithMods(command.ModCtrl),
		}},
	})

	m := ui.New(e, km)

	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = m2.(ui.Model)

	// Enter insert mode
	m = sendKey(m, 'i')

	for i := range 50 {
		for _, ch := range "hello" {
			m = sendKey(m, ch)
		}
		m = sendSpecial(m, tea.KeyEnter)

		result := m.View().Content
		if result == "" {
			t.Errorf("iteration %d: empty render", i)
			return
		}
	}
}

func TestCursorShapeRender(t *testing.T) {
	t.Run("hides block cursor escape", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		e := view.NewEditor(t.TempDir())
		e.SetMode(view.ModeInsert)
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		cur := m.View().Cursor

		assert.Nil(t, cur)
	})

	t.Run("bar cursor uses steady terminal cursor", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		e := view.NewEditor(t.TempDir())
		e.Options().CursorShape.Insert = view.CursorKindBar
		e.SetMode(view.ModeInsert)
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		cur := m.View().Cursor

		assert.NotNil(t, cur)
		assert.Equal(t, tea.CursorBar, cur.Shape)
		assert.False(t, cur.Blink)
	})

	t.Run("underline cursor uses underline shape", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		e := view.NewEditor(t.TempDir())
		e.Options().CursorShape.Insert = view.CursorKindUnderline
		e.SetMode(view.ModeInsert)
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		cur := m.View().Cursor

		assert.NotNil(t, cur)
		assert.Equal(t, tea.CursorUnderline, cur.Shape)
	})
}

func TestCursorShapeUnfocused(t *testing.T) {
	t.Run("unfocused uses underline cursor", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		e := view.NewEditor(t.TempDir())
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
		m2, _ := m.Update(tea.BlurMsg{})
		m = m2.(ui.Model)

		cur := m.View().Cursor

		assert.NotNil(t, cur)
		assert.Equal(t, tea.CursorUnderline, cur.Shape)
	})
}

func TestInvalidThemeFallback(t *testing.T) {
	t.Run("falls back to default on bad theme", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		e := view.NewEditor(t.TempDir())
		e.Options().Theme = "nonexistent-theme-xyz"
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
		out := m.View().Content
		assert.NotEmpty(t, out)
	})
}

func TestThemeRender(t *testing.T) {
	t.Run("applies background style", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		path := filepath.Join(root, "note.txt")
		err := os.WriteFile(path, []byte("plain\n"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(root)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		e.Options().Theme = "mocha"
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := m.View().Content

		assert.Contains(t, out, "\x1b[48;2;30;30;46m")
		firstLine, _, _ := strings.Cut(out, "\n")
		assert.NotContains(t, firstLine, "\x1b[49m")
	})

	t.Run("applies text style", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		path := filepath.Join(root, "note.txt")
		err := os.WriteFile(path, []byte("plain\n"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(root)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		e.Options().Theme = "mocha"
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := m.View().Content

		assert.Contains(t, out, "\x1b[38;2;205;214;244m")
	})

	t.Run("applies diagnostic underline", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		path := filepath.Join(root, "note.txt")
		err := os.WriteFile(path, []byte("plain\n"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(root)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.ReplaceDiagnostics("test", []view.Diagnostic{{
			Range:    view.DiagnosticRange{From: 0, To: 5},
			Severity: view.DiagnosticSeverityError,
		}})
		e.Options().Theme = "mocha"
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := m.View().Content

		assert.Contains(t, out, "\x1b[4:3m")
	})

	t.Run("renders diagnostic gutter", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		path := filepath.Join(root, "note.txt")
		err := os.WriteFile(path, []byte("plain\n"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(root)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.ReplaceDiagnostics("test", []view.Diagnostic{{
			Range:    view.DiagnosticRange{From: 0, To: 5},
			Severity: view.DiagnosticSeverityError,
		}})
		e.Options().Theme = "mocha"
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := m.View().Content

		assert.Contains(t, stripANSI(out), "\u25cf")
		assert.Contains(t, out, "\x1b[38;2;243;139;168m")
	})

	t.Run("renders diagnostic popup at cursor", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		path := filepath.Join(root, "note.txt")
		err := os.WriteFile(path, []byte("plain\n"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(root)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.ReplaceDiagnostics("test", []view.Diagnostic{{
			Range:    view.DiagnosticRange{From: 0, To: 5},
			Severity: view.DiagnosticSeverityError,
			Message:  "unused value",
		}})
		e.Options().Theme = "mocha"
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := m.View().Content

		assert.Contains(t, out, "\x1b[38;2;243;139;168m")
		assert.Contains(t, stripANSI(out), "unused value")
		assert.Equal(t, 1, strings.Count(stripANSI(out), "unused value"))
	})

	t.Run("popup warning severity", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		path := filepath.Join(root, "note.txt")
		err := os.WriteFile(path, []byte("plain\n"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(root)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.ReplaceDiagnostics("test", []view.Diagnostic{{
			Range:    view.DiagnosticRange{From: 0, To: 5},
			Severity: view.DiagnosticSeverityWarning,
			Source:   "lint",
			Message:  "warning msg",
		}})
		e.Options().Theme = "mocha"
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "lint: warning msg")
	})

	t.Run("popup info severity", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		path := filepath.Join(root, "note.txt")
		err := os.WriteFile(path, []byte("plain\n"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(root)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.ReplaceDiagnostics("test", []view.Diagnostic{{
			Range:    view.DiagnosticRange{From: 0, To: 5},
			Severity: view.DiagnosticSeverityInfo,
			Message:  "info msg",
		}})
		e.Options().Theme = "mocha"
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "info msg")
	})

	t.Run("popup hint severity", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		path := filepath.Join(root, "note.txt")
		err := os.WriteFile(path, []byte("plain\n"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(root)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.ReplaceDiagnostics("test", []view.Diagnostic{{
			Range:    view.DiagnosticRange{From: 0, To: 5},
			Severity: view.DiagnosticSeverityHint,
			Message:  "hint msg",
		}})
		e.Options().Theme = "mocha"
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "hint msg")
	})

	t.Run("hides diagnostic popup off range", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		path := filepath.Join(root, "note.txt")
		err := os.WriteFile(path, []byte("plain\nnext\n"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(root)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		doc.ReplaceDiagnostics("test", []view.Diagnostic{{
			Range:    view.DiagnosticRange{From: 0, To: 5},
			Severity: view.DiagnosticSeverityWarning,
			Message:  "cursor warning",
		}})
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetSelectionFor(v.ID(), core.PointSelection(6))
		e.Options().Theme = "mocha"
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)

		assert.NotContains(t, out, "cursor warning")
	})

	t.Run("renders rulers on short lines", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		path := filepath.Join(root, "note.txt")
		err := os.WriteFile(path, []byte("a\nb\n"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(root)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		e.Options().Theme = "mocha"
		e.Options().Rulers = []int{10}
		m := resize(ui.New(e, command.NewKeymaps()), 40, 10)

		out := m.View().Content

		marker := "\x1b[48;2;49;50;68m"
		assert.Contains(t, out, marker)
		for line := range strings.SplitSeq(out, "\n") {
			pfx, _, ok := strings.Cut(line, marker)
			if !ok {
				continue
			}
			assert.Equal(t, 16, ansi.StringWidth(pfx))
		}
	})

	t.Run("renders rulers cleanly", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		path := filepath.Join(root, "note.txt")
		err := os.WriteFile(path, []byte("界\n"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(root)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		e.Options().Rulers = []int{2}
		m := resize(ui.New(e, command.NewKeymaps()), 40, 10)

		out := stripANSI(m.View().Content)

		assertRenderedWidth(t, out, 40)
	})

	t.Run("keeps wide popup within width", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		_ = km.Register("wide_popup_item", command.Command{
			DocString: "Wide 項目",
			Run: func(*view.Editor, *command.Args) command.Result {
				return command.Result{}
			},
			Modes: []string{"NOR"},
			Keys: map[string][]command.KeyBinding{"*": {[][]command.KeyEvent{{
				char(' '), char('界'),
			}}}},
		})
		m := resize(ui.New(e, km), 40, 10)

		out := stripANSI(sendKey(m, ' ').View().Content)

		assert.Contains(t, out, "界")
		assertRenderedWidth(t, out, 40)
	})

	t.Run("applies line number styles", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		path := filepath.Join(root, "main.go")
		err := os.WriteFile(path, []byte("package p\n\nfunc f() {}\n"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(root)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		e.Options().Theme = "mocha"
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := m.View().Content

		assert.Contains(t, out, "\x1b[38;2;180;190;254m")
		assert.Contains(t, out, "\x1b[38;2;69;71;90m")
	})

	t.Run("highlights search matches", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		path := filepath.Join(root, "note.txt")
		err := os.WriteFile(
			path, []byte("hello world\nhello again\n"), 0o644,
		)
		assert.NoError(t, err)
		e := view.NewEditor(root)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		e.Registers().Set('/', "hello")
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		_ = m.View().Content
	})

	t.Run("applies selection and cursor styles", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		path := filepath.Join(root, "note.txt")
		err := os.WriteFile(path, []byte("abcdef\n"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(root)
		v, err := e.OpenFile(path)
		assert.NoError(t, err)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		sel, err := core.NewSelection([]core.Range{
			core.NewRange(0, 2),
			core.NewRange(3, 5),
		}, 1)
		assert.NoError(t, err)
		doc.SetSelectionFor(v.ID(), sel)
		e.Options().Theme = "mocha"
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := m.View().Content

		assert.Contains(t, out, "\x1b[48;2;69;71;90m")
		assert.Contains(t, out, "\x1b[48;2;245;224;220m")
	})

	t.Run("non-block cursor still painted", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		path := filepath.Join(root, "note.txt")
		err := os.WriteFile(path, []byte("abcdef\n"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(root)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		e.Options().Theme = "mocha"
		e.Options().CursorShape.Select = view.CursorKindUnderline
		e.SetMode(view.ModeSelect)
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := m.View().Content

		assert.Contains(t, out, "\x1b[48;2;69;71;90m")
	})

	t.Run("insert cursor stays unpainted", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		path := filepath.Join(root, "note.txt")
		err := os.WriteFile(path, []byte("abcdef\n"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(root)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		e.Options().Theme = "mocha"
		e.Options().CursorShape.Insert = view.CursorKindBar
		e.SetMode(view.ModeInsert)
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := m.View().Content

		assert.NotContains(t, out, "\x1b[48;2;69;71;90m")
	})
}

func TestBaseStyleAtCases(t *testing.T) {
	t.Run("selected syntax-highlighted text", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		path := filepath.Join(root, "main.go")
		err := os.WriteFile(path, []byte("package main\n"), 0o644)
		assert.NoError(t, err)
		e := view.NewEditor(root)
		v, err := e.OpenFile(path)
		assert.NoError(t, err)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		sel, err := core.NewSelection([]core.Range{core.NewRange(0, 7)}, 0)
		assert.NoError(t, err)
		doc.SetSelectionFor(v.ID(), sel)
		e.Options().Theme = "mocha"
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
		out := m.View().Content
		assert.NotEmpty(t, out)
	})

	t.Run("selected tab with indent guides", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		e := view.NewEditor(t.TempDir())
		e.Options().IndentGuides = view.IndentGuides{Render: true}
		e.Options().Theme = "mocha"
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		rope := doc.Text()
		cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
			core.TextChange(0, 0, "\thello\n"),
		})
		assert.NoError(t, err)
		assert.NoError(t, e.Apply(core.NewTransaction(rope).WithChanges(cs)))
		v, ok := e.FocusedView()
		assert.True(t, ok)
		sel, err := core.NewSelection([]core.Range{core.NewRange(0, 1)}, 0)
		assert.NoError(t, err)
		doc.SetSelectionFor(v.ID(), sel)
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
		out := m.View().Content
		assert.NotEmpty(t, out)
	})

	t.Run("selected space with whitespace render", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		e := view.NewEditor(t.TempDir())
		wsr := view.WhitespaceRenderAll
		e.Options().Whitespace.Render = view.WhitespaceRender{Default: &wsr}
		e.Options().Theme = "mocha"
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		rope := doc.Text()
		cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
			core.TextChange(0, 0, "  hello\n"),
		})
		assert.NoError(t, err)
		assert.NoError(t, e.Apply(core.NewTransaction(rope).WithChanges(cs)))
		v, ok := e.FocusedView()
		assert.True(t, ok)
		sel, err := core.NewSelection([]core.Range{core.NewRange(0, 2)}, 0)
		assert.NoError(t, err)
		doc.SetSelectionFor(v.ID(), sel)
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
		out := m.View().Content
		assert.NotEmpty(t, out)
	})
}

func TestSplitViewRender(t *testing.T) {
	t.Run("vsplit and hsplit render separators", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		v, ok := e.FocusedView()
		assert.True(t, ok)
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
		e.VSplit(v.DocID())
		e.HSplit(v.DocID())
		out := stripANSI(m.View().Content)
		assert.NotEmpty(t, out)
	})
}

func TestIndentGuideRender(t *testing.T) {
	t.Run("renders indent guides", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.Options().IndentGuides = view.IndentGuides{Render: true}
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		rope := doc.Text()
		cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
			core.TextChange(0, 0, "\thello\n\tworld\n"),
		})
		assert.NoError(t, err)
		assert.NoError(t, e.Apply(core.NewTransaction(rope).WithChanges(cs)))
		out := stripANSI(m.View().Content)
		assert.NotEmpty(t, out)
	})
}

func TestWhitespaceRender(t *testing.T) {
	t.Run("renders visible whitespace chars", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		wsr := view.WhitespaceRenderAll
		e.Options().Whitespace.Render = view.WhitespaceRender{Default: &wsr}
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		rope := doc.Text()
		cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
			core.TextChange(0, 0, "hello world end\t!\n"),
		})
		assert.NoError(t, err)
		assert.NoError(t, e.Apply(core.NewTransaction(rope).WithChanges(cs)))
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "hello")
	})
}

func TestMouseDragNoop(t *testing.T) {
	t.Run("motion without prior click is noop", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.Options().Mouse = true
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
		m2, _ := m.Update(tea.MouseMotionMsg{
			X: 10, Y: 5, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		out := stripANSI(m.View().Content)
		assert.NotEmpty(t, out)
	})

	t.Run("drag after click extends selection", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.Options().Mouse = true
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		rope := doc.Text()
		cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
			core.TextChange(0, 0, "hello world\n"),
		})
		assert.NoError(t, err)
		assert.NoError(t, e.Apply(core.NewTransaction(rope).WithChanges(cs)))
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
		_ = m.View() // populate render cache so click can resolve position
		m2, _ := m.Update(tea.MouseClickMsg{X: 5, Y: 0, Button: tea.MouseLeft})
		m = m2.(ui.Model)
		m2, _ = m.Update(tea.MouseMotionMsg{X: 8, Y: 0, Button: tea.MouseLeft})
		m = m2.(ui.Model)
		m2, _ = m.Update(tea.MouseReleaseMsg{Button: tea.MouseLeft})
		m = m2.(ui.Model)
		out := stripANSI(m.View().Content)
		assert.NotEmpty(t, out)
	})

	t.Run("release without click is noop", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.Options().Mouse = true
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
		m2, _ := m.Update(tea.MouseReleaseMsg{Button: tea.MouseLeft})
		m = m2.(ui.Model)
		out := stripANSI(m.View().Content)
		assert.NotEmpty(t, out)
	})
}

func TestCursorcolumnRender(t *testing.T) {
	t.Run("cursorcolumn highlights column", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.Options().Cursorcolumn = true
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		rope := doc.Text()
		cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
			core.TextChange(0, 0, "hello world\n"),
		})
		assert.NoError(t, err)
		assert.NoError(t, e.Apply(core.NewTransaction(rope).WithChanges(cs)))
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "hello")
	})

	t.Run("multi-cursor secondary column", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.Options().Cursorcolumn = true
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		rope := doc.Text()
		cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
			core.TextChange(0, 0, "hello world\n"),
		})
		assert.NoError(t, err)
		assert.NoError(t, e.Apply(core.NewTransaction(rope).WithChanges(cs)))
		v, ok := e.FocusedView()
		assert.True(t, ok)
		sel, err := core.NewSelection([]core.Range{
			core.PointRange(0),
			core.PointRange(3),
		}, 0)
		assert.NoError(t, err)
		doc.SetSelectionFor(v.ID(), sel)
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
		out2 := stripANSI(m.View().Content)
		assert.Contains(t, out2, "hello")
	})
}

func TestRelativeLineNumberRender(t *testing.T) {
	t.Run("renders distance above cursor", func(t *testing.T) {
		e := editorWithText(t, "aa\nbb\ncc\n")
		e.Options().LineNumber = view.LineNumberRelative
		setSelection(t, e, []core.Range{core.PointRange(6)}, 0)
		m := resize(ui.New(e, command.NewKeymaps()), 80, 10)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "    2  aa")
		assert.Contains(t, out, "    1  bb")
		assert.Contains(t, out, "    3  cc")
	})
}

func TestSoftWrapRender(t *testing.T) {
	t.Run("renders continuation rows", func(t *testing.T) {
		e := editorWithText(t, "alpha bravo charlie delta echo\n")
		enabled := true
		maxWrap := 4
		indicator := ">> "
		e.Options().SoftWrap.Enable = &enabled
		e.Options().SoftWrap.MaxWrap = &maxWrap
		e.Options().SoftWrap.WrapIndicator = &indicator
		m := resize(ui.New(e, command.NewKeymaps()), 18, 8)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "alpha")
		assert.Contains(t, out, ">>")
	})
}

func TestHorizontalScrollRender(t *testing.T) {
	t.Run("scan prefix on scrolled long line", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		rope := doc.Text()
		const longLine = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" +
			"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" +
			"xxxxxxxxxxxxxxx\n"
		cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
			core.TextChange(0, 0, longLine),
		})
		assert.NoError(t, err)
		assert.NoError(t, e.Apply(core.NewTransaction(rope).WithChanges(cs)))
		v, ok := e.FocusedView()
		assert.True(t, ok)
		action.MoveLineEnd(e)
		v.SetOffset(view.Position{HorizontalOffset: 50})
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
		out := stripANSI(m.View().Content)
		assert.NotEmpty(t, out)
	})

	t.Run("cursorcolumn with scrolled long line", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.Options().Cursorcolumn = true
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		rope := doc.Text()
		const longLine = "abcdefghijabcdefghijabcdefghijabcdefghij\n"
		cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
			core.TextChange(0, 0, longLine),
		})
		assert.NoError(t, err)
		assert.NoError(t, e.Apply(core.NewTransaction(rope).WithChanges(cs)))
		v, ok := e.FocusedView()
		assert.True(t, ok)
		setSelection(t, e,
			[]core.Range{core.PointRange(24), core.PointRange(28)},
			0,
		)
		v.SetOffset(view.Position{HorizontalOffset: 10})
		m := resize(ui.New(e, command.NewKeymaps()), 20, 8)

		out := stripANSI(m.View().Content)

		assert.NotEmpty(t, out)
	})
}

func TestDocumentHighlightAndLinkRender(t *testing.T) {
	t.Run("renders with document highlights set", func(t *testing.T) {
		e := editorWithText(t, "hello world\n")
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetDocumentHighlights(v.ID(), []view.DocumentHighlight{
			{From: 0, To: 5},
			{From: 6, To: 11},
		})
		doc.SetDocumentLinks([]view.DocumentLink{
			{From: 0, To: 5, Target: "/some/path"},
		})
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)

		assert.NotEmpty(t, out)
	})
}

func TestTextAnnotationRender(t *testing.T) {
	t.Run("renders inlay hints and color swatches", func(t *testing.T) {
		e := editorWithText(t, "hello\n")
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetInlayHints(v.ID(), []view.InlayHint{
			{Pos: 5, Label: ": string", Kind: "type"},
		})
		doc.SetDocumentColors([]view.DocumentColor{
			{From: 0, To: 5, Red: 255},
		})
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "\u25a0hello: string")
	})

	t.Run("renders parameter and unknown hint kinds", func(t *testing.T) {
		e := editorWithText(t, "hello\n")
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc.SetInlayHints(v.ID(), []view.InlayHint{
			{Pos: 5, Label: ": T", Kind: "parameter"},
			{Pos: 5, Label: ": U", Kind: "other"},
		})
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)

		assert.Contains(t, out, ": T")
	})
}

func TestSearchInvalidRegex(t *testing.T) {
	t.Run("invalid search pattern does not panic", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		e := view.NewEditor(t.TempDir())
		e.Registers().Write('/', []string{"["})
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := m.View().Content

		assert.NotEmpty(t, out)
	})
}
