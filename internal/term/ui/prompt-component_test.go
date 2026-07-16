package ui_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/builtin"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

func TestPromptCompletion(t *testing.T) {
	t.Run("completions update as input changes", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_ = km.Register("command_mode", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				return command.Result{Continuation: m.CmdModeAction()(e)}
			},
			Modes: []string{"NOR"},
			Keys: map[string][]command.KeyBinding{"*": {[][]command.KeyEvent{
				{char(':')},
			}}},
		})
		_ = km.Register("alpha", testCommand("alpha"))
		_ = km.Register("beta", testCommand("beta"))
		m = resize(m, 60, 12)

		m = sendKey(m, ':')
		m = sendKey(m, 'a')
		withA := m.View().Content
		m = sendKey(m, 'l')
		withAl := m.View().Content

		assert.Contains(t, withA, "alpha")
		assert.NotContains(t, withA, "beta")
		assert.Contains(t, withAl, "alpha")
		assert.NotContains(t, withAl, "beta")
		assert.Contains(t, stripANSI(withAl), "╭")
		for line := range strings.SplitSeq(stripANSI(withAl), "\n") {
			if strings.Contains(line, "alpha") ||
				strings.Contains(line, "╭") ||
				strings.Contains(line, "╰") {
				assert.Equal(t, 60, ansi.StringWidth(line))
			}
		}
	})

	t.Run("tab inserts selected command completion", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_ = km.Register("command_mode", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				return command.Result{Continuation: m.CmdModeAction()(e)}
			},
			Modes: []string{"NOR"},
			Keys: map[string][]command.KeyBinding{"*": {[][]command.KeyEvent{
				{char(':')},
			}}},
		})
		_ = km.Register("alpha", testCommand("alpha"))
		m = resize(m, 60, 12)

		m = sendKey(m, ':')
		m = sendKey(m, 'a')
		m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		m = m2.(ui.Model)

		assert.Contains(t, stripANSI(m.View().Content), ": alpha")
	})

	t.Run("tab advances the caret", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_ = km.Register("command_mode", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				return command.Result{Continuation: m.CmdModeAction()(e)}
			},
			Modes: []string{"NOR"},
			Keys: map[string][]command.KeyBinding{"*": {[][]command.KeyEvent{
				{char(':')},
			}}},
		})
		_ = km.Register("alpha", testCommand("alpha"))
		m = resize(m, 60, 12)

		m = sendKey(m, ':')
		m = sendKey(m, 'a')
		m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		m = m2.(ui.Model)

		// typing after tab lands at the end, not mid-word where the caret was
		m = sendKey(m, 'X')
		assert.Contains(t, stripANSI(m.View().Content), ": alphaX")
	})

	t.Run("ignores command args without completer", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_ = km.Register("command_mode", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				return command.Result{Continuation: m.CmdModeAction()(e)}
			},
			Modes: []string{"NOR"},
			Keys: map[string][]command.KeyBinding{"*": {[][]command.KeyEvent{
				{char(':')},
			}}},
		})
		_ = km.Register("alpha", testCommand("alpha"))
		m = resize(m, 60, 12)

		m = sendKey(m, ':')
		for _, ch := range "alpha " {
			m = sendKey(m, ch)
		}

		assert.Contains(t, stripANSI(m.View().Content), ": alpha ")
	})

	t.Run("renders completion box background", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		e := view.NewEditor(t.TempDir())
		e.Options().Theme = "mocha"
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_ = km.Register("command_mode", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				return command.Result{Continuation: m.CmdModeAction()(e)}
			},
			Modes: []string{"NOR"},
			Keys: map[string][]command.KeyBinding{"*": {[][]command.KeyEvent{
				{char(':')},
			}}},
		})
		_ = km.Register("alpha", testCommand("alpha"))
		m = resize(m, 60, 12)

		m = sendKey(m, ':')
		m = sendKey(m, 'a')

		content := m.View().Content
		assert.Regexp(t,
			regexp.MustCompile(`\x1b\[[0-9;]*48;2;49;50;68[0-9;]*m╭`),
			content,
		)
		// ANSI style carries across lines when unchanged, so the menu bg escape
		// before "alpha" must be found over the whole stream, not re-split per
		// line
		idx := strings.Index(content, "alpha")
		if !assert.GreaterOrEqual(t, idx, 0) {
			return
		}
		before := content[:idx]
		bgIdx := strings.LastIndex(before, "48;2;49;50;68m")
		if assert.GreaterOrEqual(t, bgIdx, 0) {
			after := before[bgIdx+len("48;2;49;50;68m"):]
			for _, m := range regexp.MustCompile(`48;2;\d+;\d+;\d+m`).
				FindAllString(after, -1) {
				assert.Equal(t, "48;2;49;50;68m", m)
			}
		}
	})

	t.Run("completes file args", func(t *testing.T) {
		root := t.TempDir()
		err := os.WriteFile(
			filepath.Join(root, "main.go"), []byte("package main\n"), 0o644,
		)
		assert.NoError(t, err)
		e := view.NewEditor(root)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err = builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 60, 12)

		m = sendKey(m, ':')
		for _, ch := range "open m" {
			m = sendKey(m, ch)
		}

		assert.Contains(t, m.View().Content, "main.go")
	})
}

func TestPromptCmdAccept(t *testing.T) {
	t.Run("enter submits command prompt", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 60, 12)

		m = sendKey(m, ':')
		for _, ch := range "bad_command" {
			m = sendKey(m, ch)
		}
		m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		m = m2.(ui.Model)

		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "error")
	})
}

func TestPromptKeyEditing(t *testing.T) {
	t.Run("escape closes prompt", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 60, 12)

		m = sendKey(m, ':')
		m = sendSpecial(m, tea.KeyEscape)

		assert.NotRegexp(t,
			regexp.MustCompile(`(?m)^:`), stripANSI(m.View().Content))
	})

	t.Run("backspace removes input", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 60, 12)

		m = sendKey(m, ':')
		m = sendKey(m, 'a')
		m = sendKey(m, 'b')
		m = sendSpecial(m, tea.KeyBackspace)

		out := stripANSI(m.View().Content)
		assert.Contains(t, out, ": a")
		assert.NotContains(t, out, ": ab")
	})

	t.Run("ctrl h removes input", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 60, 12)

		m = sendKey(m, ':')
		m = sendKey(m, 'x')
		m2, _ := m.Update(tea.KeyPressMsg{
			Code: 'h',
			Mod:  tea.ModCtrl,
		})
		m = m2.(ui.Model)

		out := stripANSI(m.View().Content)
		assert.Contains(t, out, ":")
		assert.NotContains(t, out, ": x")
	})

	t.Run("shift tab wraps completions", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 60, 12)

		m = sendKey(m, ':')
		m = sendKey(m, 'o')
		m2, _ := m.Update(tea.KeyPressMsg{
			Code: tea.KeyTab,
			Mod:  tea.ModShift,
		})
		m = m2.(ui.Model)

		assert.Contains(t, stripANSI(m.View().Content), ":")
	})

	t.Run("long input scrolls instead of wrapping", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 20, 12)

		m = sendKey(m, ':')
		for _, ch := range "abcdefghijklmnopqrstuvwxyz" {
			m = sendKey(m, ch)
		}

		lines := strings.Split(stripANSI(m.View().Content), "\n")
		promptLine := lines[len(lines)-1]
		assert.LessOrEqual(t, runewidth.StringWidth(promptLine), 20)
		assert.Contains(t, promptLine, "…")
		assert.NotContains(t, promptLine, "abc")
		assert.Contains(t, promptLine, "xyz")
	})
}

func TestPromptEditing(t *testing.T) {
	t.Run("inserts at the caret", func(t *testing.T) {
		m, _ := cmdPrompt(t)
		m = typeString(m, "abc")
		m = sendSpecial(m, tea.KeyLeft)
		m = sendSpecial(m, tea.KeyLeft)
		m = sendKey(m, 'X')
		assert.Equal(t, ": aXbc", promptText(m))
	})

	t.Run("left and right move the caret", func(t *testing.T) {
		m, _ := cmdPrompt(t)
		m = typeString(m, "ab")
		m = sendSpecial(m, tea.KeyLeft)
		m = sendSpecial(m, tea.KeyRight)
		m = sendKey(m, 'X')
		assert.Equal(t, ": abX", promptText(m))
	})

	t.Run("home and end jump to the ends", func(t *testing.T) {
		m, _ := cmdPrompt(t)
		m = typeString(m, "abc")
		m = sendSpecial(m, tea.KeyHome)
		m = sendKey(m, 'X')
		m = sendSpecial(m, tea.KeyEnd)
		m = sendKey(m, 'Y')
		assert.Equal(t, ": XabcY", promptText(m))
	})

	t.Run("ctrl a and ctrl e jump to the ends", func(t *testing.T) {
		m, _ := cmdPrompt(t)
		m = typeString(m, "abc")
		m = sendModified(m, 'a', tea.ModCtrl)
		m = sendKey(m, 'X')
		m = sendModified(m, 'e', tea.ModCtrl)
		m = sendKey(m, 'Y')
		assert.Equal(t, ": XabcY", promptText(m))
	})

	t.Run("delete removes the char after caret", func(t *testing.T) {
		m, _ := cmdPrompt(t)
		m = typeString(m, "abc")
		m = sendSpecial(m, tea.KeyLeft)
		m = sendSpecial(m, tea.KeyLeft)
		m = sendSpecial(m, tea.KeyDelete)
		assert.Equal(t, ": ac", promptText(m))
	})

	t.Run("ctrl d removes the char after caret", func(t *testing.T) {
		m, _ := cmdPrompt(t)
		m = typeString(m, "abc")
		m = sendSpecial(m, tea.KeyHome)
		m = sendModified(m, 'd', tea.ModCtrl)
		assert.Equal(t, ": bc", promptText(m))
	})

	t.Run("ctrl w deletes the word before caret", func(t *testing.T) {
		m, _ := cmdPrompt(t)
		m = typeString(m, "foo bar")
		m = sendModified(m, 'w', tea.ModCtrl)
		assert.Equal(t, ": foo", promptText(m))
	})

	t.Run("alt backspace deletes prior word", func(t *testing.T) {
		m, _ := cmdPrompt(t)
		m = typeString(m, "foo bar")
		m = sendModified(m, tea.KeyBackspace, tea.ModAlt)
		assert.Equal(t, ": foo", promptText(m))
	})

	t.Run("ctrl delete deletes the word after caret", func(t *testing.T) {
		m, _ := cmdPrompt(t)
		m = typeString(m, "foo bar")
		m = sendSpecial(m, tea.KeyHome)
		m = sendModified(m, tea.KeyDelete, tea.ModCtrl)
		assert.Equal(t, ":  bar", promptText(m))
	})

	t.Run("ctrl left moves by word", func(t *testing.T) {
		m, _ := cmdPrompt(t)
		m = typeString(m, "foo bar")
		m = sendModified(m, tea.KeyLeft, tea.ModCtrl)
		m = sendKey(m, 'X')
		assert.Equal(t, ": foo Xbar", promptText(m))
	})

	t.Run("ctrl k kills to end of line", func(t *testing.T) {
		m, _ := cmdPrompt(t)
		m = typeString(m, "abc")
		m = sendSpecial(m, tea.KeyLeft)
		m = sendSpecial(m, tea.KeyLeft)
		m = sendModified(m, 'k', tea.ModCtrl)
		assert.Equal(t, ": a", promptText(m))
	})

	t.Run("ctrl u kills to start of line", func(t *testing.T) {
		m, _ := cmdPrompt(t)
		m = typeString(m, "abc")
		m = sendSpecial(m, tea.KeyLeft)
		m = sendModified(m, 'u', tea.ModCtrl)
		assert.Equal(t, ": c", promptText(m))
	})
}

func TestPromptCursor(t *testing.T) {
	t.Run("tracks the caret with configured shape", func(t *testing.T) {
		m, _ := cmdPrompt(t)
		m = typeString(m, "ab")

		cur := m.View().Cursor
		assert.NotNil(t, cur)
		assert.Equal(t, tea.CursorBar, cur.Shape)
		assert.Equal(t, 4, cur.Position.X) // ": " + "ab"

		m = sendSpecial(m, tea.KeyLeft)
		assert.Equal(t, 3, m.View().Cursor.Position.X)
	})

	t.Run("honors a block insert cursor", func(t *testing.T) {
		m, e := cmdPrompt(t)
		e.Options().CursorShape.Insert = view.CursorKindBlock
		m = typeString(m, "ab")

		cur := m.View().Cursor
		assert.NotNil(t, cur)
		assert.Equal(t, tea.CursorBlock, cur.Shape)
	})

	t.Run("hidden insert cursor shows none", func(t *testing.T) {
		m, e := cmdPrompt(t)
		e.Options().CursorShape.Insert = view.CursorKindHidden
		m = typeString(m, "ab")
		assert.Nil(t, m.View().Cursor)
	})
}

func TestRegexPromptAccept(t *testing.T) {
	t.Run("enter submits regex prompt", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 60, 12)

		m = sendKey(m, 's')
		for _, ch := range "hello" {
			m = sendKey(m, ch)
		}
		m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		m = m2.(ui.Model)

		out := stripANSI(m.View().Content)
		assert.NotEmpty(t, out)
	})

	t.Run("enter with empty regex prompt", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 60, 12)

		m = sendKey(m, 's')
		m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		m = m2.(ui.Model)

		out := stripANSI(m.View().Content)
		assert.NotEmpty(t, out)
	})
}

func TestSearchPromptAccept(t *testing.T) {
	t.Run("enter submits search pattern", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 60, 12)

		m = sendKey(m, '/')
		assert.Contains(t, stripANSI(m.View().Content),
			"search-forward: ")
		for _, ch := range "hello" {
			m = sendKey(m, ch)
		}
		m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		m = m2.(ui.Model)

		out := stripANSI(m.View().Content)
		assert.NotEmpty(t, out)
	})

	t.Run("backward search prompt render", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 60, 12)

		m = sendKey(m, '?')
		m = sendKey(m, 'x')
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "search-backward: x")
	})

	t.Run("backward search enter submits", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 60, 12)

		m = sendKey(m, '?')
		for _, ch := range "hello" {
			m = sendKey(m, ch)
		}
		m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		m = m2.(ui.Model)

		out := stripANSI(m.View().Content)
		assert.NotEmpty(t, out)
	})

	t.Run("search enter with empty pattern", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 60, 12)

		m = sendKey(m, '/')
		m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		m = m2.(ui.Model)

		out := stripANSI(m.View().Content)
		assert.NotEmpty(t, out)
	})

	t.Run("empty search repeats prior pattern", func(t *testing.T) {
		dir := t.TempDir()
		source := filepath.Join(dir, "source.go")
		assert.NoError(t,
			os.WriteFile(source, []byte("x foo y foo z\n"), 0o600))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(source)
		assert.NoError(t, err)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err = builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 60, 12)

		m = sendKey(m, '/')
		for _, ch := range "foo" {
			m = sendKey(m, ch)
		}
		m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		m = m2.(ui.Model)

		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		first := doc.SelectionFor(v.ID()).Primary().Cursor(doc.Text())
		assert.Equal(t, 2, first)

		m = sendKey(m, '/')
		_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

		second := doc.SelectionFor(v.ID()).Primary().Cursor(doc.Text())
		assert.Equal(t, 8, second)
	})
}

func TestSearchPromptError(t *testing.T) {
	t.Run("invalid regex shows error", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 60, 12)

		m = sendKey(m, '/')
		for _, ch := range "[invalid" {
			m = sendKey(m, ch)
		}
		m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		m = m2.(ui.Model)

		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "error")
	})
}

func TestRegexFnError(t *testing.T) {
	t.Run("invalid regex fn shows error", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 60, 12)

		m = sendKey(m, 's')
		for _, ch := range "[invalid" {
			m = sendKey(m, ch)
		}
		m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		m = m2.(ui.Model)

		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "error")
	})
}

func TestPromptHandlesMouse(t *testing.T) {
	t.Run("mouse ignored while prompt open", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 60, 12)

		m = sendKey(m, ':')
		m2, _ := m.Update(tea.MouseClickMsg{X: 5, Y: 5, Button: tea.MouseLeft})
		m = m2.(ui.Model)

		out := m.View().Content
		assert.Contains(t, out, ":")
	})

	t.Run("mouse motion is consumed", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 60, 12)

		m = sendKey(m, ':')
		m2, _ := m.Update(tea.MouseMotionMsg{X: 5, Y: 5})
		m = m2.(ui.Model)

		out := m.View().Content
		assert.Contains(t, out, ":")
	})
}

func TestRedrawSignal(t *testing.T) {
	t.Run("redraw via prompt clears screen", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 60, 12)

		m = sendKey(m, ':')
		for _, ch := range "redraw" {
			m = sendKey(m, ch)
		}
		m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		m = m2.(ui.Model)

		out := stripANSI(m.View().Content)
		assert.NotEmpty(t, out)
	})
}

func cmdPrompt(t *testing.T) (ui.Model, *view.Editor) {
	t.Helper()
	e := view.NewEditor(t.TempDir())
	km := command.NewKeymaps()
	m := ui.New(e, km)
	_, err := builtin.Register(m, km)
	assert.NoError(t, err)
	m = resize(m, 40, 12)
	return sendKey(m, ':'), e
}

func typeString(m ui.Model, s string) ui.Model {
	for _, ch := range s {
		m = sendKey(m, ch)
	}
	return m
}

func promptText(m ui.Model) string {
	content := strings.TrimRight(stripANSI(m.View().Content), "\n")
	lines := strings.Split(content, "\n")
	return strings.TrimRight(lines[len(lines)-1], " ")
}

func testCommand(name string) command.Command {
	return command.Command{
		Run: func(
			*view.Editor, *command.Args,
		) command.Result {
			return command.Result{}
		},
		Aliases:   []string{name},
		Signature: command.DefaultSignature(),
	}
}
