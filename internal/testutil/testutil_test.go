package testutil_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/testutil"
)

func TestFakeClipboard(t *testing.T) {
	clip := testutil.NewFakeClipboard()
	assert.True(t, clip.Available())

	assert.NoError(t, clip.Write("hello"))
	assert.NoError(t, clip.WritePrimary("world"))

	sys, err := clip.Read()
	assert.NoError(t, err)
	assert.Equal(t, "hello", sys)

	pri, err := clip.ReadPrimary()
	assert.NoError(t, err)
	assert.Equal(t, "world", pri)
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
