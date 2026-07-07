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

func TestGitHelpers(t *testing.T) {
	testutil.RequireGit(t)
	repo := testutil.GitRepo(t)

	path := testutil.GitCommitFile(t, repo, "note.txt", "hello\n")
	testutil.WriteFile(t, filepath.Join(repo, "other.txt"), "world\n")
	testutil.RunGit(t, repo, "add", "other.txt")

	data, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, "hello\n", string(data))

	out, err := exec.Command("git", "-C", repo, "status", "--short").Output()
	assert.NoError(t, err)
	assert.Equal(t, "A  other.txt\n", string(out))
}
