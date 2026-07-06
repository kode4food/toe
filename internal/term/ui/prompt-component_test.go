package ui_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/defaults"
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

		assert.Contains(t, m.View().Content, ":alpha")
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

		assert.Contains(t, m.View().Content, ":alpha ")
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

		assert.Regexp(t,
			regexp.MustCompile(`\x1b\[[0-9;]*48;2;49;50;68[0-9;]*m╭`),
			m.View().Content,
		)
		for raw := range strings.SplitSeq(m.View().Content, "\n") {
			if !strings.Contains(raw, "alpha") {
				continue
			}
			before, _, _ := strings.Cut(raw, "alpha")
			assert.Contains(t, before, "48;2;49;50;68m")
			idx := strings.LastIndex(before, "48;2;49;50;68m")
			if idx >= 0 {
				assert.NotContains(t, before[idx:], "49m")
			}
			return
		}
		assert.Contains(t, m.View().Content, "alpha")
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
		_, err = defaults.RegisterDefaults(m, km)
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
		_, err := defaults.RegisterDefaults(m, km)
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
		_, err := defaults.RegisterDefaults(m, km)
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
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 60, 12)

		m = sendKey(m, ':')
		m = sendKey(m, 'a')
		m = sendKey(m, 'b')
		m = sendSpecial(m, tea.KeyBackspace)

		out := stripANSI(m.View().Content)
		assert.Contains(t, out, ":a")
		assert.NotContains(t, out, ":ab")
	})

	t.Run("ctrl h removes input", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
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
		assert.NotContains(t, out, ":x")
	})

	t.Run("shift tab wraps completions", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
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
}

func TestRegexPromptAccept(t *testing.T) {
	t.Run("enter submits regex prompt", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
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
		_, err := defaults.RegisterDefaults(m, km)
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
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 60, 12)

		m = sendKey(m, '/')
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
		_, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)
		m = resize(m, 60, 12)

		m = sendKey(m, '?')
		m = sendKey(m, 'x')
		out := m.View().Content
		assert.Contains(t, out, "?x")
	})

	t.Run("backward search enter submits", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := defaults.RegisterDefaults(m, km)
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
		_, err := defaults.RegisterDefaults(m, km)
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
		_, err = defaults.RegisterDefaults(m, km)
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
		_, err := defaults.RegisterDefaults(m, km)
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
		_, err := defaults.RegisterDefaults(m, km)
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
		_, err := defaults.RegisterDefaults(m, km)
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
		_, err := defaults.RegisterDefaults(m, km)
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
		_, err := defaults.RegisterDefaults(m, km)
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
