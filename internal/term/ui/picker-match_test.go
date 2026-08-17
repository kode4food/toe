package ui_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/builtin/files"
	"github.com/kode4food/toe/internal/term/command"

	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

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

type sectionPickerSource struct {
	ui.PickerBase
	rows int
}

func (s sectionPickerSource) Load(*view.Editor) ui.PickerLoad {
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
			SortKey: name,
			Group:   i / (s.rows / 2),
		}))
	}
	return ui.PickerLoad{Items: items, Stop: func() {}}
}

func (sectionPickerSource) Accept(
	*view.Editor, *ui.PickerItem, ui.PickerAcceptAction,
) {
}

func (sectionPickerSource) SkipPreview() {}

func sectionPickerModel(t testing.TB, rows int) ui.Model {
	t.Helper()
	src := sectionPickerSource{
		PickerBase: ui.PickerBase{
			Ident: "sections",
			Label: "Sections",
			Cols:  []string{"name"},
		},
		rows: rows,
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
