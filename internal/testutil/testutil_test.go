package testutil_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/testutil"
)

func TestWriteFakeClipboardTools(t *testing.T) {
	t.Run("pbcopy/pbpaste round-trips text", func(t *testing.T) {
		clip := filepath.Join(t.TempDir(), "clip.txt")
		testutil.WriteFakeClipboardTools(t, clip)

		assert.NoError(t, os.WriteFile(clip, []byte("hello"), 0o644))
		out, err := exec.Command("pbpaste").Output()
		assert.NoError(t, err)
		assert.Equal(t, "hello", string(out))
	})

	t.Run("installs xclip that round-trips text", func(t *testing.T) {
		clip := filepath.Join(t.TempDir(), "clip.txt")
		testutil.WriteFakeClipboardTools(t, clip)

		assert.NoError(t, os.WriteFile(clip, []byte("world"), 0o644))
		out, err := exec.Command("xclip", "-o").Output()
		assert.NoError(t, err)
		assert.Equal(t, "world", string(out))
	})
}
