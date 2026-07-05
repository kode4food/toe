package vcs_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/vcs"
	"github.com/kode4food/toe/internal/view"
)

func TestGit(t *testing.T) {
	testutil.RequireGit(t)

	t.Run("diff base returns committed content", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		path := testutil.GitCommitFile(t, repo, "a.txt", "one\ntwo\n")
		testutil.WriteFile(t, path, "one\nchanged\n")

		base, err := vcs.Git{}.DiffBase(path)
		assert.NoError(t, err)
		assert.Equal(t, "one\ntwo\n", string(base))
	})

	t.Run("diff base fails outside a repo", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "a.txt")
		testutil.WriteFile(t, path, "text\n")

		_, err := vcs.Git{}.DiffBase(path)
		assert.Error(t, err)
	})

	t.Run("diff base fails for untracked file", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		testutil.GitCommitFile(t, repo, "a.txt", "one\n")
		path := filepath.Join(repo, "new.txt")
		testutil.WriteFile(t, path, "new\n")

		_, err := vcs.Git{}.DiffBase(path)
		assert.Error(t, err)
	})

	t.Run("head name reports branch", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		path := testutil.GitCommitFile(t, repo, "a.txt", "one\n")

		name, err := vcs.Git{}.HeadName(path)
		assert.NoError(t, err)
		assert.Equal(t, "main", name)
	})

	t.Run("head name reports short hash when detached", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		path := testutil.GitCommitFile(t, repo, "a.txt", "one\n")
		testutil.RunGit(t, repo, "checkout", "--detach")

		name, err := vcs.Git{}.HeadName(path)
		assert.NoError(t, err)
		assert.NotEqual(t, "main", name)
		assert.NotEmpty(t, name)
	})

	t.Run("changed files reports status", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		modified := testutil.GitCommitFile(t, repo, "modified.txt", "one\n")
		deleted := testutil.GitCommitFile(t, repo, "deleted.txt", "gone\n")
		renamed := testutil.GitCommitFile(t,
			repo, "renamed.txt", "stable content\n")

		testutil.WriteFile(t, modified, "one\nmore\n")
		assert.NoError(t, os.Remove(deleted))
		testutil.RunGit(t, repo, "mv", "renamed.txt", "moved.txt")
		untracked := filepath.Join(repo, "untracked.txt")
		testutil.WriteFile(t, untracked, "new\n")

		changes, err := vcs.Git{}.ChangedFiles(repo)
		assert.NoError(t, err)

		kinds := map[string]view.FileChangeKind{}
		for _, c := range changes {
			kinds[filepath.Base(c.Path)] = c.Kind
		}
		assert.Equal(t, view.FileChangeModified, kinds["modified.txt"])
		assert.Equal(t, view.FileChangeDeleted, kinds["deleted.txt"])
		assert.Equal(t, view.FileChangeRenamed, kinds["moved.txt"])
		assert.Equal(t, view.FileChangeUntracked, kinds["untracked.txt"])

		for _, c := range changes {
			if c.Kind == view.FileChangeRenamed {
				assert.Equal(t, filepath.Base(renamed),
					filepath.Base(c.FromPath),
				)
			}
		}
	})

	t.Run("changed files reports conflicts", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		path := testutil.GitCommitFile(t, repo, "a.txt", "base\n")
		testutil.RunGit(t, repo, "checkout", "-b", "other")
		testutil.WriteFile(t, path, "theirs\n")
		testutil.RunGit(t, repo, "commit", "-am", "theirs")
		testutil.RunGit(t, repo, "checkout", "main")
		testutil.WriteFile(t, path, "ours\n")
		testutil.RunGit(t, repo, "commit", "-am", "ours")
		out, _ := exec.Command(
			"git", "-C", repo, "merge", "other",
		).CombinedOutput()
		assert.Contains(t, string(out), "CONFLICT")

		changes, err := vcs.Git{}.ChangedFiles(repo)
		assert.NoError(t, err)
		assert.Len(t, changes, 1)
		assert.Equal(t, view.FileChangeConflict, changes[0].Kind)
	})

	t.Run("registry falls through on failure", func(t *testing.T) {
		dir := t.TempDir()
		reg := vcs.NewRegistry()
		_, ok := reg.DiffBase(filepath.Join(dir, "nope.txt"))
		assert.False(t, ok)
		_, ok = reg.HeadName(filepath.Join(dir, "nope.txt"))
		assert.False(t, ok)
		_, ok = reg.ChangedFiles(dir)
		assert.False(t, ok)
	})
}
