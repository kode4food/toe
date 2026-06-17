package loader_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/loader"
)

func TestLoadRuntimeFile(t *testing.T) {
	root := t.TempDir()
	rt := filepath.Join(root, "runtime")
	dir := filepath.Join(rt, "queries", "go")
	err := os.MkdirAll(dir, 0o755)
	assert.NoError(t, err)
	err = os.WriteFile(
		filepath.Join(dir, "highlights.scm"), []byte("query"), 0o644,
	)
	assert.NoError(t, err)
	t.Setenv(loader.RuntimeEnv, rt)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	text, err := loader.LoadRuntimeFile("go", "highlights.scm")

	assert.NoError(t, err)
	assert.Equal(t, "query", text)
}

func TestLoadQuery(t *testing.T) {
	root := t.TempDir()
	rt := filepath.Join(root, "runtime")
	dir := filepath.Join(rt, "queries", "go")
	err := os.MkdirAll(dir, 0o755)
	assert.NoError(t, err)
	err = os.WriteFile(
		filepath.Join(dir, "injections.scm"), []byte("inject"), 0o644,
	)
	assert.NoError(t, err)
	t.Setenv(loader.RuntimeEnv, rt)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	text, err := loader.LoadQuery("go", loader.QueryInjections)

	assert.NoError(t, err)
	assert.Equal(t, "inject", text)
}
