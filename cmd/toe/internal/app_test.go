package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	app "github.com/kode4food/toe/cmd/toe/internal"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/view"
)

func TestNew(t *testing.T) {
	cwd := t.TempDir()

	t.Run("no args uses cwd", func(t *testing.T) {
		a, err := app.New(nil, cwd)
		assert.NoError(t, err)
		assert.Equal(t, cwd, a.Root)
		assert.False(t, a.ShowPicker)
		assert.Empty(t, a.Files)
	})

	t.Run("strips --config and path", func(t *testing.T) {
		a, err := app.New([]string{"--config", "/etc/toe.toml"}, cwd)
		assert.NoError(t, err)
		assert.Equal(t, "/etc/toe.toml", a.ConfigPath)
		assert.Empty(t, a.Files)
	})

	t.Run("--config at end without value", func(t *testing.T) {
		a, err := app.New([]string{"--config"}, cwd)
		assert.NoError(t, err)
		assert.Equal(t, "", a.ConfigPath)
		assert.Empty(t, a.Files)
	})

	t.Run("first arg as dir becomes session root", func(t *testing.T) {
		dir := t.TempDir()
		a, err := app.New([]string{dir}, cwd)
		assert.NoError(t, err)
		abs, _ := filepath.Abs(dir)
		assert.Equal(t, abs, a.Root)
		assert.True(t, a.ShowPicker)
		assert.Empty(t, a.Files)
	})

	t.Run("dir with trailing file args", func(t *testing.T) {
		dir := t.TempDir()
		a, err := app.New([]string{dir, "a.go", "b.go"}, cwd)
		assert.NoError(t, err)
		abs, _ := filepath.Abs(dir)
		assert.Equal(t, abs, a.Root)
		assert.Equal(t, []string{"a.go", "b.go"}, a.Files)
	})

	t.Run("non-dir first arg stays in files", func(t *testing.T) {
		a, err := app.New([]string{"main.go", "other.go"}, cwd)
		assert.NoError(t, err)
		assert.Equal(t, cwd, a.Root)
		assert.False(t, a.ShowPicker)
		assert.Equal(t, []string{"main.go", "other.go"}, a.Files)
	})

	t.Run("external file changes root", func(t *testing.T) {
		assert.NoError(t, os.Mkdir(filepath.Join(cwd, ".git"), 0o755))
		dir := t.TempDir()
		path := filepath.Join(dir, "outside.txt")
		assert.NoError(t, os.WriteFile(path, []byte("outside"), 0o644))

		a, err := app.New([]string{path}, cwd)

		assert.NoError(t, err)
		assert.Equal(t, dir, a.Root)
		assert.Equal(t, []string{path}, a.Files)
	})
}

func TestStart(t *testing.T) {
	binding := `(toe/bind :modes :normal :keys "C-A-x" (toe/write))`

	t.Run("opens the named file", func(t *testing.T) {
		dir := workspace(t)
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("package main\n"), 0o644))

		a := start(t, dir, path)

		doc := a.Editor.FocusedDocument()
		assert.NotNil(t, doc)
		assert.Equal(t, path, doc.Path())
	})

	t.Run("directory as non-first arg errors", func(t *testing.T) {
		dir := workspace(t)
		a, err := app.New([]string{"main.go", t.TempDir()}, dir)
		assert.NoError(t, err)
		defer func() { _ = a.Stop() }()

		err = a.Start(context.Background())

		assert.True(t, errors.Is(err, app.ErrDirectoryArgument))
	})

	t.Run("workspace config is applied", func(t *testing.T) {
		dir := workspace(t)
		writeWorkspaceFile(t, dir, "config.toml", "[editor]\nmouse = false\n")
		assert.NoError(t, loader.TrustWorkspace(dir))

		a := start(t, dir)

		assert.False(t, a.Editor.Options().Mouse)
	})

	t.Run("explicit config file is applied", func(t *testing.T) {
		dir := workspace(t)
		cfg := filepath.Join(t.TempDir(), "custom.toml")
		assert.NoError(t, os.WriteFile(
			cfg, []byte("[editor]\nmouse = false\n"), 0o644,
		))

		a := start(t, dir, "--config", cfg)

		assert.False(t, a.Editor.Options().Mouse)
	})

	t.Run("missing config is silently skipped", func(t *testing.T) {
		dir := workspace(t)
		cfg := filepath.Join(t.TempDir(), "none.toml")

		a := start(t, dir, "--config", cfg)

		assert.True(t, a.Editor.Options().Mouse)
	})

	t.Run("reload re-applies workspace config", func(t *testing.T) {
		dir := workspace(t)
		assert.NoError(t, loader.TrustWorkspace(dir))
		a := start(t, dir)
		writeWorkspaceFile(t, dir, "config.toml", "[editor]\nmouse = false\n")

		assert.NoError(t, a.Editor.ReloadConfig())

		assert.False(t, a.Editor.Options().Mouse)
	})

	t.Run("missing file is ok", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		start(t, workspace(t))
	})

	t.Run("evaluates Ale", func(t *testing.T) {
		writeUserInitFile(t, binding)
		start(t, workspace(t))
	})

	t.Run("returns script error", func(t *testing.T) {
		path := writeUserInitFile(t, `(`)
		dir := workspace(t)
		a, err := app.New(nil, dir)
		assert.NoError(t, err)
		defer func() { _ = a.Stop() }()

		err = a.Start(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), path+":\n")
	})

	t.Run("user binding blocks workspace", func(t *testing.T) {
		writeUserInitFile(t, binding)
		dir := workspace(t)
		path := writeWorkspaceFile(t, dir, "init.ale", binding)
		assert.NoError(t, loader.TrustWorkspace(dir))
		a, err := app.New(nil, dir)
		assert.NoError(t, err)
		defer func() { _ = a.Stop() }()

		err = a.Start(context.Background())

		assert.Contains(t, err.Error(), i18n.Text(i18n.ErrorBindingExists))
		assert.Contains(t, err.Error(), path+":\n")
	})

	t.Run("skips untrusted workspace", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		dir := workspace(t)
		writeWorkspaceFile(t, dir, "init.ale", `(`)
		start(t, dir)
	})

	t.Run("trusted workspace restores the session", func(t *testing.T) {
		dir := workspace(t)
		assert.NoError(t, loader.TrustWorkspace(dir))
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("package main\n"), 0o644))
		saveSessionFor(t, dir, path)

		a := start(t, dir)

		doc := a.Editor.FocusedDocument()
		assert.NotNil(t, doc)
		assert.Equal(t, path, doc.Path())
	})

	t.Run("untrusted workspace ignores the session", func(t *testing.T) {
		dir := workspace(t)
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("package main\n"), 0o644))
		saveSessionFor(t, dir, path)

		a := start(t, dir)

		doc := a.Editor.FocusedDocument()
		assert.NotNil(t, doc)
		assert.Equal(t, "", doc.Path())
	})

	t.Run("directory argument clears the picker", func(t *testing.T) {
		dir := workspace(t)
		assert.NoError(t, loader.TrustWorkspace(dir))
		saveSessionFor(t, dir)
		a, err := app.New([]string{dir}, t.TempDir())
		assert.NoError(t, err)
		defer func() { _ = a.Stop() }()

		assert.NoError(t, a.Start(context.Background()))

		assert.Equal(t, dir, a.Root)
		assert.False(t, a.ShowPicker)
	})
}

func TestStop(t *testing.T) {
	t.Run("session holds only options off the base", func(t *testing.T) {
		dir := workspace(t)
		assert.NoError(t, loader.TrustWorkspace(dir))
		a := start(t, dir)
		a.Editor.Options().ScrollOff = 12

		assert.NoError(t, a.Stop())

		opts := savedSessionOptions(t, view.WorkspaceSessionFile(dir))
		assert.Equal(t, map[string]string{"scrolloff": "12"}, opts)
	})

	t.Run("unchanged options save nothing", func(t *testing.T) {
		dir := workspace(t)
		assert.NoError(t, loader.TrustWorkspace(dir))
		a := start(t, dir)

		assert.NoError(t, a.Stop())

		opts := savedSessionOptions(t, view.WorkspaceSessionFile(dir))
		assert.Empty(t, opts)
	})

	t.Run("file arguments keep existing session file", func(t *testing.T) {
		dir := workspace(t)
		assert.NoError(t, loader.TrustWorkspace(dir))
		path := filepath.Join(dir, "main.go")
		assert.NoError(t, os.WriteFile(path, []byte("package main\n"), 0o644))
		saveSessionFor(t, dir, path)
		sessionPath := view.WorkspaceSessionFile(dir)
		before, err := os.ReadFile(sessionPath)
		assert.NoError(t, err)
		a := start(t, dir, path)
		a.Editor.Options().ScrollOff = 12

		assert.NoError(t, a.Stop())

		after, err := os.ReadFile(sessionPath)
		assert.NoError(t, err)
		assert.Equal(t, string(before), string(after))
	})

	t.Run("untrusted keeps existing session file", func(t *testing.T) {
		dir := workspace(t)
		sessionPath := view.WorkspaceSessionFile(dir)
		assert.NoError(t, os.MkdirAll(filepath.Dir(sessionPath), 0o755))
		assert.NoError(t,
			os.WriteFile(sessionPath, []byte("original"), 0o644))
		a := start(t, dir)

		assert.NoError(t, a.Stop())

		data, err := os.ReadFile(sessionPath)
		assert.NoError(t, err)
		assert.Equal(t, "original", string(data))
	})
}

func TestWorkspaceTrusted(t *testing.T) {
	t.Run("untrusted workspace returns false", func(t *testing.T) {
		a := start(t, workspace(t))
		assert.False(t, a.WorkspaceTrusted())
	})

	t.Run("trusted workspace returns true", func(t *testing.T) {
		dir := workspace(t)
		assert.NoError(t, loader.TrustWorkspace(dir))
		a := start(t, dir)
		assert.True(t, a.WorkspaceTrusted())
	})
}

func TestRun(t *testing.T) {
	t.Run("--health flag runs health check", func(t *testing.T) {
		var b bytes.Buffer
		err := app.Run([]string{"--health"}, &b)
		assert.NoError(t, err)
		assert.Contains(t, b.String(), "toe health: ok")
	})

	t.Run("directory as non-first arg errors", func(t *testing.T) {
		dir := workspace(t)
		outer := t.TempDir()
		err := app.Run([]string{dir, outer}, nil)
		assert.True(t, errors.Is(err, app.ErrDirectoryArgument))
	})
}

// workspace returns a temp directory that is the current directory, holds a
// git marker so it reads as a workspace root, and has private trust and config
// stores
func workspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	assert.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o755))
	return dir
}

// start builds an app over dir and brings it up, stopping it on cleanup
func start(t *testing.T, dir string, args ...string) *app.App {
	t.Helper()
	a, err := app.New(args, dir)
	assert.NoError(t, err)
	assert.NoError(t, a.Start(context.Background()))
	t.Cleanup(func() { _ = a.Stop() })
	return a
}

// writeWorkspaceFile writes a file into dir's workspace directory
func writeWorkspaceFile(t *testing.T, dir, name, src string) string {
	t.Helper()
	wdir := filepath.Join(dir, loader.WorkspaceDirName)
	assert.NoError(t, os.MkdirAll(wdir, 0o755))
	path := filepath.Join(wdir, name)
	assert.NoError(t, os.WriteFile(path, []byte(src), 0o644))
	return path
}

// writeUserInitFile writes init.ale into a private user config directory
func writeUserInitFile(t *testing.T, src string) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfg := filepath.Join(dir, loader.DirName)
	assert.NoError(t, os.MkdirAll(cfg, 0o755))
	path := filepath.Join(cfg, "init.ale")
	assert.NoError(t, os.WriteFile(path, []byte(src), 0o644))
	return path
}

// saveSessionFor writes a session for dir holding the given paths
func saveSessionFor(t *testing.T, dir string, paths ...string) {
	t.Helper()
	e := view.NewEditor(dir)
	for _, path := range paths {
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
	}
	assert.NoError(t,
		e.SaveSession(view.WorkspaceSessionFile(dir), nil))
}

func savedSessionOptions(t *testing.T, path string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(path)
	assert.NoError(t, err)
	var sess struct {
		Options map[string]string `json:"options"`
	}
	assert.NoError(t, json.Unmarshal(data, &sess))
	return sess.Options
}
