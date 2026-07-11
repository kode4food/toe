//go:build !windows && !darwin

package action_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view/action"
)

func TestSystemClipboardProviderSelection(t *testing.T) {
	t.Run("prefers wayland when both are available", func(t *testing.T) {
		clipFile := filepath.Join(t.TempDir(), "clip.txt")
		bin := fakeClipboardTools(t, clipFile, "wl-copy", "wl-paste", "xclip")
		t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
		t.Setenv("WAYLAND_DISPLAY", "wayland-0")
		t.Setenv("DISPLAY", ":0")

		clip := action.NewSystemClipboard()
		assert.True(t, clip.Available())
		assert.NoError(t, clip.Write("hello"))
		got, err := clip.Read()
		assert.NoError(t, err)
		assert.Equal(t, "hello", got)
	})

	t.Run("falls back to xclip without wayland", func(t *testing.T) {
		clipFile := filepath.Join(t.TempDir(), "clip.txt")
		bin := fakeClipboardTools(t, clipFile, "xclip")
		t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
		t.Setenv("WAYLAND_DISPLAY", "")
		t.Setenv("DISPLAY", ":0")

		clip := action.NewSystemClipboard()
		assert.True(t, clip.Available())
		assert.NoError(t, clip.Write("world"))
		got, err := clip.Read()
		assert.NoError(t, err)
		assert.Equal(t, "world", got)
	})

	t.Run("unavailable with no tools on path", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		t.Setenv("WAYLAND_DISPLAY", "")
		t.Setenv("DISPLAY", "")

		clip := action.NewSystemClipboard()

		assert.False(t, clip.Available())
	})
}

func fakeClipboardTools(t *testing.T, clipFile string, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range names {
		var body string
		switch name {
		case "wl-paste":
			body = "#!/bin/sh\ncat '" + clipFile + "'\n"
		case "wl-copy":
			body = "#!/bin/sh\ncat > '" + clipFile + "'\n"
		default:
			body = "#!/bin/sh\n" +
				"for a in \"$@\"; do [ \"$a\" = \"-o\" ] && cat '" +
				clipFile + "' && exit 0; done\n" +
				"cat > '" + clipFile + "'\n"
		}
		path := filepath.Join(dir, name)
		assert.NoError(t, os.WriteFile(path, []byte(body), 0o755))
	}
	return dir
}
