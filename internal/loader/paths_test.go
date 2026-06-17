package loader_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/loader"
)

func TestPaths(t *testing.T) {
	t.Run("resolves config file", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", root)

		path, ok := loader.ConfigFile()

		assert.True(t, ok)
		assert.Equal(t,
			filepath.Join(root, loader.DirName, "config.toml"), path)
	})

	t.Run("resolves languages file", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", root)

		path, ok := loader.LanguagesFile()

		assert.True(t, ok)
		assert.Equal(t,
			filepath.Join(root, loader.DirName, "languages.toml"), path,
		)
	})

	t.Run("resolves config ignore file", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", root)

		path := loader.ConfigIgnoreFile()

		assert.Equal(t, filepath.Join(root, loader.DirName, "ignore"), path)
	})

	t.Run("resolves log file", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CACHE_HOME", root)

		path, ok := loader.LogFile()

		assert.True(t, ok)
		assert.Equal(t,
			filepath.Join(root, loader.DirName, loader.LogFileName), path)
	})

	t.Run("resolves data dir via XDG", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_DATA_HOME", root)

		dir, ok := loader.DataDir()

		assert.True(t, ok)
		assert.Equal(t, filepath.Join(root, loader.DirName), dir)
	})

	t.Run("resolves cache dir via XDG", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CACHE_HOME", root)

		dir, ok := loader.CacheDir()

		assert.True(t, ok)
		assert.Equal(t, filepath.Join(root, loader.DirName), dir)
	})

	t.Run("resolves config dir via XDG", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", root)

		dir, ok := loader.ConfigDir()

		assert.True(t, ok)
		assert.Equal(t, filepath.Join(root, loader.DirName), dir)
	})

	t.Run("resolves workspace config file", func(t *testing.T) {
		root := t.TempDir()
		work := filepath.Join(root, "work")
		err := os.MkdirAll(filepath.Join(work, ".git"), 0o755)
		assert.NoError(t, err)

		path := loader.WorkspaceConfigFile(work)

		assert.Equal(t,
			filepath.Join(work, "."+loader.DirName, "config.toml"), path)
	})

	t.Run("resolves workspace languages file", func(t *testing.T) {
		root := t.TempDir()
		work := filepath.Join(root, "work")
		err := os.MkdirAll(filepath.Join(work, ".git"), 0o755)
		assert.NoError(t, err)

		path := loader.WorkspaceLanguagesFile(work)

		assert.Equal(t,
			filepath.Join(work, "."+loader.DirName, "languages.toml"), path)
	})
}

func TestRuntimeFile(t *testing.T) {
	t.Run("uses runtime env path", func(t *testing.T) {
		root := t.TempDir()
		rt := filepath.Join(root, "runtime")
		err := os.MkdirAll(rt, 0o755)
		assert.NoError(t, err)
		path := filepath.Join(rt, "tutor")
		err = os.WriteFile(path, []byte("learn"), 0o644)
		assert.NoError(t, err)
		t.Setenv(loader.RuntimeEnv, rt)
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv(loader.DefaultRuntimeEnv, t.TempDir())

		assert.Equal(t, path, loader.RuntimeFile("tutor"))
	})

	t.Run("expands runtime env home path", func(t *testing.T) {
		home := t.TempDir()
		rt := filepath.Join(home, "runtime")
		err := os.MkdirAll(rt, 0o755)
		assert.NoError(t, err)
		path := filepath.Join(rt, "tutor")
		err = os.WriteFile(path, []byte("learn"), 0o644)
		assert.NoError(t, err)
		t.Setenv("HOME", home)
		t.Setenv(loader.RuntimeEnv, "~/runtime")
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv(loader.DefaultRuntimeEnv, t.TempDir())

		assert.Equal(t, path, loader.RuntimeFile("tutor"))
	})
}

func TestWorkspace(t *testing.T) {
	root := t.TempDir()
	work := filepath.Join(root, "work")
	cwd := filepath.Join(work, "src")
	err := os.MkdirAll(filepath.Join(work, ".git"), 0o755)
	assert.NoError(t, err)
	err = os.MkdirAll(cwd, 0o755)
	assert.NoError(t, err)

	found, fallback := loader.FindWorkspace(cwd)

	assert.False(t, fallback)
	assert.Equal(t, work, found)
}

func TestWorkspaceTrust(t *testing.T) {
	root := t.TempDir()
	work := filepath.Join(root, "work")
	cwd := filepath.Join(work, "src")
	err := os.MkdirAll(filepath.Join(work, ".git"), 0o755)
	assert.NoError(t, err)
	err = os.MkdirAll(cwd, 0o755)
	assert.NoError(t, err)
	dataRoot := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataRoot)

	err = loader.TrustWorkspace(cwd)

	assert.NoError(t, err)
	path, ok := loader.WorkspaceTrustFile()
	assert.True(t, ok)
	data, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, work+"\n", string(data))

	err = loader.UntrustWorkspace(cwd)

	assert.NoError(t, err)
	data, err = os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, "", string(data))
}

func TestQueryWorkspaceTrust(t *testing.T) {
	root := t.TempDir()
	work := filepath.Join(root, "work")
	cwd := filepath.Join(work, "src")
	err := os.MkdirAll(filepath.Join(work, ".git"), 0o755)
	assert.NoError(t, err)
	err = os.MkdirAll(cwd, 0o755)
	assert.NoError(t, err)
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	status := loader.QueryWorkspaceTrust(cwd, false)

	assert.Equal(t, loader.TrustUntrusted, status)

	err = loader.TrustWorkspace(cwd)
	assert.NoError(t, err)

	status = loader.QueryWorkspaceTrust(cwd, false)

	assert.Equal(t, loader.TrustTrusted, status)
	assert.Equal(t, loader.TrustTrusted, loader.QueryWorkspaceTrust(cwd, true))
}

func TestExplicitWorkspaceTrust(t *testing.T) {
	root := t.TempDir()
	work := filepath.Join(root, "work")
	cwd := filepath.Join(work, "src")
	err := os.MkdirAll(filepath.Join(work, ".git"), 0o755)
	assert.NoError(t, err)
	err = os.MkdirAll(cwd, 0o755)
	assert.NoError(t, err)
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	status := loader.QueryWorkspaceTrustWithExplicitUntrust(cwd, false)

	assert.Equal(t, loader.TrustDenyOnce, status)

	err = loader.TrustWorkspace(cwd)
	assert.NoError(t, err)

	status = loader.QueryWorkspaceTrustWithExplicitUntrust(cwd, false)

	assert.Equal(t, loader.TrustAllowAlways, status)

	err = loader.ExcludeWorkspace(cwd)
	assert.NoError(t, err)

	status = loader.QueryWorkspaceTrustWithExplicitUntrust(cwd, false)

	assert.Equal(t, loader.TrustDenyAlways, status)
	assert.Equal(t, loader.TrustAllowAlways,
		loader.QueryWorkspaceTrustWithExplicitUntrust(cwd, true),
	)
}
