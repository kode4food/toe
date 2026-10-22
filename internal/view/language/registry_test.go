package language_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/view/language"
)

const (
	cachedLangsHash = `
[[language]]
name = "cached"
file-types = ["cached"]
comment-token = "#"
`
	cachedLangsSlash = `
[[language]]
name = "cached"
file-types = ["cached"]
comment-token = "//"
`
)

func TestRegistry(t *testing.T) {
	t.Run("keeps config until invalidated", func(t *testing.T) {
		path := writeCachedLangs(t)
		registry := new(language.Registry)
		assert.Equal(t,
			[]string{"#"}, registry.Get("cached").CommentTokens,
		)

		rewriteLangs(t, path, cachedLangsSlash)
		assert.Equal(t,
			[]string{"#"}, registry.Get("cached").CommentTokens,
		)

		registry.Invalidate()
		assert.Equal(t,
			[]string{"//"}, registry.Get("cached").CommentTokens,
		)
	})

	t.Run("loaded file change reloads", func(t *testing.T) {
		path := writeCachedLangs(t)
		registry := new(language.Registry)
		assert.Equal(t,
			[]string{"#"}, registry.Get("cached").CommentTokens,
		)

		rewriteLangs(t, path, cachedLangsSlash)
		registry.FileChanged(path)
		assert.Equal(t,
			[]string{"//"}, registry.Get("cached").CommentTokens,
		)
	})

	t.Run("unrelated file change keeps", func(t *testing.T) {
		path := writeCachedLangs(t)
		registry := new(language.Registry)
		assert.Equal(t,
			[]string{"#"}, registry.Get("cached").CommentTokens,
		)

		rewriteLangs(t, path, cachedLangsSlash)
		other := filepath.Join(t.TempDir(), "languages.toml")
		registry.FileChanged(other)
		assert.Equal(t,
			[]string{"#"}, registry.Get("cached").CommentTokens,
		)
	})

	t.Run("detects from cached config", func(t *testing.T) {
		path := writeCachedLangs(t)
		registry := new(language.Registry)
		args := language.DetectArgs{Path: "a.cached", Default: "x"}
		assert.Equal(t, "cached", registry.Detect(args))

		assert.NoError(t, os.Remove(path))
		assert.Equal(t, "cached", registry.Detect(args))

		registry.Invalidate()
		assert.Equal(t, "x", registry.Detect(args))
	})

	t.Run("unknown name is empty", func(t *testing.T) {
		writeCachedLangs(t)
		l := new(language.Registry).Get("no-such-lang")
		assert.NotNil(t, l)
		assert.Equal(t, "", l.Name)
	})
}

func writeCachedLangs(t *testing.T) string {
	t.Helper()
	setUserLangs(t, cachedLangsHash)
	path, ok := loader.LanguagesFile()
	assert.True(t, ok)
	return path
}

func rewriteLangs(t *testing.T, path, text string) {
	t.Helper()
	assert.NoError(t, os.WriteFile(path, []byte(text), 0o644))
}
