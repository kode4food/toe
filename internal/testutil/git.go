package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// GitName is a file's name relative to the repository root
type GitName string

// RequireGit skips the test when no git binary is on PATH
func RequireGit(t testing.TB) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}
}

// GitRepo creates a temporary git repository with commit identity configured
func GitRepo(t testing.TB) string {
	t.Helper()
	if testing.Short() {
		t.Skip("slow: runs git")
	}
	dir := t.TempDir()
	RunGit(t, dir, "init", "-b", "main")
	RunGit(t, dir, "config", "user.email", "toe@example.com")
	RunGit(t, dir, "config", "user.name", "toe")
	RunGit(t, dir, "config", "commit.gpgsign", "false")
	return dir
}

// GitCommitFile writes and commits a file, returning its absolute path
func GitCommitFile(
	t testing.TB, repo string, name GitName, data []byte,
) string {
	t.Helper()
	path := filepath.Join(repo, string(name))
	WriteFile(t, path, data)
	RunGit(t, repo, "add", string(name))
	RunGit(t, repo, "commit", "-m", "add "+string(name))
	return path
}

// WriteFile writes data to path, failing the test on error
func WriteFile(t testing.TB, path string, data []byte) {
	t.Helper()
	assert.NoError(t, os.WriteFile(path, data, 0o644))
}

// RunGit runs a git command in dir, failing the test on error
func RunGit(t testing.TB, dir string, args ...string) {
	t.Helper()
	all := append([]string{"-C", dir}, args...)
	out, err := exec.Command("git", all...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
