package ui_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"

	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

const testPickerPreviewWidth = 100

func TestPickerPreview(t *testing.T) {
	t.Run("keeps themed background across short rows", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		path := filepath.Join(tmp, "note.txt")
		err := os.WriteFile(path, []byte("plain\n"), 0o644)
		assert.NoError(t, err)

		e := view.NewEditor(tmp)
		cfg := e.Config()
		cfg.Theme.Name = "mocha"
		e.SetConfig(cfg)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker",
			m.PickerAction(ui.FilePickerInDir(tmp)),
			[]command.KeyEvent{command.Char('p')},
		)

		m = resize(m, 100, 18)
		m = sendKey(m, 'p')
		out := m.View().Content

		// mocha ui.background = base = rgb(30,30,46); ui.popup = surface0 =
		// rgb(49,50,68). Content cells must carry the popup bg, not the editor
		// bg or terminal default (\x1b[49m)
		assert.NotContains(t, out, "48;2;30;30;46mplain")
		for line := range strings.SplitSeq(out, "\n") {
			if strings.Contains(line, "plain") {
				assert.NotContains(t, line, "\x1b[49m")
				assert.Contains(t, line, "48;2;49;50;68")
				return
			}
		}
		assert.Contains(t, out, "plain")
	})

	t.Run("syntax cells carry popup background", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		path := filepath.Join(tmp, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("package main\n"), 0o644))

		e := view.NewEditor(tmp)
		cfg := e.Config()
		cfg.Theme.Name = "mocha"
		e.SetConfig(cfg)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker",
			m.PickerAction(ui.FilePickerInDir(tmp)),
			[]command.KeyEvent{command.Char('p')},
		)

		m = resize(m, 100, 18)
		m = sendKey(m, 'p')
		out := m.View().Content

		// "package" is syntax-colored (mauve fg); its cells must still carry
		// the popup bg (49;50;68), not terminal default from a bg reset
		for line := range strings.SplitSeq(out, "\n") {
			if strings.Contains(line, "package") {
				assert.NotContains(t, line, "\x1b[49m")
				assert.Contains(t, line, "48;2;49;50;68")
				return
			}
		}
		assert.Contains(t, out, "package")
	})

	t.Run("styles full row right padding", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		line := "package main // " + strings.Repeat("x", 160)
		text := strings.Repeat(line+"\n", 40)
		path := filepath.Join(tmp, "main.go")
		err := os.WriteFile(path, []byte(text), 0o644)
		assert.NoError(t, err)

		e := view.NewEditor(tmp)
		cfg := e.Config()
		cfg.Theme.Name = "mocha"
		e.SetConfig(cfg)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker",
			m.PickerAction(ui.FilePickerInDir(tmp)),
			[]command.KeyEvent{command.Char('p')},
		)

		m = resize(m, 100, 18)
		m = sendKey(m, 'p')

		for raw := range strings.SplitSeq(m.View().Content, "\n") {
			if !strings.Contains(raw, "package") {
				continue
			}
			border := strings.LastIndex(raw, "│")
			assert.NotEqual(t, -1, border)
			tail := raw[max(0, border-80):]
			// right padding must not carry the editor document background;
			// the pane provides the background via its outer render
			assert.NotRegexp(t,
				regexp.MustCompile(`48;2;30;30;46[0-9;]*m +\x1b\[[0-9;]*m│`),
				tail,
			)
			return
		}
		assert.Contains(t, m.View().Content, "package")
	})

	t.Run("clips long lines", func(t *testing.T) {
		tmp := t.TempDir()
		writeLanguageConfig(t, t.TempDir(), "markdown", true)
		long := "\t" + strings.Repeat("x", 200)
		err := os.WriteFile(filepath.Join(tmp, "main.go"), []byte(long), 0o644)
		assert.NoError(t, err)

		e := view.NewEditor(tmp)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker",
			m.PickerAction(ui.FilePickerInDir(tmp)),
			[]command.KeyEvent{command.Char('p')},
		)

		width := testPickerPreviewWidth
		m = resize(m, width, 24)
		m = sendKey(m, 'p')
		out := stripANSI(m.View().Content)

		assert.NotContains(t, out, long)
		assert.NotContains(t, out, "\t")
		assert.NotContains(t, out, "↪")
		for line := range strings.SplitSeq(out, "\n") {
			assert.LessOrEqual(t, len([]rune(line)), width)
		}
	})

	t.Run("soft wraps markdown", func(t *testing.T) {
		tmp := t.TempDir()
		writeLanguageConfig(t, t.TempDir(), "markdown", true)
		text := strings.Repeat("word ", 80)
		path := filepath.Join(tmp, "notes.md")
		err := os.WriteFile(path, []byte(text), 0o644)
		assert.NoError(t, err)

		e := view.NewEditor(tmp)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker",
			m.PickerAction(ui.FilePickerInDir(tmp)),
			[]command.KeyEvent{command.Char('p')},
		)

		width := testPickerPreviewWidth
		m = resize(m, width, 24)
		m = sendKey(m, 'p')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "↪ ")
		for line := range strings.SplitSeq(out, "\n") {
			assert.LessOrEqual(t, len([]rune(line)), width)
		}
	})

	t.Run("retains indent on wrap", func(t *testing.T) {
		tmp := t.TempDir()
		cfgRoot := t.TempDir()
		writeConfig(t, cfgRoot, `
[editor.soft-wrap]
enable = true
wrap-indicator = "» "
max-indent-retain = 8
`)
		text := "    " + strings.Repeat("word ", 20)
		path := filepath.Join(tmp, "notes.md")
		err := os.WriteFile(path, []byte(text), 0o644)
		assert.NoError(t, err)
		t.Setenv("XDG_CONFIG_HOME", cfgRoot)

		e := view.NewEditor(tmp)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker",
			m.PickerAction(ui.FilePickerInDir(tmp)),
			[]command.KeyEvent{command.Char('p')},
		)

		m = resize(m, 100, 18)
		m = sendKey(m, 'p')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "    » ")
	})

	t.Run("wraps makefile", func(t *testing.T) {
		tmp := t.TempDir()
		cfgRoot := t.TempDir()
		writeConfig(t, cfgRoot, `
[editor.soft-wrap]
enable = true
wrap-indicator = "↪ "
max-indent-retain = 40
`)
		text := "install: test\n\tgo install " +
			strings.Repeat("github.com/kode4food/toe/cmd/toe ", 8)
		path := filepath.Join(tmp, "Makefile")
		err := os.WriteFile(path, []byte(text), 0o644)
		assert.NoError(t, err)
		t.Setenv("XDG_CONFIG_HOME", cfgRoot)

		e := view.NewEditor(tmp)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker",
			m.PickerAction(ui.FilePickerInDir(tmp)),
			[]command.KeyEvent{command.Char('p')},
		)

		m = resize(m, 100, 18)
		m = sendKey(m, 'p')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "Makefile")
		assert.Contains(t, out, "go install")
	})

	t.Run("caps wrap rows to height", func(t *testing.T) {
		tmp := t.TempDir()
		cfgRoot := t.TempDir()
		writeConfig(t, cfgRoot, `
[editor.soft-wrap]
enable = true
wrap-indicator = "↪ "
`)
		// A single very long line wraps into many visual rows. The preview must
		// not render more rows than fit its area, so the count of wrap markers
		// stays bounded by the preview height rather than the line length
		path := filepath.Join(tmp, "notes.md")
		err := os.WriteFile(path, []byte(strings.Repeat("word ", 200)), 0o644)
		assert.NoError(t, err)
		t.Setenv("XDG_CONFIG_HOME", cfgRoot)

		e := view.NewEditor(tmp)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker",
			m.PickerAction(ui.FilePickerInDir(tmp)),
			[]command.KeyEvent{command.Char('p')},
		)

		height := 18
		m = resize(m, 100, height)
		m = sendKey(m, 'p')
		out := stripANSI(m.View().Content)

		assert.GreaterOrEqual(t, strings.Count(out, "↪ "), 1)
		assert.LessOrEqual(t, strings.Count(out, "↪ "), height)
	})

	t.Run("range highlight keeps syntax foreground", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		path := filepath.Join(tmp, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("package main\n"), 0o644))

		e := view.NewEditor(tmp)
		cfg := e.Config()
		cfg.Theme.Name = "mocha"
		e.SetConfig(cfg)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "buffer_picker", m.PickerAction(ui.BufferPicker),
			[]command.KeyEvent{command.Char('b')},
		)

		m = resize(m, 100, 30)
		m = sendKey(m, 'b')
		for _, ch := range "main" {
			m = sendKey(m, ch)
		}
		out := m.View().Content

		// On the highlighted preview row the "package" keyword keeps its mocha
		// syntax foreground (mauve 203;166;247) with the highlight background
		// (surface1 69;71;90) overlaid behind it. The old strip-and-restyle
		// path could not produce a syntax foreground under the highlight.
		// The two SGR codes may be separate escapes or combined — check both
		// are present on a single line containing "package"
		found := false
		for line := range strings.SplitSeq(out, "\n") {
			if strings.Contains(line, "package") &&
				strings.Contains(line, "203;166;247") &&
				strings.Contains(line, "69;71;90") {
				found = true
				break
			}
		}
		assert.True(t, found,
			"expected a line with 'package' to carry both the syntax fg "+
				"(203;166;247) and the highlight bg (69;71;90)")
	})

	t.Run("re-renders to new width after resize", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		// a long in-memory line whose preview is wider than the narrow window;
		// after a resize the cached spans must lay out at the new width, so no
		// frame line may exceed it
		path := filepath.Join(tmp, "long.go")
		long := "package main // " + strings.Repeat("x", 300)
		assert.NoError(t, os.WriteFile(path, []byte(long+"\n"), 0o644))

		e := view.NewEditor(tmp)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "buffer_picker", m.PickerAction(ui.BufferPicker),
			[]command.KeyEvent{command.Char('b')},
		)

		m = resize(m, 120, 30)
		m = sendKey(m, 'b')
		for _, ch := range "long" {
			m = sendKey(m, ch)
		}
		_ = m.View().Content // render wide, populating the span cache

		const narrow = 60
		m = resize(m, narrow, 30)
		out := stripANSI(m.View().Content)
		for line := range strings.SplitSeq(out, "\n") {
			assert.LessOrEqual(t, len([]rune(line)), narrow)
		}
	})

	t.Run("wheel scrolls the preview and pins the bottom", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		var b strings.Builder
		for i := range 80 {
			_, _ = fmt.Fprintf(&b, "LINE-%02d\n", i)
		}
		path := filepath.Join(tmp, "big.txt")
		assert.NoError(t, os.WriteFile(path, []byte(b.String()), 0o644))

		// file picker over a fresh editor: the scratch buffer behind the
		// overlay is empty, so only the preview pane shows the LINE-NN lines
		e := view.NewEditor(tmp)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker", m.PickerAction(ui.FilePickerInDir(tmp)),
			[]command.KeyEvent{command.Char('p')},
		)
		m = resize(m, 120, 30)
		m = sendKey(m, 'p')

		// X=90 is in the preview pane (list is on the left half)
		wheel := func(dir tea.MouseButton, n int) {
			for range n {
				m2, _ := m.Update(tea.MouseWheelMsg{X: 90, Y: 10, Button: dir})
				m = m2.(ui.Model)
			}
		}

		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "LINE-00")
		assert.NotContains(t, out, "LINE-79")

		// scroll down hard: the last line is pinned to the bottom of the pane,
		// the top has scrolled away, and content has not run off the top
		wheel(tea.MouseWheelDown, 40)
		out = stripANSI(m.View().Content)
		assert.NotContains(t, out, "LINE-00")
		assert.Contains(t, out, "LINE-79")

		// scrolling back the same amount restores the top (no runaway offset)
		wheel(tea.MouseWheelUp, 40)
		out = stripANSI(m.View().Content)
		assert.Contains(t, out, "LINE-00")
		assert.NotContains(t, out, "LINE-79")
	})
}
