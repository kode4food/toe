package ui_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

// explorerModel opens a FileExplorer rooted at dir (the editor's cwd) and
// returns the mounted model after one render-sized resize
func explorerModel(t *testing.T, dir string) ui.Model {
	t.Helper()
	e := view.NewEditor(dir)
	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(
		km, "explorer", m.PickerAction(ui.FileExplorer),
		[]command.KeyEvent{command.Char('e')},
	)
	m = resize(m, 100, 30)
	return sendKey(m, 'e')
}

// paletteModel registers a probe command that switches to insert mode, then
// binds 'p' to open the command palette. The editor is returned so the test
// can observe the probe command's effect after it is accepted
func paletteModel(t *testing.T) (ui.Model, *view.Editor) {
	t.Helper()
	e := view.NewEditor(t.TempDir())
	km := command.NewKeymaps()
	m := ui.New(e, km)
	km.Register("palette_probe", command.Command{
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			e.SetMode(view.ModeInsert)
			return command.Result{}
		},
		Aliases: []string{"palette_probe"},
		Modes:   []string{"NOR"},
	})
	bindNormalTestAction(
		km, "open_palette",
		m.PickerAction(func(ed *view.Editor) *ui.Picker {
			return ui.CommandPalettePicker(ed, km)
		}),
		[]command.KeyEvent{command.Char('p')},
	)
	m = resize(m, 100, 30)
	return sendKey(m, 'p'), e
}

// jumplistModel opens a file, records a jump to the TARGET line, and returns a
// model with the jumplist picker open plus the editor and recorded anchor
func jumplistModel(t *testing.T) (ui.Model, *view.Editor, int) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	assert.NoError(t, os.WriteFile(path, []byte("l0\nl1\nTARGET\n"), 0o644))

	e := view.NewEditor(dir)
	v, err := e.OpenFile(path)
	assert.NoError(t, err)
	const anchor = 6 // start of "TARGET"
	v.PushJump(v.DocID(), anchor)

	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(
		km, "jumplist", m.PickerAction(ui.JumplistPicker),
		[]command.KeyEvent{command.Char('j')},
	)
	m = resize(m, 120, 30)
	return sendKey(m, 'j'), e, anchor
}

// globalSearchModel writes two files, opens the global-search picker, and types
// the query. openPickerAndFeed drains the dynamic source's async feed per key
func globalSearchModel(t *testing.T, query string) (ui.Model, *view.Editor) {
	t.Helper()
	dir := t.TempDir()
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "a.txt"), []byte("alpha\nfindme here\n"), 0o644,
	))
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "b.txt"), []byte("beta\n"), 0o644,
	))

	e := view.NewEditor(dir)
	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(
		km, "global_search", m.GlobalSearchAction(),
		[]command.KeyEvent{command.Char('s')},
	)
	m = resize(m, 120, 30)
	m = openPickerAndFeed(m, 's')
	for _, ch := range query {
		m = openPickerAndFeed(m, ch)
	}
	return m, e
}
