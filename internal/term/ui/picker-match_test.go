package ui_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/builtin/files"
	"github.com/kode4food/toe/internal/term/command"

	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

type sectionPickerSource struct {
	ui.PickerBase
	rows int
}

func (s sectionPickerSource) Load() ui.PickerLoad {
	var slab ui.PickerItemSlab
	items := []*ui.PickerItem{
		slab.Add(ui.PickerItem{Display: "First Group", Section: true}),
		slab.Add(ui.PickerItem{
			Display: "Second Group", Group: 1, Section: true,
		}),
	}
	for i := range s.rows {
		name := fmt.Sprintf("row-%02d", i)
		items = append(items, slab.Add(ui.PickerItem{
			Display: name,
			Content: name,
			Group:   i / (s.rows / 2),
		}))
	}
	return ui.PickerLoad{Items: items, Stop: func() {}}
}

func (sectionPickerSource) Accept(*ui.PickerItem, ui.PickerAcceptAction) {
}

func (sectionPickerSource) SkipPreview() {}

func TestPickerMatch(t *testing.T) {
	t.Run("file picker page keys", func(t *testing.T) {
		tmp := t.TempDir()
		for i := range 30 {
			name := fmt.Sprintf("file-%02d.go", i)
			err := os.WriteFile(
				filepath.Join(tmp, name), []byte("package p\n"), 0o644,
			)
			assert.NoError(t, err)
		}

		e := view.NewEditor(tmp)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker",
			m.PickerAction(files.NewFilePickerInDir(tmp)),
			[]command.KeyEvent{char('p')},
		)

		m = resize(m, 70, 20)
		m = sendKeyAndFeed(m, 'p')
		_ = m.View()

		m = sendSpecialText(m, tea.KeyPgDown, "pgdown")
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, " > \U000f07d3 file-13.go")

		m = sendSpecialText(m, tea.KeyPgUp, "pgup")
		out = stripANSI(m.View().Content)
		assert.Contains(t, out, " > \U000f07d3 file-00.go")
	})

	t.Run("buffer picker filters by field", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "alpha.go")
		err := os.WriteFile(path, []byte("package alpha\n"), 0o644)
		assert.NoError(t, err)

		e := view.NewEditor(tmp)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "buffer_picker", m.PickerAction(bufferPicker),
			[]command.KeyEvent{char('b')},
		)

		m = resize(m, 100, 30)
		m = sendKey(m, 'b')
		for _, ch := range "%path alpha" {
			m = sendKey(m, ch)
		}
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "alpha.go")
		assert.NotContains(t, out, "[scratch]")
	})

	t.Run("narrowed equals rebuilt", func(t *testing.T) {
		narrowed := stripANSI(typeQuery(narrowPicker(t), "ale").View().Content)
		m := typeQuery(narrowPicker(t), "alex")
		rebuilt := stripANSI(sendSpecial(m, tea.KeyBackspace).View().Content)
		assert.Equal(t, rebuilt, narrowed)
		assert.Contains(t, narrowed, "ale.go")
		assert.NotContains(t, narrowed, "main.go")
	})

	t.Run("column query is not narrowed", func(t *testing.T) {
		m := typeQuery(narrowPicker(t), "%path ale")
		direct := stripANSI(m.View().Content)
		m = typeQuery(narrowPicker(t), "%path alex")
		back := stripANSI(sendSpecial(m, tea.KeyBackspace).View().Content)
		assert.Equal(t, back, direct)
		assert.Contains(t, direct, "ale.go")
	})

	t.Run("clearing restores every row", func(t *testing.T) {
		full := stripANSI(narrowPicker(t).View().Content)
		m := typeQuery(narrowPicker(t), "ale")
		for range 3 {
			m = sendSpecial(m, tea.KeyBackspace)
		}
		assert.Equal(t, full, stripANSI(m.View().Content))
	})

	t.Run("ctrl+w rubs out the last word", func(t *testing.T) {
		kept := typeQuery(narrowPicker(t), "%path ")
		oneWord := stripANSI(kept.View().Content)
		m := typeQuery(narrowPicker(t), "%path ale")

		m = sendModified(m, 'w', tea.ModCtrl)

		assert.Equal(t, oneWord, stripANSI(m.View().Content))
	})

	t.Run("ctrl+w on one word restores every row", func(t *testing.T) {
		full := stripANSI(narrowPicker(t).View().Content)
		m := typeQuery(narrowPicker(t), "ale")

		m = sendModified(m, 'w', tea.ModCtrl)

		assert.Equal(t, full, stripANSI(m.View().Content))
	})

	t.Run("ctrl+w on empty query is a no-op", func(t *testing.T) {
		full := stripANSI(narrowPicker(t).View().Content)

		m := sendModified(narrowPicker(t), 'w', tea.ModCtrl)

		assert.Equal(t, full, stripANSI(m.View().Content))
	})

	t.Run("ctrl+w skips trailing spaces", func(t *testing.T) {
		kept := typeQuery(narrowPicker(t), "%path ")
		oneWord := stripANSI(kept.View().Content)
		m := typeQuery(narrowPicker(t), "%path ale  ")

		m = sendModified(m, 'w', tea.ModCtrl)

		assert.Equal(t, oneWord, stripANSI(m.View().Content))
	})
}

func TestPickerMatchPath(t *testing.T) {
	t.Run("matches the path as written", func(t *testing.T) {
		m := typeQuery(treePicker(t), "cmd/main")

		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "main.go")
		assert.NotContains(t, out, "readme.md")
	})

	t.Run("a space never matches the label", func(t *testing.T) {
		m := typeQuery(treePicker(t), " ")

		assert.NotContains(t, stripANSI(m.View().Content), "main.go")
	})

	t.Run("highlights the name in the label", func(t *testing.T) {
		m := typeQuery(treePicker(t), "main")

		row := rawLineContaining(t, m.View().Content, ".go")
		assert.Contains(t, stripANSI(row), "main.go")
		assert.NotContains(t, row, "main.go")
	})
}

func TestPickerSectionScroll(t *testing.T) {
	t.Run("paging back up reveals the first header", func(t *testing.T) {
		m := sectionPickerModel(t, 40)
		assert.Contains(t, stripANSI(m.View().Content), "First Group")

		m = sendSpecialText(m, tea.KeyPgDown, "pgdown")
		m = sendSpecialText(m, tea.KeyPgDown, "pgdown")
		assert.NotContains(t, stripANSI(m.View().Content), "First Group")

		m = sendSpecialText(m, tea.KeyPgUp, "pgup")
		m = sendSpecialText(m, tea.KeyPgUp, "pgup")

		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "First Group")
		assert.Contains(t, out, "> row-00")
	})

	t.Run("arrowing up reveals the first header", func(t *testing.T) {
		m := sectionPickerModel(t, 40)
		assert.Contains(t, stripANSI(m.View().Content), "First Group")
		for range 25 {
			m = sendSpecialText(m, tea.KeyDown, "down")
		}
		assert.NotContains(t, stripANSI(m.View().Content), "First Group")

		for range 25 {
			m = sendSpecialText(m, tea.KeyUp, "up")
		}

		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "First Group")
		assert.Contains(t, out, "> row-00")
	})

	t.Run("group header scrolls in with its row", func(t *testing.T) {
		m := sectionPickerModel(t, 40)
		assert.Contains(t, stripANSI(m.View().Content), "First Group")
		for range 25 {
			m = sendSpecialText(m, tea.KeyDown, "down")
		}
		for range 6 {
			m = sendSpecialText(m, tea.KeyUp, "up")
		}

		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "Second Group")
	})
}

func TestPickerSectionFilter(t *testing.T) {
	t.Run("spaces never match a header", func(t *testing.T) {
		m := sectionPickerModel(t, 6)

		m = typeQuery(m, "  ")

		out := stripANSI(m.View().Content)
		assert.NotContains(t, out, "First Group")
		assert.NotContains(t, out, "Second Group")
	})

	t.Run("query matching a header", func(t *testing.T) {
		m := sectionPickerModel(t, 6)

		m = typeQuery(m, "ro")

		out := stripANSI(m.View().Content)
		assert.Equal(t, 1, strings.Count(out, "First Group"))
		assert.Equal(t, 1, strings.Count(out, "Second Group"))
	})
}

func treePicker(t *testing.T) ui.Model {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("COLORTERM", "truecolor")
	tmp := t.TempDir()
	for _, rel := range []string{"cmd/toe/main.go", "readme.md"} {
		path := filepath.Join(tmp, rel)
		assert.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		assert.NoError(t, os.WriteFile(path, nil, 0o644))
	}
	e := view.NewEditor(tmp)
	e.Options().Theme = "mocha"
	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(
		km, "file_picker",
		m.PickerAction(files.NewFilePickerInDir(tmp)),
		[]command.KeyEvent{char('p')},
	)
	return sendKeyAndFeed(resize(m, 70, 20), 'p')
}

func narrowPicker(t *testing.T) ui.Model {
	t.Helper()
	m := feedPickerModel(t, []string{
		"internal/term/ale.go",
		"internal/lsp/capabilities.go",
		"internal/view/action/selection-lines.go",
		"alembic/config.toml",
		"docs/scale-guide.md",
		"cmd/toe/main.go",
	})
	return sendKeyAndFeed(m, 'p')
}

func typeQuery(m ui.Model, query string) ui.Model {
	for _, ch := range query {
		m = sendKey(m, ch)
	}
	return m
}

func sectionPickerModel(t testing.TB, rows int) ui.Model {
	t.Helper()
	src := sectionPickerSource{
		Ident: "sections",
		Label: "Sections",
		Cols:  []string{"name"},
		rows:  rows,
	}
	e := view.NewEditor(t.TempDir())
	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(
		km, "section_picker",
		m.PickerAction(func(*view.Editor) *ui.Picker {
			return ui.NewPicker(e, src)
		}),
		[]command.KeyEvent{char('p')},
	)
	return sendKeyAndFeed(resize(m, 70, 20), 'p')
}
