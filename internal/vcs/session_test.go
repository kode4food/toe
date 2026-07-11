package vcs_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/vcs"
	"github.com/kode4food/toe/internal/view"
)

func TestSession(t *testing.T) {
	testutil.RequireGit(t)

	t.Run("serves hunks for an edited document", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		path := testutil.GitCommitFile(t, repo, "a.txt", "one\ntwo\nthree\n")
		testutil.WriteFile(t, path, "one\nchanged\nthree\n")

		e := view.NewEditor(repo)
		s := vcs.Attach(e)
		defer s.Close()
		assert.Same(t, s, e.VersionControl())

		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		hunks := waitHunks(t, s, doc)
		assert.Equal(t, []view.DiffHunk{
			{BaseFrom: 1, BaseTo: 2, From: 1, To: 2},
		}, hunks)

		base, ok := s.DiffBase(doc)
		assert.True(t, ok)
		assert.Equal(t, "one\ntwo\nthree\n", base)

		name, ok := s.HeadName(doc)
		assert.True(t, ok)
		assert.Equal(t, "main", name)
	})

	t.Run("tracks documents open before attach", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		path := testutil.GitCommitFile(t, repo, "a.txt", "one\n")
		testutil.WriteFile(t, path, "changed\n")

		e := view.NewEditor(repo)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)

		s := vcs.Attach(e)
		defer s.Close()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		assert.Len(t, waitHunks(t, s, doc), 1)
	})

	t.Run("no diff state for scratch documents", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		s := vcs.Attach(e)
		defer s.Close()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		assert.Empty(t, s.DiffHunks(doc))
		_, ok = s.DiffBase(doc)
		assert.False(t, ok)
		_, ok = s.HeadName(doc)
		assert.False(t, ok)
		s.DocumentSaved(doc)
	})

	t.Run("changed files uses editor cwd", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		testutil.GitCommitFile(t, repo, "a.txt", "one\n")
		testutil.WriteFile(t, filepath.Join(repo, "new.txt"), "new\n")

		e := view.NewEditor(repo)
		s := vcs.Attach(e)
		defer s.Close()

		changes, err := s.ChangedFiles()
		assert.NoError(t, err)
		assert.Len(t, changes, 1)
		assert.Equal(t, view.FileChangeUntracked, changes[0].Kind)
	})

	t.Run("refreshes after external head movement", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		path := testutil.GitCommitFile(t, repo, "a.txt", "one\n")

		e := view.NewEditor(repo)
		s := vcs.Attach(e)
		defer s.Close()
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		waitBase(t, s, doc, "one\n")

		testutil.WriteFile(t, path, "two\n")
		testutil.RunGit(t, repo, "add", "a.txt")
		testutil.RunGit(t, repo, "commit", "-m", "external")

		s.Refresh()

		assert.Equal(t, "two\n", waitBase(t, s, doc, "two\n"))
		assert.Equal(t, []view.DiffHunk{
			{BaseFrom: 0, BaseTo: 1, From: 0, To: 1},
		}, waitHunks(t, s, doc))
	})

	t.Run("edits and lifecycle flow through the differ", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		path := testutil.GitCommitFile(t, repo, "a.txt", "one\ntwo\nthree\n")

		e := view.NewEditor(repo)
		s := vcs.Attach(e)
		defer s.Close()

		v, err := e.OpenFile(path)
		assert.NoError(t, err)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		waitDiffer(t, s, doc)

		// consume any pending update token, then edit the document
		select {
		case <-s.Updates():
		default:
		}
		tx := core.NewTransaction(doc.Text()).WithChanges(
			mustChangeSet(t, doc.Text(), "one\nCHANGED\nthree\n"),
		)
		assert.NoError(t, e.Apply(tx))
		assert.Equal(t, []view.DiffHunk{
			{BaseFrom: 1, BaseTo: 2, From: 1, To: 2},
		}, waitHunks(t, s, doc))

		// saving refreshes the diff base from HEAD; hunks remain
		select {
		case <-s.Updates():
		default:
		}
		assert.NoError(t, e.Save(false))
		select {
		case <-s.Updates():
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for save update")
		}
		assert.Len(t, waitHunks(t, s, doc), 1)

		// closing the document tears down its differ
		e.CloseView(v.ID())
		assert.Empty(t, s.DiffHunks(doc))
	})

	t.Run("computes hunks for unopened paths", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		path := testutil.GitCommitFile(t, repo, "a.txt", "one\ntwo\nthree\n")
		testutil.WriteFile(t, path, "one\nCHANGED\nthree\n")

		e := view.NewEditor(repo)
		s := vcs.Attach(e)
		defer s.Close()

		assert.Equal(t, []view.DiffHunk{
			{BaseFrom: 1, BaseTo: 2, From: 1, To: 2},
		}, s.DiffHunksForPath(path))
		assert.NoError(t, os.Remove(path))
		assert.Nil(t, s.DiffHunksForPath(path))
		assert.Nil(t, s.DiffHunksForPath(filepath.Join(repo, "nope.txt")))
	})

	t.Run("changed files fails outside a repo", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		s := vcs.Attach(e)
		defer s.Close()
		_, err := s.ChangedFiles()
		assert.Error(t, err)
	})
}

func mustChangeSet(
	t *testing.T, text core.Rope, replacement string,
) core.ChangeSet {
	t.Helper()
	cs, err := core.NewChangeSetFromChanges(text, []core.Change{
		core.TextChange(0, text.LenChars(), replacement),
	})
	assert.NoError(t, err)
	return cs
}

func waitDiffer(t *testing.T, s *vcs.Session, doc *view.Document) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := s.DiffBase(doc); ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for differ")
}

func waitBase(
	t *testing.T, s *vcs.Session, doc *view.Document, want string,
) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if base, ok := s.DiffBase(doc); ok && base == want {
			return base
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for base")
	return ""
}

func waitHunks(
	t *testing.T, s *vcs.Session, doc *view.Document,
) []view.DiffHunk {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if hunks := s.DiffHunks(doc); len(hunks) > 0 {
			return hunks
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for hunks")
	return nil
}
