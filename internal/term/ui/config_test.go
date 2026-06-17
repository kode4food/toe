package ui_test

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/loader"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/defaults"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

func TestConfigCommands(t *testing.T) {
	t.Run("config-open", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", root)
		e := view.NewEditor(t.TempDir())
		m := runTypable(newTestModel(e), "config-open")

		_, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		doc, ok := e.FocusedDocument()

		assert.True(t, ok)
		assert.Equal(t,
			filepath.Join(root, loader.DirName, "config.toml"), doc.Path(),
		)
	})

	t.Run("config-open-workspace", func(t *testing.T) {
		root := t.TempDir()
		work := filepath.Join(root, "work")
		cwd := filepath.Join(work, "src")
		err := os.MkdirAll(filepath.Join(work, ".git"), 0o755)
		assert.NoError(t, err)
		err = os.MkdirAll(cwd, 0o755)
		assert.NoError(t, err)
		e := view.NewEditor(cwd)
		m := runTypable(
			newTestModel(e),
			"config-open-workspace",
		)

		_, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		doc, ok := e.FocusedDocument()

		assert.True(t, ok)
		assert.Equal(t,
			filepath.Join(work, loader.WorkspaceDirName, "config.toml"),
			doc.Path(),
		)
	})

	t.Run("log-open", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CACHE_HOME", root)
		e := view.NewEditor(t.TempDir())
		m := runTypable(newTestModel(e), "log-open")

		_, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		doc, ok := e.FocusedDocument()

		assert.True(t, ok)
		assert.Equal(t,
			filepath.Join(root, loader.DirName, loader.LogFileName),
			doc.Path(),
		)
	})

	t.Run("workspace-trust", func(t *testing.T) {
		root := t.TempDir()
		work := filepath.Join(root, "work")
		cwd := filepath.Join(work, "src")
		err := os.MkdirAll(filepath.Join(work, ".git"), 0o755)
		assert.NoError(t, err)
		err = os.MkdirAll(cwd, 0o755)
		assert.NoError(t, err)
		dataRoot := t.TempDir()
		t.Setenv("XDG_DATA_HOME", dataRoot)
		e := view.NewEditor(cwd)

		_ = runTypable(newTestModel(e), "workspace-trust")

		path := filepath.Join(dataRoot, loader.DirName, "trusted_workspaces")
		data, err := os.ReadFile(path)
		assert.NoError(t, err)
		assert.Equal(t, work+"\n", string(data))
	})

	t.Run("tutor", func(t *testing.T) {
		root := t.TempDir()
		rt := filepath.Join(root, "runtime")
		err := os.MkdirAll(rt, 0o755)
		assert.NoError(t, err)
		path := filepath.Join(rt, "tutor")
		err = os.WriteFile(path, []byte("learn"), 0o644)
		assert.NoError(t, err)
		t.Setenv(loader.RuntimeEnv, rt)
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		e := view.NewEditor(t.TempDir())
		m := runTypable(newTestModel(e), "tutor")

		_, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		doc, ok := e.FocusedDocument()

		assert.True(t, ok)
		assert.Equal(t, "", doc.Path())
		assert.Equal(t, "learn", doc.Text().String())
	})

	t.Run("set", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())

		_ = runTypable(
			newTestModel(e),
			"set editor.text-width 72",
		)

		assert.Equal(t, 72, *e.Config().Editor.TextWidth)
	})

	t.Run("set: quoted spaces", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())

		_ = runTypable(
			newTestModel(e),
			"set editor.soft-wrap.wrap-indicator '» '",
		)

		assert.Equal(t, "» ", *e.Config().Editor.SoftWrap.WrapIndicator)
	})

	t.Run("set: arrays", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		m := newTestModel(e)

		_ = runTypable(m, "set editor.rulers [80, 120]")
		_ = runTypable(m, `set editor.shell ["bash", "--norc", "-c"]`)

		assert.Equal(t, []int{80, 120}, e.Config().Rulers())
		assert.Equal(t, []string{"bash", "--norc", "-c"}, e.Config().Shell())
	})

	t.Run("toggle", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())

		_ = runTypable(
			newTestModel(e),
			"toggle editor.soft-wrap.enable",
		)

		assert.True(t, *e.Config().Editor.SoftWrap.Enable)
	})

	t.Run("config-reload", func(t *testing.T) {
		root := t.TempDir()
		dir := filepath.Join(root, loader.DirName)
		err := os.MkdirAll(dir, 0o755)
		assert.NoError(t, err)
		err = os.WriteFile(filepath.Join(dir, "config.toml"), []byte(`
[editor]
text-width = 72
`), 0o644)
		assert.NoError(t, err)
		t.Setenv("XDG_CONFIG_HOME", root)
		e := view.NewEditor(t.TempDir())
		cfg := e.Config()
		cfg.Editor.TextWidth = new(80)
		e.SetConfig(cfg)

		_ = runTypable(newTestModel(e), "config-reload")

		assert.Equal(t, 72, *e.Config().Editor.TextWidth)
	})

	t.Run("theme: set", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		e := view.NewEditor(t.TempDir())

		_ = runTypable(newTestModel(e), "theme latte")

		assert.Equal(t, "latte", e.Config().Theme.Name)
	})

	t.Run("theme: unsupported theme", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		e := view.NewEditor(t.TempDir())

		m := resize(newTestModel(e), 80, 24)
		m = runTypable(m, "theme bad")

		assert.NotEqual(t, "bad", e.Config().Theme.Name)
		assert.Contains(t, m.View().Content, "theme not found")
	})

	t.Run("theme: RGB without true color", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "")
		t.Setenv("WSL_DISTRO_NAME", "")
		e := view.NewEditor(t.TempDir())
		cfg := e.Config()
		cfg.Theme.Name = "latte"
		e.SetConfig(cfg)
		m := resize(newTestModel(e), 80, 24)

		m = runTypable(m, "theme mocha")

		assert.NotEqual(t, "mocha", e.Config().Theme.Name)
		assert.Equal(t, "latte", e.Config().Theme.Name)
		assert.Contains(t, m.View().Content, "theme requires true color")
	})

	t.Run("theme: RGB with true color", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		t.Setenv("WSL_DISTRO_NAME", "")
		e := view.NewEditor(t.TempDir())

		_ = runTypable(newTestModel(e), "theme mocha")

		assert.Equal(t, "mocha", e.Config().Theme.Name)
	})

	t.Run("theme: default alias", func(t *testing.T) {
		t.Setenv(loader.RuntimeEnv, t.TempDir())
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		e := view.NewEditor(t.TempDir())

		_ = runTypable(newTestModel(e), "theme default")

		assert.Equal(t, "mocha", e.Config().Theme.Name)
	})

	t.Run("theme: reports active", func(t *testing.T) {
		t.Setenv(loader.RuntimeEnv, t.TempDir())
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		e := view.NewEditor(t.TempDir())
		m := resize(newTestModel(e), 80, 24)

		m = runTypable(m, "theme")

		assert.Contains(t, m.View().Content, "mocha")
	})
}

func runTypable(m ui.Model, cmd string) ui.Model {
	return m.ExecTypable(cmd)
}

func newTestModel(e *view.Editor) ui.Model {
	km := command.NewKeymaps()
	m := ui.New(e, km)
	defaults.RegisterDefaults(m, km)
	return m
}
