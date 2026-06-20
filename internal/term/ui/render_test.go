package ui_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

type bindTestActionArgs struct {
	km   *command.Keymaps
	mode string
	name string
	fn   command.KeyAction
	seqs [][]command.KeyEvent
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
		seqs: [][]command.KeyEvent{{command.Char('i')}},
	})
	bindTestAction(bindTestActionArgs{
		km: km, mode: "INS", name: "insert_newline",
		fn:   act(action.InsertNewline),
		seqs: [][]command.KeyEvent{{command.Special("ret")}},
	})
	bindTestAction(bindTestActionArgs{
		km: km, mode: "INS", name: "insert_newline",
		fn: act(action.InsertNewline),
		seqs: [][]command.KeyEvent{{
			command.Char('j').WithMods(command.ModCtrl),
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
}

func TestModeColorRender(t *testing.T) {
	t.Run("applies mode color", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		e := view.NewEditor(t.TempDir())
		cfg := e.Config()
		cfg.Theme.Name = "mocha"
		e.SetConfig(cfg)
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := m.View().Content

		assert.Contains(t, out, "\x1b[48;2;245;224;220m NOR ")
	})
}

func TestStatuslineConfigRender(t *testing.T) {
	t.Run("uses configured mode label", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		e := view.NewEditor(t.TempDir())
		e.Options().StatusLine.Mode.Normal = "NORMAL"
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := m.View().Content

		assert.Contains(t, out, " NORMAL ")
		assert.NotContains(t, out, " NOR ")
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
		cfg := e.Config()
		cfg.Theme.Name = "mocha"
		e.SetConfig(cfg)
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
		cfg := e.Config()
		cfg.Theme.Name = "mocha"
		e.SetConfig(cfg)
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := m.View().Content

		assert.Contains(t, out, "\x1b[38;2;205;214;244m")
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
		cfg := e.Config()
		cfg.Theme.Name = "mocha"
		e.SetConfig(cfg)
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
			assert.Equal(t, 13, ansi.StringWidth(pfx))
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
				command.Char(' '), command.Char('界'),
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
		cfg := e.Config()
		cfg.Theme.Name = "mocha"
		e.SetConfig(cfg)
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := m.View().Content

		assert.Contains(t, out, "\x1b[38;2;180;190;254m")
		assert.Contains(t, out, "\x1b[38;2;69;71;90m")
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
		cfg := e.Config()
		cfg.Theme.Name = "mocha"
		e.SetConfig(cfg)
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := m.View().Content

		assert.Contains(t, out, "\x1b[48;2;69;71;90m")
		assert.Contains(t, out, "\x1b[48;2;245;224;220m")
	})

	t.Run("applies commandline styles", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		e := view.NewEditor(root)
		cfg := e.Config()
		cfg.Theme.Name = "mocha"
		e.SetConfig(cfg)
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		prompt := sendKey(m, ':').View().Content
		errOut := runTypable(m, "not-a-command").View().Content

		assert.Contains(t, prompt, "\x1b[38;2;205;214;244m")
		assert.Contains(t, errOut, "\x1b[38;2;243;139;168m")
	},
	)
}

func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)
	return strings.TrimRight(re.ReplaceAllString(s, ""), "\n")
}

func assertPromptCountRightPadding(t *testing.T, out string) {
	t.Helper()
	re := regexp.MustCompile(`[0-9]+/[0-9]+ `)
	assert.True(t, re.MatchString(out))
}

func assertRenderedWidth(t *testing.T, out string, w int) {
	t.Helper()
	for line := range strings.SplitSeq(out, "\n") {
		assert.LessOrEqual(t, ansi.StringWidth(line), w)
	}
}

func writeLanguageConfig(t *testing.T, root, lang string, enabled bool) {
	t.Helper()
	dir := filepath.Join(root, loader.DirName)
	err := os.MkdirAll(dir, 0o755)
	assert.NoError(t, err)
	text := "[[language]]\n" +
		"name = \"" + lang + "\"\n" +
		fmt.Sprintf("soft-wrap.enable = %t\n", enabled)
	err = os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	)
	assert.NoError(t, err)
	t.Setenv("XDG_CONFIG_HOME", root)
}

func writeConfig(t *testing.T, root, text string) {
	t.Helper()
	dir := filepath.Join(root, loader.DirName)
	err := os.MkdirAll(dir, 0o755)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "config.toml"), []byte(text), 0o644)
	assert.NoError(t, err)
}

func writeConfigIgnore(t *testing.T, root, text string) {
	t.Helper()
	dir := filepath.Join(root, loader.DirName)
	err := os.MkdirAll(dir, 0o755)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "ignore"), []byte(text), 0o644)
	assert.NoError(t, err)
	t.Setenv("XDG_CONFIG_HOME", root)
}

// sendKey simulates a single typed character key press
func sendKey(m ui.Model, ch rune) ui.Model {
	m2, _ := m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	return m2.(ui.Model)
}

func openPickerAndFeed(m ui.Model, ch rune) ui.Model {
	m2, cmd := m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	m = m2.(ui.Model)
	for cmd != nil {
		msg := cmd()
		if msg == nil {
			break
		}
		m2, cmd = m.Update(msg)
		m = m2.(ui.Model)
	}
	return m
}

// sendSpecial simulates a special key press (Enter, Backspace, etc.)
func sendSpecial(m ui.Model, k rune) ui.Model {
	m2, _ := m.Update(tea.KeyPressMsg{Code: k})
	return m2.(ui.Model)
}

func sendSpecialText(m ui.Model, k rune, text string) ui.Model {
	m2, _ := m.Update(tea.KeyPressMsg{Code: k, Text: text})
	return m2.(ui.Model)
}

func resize(m ui.Model, w, h int) ui.Model {
	m2, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return m2.(ui.Model)
}

func bindTestAction(args bindTestActionArgs) {
	_ = args.km.Register(args.name, command.Command{
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			return command.Result{Continuation: args.fn(e)}
		},
		Modes: []string{args.mode},
		Keys:  map[string][]command.KeyBinding{"*": {args.seqs}},
	})
}

func bindNormalTestAction(
	km *command.Keymaps, name string, fn command.KeyAction,
	seqs ...[]command.KeyEvent,
) {
	bindTestAction(bindTestActionArgs{
		km: km, mode: "NOR", name: name, fn: fn, seqs: seqs,
	})
}
