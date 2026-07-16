package ui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/vcs"
	"github.com/kode4food/toe/internal/view"
)

func TestDiffGutter(t *testing.T) {
	testutil.RequireGit(t)

	t.Run("marks changed and added lines", func(t *testing.T) {
		e, s := repoEditor(t,
			"one\ntwo\nthree\n", "one\nCHANGED\nthree\nadded\n",
		)
		defer s.Close()
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "▍")
	})

	t.Run("marks removed lines", func(t *testing.T) {
		e, s := repoEditor(t, "one\ntwo\nthree\n", "one\nthree\n")
		defer s.Close()
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "▔")
	})

	t.Run("clean file renders no markers", func(t *testing.T) {
		text := "one\ntwo\nthree\n"
		e, s := repoEditorNoWait(t, text, text)
		defer s.Close()
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)
		assert.NotContains(t, out, "▍")
		assert.NotContains(t, out, "▔")
	})

	t.Run("statusline shows head name", func(t *testing.T) {
		e, s := repoEditor(t, "one\ntwo\n", "one\nCHANGED\n")
		defer s.Close()
		e.Options().StatusLine.Right = []view.StatusLineItem{
			{Element: view.StatusLineVersionControl},
		}
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		out := stripANSI(m.View().Content)
		assert.Contains(t, out, " main ")
	})
}

func TestChangedFilePicker(t *testing.T) {
	testutil.RequireGit(t)

	t.Run("lists changed files with kinds", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		testutil.GitCommitFile(t, repo, "modified.txt", "one\n")
		testutil.WriteFile(t, filepath.Join(repo, "modified.txt"), "two\n")
		testutil.WriteFile(t, filepath.Join(repo, "untracked.txt"), "new\n")
		testutil.WriteFile(t, filepath.Join(repo, "staged.txt"), "new\n")
		testutil.RunGit(t, repo, "add", "staged.txt")

		m := changedFilePicker(t, repo)

		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "modified.txt")
		assert.Contains(t, out, "untracked.txt")
		assert.Contains(t, out, "\uf420 untracked.txt") //  nf-oct-question
		assert.Contains(t, out, "staged.txt")
		assert.Contains(t, out, "\uf457 staged.txt") //  nf-oct-diff_added
	})

	t.Run("preview opens on the first change", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		lines := make([]string, 60)
		for i := range lines {
			lines[i] = "line\n"
		}
		committed := strings.Join(lines, "")
		testutil.GitCommitFile(t, repo, "deep.txt", committed)
		lines[49] = "CHANGED-DEEP\n"
		testutil.WriteFile(
			t, filepath.Join(repo, "deep.txt"), strings.Join(lines, ""),
		)

		m := changedFilePicker(t, repo)

		// line 50 is far below the preview fold; it only shows when the preview
		// is centered on the first hunk
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "CHANGED-DEEP")
		assert.Contains(t, out, "▍")
	})

	t.Run("lists deleted and renamed files", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		deleted := testutil.GitCommitFile(t, repo, "deleted.txt", "gone\n")
		testutil.GitCommitFile(t, repo, "old.txt", "moved\n")
		assert.NoError(t, os.Remove(deleted))
		testutil.RunGit(t, repo, "mv", "old.txt", "new.txt")

		m := changedFilePicker(t, repo)

		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "\uf458 deleted.txt")
		assert.Contains(t, out, "\uf45a old.txt \u2192 new.txt")
	})

	t.Run("accept opens changed file", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		testutil.GitCommitFile(t, repo, "modified.txt", "one\n")
		testutil.WriteFile(t, filepath.Join(repo, "modified.txt"), "two\n")

		m := changedFilePicker(t, repo)
		_ = sendSpecial(m, tea.KeyEnter)

		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "two")
	})

	t.Run("accept selects the first hunk", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		lines := make([]string, 60)
		for i := range lines {
			lines[i] = "line\n"
		}
		testutil.GitCommitFile(t, repo, "deep.txt", strings.Join(lines, ""))
		lines[49] = "CHANGED-DEEP\n"
		testutil.WriteFile(
			t, filepath.Join(repo, "deep.txt"), strings.Join(lines, ""),
		)

		e := view.NewEditor(repo)
		s := vcs.Attach(e)
		t.Cleanup(s.Close)
		m := ui.New(e, command.NewKeymaps()).
			WithInitialPicker(ui.NewChangedFilePicker)
		m = updateAndFeed(m, tea.WindowSizeMsg{Width: 120, Height: 24})
		_ = sendSpecial(m, tea.KeyEnter)

		v, ok := e.FocusedView()
		assert.True(t, ok)
		doc, ok := e.Document(v.DocID())
		assert.True(t, ok)
		line, err := doc.SelectionFor(v.ID()).Primary().CursorLine(doc.Text())
		assert.NoError(t, err)
		assert.Equal(t, 49, line)
	})
}

// changedFilePicker opens the changed-file picker over repo and drains its item
// feed so the rendered frame is complete
func changedFilePicker(t *testing.T, repo string) ui.Model {
	t.Helper()
	e := view.NewEditor(repo)
	s := vcs.Attach(e)
	t.Cleanup(s.Close)
	m := ui.New(e, command.NewKeymaps()).
		WithInitialPicker(ui.NewChangedFilePicker)
	return updateAndFeed(m, tea.WindowSizeMsg{Width: 120, Height: 24})
}

// repoEditor opens an editor on a repo file whose work-tree content
// differs from HEAD, waiting until diff hunks are available
func repoEditor(
	t *testing.T, committed, current string,
) (*view.Editor, *vcs.Session) {
	t.Helper()
	e, s := repoEditorNoWait(t, committed, current)
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(s.DiffHunks(doc)) > 0 {
			return e, s
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for diff hunks")
	return nil, nil
}

func repoEditorNoWait(
	t *testing.T, committed, current string,
) (*view.Editor, *vcs.Session) {
	t.Helper()
	repo := testutil.GitRepo(t)
	path := testutil.GitCommitFile(t, repo, "file.txt", committed)
	testutil.WriteFile(t, path, current)
	e := view.NewEditor(repo)
	s := vcs.Attach(e)
	_, err := e.OpenFile(path)
	assert.NoError(t, err)
	return e, s
}
