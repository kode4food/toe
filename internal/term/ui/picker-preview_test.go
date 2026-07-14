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

	"github.com/kode4food/toe/internal/term/builtin/files"
	"github.com/kode4food/toe/internal/term/command"

	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

// pathPickerSource is a minimal PickerSource that returns one item with a
// Location.Target.Path set to the given path
type pathPickerSource struct{ path string }

func (p *pathPickerSource) Title() string {
	return "test"
}

func (p *pathPickerSource) Columns() []string {
	return []string{"name"}
}

func (p *pathPickerSource) MatchColumn() int {
	return 0
}

func (p *pathPickerSource) ColumnProportions() []int {
	return []int{1}
}

func (p *pathPickerSource) Accept(
	*view.Editor, ui.PickerItem, ui.PickerAcceptAction,
) {
}

func (p *pathPickerSource) Load(
	*view.Editor,
) ([]ui.PickerItem, <-chan ui.PickerItem, ui.StopFunc) {
	items := []ui.PickerItem{{
		Display:  "item",
		Columns:  []string{"item"},
		SortKey:  "item",
		Location: ui.PickerLocation{Target: ui.PickerTarget{Path: p.path}},
	}}
	return items, nil, func() {}
}

func (p *pathPickerSource) Match(_ string, _ ui.PickerItem) (int, []int, bool) {
	return 0, nil, true
}

const (
	testPickerPreviewWidth   = 100
	narrowPickerPreviewWidth = 60
)

func TestPickerPreview(t *testing.T) {
	t.Run("keeps themed short rows", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		path := filepath.Join(tmp, "note.txt")
		err := os.WriteFile(path, []byte("plain\n"), 0o644)
		assert.NoError(t, err)

		e := view.NewEditor(tmp)
		e.Options().Theme = "mocha"
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker",
			m.PickerAction(files.NewFilePickerInDir(tmp)),
			[]command.KeyEvent{char('p')},
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
		e.Options().Theme = "mocha"
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker",
			m.PickerAction(files.NewFilePickerInDir(tmp)),
			[]command.KeyEvent{char('p')},
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
		e.Options().Theme = "mocha"
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker",
			m.PickerAction(files.NewFilePickerInDir(tmp)),
			[]command.KeyEvent{char('p')},
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
			m.PickerAction(files.NewFilePickerInDir(tmp)),
			[]command.KeyEvent{char('p')},
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
			m.PickerAction(files.NewFilePickerInDir(tmp)),
			[]command.KeyEvent{char('p')},
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
		text := "    " + strings.Repeat("word ", 20)
		path := filepath.Join(tmp, "notes.md")
		err := os.WriteFile(path, []byte(text), 0o644)
		assert.NoError(t, err)
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())

		e := view.NewEditor(tmp)
		e.Options().SoftWrap.Enable = new(true)
		e.Options().SoftWrap.WrapIndicator = new("» ")
		e.Options().SoftWrap.MaxIndentRetain = new(8)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker",
			m.PickerAction(files.NewFilePickerInDir(tmp)),
			[]command.KeyEvent{char('p')},
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
			m.PickerAction(files.NewFilePickerInDir(tmp)),
			[]command.KeyEvent{char('p')},
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
			m.PickerAction(files.NewFilePickerInDir(tmp)),
			[]command.KeyEvent{char('p')},
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
		e.Options().Theme = "mocha"
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "buffer_picker", m.PickerAction(bufferPicker),
			[]command.KeyEvent{char('b')},
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
		// path could not produce a syntax foreground under the highlight. The
		// two SGR codes may be separate escapes or combined — check both are
		// present on a single line containing "package"
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
			km, "buffer_picker", m.PickerAction(bufferPicker),
			[]command.KeyEvent{char('b')},
		)

		m = resize(m, 120, 30)
		m = sendKey(m, 'b')
		for _, ch := range "long" {
			m = sendKey(m, ch)
		}
		_ = m.View().Content // render wide, populating the preview cache

		m = resize(m, narrowPickerPreviewWidth, 30)
		out := stripANSI(m.View().Content)
		for line := range strings.SplitSeq(out, "\n") {
			assert.LessOrEqual(t, len([]rune(line)), narrowPickerPreviewWidth)
		}
	})

	t.Run("wheel pins bottom", func(t *testing.T) {
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
			km, "file_picker", m.PickerAction(files.NewFilePickerInDir(tmp)),
			[]command.KeyEvent{char('p')},
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

func TestPickerPreviewPlaceholders(t *testing.T) {
	t.Run("binary file shows placeholder", func(t *testing.T) {
		tmp := t.TempDir()
		assert.NoError(t, os.WriteFile(
			filepath.Join(tmp, "binary.bin"),
			[]byte{0x00, 0x01, 0x02, 0x03},
			0o644,
		))
		e := view.NewEditor(tmp)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker", m.PickerAction(files.NewFilePickerInDir(tmp)),
			[]command.KeyEvent{char('p')},
		)
		m = resize(m, 120, 30)
		m = sendKey(m, 'p')
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "<Binary file>")
	})

	t.Run("nonexistent path shows placeholder", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		src := &pathPickerSource{path: "/no/such/file.txt"}
		bindNormalTestAction(
			km, "custom_picker",
			m.PickerAction(func(ed *view.Editor) *ui.Picker {
				return ui.NewPicker(ed, src)
			}),
			[]command.KeyEvent{char('p')},
		)
		m = resize(m, 120, 30)
		m = sendKey(m, 'p')
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "<File not found>")
	})

	t.Run("directory path shows contents", func(t *testing.T) {
		dir := t.TempDir()
		sub := filepath.Join(dir, "subdir")
		assert.NoError(t, os.Mkdir(sub, 0o755))
		assert.NoError(t, os.WriteFile(
			filepath.Join(dir, "file.txt"), []byte("text\n"), 0o644,
		))

		e := view.NewEditor(dir)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		src := &pathPickerSource{path: dir}
		bindNormalTestAction(
			km, "custom_picker2",
			m.PickerAction(func(ed *view.Editor) *ui.Picker {
				return ui.NewPicker(ed, src)
			}),
			[]command.KeyEvent{char('q')},
		)
		m = resize(m, 120, 30)
		m = sendKey(m, 'q')
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "subdir/")
		assert.Contains(t, out, "file.txt")
	})

	t.Run("large file shows placeholder", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "huge.txt")
		f, err := os.Create(path)
		assert.NoError(t, err)
		_, err = f.Write(make([]byte, 10*1024*1024+1))
		assert.NoError(t, err)
		assert.NoError(t, f.Close())

		e := view.NewEditor(tmp)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker", m.PickerAction(files.NewFilePickerInDir(tmp)),
			[]command.KeyEvent{char('p')},
		)
		m = resize(m, 120, 30)
		m = sendKey(m, 'p')
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "<File too large to preview>")
	})

	t.Run("invalidates not found preview", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "later.txt")
		e := view.NewEditor(tmp)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		src := &pathPickerSource{path: path}
		bindNormalTestAction(
			km, "custom_picker3",
			m.PickerAction(func(ed *view.Editor) *ui.Picker {
				return ui.NewPicker(ed, src)
			}),
			[]command.KeyEvent{char('p')},
		)
		m = resize(m, 120, 30)
		m = sendKey(m, 'p')
		assert.Contains(t, stripANSI(m.View().Content), "<File not found>")

		assert.NoError(t, os.WriteFile(path, []byte("now here\n"), 0o644))
		m = sendKey(m, 'x')
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "now here")
	})
}
