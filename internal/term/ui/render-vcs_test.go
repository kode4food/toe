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

func TestVersionControlFileWatch(t *testing.T) {
	if testing.Short() {
		t.Skip("slow: real filesystem watch with a multi-second timeout")
	}
	testutil.RequireGit(t)

	t.Run("refreshes on an external commit", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		path := testutil.GitCommitFile(t, repo, "a.txt", "one\n")
		e := view.NewEditor(repo)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		s := vcs.Attach(e)
		defer s.Close()
		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		assert.Eventually(t, func() bool {
			base, ok := s.DiffBase(doc)
			return ok && base == "one\n"
		}, time.Second, 5*time.Millisecond)

		// the watcher must already be registered before the external
		// change happens, exactly as it would in a running editor
		m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

		testutil.WriteFile(t, path, "two\n")
		testutil.RunGit(t, repo, "add", "a.txt")
		testutil.RunGit(t, repo, "commit", "-m", "external")
		drainFileWatch(t, m)

		assert.Eventually(t, func() bool {
			base, ok := s.DiffBase(doc)
			return ok && base == "two\n"
		}, time.Second, 5*time.Millisecond)
	})
}

func TestChangedFilePicker(t *testing.T) {
	if testing.Short() {
		t.Skip("slow: many subtests each shell out to a real git repo")
	}
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

	t.Run("groups staged apart from unstaged", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		testutil.GitCommitFile(t, repo, "both.txt", "one\n")
		testutil.WriteFile(t, filepath.Join(repo, "both.txt"), "two\n")
		testutil.RunGit(t, repo, "add", "both.txt")
		testutil.WriteFile(t, filepath.Join(repo, "both.txt"), "three\n")

		m := changedFilePicker(t, repo)

		out := stripANSI(m.View().Content)
		staged := sectionRow(out, "Staged Changes")
		unstaged := sectionRow(out, "Changes")
		assert.GreaterOrEqual(t, staged, 0)
		assert.Greater(t, unstaged, staged)
		// edited in the index and again in the tree, so it lands in both
		assert.Equal(t, 2, strings.Count(out, "both.txt"))
	})

	t.Run("section hides when its group empties", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		testutil.WriteFile(t, filepath.Join(repo, "solo.txt"), "new\n")

		m := changedFilePicker(t, repo)

		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "Changes")
		assert.NotContains(t, out, "Staged Changes")
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

		// line 50 is far below the preview fold; it only shows when the diff
		// preview anchors on the first hunk. The inline diff shows the removed
		// base line and the added working line with sign-column markers
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "+ CHANGED-DEEP")
		assert.Contains(t, out, "- line")
	})

	t.Run("added file preview shows all additions", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		testutil.GitCommitFile(t, repo, "tracked.txt", "keep\n")
		testutil.WriteFile(
			t, filepath.Join(repo, "added.txt"), "alpha\nbeta\n",
		)

		m := changedFilePicker(t, repo)

		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "+ alpha")
		assert.Contains(t, out, "+ beta")
	})

	t.Run("deleted file preview shows removed base", func(t *testing.T) {
		repo := testutil.GitRepo(t)
		gone := testutil.GitCommitFile(t, repo, "gone.txt", "first\nsecond\n")
		assert.NoError(t, os.Remove(gone))

		m := changedFilePicker(t, repo)

		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "- first")
		assert.Contains(t, out, "- second")
		assert.NotContains(t, out, "<File not found>")
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
		assert.Contains(t, out, "\uf45a new.txt")
		assert.NotContains(t, out, "old.txt")
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

	t.Run("click selects the clicked file", func(t *testing.T) {
		// the changed-file picker uses empty-label columns, so it renders no
		// header row; a click must resolve to the row actually under it
		repo := testutil.GitRepo(t)
		for _, f := range []struct{ name, body string }{
			{"a.txt", "AAA\n"}, {"b.txt", "BBB\n"}, {"c.txt", "CCC\n"},
		} {
			testutil.GitCommitFile(t, repo, f.name, "base\n")
			testutil.WriteFile(t, filepath.Join(repo, f.name), f.body)
		}

		m := changedFilePicker(t, repo)
		// first item (a.txt) is selected on open
		assert.Contains(t, stripANSI(m.View().Content), "+ AAA")

		lines := strings.Split(m.View().Content, "\n")
		clickX, clickY := -1, -1
		for y, line := range lines {
			if col := strings.Index(stripANSI(line), "b.txt"); col >= 0 {
				clickX, clickY = col, y
				break
			}
		}
		assert.GreaterOrEqual(t, clickY, 0)

		m2, _ := m.Update(tea.MouseClickMsg{
			X: clickX, Y: clickY, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)

		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "+ BBB")
		assert.NotContains(t, out, "+ AAA")
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

		v := e.FocusedView()
		assert.NotNil(t, v)
		doc := e.Document(v.DocID())
		assert.NotNil(t, doc)
		line, err := doc.SelectionFor(v.ID()).Primary().CursorLine(doc.Text())
		assert.NoError(t, err)
		assert.Equal(t, 49, line)
	})
}

func changedFilePicker(t *testing.T, repo string) ui.Model {
	t.Helper()
	e := view.NewEditor(repo)
	s := vcs.Attach(e)
	t.Cleanup(s.Close)
	m := ui.New(e, command.NewKeymaps()).
		WithInitialPicker(ui.NewChangedFilePicker)
	return updateAndFeed(m, tea.WindowSizeMsg{Width: 120, Height: 24})
}

func repoEditor(
	t *testing.T, committed, current string,
) (*view.Editor, *vcs.Session) {
	t.Helper()
	e, s := repoEditorNoWait(t, committed, current)
	doc := e.FocusedDocument()
	assert.NotNil(t, doc)
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

func sectionRow(out, label string) int {
	for i, line := range strings.Split(out, "\n") {
		body := strings.Trim(line, " \u2502")
		if strings.TrimSpace(body) == label {
			return i
		}
	}
	return -1
}
