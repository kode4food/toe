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
			filepath.Join(root, loader.DirName, "languages.toml"), path)
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

func TestFindWorkspaceNoMarker(t *testing.T) {
	t.Run("returns fallback when no marker", func(t *testing.T) {
		_, fallback := loader.FindWorkspace(t.TempDir())
		assert.True(t, fallback)
	})
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

func TestDirFallbacksToHome(t *testing.T) {
	t.Run("ConfigDir uses home when XDG unset", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		dir, ok := loader.ConfigDir()
		assert.True(t, ok)
		assert.Contains(t, dir, loader.DirName)
	})

	t.Run("CacheDir uses home when XDG unset", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", "")
		dir, ok := loader.CacheDir()
		assert.True(t, ok)
		assert.Contains(t, dir, loader.DirName)
	})

	t.Run("DataDir uses home when XDG unset", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "")
		dir, ok := loader.DataDir()
		assert.True(t, ok)
		assert.Contains(t, dir, loader.DirName)
	})
}

func TestRuntimeDirTildePath(t *testing.T) {
	t.Run("normalizes bare tilde to home", func(t *testing.T) {
		t.Setenv(loader.RuntimeEnv, "~")
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv(loader.DefaultRuntimeEnv, "")
		dirs := loader.RuntimeDirs()
		var found bool
		for _, d := range dirs {
			if !found && len(d) > 0 && d[len(d)-1] != '~' {
				found = true
			}
		}
		assert.True(t, found)
	})

	t.Run("tilde with no home stays as-is", func(t *testing.T) {
		t.Setenv(loader.RuntimeEnv, "~")
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Setenv(loader.DefaultRuntimeEnv, "")
		t.Setenv("HOME", "")
		dirs := loader.RuntimeDirs()
		assert.NotEmpty(t, dirs)
	})
}

func TestRuntimeFileFallback(t *testing.T) {
	t.Run("path returned when file not found", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())

		result := loader.RuntimeFile("no-such-file-xyz99.txt")

		assert.NotEmpty(t, result)
		assert.Contains(t, result, "no-such-file-xyz99.txt")
	})
}

func TestPathsWhenHomeMissing(t *testing.T) {
	t.Run("ConfigFile false when no home", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("HOME", "")

		_, ok := loader.ConfigFile()

		assert.False(t, ok)
	})

	t.Run("LanguagesFile false when no home", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("HOME", "")

		_, ok := loader.LanguagesFile()

		assert.False(t, ok)
	})

	t.Run("ConfigIgnoreFile empty when no home", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("HOME", "")

		result := loader.ConfigIgnoreFile()

		assert.Equal(t, "", result)
	})

	t.Run("LogFile false when no home", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", "")
		t.Setenv("HOME", "")

		_, ok := loader.LogFile()

		assert.False(t, ok)
	})

	t.Run("DataDir false when no home", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "")
		t.Setenv("HOME", "")

		_, ok := loader.DataDir()

		assert.False(t, ok)
	})

	t.Run("WorkspaceTrustFile false when no home", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "")
		t.Setenv("HOME", "")

		_, ok := loader.WorkspaceTrustFile()

		assert.False(t, ok)
	})

	t.Run("TrustWorkspace errors when no home", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "")
		t.Setenv("HOME", "")

		err := loader.TrustWorkspace(t.TempDir())

		assert.Equal(t, loader.ErrPathUnavailable, err)
	})

	t.Run("UntrustWorkspace errors when no home", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "")
		t.Setenv("HOME", "")

		err := loader.UntrustWorkspace(t.TempDir())

		assert.Equal(t, loader.ErrPathUnavailable, err)
	})
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
