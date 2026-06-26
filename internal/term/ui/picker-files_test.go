package ui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/defaults"

	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

func TestPickerTarget(t *testing.T) {
	t.Run("valid path", func(t *testing.T) {
		target := ui.PickerTarget{Path: "main.go"}
		assert.True(t, target.Valid())
	})

	t.Run("valid document id", func(t *testing.T) {
		target := ui.PickerTarget{ID: view.DocumentId(1)}
		assert.True(t, target.Valid())
	})

	t.Run("empty target", func(t *testing.T) {
		target := ui.PickerTarget{ID: view.InvalidDocumentId}
		assert.False(t, target.Valid())
	})
}

func TestPickerFiles(t *testing.T) {
	t.Run("accepts file", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "main.go")
		err := os.WriteFile(path, []byte("package main\n"), 0o644)
		assert.NoError(t, err)

		e := view.NewEditor(tmp)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker",
			m.PickerAction(ui.FilePickerInDir(tmp)),
			[]command.KeyEvent{char('p')},
		)

		m = resize(m, 100, 30)
		m = sendKey(m, 'p')
		m = sendSpecial(m, tea.KeyEnter)
		out := stripANSI(m.View().Content)
		doc, ok := e.FocusedDocument()
		if !assert.True(t, ok) {
			return
		}
		wantPath, err := filepath.EvalSymlinks(path)
		assert.NoError(t, err)
		gotPath, err := filepath.EvalSymlinks(doc.Path())
		assert.NoError(t, err)

		assert.Equal(t, wantPath, gotPath)
		assert.Contains(t, out, "package main")
		assert.NotContains(t, out, "┐┌")
		assert.NotContains(t, out, "\x1b")
	})

	t.Run("respects ignore files", func(t *testing.T) {
		tmp := t.TempDir()
		cfgRoot := t.TempDir()
		writeConfigIgnore(t, cfgRoot, "config_ignored.go\n")
		path := filepath.Join(tmp, ".gitignore")
		err := os.WriteFile(path, []byte("ignored.go\n"), 0o644)
		assert.NoError(t, err)
		path = filepath.Join(tmp, "ignored.go")
		err = os.WriteFile(path, []byte("package ignored\n"), 0o644)
		assert.NoError(t, err)
		path = filepath.Join(tmp, "visible.go")
		err = os.WriteFile(path, []byte("package visible\n"), 0o644)
		assert.NoError(t, err)
		path = filepath.Join(tmp, ".hidden.go")
		err = os.WriteFile(path, []byte("package hidden\n"), 0o644)
		assert.NoError(t, err)
		err = os.WriteFile(
			filepath.Join(tmp, "archive.zip"), []byte("PK"), 0o644,
		)
		assert.NoError(t, err)
		path = filepath.Join(tmp, "config_ignored.go")
		err = os.WriteFile(path, []byte("package c\n"), 0o644)
		assert.NoError(t, err)
		sub := filepath.Join(tmp, "sub")
		err = os.MkdirAll(sub, 0o755)
		assert.NoError(t, err)
		path = filepath.Join(sub, ".gitignore")
		err = os.WriteFile(path, []byte("nested.go\n"), 0o644)
		assert.NoError(t, err)
		path = filepath.Join(sub, "nested.go")
		err = os.WriteFile(path, []byte("package n\n"), 0o644)
		assert.NoError(t, err)
		path = filepath.Join(sub, "shown.go")
		err = os.WriteFile(path, []byte("package s\n"), 0o644)
		assert.NoError(t, err)

		e := view.NewEditor(tmp)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker",
			m.PickerAction(ui.FilePickerInDir(tmp)),
			[]command.KeyEvent{char('p')},
		)

		m = resize(m, 100, 30)
		m = sendKey(m, 'p')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "visible.go")
		assert.NotContains(t, out, "ignored.go")
		assert.NotContains(t, out, ".hidden.go")
		assert.NotContains(t, out, "archive.zip")
		assert.NotContains(t, out, "config_ignored.go")
		assert.NotContains(t, out, "nested.go")
		assert.Contains(t, out, "shown.go")
	})

	t.Run("workspace picker starts at workspace root", func(t *testing.T) {
		tmp := t.TempDir()
		work := filepath.Join(tmp, "work")
		cwd := filepath.Join(work, "src")
		err := os.MkdirAll(filepath.Join(work, ".git"), 0o755)
		assert.NoError(t, err)
		err = os.MkdirAll(cwd, 0o755)
		assert.NoError(t, err)
		err = os.WriteFile(
			filepath.Join(work, "root.go"), []byte("package root\n"), 0o644,
		)
		assert.NoError(t, err)
		err = os.WriteFile(
			filepath.Join(cwd, "child.go"), []byte("package child\n"), 0o644,
		)
		assert.NoError(t, err)

		e := view.NewEditor(cwd)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker", m.PickerAction(ui.FilePicker),
			[]command.KeyEvent{char('p')},
		)

		m = resize(m, 100, 30)
		m = sendKey(m, 'p')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "root.go")
		assert.Contains(t, out, "src/child.go")
	})

	t.Run("cwd picker stays in current directory", func(t *testing.T) {
		tmp := t.TempDir()
		work := filepath.Join(tmp, "work")
		cwd := filepath.Join(work, "src")
		err := os.MkdirAll(filepath.Join(work, ".git"), 0o755)
		assert.NoError(t, err)
		err = os.MkdirAll(cwd, 0o755)
		assert.NoError(t, err)
		err = os.WriteFile(
			filepath.Join(work, "root.go"), []byte("package root\n"), 0o644,
		)
		assert.NoError(t, err)
		err = os.WriteFile(
			filepath.Join(cwd, "child.go"), []byte("package child\n"), 0o644,
		)
		assert.NoError(t, err)

		e := view.NewEditor(cwd)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker_cwd", m.PickerAction(ui.FilePickerInCWD),
			[]command.KeyEvent{char('p')},
		)

		m = resize(m, 100, 30)
		m = sendKey(m, 'p')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "child.go")
		assert.NotContains(t, out, "root.go")
		assert.NotContains(t, out, "src/child.go")
	})

	t.Run("space capital f opens cwd picker", func(t *testing.T) {
		tmp := t.TempDir()
		work := filepath.Join(tmp, "work")
		cwd := filepath.Join(work, "src")
		err := os.MkdirAll(filepath.Join(work, ".git"), 0o755)
		assert.NoError(t, err)
		err = os.MkdirAll(cwd, 0o755)
		assert.NoError(t, err)
		err = os.WriteFile(
			filepath.Join(work, "root.go"), []byte("package root\n"), 0o644,
		)
		assert.NoError(t, err)
		err = os.WriteFile(
			filepath.Join(cwd, "child.go"), []byte("package child\n"), 0o644,
		)
		assert.NoError(t, err)

		e := view.NewEditor(cwd)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err = defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)

		m = resize(m, 100, 30)
		m = sendKey(m, ' ')
		m = sendKey(m, 'F')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "child.go")
		assert.NotContains(t, out, "root.go")
		assert.NotContains(t, out, "src/child.go")
	})

	t.Run("current directory below hidden parent", func(t *testing.T) {
		tmp := t.TempDir()
		cwd := filepath.Join(tmp, ".config", "toe")
		err := os.MkdirAll(cwd, 0o755)
		assert.NoError(t, err)
		err = os.WriteFile(
			filepath.Join(cwd, "config.toml"), []byte("[editor]\n"), 0o644,
		)
		assert.NoError(t, err)
		err = os.WriteFile(
			filepath.Join(cwd, "languages.toml"), []byte("[[language]]\n"),
			0o644,
		)
		assert.NoError(t, err)

		e := view.NewEditor(cwd)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker_cwd", m.PickerAction(ui.FilePickerInCWD),
			[]command.KeyEvent{char('p')},
		)

		m = resize(m, 100, 30)
		m = sendKey(m, 'p')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "config.toml")
		assert.Contains(t, out, "languages.toml")
	})

	t.Run("space f below hidden parent", func(t *testing.T) {
		tmp := t.TempDir()
		cwd := filepath.Join(tmp, ".config", "toe")
		err := os.MkdirAll(cwd, 0o755)
		assert.NoError(t, err)
		err = os.WriteFile(
			filepath.Join(cwd, "config.toml"), []byte("[editor]\n"), 0o644,
		)
		assert.NoError(t, err)
		err = os.WriteFile(
			filepath.Join(cwd, "languages.toml"), []byte("[[language]]\n"),
			0o644,
		)
		assert.NoError(t, err)

		e := view.NewEditor(cwd)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		_, err = defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)

		m = resize(m, 100, 30)
		m = sendKey(m, ' ')
		m = sendKey(m, 'f')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "config.toml")
		assert.Contains(t, out, "languages.toml")
	})

	t.Run("symlinked current directory", func(t *testing.T) {
		tmp := t.TempDir()
		target := filepath.Join(tmp, ".dotfiles", "config", "toe")
		err := os.MkdirAll(target, 0o755)
		assert.NoError(t, err)
		err = os.WriteFile(
			filepath.Join(target, "config.toml"), []byte("[editor]\n"), 0o644,
		)
		assert.NoError(t, err)
		err = os.WriteFile(
			filepath.Join(target, "languages.toml"), []byte("[[language]]\n"),
			0o644,
		)
		assert.NoError(t, err)
		link := filepath.Join(tmp, ".config", "toe")
		err = os.MkdirAll(filepath.Dir(link), 0o755)
		assert.NoError(t, err)
		err = os.Symlink(target, link)
		assert.NoError(t, err)

		e := view.NewEditor(link)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker_cwd", m.PickerAction(ui.FilePickerInCWD),
			[]command.KeyEvent{char('p')},
		)

		m = resize(m, 100, 30)
		m = sendKey(m, 'p')
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "config.toml")
		assert.Contains(t, out, "languages.toml")
	})

	t.Run("symlinked initial directory", func(t *testing.T) {
		tmp := t.TempDir()
		target := filepath.Join(tmp, ".dotfiles", "config", "toe")
		err := os.MkdirAll(target, 0o755)
		assert.NoError(t, err)
		err = os.WriteFile(
			filepath.Join(target, "config.toml"), []byte("[editor]\n"), 0o644,
		)
		assert.NoError(t, err)
		err = os.WriteFile(
			filepath.Join(target, "languages.toml"), []byte("[[language]]\n"),
			0o644,
		)
		assert.NoError(t, err)
		link := filepath.Join(tmp, ".config", "toe")
		err = os.MkdirAll(filepath.Dir(link), 0o755)
		assert.NoError(t, err)
		err = os.Symlink(target, link)
		assert.NoError(t, err)

		e := view.NewEditor(link)
		m := ui.New(e, command.NewKeymaps()).WithInitialPicker(
			ui.FilePickerInDir(link),
		)

		m = resize(m, 100, 30)
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "config.toml")
		assert.Contains(t, out, "languages.toml")
	})

	t.Run("broken symlink root falls back", func(t *testing.T) {
		tmp := t.TempDir()
		link := filepath.Join(tmp, "missing")
		err := os.Symlink(filepath.Join(tmp, "target"), link)
		assert.NoError(t, err)

		e := view.NewEditor(tmp)
		m := ui.New(e, command.NewKeymaps()).WithInitialPicker(
			ui.FilePickerInDir(link),
		)

		m = resize(m, 100, 30)
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "0/0")
	})

	t.Run("file symlink root falls back", func(t *testing.T) {
		tmp := t.TempDir()
		target := filepath.Join(tmp, "target.go")
		err := os.WriteFile(target, []byte("package target\n"), 0o644)
		assert.NoError(t, err)
		link := filepath.Join(tmp, "link.go")
		err = os.Symlink(target, link)
		assert.NoError(t, err)

		e := view.NewEditor(tmp)
		m := ui.New(e, command.NewKeymaps()).WithInitialPicker(
			ui.FilePickerInDir(link),
		)

		m = resize(m, 100, 30)
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "0/0")
	})

	t.Run("file feed continues after first batch", func(t *testing.T) {
		tmp := t.TempDir()
		for i := range 12 {
			name := filepath.Join(tmp, "file-"+string(rune('a'+i))+".go")
			err := os.WriteFile(name, []byte("package main\n"), 0o644)
			assert.NoError(t, err)
		}

		e := view.NewEditor(tmp)
		m := resize(ui.New(e, command.NewKeymaps()), 100, 6)
		p := ui.FilePickerInDir(tmp)(e)

		out := stripANSI(m.View().Content)

		assert.NotNil(t, p)
		assert.NotEmpty(t, out)
	})

	t.Run("dynamic feed displays first open batch", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		src := newControlledDynamicSource()
		bindNormalTestAction(
			km, "dynamic_picker",
			m.PickerAction(func(e *view.Editor) *ui.Picker {
				return ui.NewPicker(e, src)
			}),
			[]command.KeyEvent{char('p')},
		)

		m = resize(m, 80, 20)
		m = sendKey(m, 'p')
		next, cmd := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
		m = next.(ui.Model)
		assert.NotNil(t, cmd)

		msg := runTestCmd(t, cmd)
		next, cmd = m.Update(msg)
		m = next.(ui.Model)
		assert.NotNil(t, cmd)

		src.ch <- ui.PickerItem{Display: "alpha.go:1"}
		msg = runTestCmd(t, cmd)
		next, _ = m.Update(msg)
		m = next.(ui.Model)
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "alpha.go:1")
	})

	t.Run("buffer picker highlights hidden cursor", func(t *testing.T) {
		tmp := t.TempDir()
		alpha := filepath.Join(tmp, "alpha.go")
		beta := filepath.Join(tmp, "beta.go")
		lines := make([]string, 30)
		for i := range lines {
			lines[i] = "line"
		}
		lines[0] = "line-00 start"
		lines[18] = "line-18 target"
		err := os.WriteFile(alpha, []byte(strings.Join(lines, "\n")), 0o644)
		assert.NoError(t, err)
		err = os.WriteFile(beta, []byte("package beta\n"), 0o644)
		assert.NoError(t, err)

		e := view.NewEditor(tmp)
		v, err := e.OpenFile(alpha)
		assert.NoError(t, err)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		pos, err := doc.Text().LineToChar(18)
		assert.NoError(t, err)
		doc.SetSelectionFor(v.ID(), core.PointSelection(pos))

		_, err = e.OpenFile(beta)
		assert.NoError(t, err)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "buffer_picker", m.PickerAction(ui.BufferPicker),
			[]command.KeyEvent{char('b')},
		)

		m = resize(m, 100, 16)
		m = sendKey(m, 'b')
		for _, ch := range "alpha" {
			m = sendKey(m, ch)
		}
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "line-18 target")
		assert.NotContains(t, out, "line-00 start")
	})
}

func runTestCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(time.Second):
		t.Fatal("command did not return")
		return nil
	}
}

type controlledDynamicSource struct {
	ch    chan ui.PickerItem
	query string
}

func newControlledDynamicSource() *controlledDynamicSource {
	return &controlledDynamicSource{ch: make(chan ui.PickerItem, 1)}
}

func (c *controlledDynamicSource) Title() string { return "Dynamic" }

func (c *controlledDynamicSource) Columns() []string { return []string{"path"} }

func (c *controlledDynamicSource) Primary() int { return 0 }

func (c *controlledDynamicSource) Search(query string) { c.query = query }

func (c *controlledDynamicSource) Load(
	_ *view.Editor,
) ([]ui.PickerItem, <-chan ui.PickerItem, ui.StopFunc) {
	if c.query == "" {
		return nil, nil, func() {}
	}
	return nil, c.ch, func() {}
}

func (c *controlledDynamicSource) Accept(_ *view.Editor, _ ui.PickerItem) {}
