package ui_test

import (
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

func TestFileExplorer(t *testing.T) {
	t.Run("lists dirs and files", func(t *testing.T) {
		dir := t.TempDir()
		assert.NoError(t, os.Mkdir(filepath.Join(dir, "sub"), 0o755))
		assert.NoError(t, os.WriteFile(
			filepath.Join(dir, "alpha.txt"), []byte("x"), 0o644,
		))
		out := stripANSI(explorerModel(t, dir).View().Content)
		assert.Contains(t, out, "alpha.txt")
		assert.Contains(t, out, "sub/")
		assert.Contains(t, out, "../")
	})

	t.Run("accepts a file and opens it", func(t *testing.T) {
		dir := t.TempDir()
		assert.NoError(t, os.WriteFile(
			filepath.Join(dir, "alpha.txt"), []byte("ALPHACONTENT"), 0o644,
		))
		m := explorerModel(t, dir)
		for _, ch := range "alpha" {
			m = sendKey(m, ch)
		}
		m = sendSpecial(m, tea.KeyEnter)
		assert.Contains(t, stripANSI(m.View().Content), "ALPHACONTENT")
	})

	t.Run("accepting a binary file shows an error", func(t *testing.T) {
		dir := t.TempDir()
		assert.NoError(t, os.WriteFile(
			filepath.Join(dir, "alpha.bin"), []byte("bin\x00ary"), 0o644,
		))
		m := explorerModel(t, dir)
		for _, ch := range "alpha" {
			m = sendKey(m, ch)
		}
		m = sendSpecial(m, tea.KeyEnter)
		// the error must render on this very frame, with no further key
		// press needed to flush the pending status message
		assert.Contains(t, stripANSI(m.View().Content), "error")
	})

	t.Run("dir preview lists contents", func(t *testing.T) {
		dir := t.TempDir()
		assert.NoError(t, os.Mkdir(filepath.Join(dir, "sub"), 0o755))
		assert.NoError(t, os.WriteFile(
			filepath.Join(dir, "sub", "inner.txt"), []byte("y"), 0o644,
		))
		m := explorerModel(t, dir)
		for _, ch := range "sub" {
			m = sendKey(m, ch)
		}
		// "sub/" is now selected; its preview pane lists the directory contents
		assert.Contains(t, stripANSI(m.View().Content), "inner.txt")
	})

	t.Run("navigates into a directory", func(t *testing.T) {
		dir := t.TempDir()
		assert.NoError(t, os.Mkdir(filepath.Join(dir, "sub"), 0o755))
		assert.NoError(t, os.WriteFile(
			filepath.Join(dir, "sub", "inner.txt"), []byte("y"), 0o644,
		))
		m := explorerModel(t, dir)
		for _, ch := range "sub" {
			m = sendKey(m, ch)
		}
		m = sendSpecial(m, tea.KeyEnter)
		// the explorer is now rooted in sub, listing its file
		assert.Contains(t, stripANSI(m.View().Content), "inner.txt")
	})

	t.Run("flattens single-child dirs", func(t *testing.T) {
		dir := t.TempDir()
		nested := filepath.Join(dir, "one", "two")
		assert.NoError(t, os.MkdirAll(nested, 0o755))
		assert.NoError(t, os.WriteFile(
			filepath.Join(nested, "inner.txt"), []byte("y"), 0o644,
		))

		m := explorerModel(t, dir)
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "one/two/")

		for _, ch := range "one" {
			m = sendKey(m, ch)
		}
		m = sendSpecial(m, tea.KeyEnter)
		assert.Contains(t, stripANSI(m.View().Content), "inner.txt")
	})

	t.Run("shows hidden files by default", func(t *testing.T) {
		dir := t.TempDir()
		assert.NoError(t, os.WriteFile(
			filepath.Join(dir, ".hidden.txt"), []byte("x"), 0o644,
		))

		out := stripANSI(explorerModel(t, dir).View().Content)

		assert.Contains(t, out, ".hidden.txt")
	})

	t.Run("hides hidden files when configured", func(t *testing.T) {
		dir := t.TempDir()
		assert.NoError(t, os.WriteFile(
			filepath.Join(dir, ".hidden.txt"), []byte("x"), 0o644,
		))
		opts := files.DefaultFileExplorerOptions()
		opts.Hidden = true

		out := stripANSI(explorerModel(t, dir, opts).View().Content)

		assert.NotContains(t, out, ".hidden.txt")
	})

	t.Run("shows ignored files by default", func(t *testing.T) {
		dir := t.TempDir()
		assert.NoError(t, os.WriteFile(
			filepath.Join(dir, ".ignore"), []byte("ignored.txt\n"), 0o644,
		))
		assert.NoError(t, os.WriteFile(
			filepath.Join(dir, "ignored.txt"), []byte("x"), 0o644,
		))

		out := stripANSI(explorerModel(t, dir).View().Content)

		assert.Contains(t, out, "ignored.txt")
	})

	t.Run("respects ignore when configured", func(t *testing.T) {
		dir := t.TempDir()
		assert.NoError(t, os.WriteFile(
			filepath.Join(dir, ".ignore"), []byte("ignored.txt\n"), 0o644,
		))
		assert.NoError(t, os.WriteFile(
			filepath.Join(dir, "ignored.txt"), []byte("x"), 0o644,
		))
		opts := files.DefaultFileExplorerOptions()
		opts.Ignore = true

		out := stripANSI(explorerModel(t, dir, opts).View().Content)

		assert.NotContains(t, out, "ignored.txt")
	})

	t.Run("disables directory flattening", func(t *testing.T) {
		dir := t.TempDir()
		nested := filepath.Join(dir, "one", "two")
		assert.NoError(t, os.MkdirAll(nested, 0o755))
		assert.NoError(t, os.WriteFile(
			filepath.Join(nested, "inner.txt"), []byte("y"), 0o644,
		))
		opts := files.DefaultFileExplorerOptions()
		opts.FlattenDirs = false

		out := stripANSI(explorerModel(t, dir, opts).View().Content)

		assert.Contains(t, out, "one/")
		assert.NotContains(t, out, "one/two/")
	})

	t.Run("follows symlinked directories", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "target", "child")
		assert.NoError(t, os.MkdirAll(target, 0o755))
		assert.NoError(t, os.WriteFile(
			filepath.Join(target, "inner.txt"), []byte("y"), 0o644,
		))
		link := filepath.Join(dir, "linked")
		if err := os.Symlink(filepath.Join(dir, "target"), link); err != nil {
			t.Skip("symlink unavailable")
		}
		opts := files.DefaultFileExplorerOptions()
		opts.FollowSymlinks = true

		out := stripANSI(explorerModel(t, dir, opts).View().Content)

		assert.Contains(t, out, "linked/child/")
	})
}

func TestFileExplorerInPaneDir(t *testing.T) {
	t.Run("document path", func(t *testing.T) {
		dir := t.TempDir()
		sub := filepath.Join(dir, "nested")
		assert.NoError(t, os.Mkdir(sub, 0o755))
		path := filepath.Join(sub, "sibling.txt")
		assert.NoError(t, os.WriteFile(path, []byte("x"), 0o644))

		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "explorer_buf",
			m.PickerAction(bufferDirExplorer),
			[]command.KeyEvent{char('e')},
		)
		m = resize(m, 100, 30)
		m = sendKey(m, 'e')
		assert.Contains(t, stripANSI(m.View().Content), "sibling.txt")
	})

	t.Run("image path", func(t *testing.T) {
		dir := t.TempDir()
		sub := filepath.Join(dir, "images")
		assert.NoError(t, os.Mkdir(sub, 0o755))
		assert.NoError(t, os.WriteFile(
			filepath.Join(sub, "sibling.txt"), []byte("x"), 0o644,
		))

		e := view.NewEditor(dir)
		openRenderImagePane(t, e, writeRenderImage(t, sub, 4, 4, nil))
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindTestAction(bindTestActionArgs{
			km:   km,
			mode: "IMG",
			name: "explorer_pane",
			fn:   m.PickerAction(bufferDirExplorer),
			seqs: [][]command.KeyEvent{{char('e')}},
		})

		m = resize(m, 100, 30)
		m = sendKey(m, 'e')

		assert.Contains(t, stripANSI(m.View().Content), "sibling.txt")
	})

	t.Run("scratch falls back to cwd", func(t *testing.T) {
		dir := t.TempDir()
		assert.NoError(t, os.WriteFile(
			filepath.Join(dir, "cwd.txt"), []byte("x"), 0o644,
		))
		e := view.NewEditor(dir)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "explorer_buf",
			m.PickerAction(bufferDirExplorer),
			[]command.KeyEvent{char('e')},
		)

		m = resize(m, 100, 30)
		m = sendKey(m, 'e')

		assert.Contains(t, stripANSI(m.View().Content), "cwd.txt")
	})
}

// explorerModel opens a FileExplorer rooted at dir (the editor's cwd) and
// returns the mounted model after one render-sized resize
func explorerModel(
	t *testing.T, dir string, opts ...files.FileExplorerOptions,
) ui.Model {
	t.Helper()
	cfg := files.DefaultFileExplorerOptions()
	if len(opts) > 0 {
		cfg = opts[0]
	}
	e := view.NewEditor(dir)
	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(
		km, "explorer",
		m.PickerAction(func(e *view.Editor) *ui.Picker {
			return files.NewFileExplorer(e, cfg)
		}),
		[]command.KeyEvent{char('e')},
	)
	m = resize(m, 100, 30)
	return sendKey(m, 'e')
}
