package app_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	app "github.com/kode4food/toe/cmd/toe/internal"
	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/term/builtin"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

func newTestApp(t *testing.T) *app.App {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	assert.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o755))
	e := view.NewEditor(dir)
	km := command.NewKeymaps()
	reg, err := builtin.Register(ui.New(e, km), km)
	assert.NoError(t, err)
	return &app.App{Root: dir, Editor: e, Reg: reg}
}

func TestParseConfigFlag(t *testing.T) {
	t.Run("strips --config and path", func(t *testing.T) {
		a := &app.App{}
		args := a.ParseConfigFlag([]string{
			"--config", "/etc/toe.toml", "file.go",
		})
		assert.Equal(t, "/etc/toe.toml", a.ConfigPath)
		assert.Equal(t, []string{"file.go"}, args)
	})

	t.Run("passes through non-config args", func(t *testing.T) {
		a := &app.App{}
		args := a.ParseConfigFlag([]string{"a.go", "b.go"})
		assert.Equal(t, "", a.ConfigPath)
		assert.Equal(t, []string{"a.go", "b.go"}, args)
	})

	t.Run("--config at end without value", func(t *testing.T) {
		a := &app.App{}
		args := a.ParseConfigFlag([]string{"--config"})
		assert.Equal(t, "", a.ConfigPath)
		assert.Empty(t, args)
	})

	t.Run("empty args", func(t *testing.T) {
		a := &app.App{}
		args := a.ParseConfigFlag(nil)
		assert.Equal(t, "", a.ConfigPath)
		assert.Empty(t, args)
	})
}

func TestResolveSession(t *testing.T) {
	cwd := t.TempDir()

	t.Run("no args uses cwd", func(t *testing.T) {
		a := &app.App{}
		assert.NoError(t, a.ResolveSession(nil, cwd))
		assert.Equal(t, cwd, a.Root)
		assert.Equal(t, "", a.PickerDir)
		assert.Empty(t, a.Files)
	})

	t.Run("first arg as dir becomes session root", func(t *testing.T) {
		dir := t.TempDir()
		a := &app.App{}
		assert.NoError(t, a.ResolveSession([]string{dir}, cwd))
		abs, _ := filepath.Abs(dir)
		assert.Equal(t, abs, a.Root)
		assert.Equal(t, dir, a.PickerDir)
		assert.Empty(t, a.Files)
	})

	t.Run("dir with trailing file args", func(t *testing.T) {
		dir := t.TempDir()
		a := &app.App{}
		assert.NoError(t, a.ResolveSession([]string{dir, "a.go", "b.go"}, cwd))
		abs, _ := filepath.Abs(dir)
		assert.Equal(t, abs, a.Root)
		assert.Equal(t, []string{"a.go", "b.go"}, a.Files)
	})

	t.Run("non-dir first arg stays in files", func(t *testing.T) {
		a := &app.App{}
		assert.NoError(t,
			a.ResolveSession([]string{"main.go", "other.go"}, cwd))
		assert.Equal(t, cwd, a.Root)
		assert.Equal(t, "", a.PickerDir)
		assert.Equal(t, []string{"main.go", "other.go"}, a.Files)
	})
}

func TestOpenEditorFiles(t *testing.T) {
	t.Run("empty files is ok", func(t *testing.T) {
		a := newTestApp(t)
		assert.NoError(t, a.OpenEditorFiles())
	})

	t.Run("directory arg returns error", func(t *testing.T) {
		a := newTestApp(t)
		a.Files = []string{a.Root}
		err := a.OpenEditorFiles()
		assert.True(t, errors.Is(err, app.ErrDirectoryArgument))
	})

	t.Run("opens existing file", func(t *testing.T) {
		a := newTestApp(t)
		path := filepath.Join(a.Root, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("package main\n"), 0o644))
		a.Files = []string{path}
		assert.NoError(t, a.OpenEditorFiles())
		doc, ok := a.Editor.FocusedDocument()
		assert.True(t, ok)
		assert.Equal(t, path, doc.Path())
	})
}

func TestInitReg(t *testing.T) {
	t.Run("wires up model and reg", func(t *testing.T) {
		dir := t.TempDir()
		a := &app.App{Root: dir, Editor: view.NewEditor(dir)}
		assert.NoError(t, a.InitReg())
		assert.NotNil(t, a.Reg)
	})
}

func TestApplyConfigFiles(t *testing.T) {
	t.Run("no config files is ok", func(t *testing.T) {
		a := newTestApp(t)
		assert.NoError(t, a.ApplyConfigFiles())
	})

	t.Run("missing config is silently skipped", func(t *testing.T) {
		a := newTestApp(t)
		a.ConfigPath = filepath.Join(t.TempDir(), "none.toml")
		assert.NoError(t, a.ApplyConfigFiles())
	})

	t.Run("explicit config file is applied", func(t *testing.T) {
		a := newTestApp(t)
		cfg := filepath.Join(t.TempDir(), "custom.toml")
		assert.NoError(t, os.WriteFile(
			cfg, []byte("[editor]\nmouse = false\n"), 0o644,
		))
		a.ConfigPath = cfg
		assert.NoError(t, a.ApplyConfigFiles())
		assert.False(t, a.Editor.Options().Mouse)
	})

	t.Run("workspace config is applied", func(t *testing.T) {
		a := newTestApp(t)
		wdir := filepath.Join(a.Root, loader.WorkspaceDirName)
		assert.NoError(t, os.MkdirAll(wdir, 0o755))
		cfg := filepath.Join(wdir, "config.toml")
		assert.NoError(t, os.WriteFile(
			cfg, []byte("[editor]\nmouse = false\n"), 0o644,
		))
		assert.NoError(t, loader.TrustWorkspace(a.Root))
		assert.NoError(t, a.ApplyConfigFiles())
		assert.False(t, a.Editor.Options().Mouse)
	})
}

func TestWorkspaceTrusted(t *testing.T) {
	t.Run("untrusted workspace returns false", func(t *testing.T) {
		a := newTestApp(t)
		assert.False(t, a.WorkspaceTrusted())
	})

	t.Run("trusted workspace returns true", func(t *testing.T) {
		a := newTestApp(t)
		assert.NoError(t, loader.TrustWorkspace(a.Root))
		assert.True(t, a.WorkspaceTrusted())
	})
}

func TestMaybeRestoreSession(t *testing.T) {
	t.Run("skipped when files were given", func(t *testing.T) {
		a := newTestApp(t)
		a.Files = []string{"file.go"}
		a.PickerDir = "some/dir"
		assert.NoError(t, a.MaybeRestoreSession(func() bool { return true }))
		assert.Equal(t, "some/dir", a.PickerDir)
	})

	t.Run("skipped when workspace not trusted", func(t *testing.T) {
		a := newTestApp(t)
		a.PickerDir = "some/dir"
		assert.NoError(t, a.MaybeRestoreSession(func() bool { return false }))
		assert.Equal(t, "some/dir", a.PickerDir)
	})

	t.Run("skipped when no session exists", func(t *testing.T) {
		a := newTestApp(t)
		a.PickerDir = "some/dir"
		assert.NoError(t, a.MaybeRestoreSession(func() bool { return true }))
		assert.Equal(t, "some/dir", a.PickerDir)
	})

	t.Run("session is restored", func(t *testing.T) {
		a := newTestApp(t)
		// Save a session with the default scratch document
		sessionPath := view.WorkspaceSessionFile(a.Root)
		assert.NoError(t,
			a.Editor.SaveSession(sessionPath, map[string]string{}))
		// Fresh editor; PickerDir should be cleared after restore
		a.Editor = view.NewEditor(a.Root)
		a.PickerDir = "some/dir"
		assert.NoError(t, a.MaybeRestoreSession(func() bool { return true }))
		assert.Equal(t, "", a.PickerDir)
	})
}

func TestInitLSP(t *testing.T) {
	t.Run("attaches LSP and returns cleanup", func(t *testing.T) {
		a := newTestApp(t)
		cleanup := a.InitLSP(context.Background())
		assert.NotNil(t, cleanup)
		cleanup()
	})

	t.Run("config reload applies workspace config", func(t *testing.T) {
		a := newTestApp(t)
		cleanup := a.InitLSP(context.Background())
		defer cleanup()
		assert.NoError(t, a.Editor.ReloadConfig())
	})
}

func TestMaybeSaveSession(t *testing.T) {
	t.Run("skipped when workspace not trusted", func(t *testing.T) {
		a := newTestApp(t)
		assert.NoError(t, a.MaybeSaveSession(map[string]string{}))
	})

	t.Run("saves when workspace trusted", func(t *testing.T) {
		a := newTestApp(t)
		assert.NoError(t, loader.TrustWorkspace(a.Root))
		assert.NoError(t, a.MaybeSaveSession(map[string]string{}))
	})
}

func TestConfigureModel(t *testing.T) {
	t.Run("no picker dir leaves model unchanged", func(t *testing.T) {
		a := newTestApp(t)
		km := command.NewKeymaps()
		a.Model = ui.New(a.Editor, km)
		assert.NoError(t, a.ConfigureModel())
	})

	t.Run("picker dir sets initial picker", func(t *testing.T) {
		a := newTestApp(t)
		km := command.NewKeymaps()
		a.Model = ui.New(a.Editor, km)
		a.PickerDir = a.Root
		assert.NoError(t, a.ConfigureModel())
	})

	t.Run("untrusted workspace startup message", func(t *testing.T) {
		dir := t.TempDir()
		assert.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
		e := view.NewEditor(dir)
		km := command.NewKeymaps()
		a := &app.App{Root: dir, Editor: e, Model: ui.New(e, km)}
		assert.NoError(t, a.ConfigureModel())
	})
}

func TestChangedOptionValues(t *testing.T) {
	t.Run("returns only changed keys", func(t *testing.T) {
		base := map[string]string{"a": "1", "b": "2", "c": "3"}
		values := map[string]string{"a": "1", "b": "99", "c": "3"}
		got := app.ChangedOptionValues(base, values)
		assert.Equal(t, map[string]string{"b": "99"}, got)
	})

	t.Run("returns all when base empty", func(t *testing.T) {
		values := map[string]string{"x": "1", "y": "2"}
		got := app.ChangedOptionValues(map[string]string{}, values)
		assert.Equal(t, values, got)
	})

	t.Run("returns empty when nothing changed", func(t *testing.T) {
		base := map[string]string{"a": "1"}
		got := app.ChangedOptionValues(base, map[string]string{"a": "1"})
		assert.Empty(t, got)
	})
}

func TestRunHealth(t *testing.T) {
	t.Run("--health flag runs health check", func(t *testing.T) {
		var b bytes.Buffer
		err := app.Run([]string{"--health"}, &b)
		assert.NoError(t, err)
		assert.Contains(t, b.String(), "toe health: ok")
	})
}

func TestRunErrors(t *testing.T) {
	t.Run("directory as non-first arg errors", func(t *testing.T) {
		dir1, dir2 := t.TempDir(), t.TempDir()
		err := app.Run([]string{dir1, dir2}, nil)
		assert.True(t, errors.Is(err, app.ErrDirectoryArgument))
	})
}
