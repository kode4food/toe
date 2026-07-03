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
	viewconfig "github.com/kode4food/toe/internal/view/config"
)

func TestConfigCommands(t *testing.T) {
	t.Run("config-open", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", root)
		e := view.NewEditor(t.TempDir())
		m := runTypable(newTestModel(t, e), "config-open")

		_, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		doc, ok := e.FocusedDocument()

		assert.True(t, ok)
		assert.Equal(t,
			filepath.Join(root, loader.DirName, "config.toml"), doc.Path())
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
			newTestModel(t, e),
			"config-open-workspace",
		)

		_, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		doc, ok := e.FocusedDocument()

		assert.True(t, ok)
		assert.Equal(t,
			filepath.Join(work, loader.WorkspaceDirName, "config.toml"),
			doc.Path())
	})

	t.Run("log-open", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CACHE_HOME", root)
		e := view.NewEditor(t.TempDir())
		m := runTypable(newTestModel(t, e), "log-open")

		_, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		doc, ok := e.FocusedDocument()

		assert.True(t, ok)
		assert.Equal(t,
			filepath.Join(root, loader.DirName, loader.LogFileName),
			doc.Path())
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

		_ = runTypable(newTestModel(t, e), "workspace-trust")

		path := filepath.Join(dataRoot, loader.DirName, "trusted_workspaces")
		data, err := os.ReadFile(path)
		assert.NoError(t, err)
		assert.Equal(t, work+"\n", string(data))
	})

	t.Run("set", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())

		_ = runTypable(
			newTestModel(t, e),
			"set editor.text-width 72",
		)

		assert.Equal(t, 72, *e.Options().TextWidth)
	})

	t.Run("set: quoted spaces", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())

		_ = runTypable(
			newTestModel(t, e),
			"set editor.soft-wrap.wrap-indicator '» '",
		)

		assert.Equal(t, "» ", *e.Options().SoftWrap.WrapIndicator)
	})

	t.Run("set: arrays", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		m := newTestModel(t, e)

		_ = runTypable(m, "set editor.rulers [80, 120]")
		_ = runTypable(m, `set editor.shell ["bash", "--norc", "-c"]`)

		assert.Equal(t, []int{80, 120}, e.Options().Rulers)
		assert.Equal(t, []string{"bash", "--norc", "-c"}, e.Options().Shell)
	})

	t.Run("toggle", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())

		_ = runTypable(
			newTestModel(t, e),
			"toggle editor.soft-wrap.enable",
		)

		assert.True(t, *e.Options().SoftWrap.Enable)
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
		e.Options().TextWidth = new(80)

		_ = runTypable(newTestModel(t, e), "config-reload")

		assert.Equal(t, 72, *e.Options().TextWidth)
	})

	t.Run("theme: set", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		e := view.NewEditor(t.TempDir())

		_ = runTypable(newTestModel(t, e), "theme latte")

		assert.Equal(t, "latte", e.Options().Theme)
	})

	t.Run("theme: unsupported theme", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		e := view.NewEditor(t.TempDir())

		m := resize(newTestModel(t, e), 80, 24)
		m = runTypable(m, "theme bad")

		assert.NotEqual(t, "bad", e.Options().Theme)
		assert.Contains(t, m.View().Content, "theme not found")
	})

	t.Run("theme: RGB without true color", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "")
		t.Setenv("WSL_DISTRO_NAME", "")
		e := view.NewEditor(t.TempDir())
		e.Options().Theme = "latte"
		m := resize(newTestModel(t, e), 80, 24)

		m = runTypable(m, "theme mocha")

		assert.NotEqual(t, "mocha", e.Options().Theme)
		assert.Equal(t, "latte", e.Options().Theme)
		assert.Contains(t, m.View().Content, "theme requires true color")
	})

	t.Run("theme: RGB with true color", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		t.Setenv("WSL_DISTRO_NAME", "")
		e := view.NewEditor(t.TempDir())

		_ = runTypable(newTestModel(t, e), "theme mocha")

		assert.Equal(t, "mocha", e.Options().Theme)
	})

	t.Run("theme: default alias", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv("COLORTERM", "truecolor")
		e := view.NewEditor(t.TempDir())

		_ = runTypable(newTestModel(t, e), "theme default")

		assert.Equal(t, "mocha", e.Options().Theme)
	})

	t.Run("theme: reports active", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		e := view.NewEditor(t.TempDir())
		m := resize(newTestModel(t, e), 80, 24)

		m = runTypable(m, "theme")

		assert.Contains(t, m.View().Content, "mocha")
	})
}

func runTypable(m ui.Model, cmd string) ui.Model {
	return m.ExecTypable(cmd)
}

func newTestModel(t *testing.T, e *view.Editor) ui.Model {
	t.Helper()
	km := command.NewKeymaps()
	m := ui.New(e, km)
	reg, err := defaults.RegisterDefaults(m, km)
	assert.NoError(t, err)
	e.SetConfigReload(func() error {
		raw, _ := viewconfig.LoadRawConfigForDir(e.Cwd())
		if raw == nil {
			raw = map[string]any{}
		}
		return reg.ApplyTOML(e, raw)
	})
	return m
}
