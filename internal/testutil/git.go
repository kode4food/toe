package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// RequireGit skips the test when no git binary is on PATH
func RequireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}
}

// GitRepo creates a temporary git repository with commit identity configured
func GitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	RunGit(t, dir, "init", "-b", "main")
	RunGit(t, dir, "config", "user.email", "toe@example.com")
	RunGit(t, dir, "config", "user.name", "toe")
	RunGit(t, dir, "config", "commit.gpgsign", "false")
	return dir
}

// GitCommitFile writes and commits a file, returning its absolute path
func GitCommitFile(t *testing.T, repo, name, content string) string {
	t.Helper()
	path := filepath.Join(repo, name)
	WriteFile(t, path, content)
	RunGit(t, repo, "add", name)
	RunGit(t, repo, "commit", "-m", "add "+name)
	return path
}

func WriteFile(t *testing.T, path, content string) {
	t.Helper()
	assert.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func RunGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	all := append([]string{"-C", dir}, args...)
	out, err := exec.Command("git", all...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
