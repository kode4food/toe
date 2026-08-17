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

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/builtin"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type promptRow struct {
	raw  string
	text string
}

func TestPromptFrame(t *testing.T) {
	t.Run("command title and breadcrumb", func(t *testing.T) {
		m, _ := cmdPrompt(t)

		assert.Contains(t, stripANSI(m.View().Content), " Command ")
		assert.Equal(t, ":", promptText(m))
	})

	t.Run("multi-key search breadcrumb", func(t *testing.T) {
		m := sendKey(sendKey(builtinModel(t), 'z'), '/')

		assert.Contains(t, stripANSI(m.View().Content), " Search forward ")
		assert.Equal(t, "z /", promptText(m))
	})

	t.Run("outside click dismisses", func(t *testing.T) {
		m, _ := cmdPrompt(t)
		m = mouse(m, tea.MouseClickMsg{X: 0, Y: 0, Button: tea.MouseLeft})

		assert.NotContains(t, stripANSI(m.View().Content), " Command ")
	})

	tests := []struct {
		name  string
		key   i18n.Key
		shell bool
		title string
	}{
		{name: "regex select", key: "prompt.select", title: "Select"},
		{name: "regex split", key: "prompt.split", title: "Split"},
		{name: "regex keep", key: "prompt.keep", title: "Keep"},
		{name: "regex remove", key: "prompt.remove", title: "Remove"},
		{
			name:  "shell pipe",
			key:   "prompt.pipe",
			shell: true,
			title: "Pipe",
		},
		{
			name:  "shell insert",
			key:   "prompt.insertOutput",
			shell: true,
			title: "Insert output",
		},
		{
			name:  "shell filter",
			key:   "prompt.filter",
			shell: true,
			title: "Filter",
		},
		{
			name:  "shell pipe to",
			key:   "prompt.pipeTo",
			shell: true,
			title: "Pipe to",
		},
		{
			name:  "shell append",
			key:   "prompt.appendOutput",
			shell: true,
			title: "Append output",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := view.NewEditor(t.TempDir())
			km := command.NewKeymaps()
			m := ui.New(e, km)
			_, err := builtin.Register(m, km)
			assert.NoError(t, err)
			m = resize(m, 60, 12)
			fn := func(*view.Editor, string) error { return nil }
			if tc.shell {
				m.ShellAction(tc.key, fn)(e)
			} else {
				m.RegexAction(tc.key, fn)(e)
			}
			m2, _ := m.Update(struct{}{})
			m = m2.(ui.Model)

			assert.Contains(t, stripANSI(m.View().Content), " "+tc.title+" ")
		})
	}
}

func TestPromptCompletion(t *testing.T) {
	t.Run("completions update as input changes", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_ = km.Register("command_mode", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				m.CmdModeAction(e)
				return command.Result{}
			},
			Modes: view.ModeNormal,
			Keys: map[view.Mode]command.KeyBinding{
				view.ModeAny: {{char(':')}},
			},
		})
		alpha := testCommand("alpha")
		alpha.Aliases = append(alpha.Aliases, "alias")
		_ = km.Register("alpha", alpha)
		_ = km.Register("beta", testCommand("beta"))
		m = resize(m, 60, 12)

		m = sendKey(m, ':')
		m = sendKey(m, 'a')
		withA := stripANSI(m.View().Content)
		m = sendKey(m, 'l')
		withAl := stripANSI(m.View().Content)

		assert.Contains(t, withA, "alpha")
		assert.Contains(t, withA, "alias")
		assert.NotContains(t, withA, "beta")
		assert.Contains(t, withAl, "alpha")
		assert.NotContains(t, withAl, "beta")
		assert.Contains(t, withAl, "╭")
		for line := range strings.SplitSeq(stripANSI(withAl), "\n") {
			if strings.Contains(line, "alpha") ||
				strings.Contains(line, "╭") ||
				strings.Contains(line, "╰") {
				assert.Equal(t, 60, ansi.StringWidth(line))
			}
		}
	})

	t.Run("filters command names by mode", func(t *testing.T) {
		root := t.TempDir()
		e := view.NewEditor(root)
		openRenderImagePane(t, e, writeRenderImage(t, root, 4, 4, nil))
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_ = km.Register("command_mode", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				m.CmdModeAction(e)
				return command.Result{}
			},
			Modes: view.ModeImage,
			Keys: map[view.Mode]command.KeyBinding{
				view.ModeAny: {{char(':')}},
			},
		})
		img := testCommand("image-only")
		img.Modes = view.ModeImage
		doc := testCommand("document-only")
		doc.Modes = view.ModeNormal
		_ = km.Register("image_only", img)
		_ = km.Register("document_only", doc)
		m = resize(m, 60, 12)

		m = sendKey(m, ':')
		m = sendKey(m, 'o')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "image-only")
		assert.NotContains(t, out, "document-only")
	})

	t.Run("lists what the frame fits", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_ = km.Register("command_mode", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				m.CmdModeAction(e)
				return command.Result{}
			},
			Modes: view.ModeNormal,
			Keys: map[view.Mode]command.KeyBinding{
				view.ModeAny: {{char(':')}},
			},
		})
		for _, name := range []string{
			"match-a", "match-b", "match-c", "match-d",
			"match-e", "match-f", "match-g", "match-h",
			"few-a", "few-command-with-long-name",
		} {
			_ = km.Register(name, testCommand(name))
		}
		m = resize(m, 60, 14)
		m = sendKey(m, ':')

		content := stripANSI(m.View().Content)
		rows := 0
		for line := range strings.SplitSeq(content, "\n") {
			if strings.Contains(line, "match-") ||
				strings.Contains(line, "few-") {
				rows++
			}
		}
		assert.Equal(t, 5, rows)
		assert.NotContains(t, content, "few-command-with-long-name")
		assert.NotContains(t, content, "┬")
		assert.NotContains(t, content, "┴")

		for _, ch := range "few" {
			m = sendKey(m, ch)
		}
		filtered := stripANSI(m.View().Content)
		assert.Contains(t, filtered, "few-command-with-long-name")
	})

	t.Run("tab accepts the first completion", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_ = km.Register("command_mode", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				m.CmdModeAction(e)
				return command.Result{}
			},
			Modes: view.ModeNormal,
			Keys: map[view.Mode]command.KeyBinding{
				view.ModeAny: {{char(':')}},
			},
		})
		_ = km.Register("alpha", testCommand("alpha"))
		m = resize(m, 60, 12)

		m = sendKey(m, ':')
		m = sendKey(m, 'a')
		m = sendSpecial(m, tea.KeyTab)

		assert.Equal(t, ": alpha", promptText(m))
	})

	t.Run("accepting advances the caret", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_ = km.Register("command_mode", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				m.CmdModeAction(e)
				return command.Result{}
			},
			Modes: view.ModeNormal,
			Keys: map[view.Mode]command.KeyBinding{
				view.ModeAny: {{char(':')}},
			},
		})
		_ = km.Register("alpha", testCommand("alpha"))
		m = resize(m, 60, 12)

		m = sendKey(m, ':')
		m = sendKey(m, 'a')
		m = sendSpecial(m, tea.KeyTab)

		// typing after acceptance lands at the end, not inside the old input
		m = sendKey(m, 'X')
		assert.Equal(t, ": alphaX", promptText(m))
	})

	t.Run("keyboard and mouse navigate", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_ = km.Register("command_mode", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				m.CmdModeAction(e)
				return command.Result{}
			},
			Modes: view.ModeNormal,
			Keys: map[view.Mode]command.KeyBinding{
				view.ModeAny: {{char(':')}},
			},
		})
		for _, name := range []string{
			"match-a", "match-b", "match-c", "match-d",
			"match-e", "match-f", "match-g", "match-h",
		} {
			_ = km.Register(name, testCommand(name))
		}
		m = resize(m, 60, 14)
		m = sendKey(m, ':')
		m = typeString(m, "match")
		_ = m.View()

		// moving the selection leaves the input alone until it is accepted
		m = sendSpecial(m, tea.KeyDown)
		m = sendSpecial(m, tea.KeyDown)
		assert.Equal(t, ": match", promptText(m))
		m = sendSpecial(m, tea.KeyUp)
		m = sendSpecial(m, tea.KeyTab)
		assert.Equal(t, ": match-b", promptText(m))

		m = sendSpecial(m, tea.KeyEscape)
		m = sendKey(m, ':')
		m = typeString(m, "match")
		_ = m.View()

		at := completionTextPoint(t, m, "match-b")
		m = mouse(m, tea.MouseWheelMsg{
			X: at.X, Y: at.Y, Button: tea.MouseWheelDown,
		})
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "match-h")
		assert.NotContains(t, out, "match-a")
		assert.Equal(t, ": match", promptText(m))

		at = completionTextPoint(t, m, "match-g")
		m = mouse(m, tea.MouseClickMsg{
			X: at.X, Y: at.Y, Button: tea.MouseLeft,
		})
		assert.Equal(t, ": match-g", promptText(m))
	})

	t.Run("enter accepts, then submits", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_ = km.Register("command_mode", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				m.CmdModeAction(e)
				return command.Result{}
			},
			Modes: view.ModeNormal,
			Keys: map[view.Mode]command.KeyBinding{
				view.ModeAny: {{char(':')}},
			},
		})
		var ran bool
		_ = km.Register("alpha", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				ran = true
				return command.Result{}
			},
			Modes:   view.ModeNormal,
			Aliases: []string{"alpha"},
		})
		m = resize(m, 60, 12)

		m = sendKey(m, ':')
		m = sendKey(m, 'a')
		m = sendSpecial(m, tea.KeyEnter)

		assert.Equal(t, ": alpha", promptText(m))
		assert.False(t, ran)

		m = sendSpecial(m, tea.KeyEnter)

		assert.True(t, ran)
	})

	t.Run("ignores command args without completer", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_ = km.Register("command_mode", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				m.CmdModeAction(e)
				return command.Result{}
			},
			Modes: view.ModeNormal,
			Keys: map[view.Mode]command.KeyBinding{
				view.ModeAny: {{char(':')}},
			},
		})
		_ = km.Register("alpha", testCommand("alpha"))
		m = resize(m, 60, 12)

		m = sendKey(m, ':')
		for _, ch := range "alpha " {
			m = sendKey(m, ch)
		}

		assert.Contains(t, stripANSI(m.View().Content), ": alpha ")
	})

	t.Run("lists the command description", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_ = km.Register("command_mode", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				m.CmdModeAction(e)
				return command.Result{}
			},
			Modes: view.ModeNormal,
			Keys: map[view.Mode]command.KeyBinding{
				view.ModeAny: {{char(':')}},
			},
		})
		alpha := testCommand("alpha")
		alpha.DocString = "does an alpha thing"
		_ = km.Register("alpha", alpha)
		m = resize(m, 60, 14)

		m = sendKey(m, ':')
		m = sendKey(m, 'a')

		var item string
		for _, row := range promptRows(m) {
			if strings.Contains(row.text, "alpha") &&
				!strings.Contains(row.text, ": ") {
				item = row.text
			}
		}
		assert.Regexp(t, `^alpha\s+does an alpha thing$`, item)
	})

	t.Run("lists the option description", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 90, 20)

		m = sendKey(m, ':')
		for _, ch := range "set scrolloff" {
			m = sendKey(m, ch)
		}

		var item string
		for _, row := range promptRows(m) {
			if strings.HasPrefix(row.text, "scrolloff") {
				item = row.text
			}
		}
		assert.Contains(t, item, "Lines of context kept")
	})

	t.Run("matches the name, not the description", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_ = km.Register("command_mode", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				m.CmdModeAction(e)
				return command.Result{}
			},
			Modes: view.ModeNormal,
			Keys: map[view.Mode]command.KeyBinding{
				view.ModeAny: {{char(':')}},
			},
		})
		alpha := testCommand("alpha")
		alpha.DocString = "zzzz unique words"
		_ = km.Register("alpha", alpha)
		m = resize(m, 60, 14)

		m = sendKey(m, ':')
		for _, ch := range "zzzz" {
			m = sendKey(m, ch)
		}

		assert.NotContains(t, stripANSI(m.View().Content), "unique words")
	})

	t.Run("completions share the input background", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		e := view.NewEditor(t.TempDir())
		e.Options().Theme = "mocha"
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_ = km.Register("command_mode", command.Command{
			Run: func(*view.Editor, *command.Args) command.Result {
				m.CmdModeAction(e)
				return command.Result{}
			},
			Modes: view.ModeNormal,
			Keys: map[view.Mode]command.KeyBinding{
				view.ModeAny: {{char(':')}},
			},
		})
		_ = km.Register("alpha", testCommand("alpha"))
		_ = km.Register("alpine", testCommand("alpine"))
		m = resize(m, 60, 14)

		m = sendKey(m, ':')
		m = sendKey(m, 'a')

		// the first row is selected, so compare against an unselected one
		var border, input, item string
		for line := range strings.SplitSeq(m.View().Content, "\n") {
			plain := stripANSI(line)
			switch {
			case strings.Contains(plain, "╭"):
				border = line
			case strings.Contains(plain, ": a"):
				input = line
			case strings.Contains(plain, "alpine"):
				item = line
			}
		}

		assert.NotEmpty(t, backgrounds(input))
		assert.Equal(t, backgrounds(input), backgrounds(item))
		assert.Equal(t, backgrounds(input), backgrounds(border))
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

	t.Run("marks current option value", func(t *testing.T) {
		cases := []struct {
			name   string
			nerd   bool
			marker string
		}{
			{name: "nerd font", nerd: true, marker: "\uf42e"},
			{name: "ascii", nerd: false, marker: "*"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Setenv("XDG_CONFIG_HOME", t.TempDir())
				t.Setenv("COLORTERM", "truecolor")
				e := view.NewEditor(t.TempDir())
				e.Options().Theme = "mocha"
				e.Options().NerdFonts = tc.nerd
				m := resize(newTestModel(t, e), 60, 12)
				m = sendKey(m, ':')
				m = typeString(m, "set line-number ")

				out := stripANSI(m.View().Content)
				assert.Contains(t, out, "absolute "+tc.marker)
			})
		}
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

	t.Run("tab does not navigate completions", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err := builtin.Register(m, km)
		assert.NoError(t, err)
		m = resize(m, 60, 12)

		m = sendKey(m, ':')
		m = sendKey(m, 'o')
		m = sendSpecial(m, tea.KeyTab)

		assert.Equal(t, ": o", promptText(m))
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

		promptLine := promptText(m)
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
		empty := m.View().Cursor.Position.X // after the ": " label
		m = typeString(m, "ab")

		cur := m.View().Cursor
		assert.NotNil(t, cur)
		assert.Equal(t, tea.CursorBar, cur.Shape)
		assert.Equal(t, empty+2, cur.Position.X)

		m = sendSpecial(m, tea.KeyLeft)
		assert.Equal(t, empty+1, m.View().Cursor.Position.X)
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
		assert.Contains(t, stripANSI(m.View().Content), " Search forward ")
		assert.Equal(t, "/", promptText(m))
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
		assert.Contains(t, out, " Search backward ")
		assert.Equal(t, "? x", promptText(m))
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

		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		v := e.FocusedView()
		assert.NotNil(t, v)
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
	rows := promptRows(m)
	if len(rows) == 0 {
		return ""
	}
	return rows[0].text
}

func promptRows(m ui.Model) []promptRow {
	var out []promptRow
	content := strings.TrimRight(m.View().Content, "\n")
	for raw := range strings.SplitSeq(content, "\n") {
		line := stripANSI(raw)
		from := strings.Index(line, tui.BorderV)
		to := strings.LastIndex(line, tui.BorderV)
		if from < 0 || to <= from {
			continue
		}
		inner := line[from+len(tui.BorderV) : to]
		out = append(out, promptRow{
			raw:  raw,
			text: strings.TrimRight(strings.TrimLeft(inner, " "), " "),
		})
	}
	return out
}

func testCommand(name string) command.Command {
	return command.Command{
		Run: func(
			*view.Editor, *command.Args,
		) command.Result {
			return command.Result{}
		},
		Aliases: []string{name},
		Modes:   command.PaneModes,
	}
}
