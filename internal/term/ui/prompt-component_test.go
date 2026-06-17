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
		km.Register("command_mode", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				return command.Result{Continuation: m.CmdModeAction()(e)}
			},
			Modes: []string{"NOR"},
			Keys: []command.KeyBinding{[][]command.KeyEvent{
				{command.Char(':')},
			}},
		})
		km.Register("alpha", testCommand("alpha"))
		km.Register("beta", testCommand("beta"))
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
		km.Register("command_mode", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				return command.Result{Continuation: m.CmdModeAction()(e)}
			},
			Modes: []string{"NOR"},
			Keys: []command.KeyBinding{[][]command.KeyEvent{
				{command.Char(':')},
			}},
		})
		km.Register("alpha", testCommand("alpha"))
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
		km.Register("command_mode", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				return command.Result{Continuation: m.CmdModeAction()(e)}
			},
			Modes: []string{"NOR"},
			Keys: []command.KeyBinding{[][]command.KeyEvent{
				{command.Char(':')},
			}},
		})
		km.Register("alpha", testCommand("alpha"))
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
		cfg := e.Config()
		cfg.Theme.Name = "mocha"
		e.SetConfig(cfg)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		km.Register("command_mode", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				return command.Result{Continuation: m.CmdModeAction()(e)}
			},
			Modes: []string{"NOR"},
			Keys: []command.KeyBinding{[][]command.KeyEvent{
				{command.Char(':')},
			}},
		})
		km.Register("alpha", testCommand("alpha"))
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

	t.Run("completes file arguments after command name", func(t *testing.T) {
		root := t.TempDir()
		err := os.WriteFile(
			filepath.Join(root, "main.go"), []byte("package main\n"), 0o644,
		)
		assert.NoError(t, err)
		e := view.NewEditor(root)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		defaults.RegisterDefaults(m, km)
		m = resize(m, 60, 12)

		m = sendKey(m, ':')
		for _, ch := range "open m" {
			m = sendKey(m, ch)
		}

		assert.Contains(t, m.View().Content, "main.go")
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
