package vcs_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/vcs"
)

func TestGitBatch(t *testing.T) {
	testutil.RequireGit(t)

	t.Run("diff base follows commits", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		path := testutil.GitCommitFile(t, repo, "a.txt", []byte("one\n"))
		git := new(vcs.Git)
		t.Cleanup(git.Close)

		base, err := git.DiffBase(path)
		assert.NoError(t, err)
		assert.Equal(t, "one\n", string(base))

		testutil.GitCommitFile(t, repo, "a.txt", []byte("two\n"))
		base, err = git.DiffBase(path)
		assert.NoError(t, err)
		assert.Equal(t, "two\n", string(base))
	})

	t.Run("index text follows staging", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		path := testutil.GitCommitFile(t, repo, "a.txt", []byte("one\n"))
		git := new(vcs.Git)
		t.Cleanup(git.Close)

		text, err := git.IndexText(repo, path)
		assert.NoError(t, err)
		assert.Equal(t, "one\n", string(text))

		testutil.WriteFile(t, path, []byte("two\n"))
		assert.NoError(t, git.Stage(repo, path))
		text, err = git.IndexText(repo, path)
		assert.NoError(t, err)
		assert.Equal(t, "two\n", string(text))
	})

	t.Run("index text fails for untracked file", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		testutil.GitCommitFile(t, repo, "a.txt", []byte("one\n"))
		path := filepath.Join(repo, "new.txt")
		testutil.WriteFile(t, path, []byte("new\n"))
		git := new(vcs.Git)
		t.Cleanup(git.Close)

		_, err := git.IndexText(repo, path)
		assert.True(t, errors.Is(err, vcs.ErrGitCommand))
		_, err = git.IndexText(repo, filepath.Join(repo, "a.txt"))
		assert.NoError(t, err)
	})

	t.Run("reads a path holding a newline", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		path := testutil.GitCommitFile(t, repo, "a\nb.txt", []byte("one\n"))
		git := new(vcs.Git)
		t.Cleanup(git.Close)

		base, err := git.DiffBase(path)
		assert.NoError(t, err)
		assert.Equal(t, "one\n", string(base))
	})

	t.Run("reads after close", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		path := testutil.GitCommitFile(t, repo, "a.txt", []byte("one\n"))
		git := new(vcs.Git)

		_, err := git.DiffBase(path)
		assert.NoError(t, err)
		git.Close()
		base, err := git.DiffBase(path)
		assert.NoError(t, err)
		assert.Equal(t, "one\n", string(base))
		git.Close()
	})
}
