package ui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/builtin/files"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
)

type countingPathSource struct {
	dir       string
	loadCalls int
}

const fileWatchTestTimeout = 2 * time.Second

func TestPickerFileWatch(t *testing.T) {
	if testing.Short() {
		t.Skip("slow: real filesystem watches with multi-second timeouts")
	}
	t.Run("preview reflects file change", func(t *testing.T) {
		tmp := resolvedTempDir(t)
		alpha := filepath.Join(tmp, "alpha.go")
		assert.NoError(t, os.WriteFile(alpha, []byte("original\n"), 0o644))
		beta := filepath.Join(tmp, "beta.go")
		assert.NoError(t, os.WriteFile(beta, []byte("beta\n"), 0o644))

		e := view.NewEditor(tmp)
		// open beta first, the watcher registers tmp when the model builds
		_, err := e.OpenFile(beta)
		assert.NoError(t, err)

		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker", m.PickerAction(files.NewFilePickerInCWD),
			[]command.KeyEvent{char('p')},
		)
		m = resize(m, 100, 20)
		m = sendKey(m, 'p')
		for _, ch := range "alpha" {
			m = sendKey(m, ch)
		}
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "original")

		assert.NoError(t, os.WriteFile(alpha, []byte("changed\n"), 0o644))
		m = drainFileWatch(t, m)

		out = stripANSI(m.View().Content)
		assert.Contains(t, out, "changed")
		assert.NotContains(t, out, "original")
	})

	t.Run("new file appears", func(t *testing.T) {
		tmp := resolvedTempDir(t)
		alpha := filepath.Join(tmp, "alpha.go")
		assert.NoError(t, os.WriteFile(alpha, []byte("package alpha\n"), 0o644))

		e := view.NewEditor(tmp)
		_, err := e.OpenFile(alpha)
		assert.NoError(t, err)

		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker", m.PickerAction(files.NewFilePickerInCWD),
			[]command.KeyEvent{char('p')},
		)
		m = resize(m, 100, 20)
		m = sendKey(m, 'p')

		out := stripANSI(m.View().Content)
		assert.NotContains(t, out, "gamma.go")

		gamma := filepath.Join(tmp, "gamma.go")
		assert.NoError(t, os.WriteFile(gamma, []byte("package gamma\n"), 0o644))
		m = drainFileWatch(t, m)

		out = stripANSI(m.View().Content)
		assert.Contains(t, out, "gamma.go")
	})

	t.Run("toggle preserves interest", func(t *testing.T) {
		tmp := resolvedTempDir(t)
		alpha := filepath.Join(tmp, "alpha.go")
		assert.NoError(t,
			os.WriteFile(alpha, []byte("package alpha\n"), 0o644),
		)

		e := view.NewEditor(tmp)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		t.Cleanup(m.Close)
		bindNormalTestAction(
			km, "file_picker", m.PickerAction(files.NewFilePickerInCWD),
			[]command.KeyEvent{char('p')},
		)
		m = resize(m, 100, 20)
		m = sendKey(m, 'p')

		for _, name := range []string{"beta.go", "gamma.go"} {
			e.Options().FileWatch = false
			m2, _ := m.Update(tea.BlurMsg{})
			m = m2.(ui.Model)
			e.Options().FileWatch = true
			m2, _ = m.Update(tea.FocusMsg{})
			m = m2.(ui.Model)
			path := filepath.Join(tmp, name)
			assert.NoError(t,
				os.WriteFile(path, []byte("package added\n"), 0o644),
			)
			m = drainFileWatch(t, m)

			assert.Contains(t, stripANSI(m.View().Content), name)
		}
	})

	t.Run("shared document stays watched", func(t *testing.T) {
		tmp := resolvedTempDir(t)
		path := filepath.Join(tmp, "shared.go")
		assert.NoError(t,
			os.WriteFile(path, []byte("package before\n"), 0o644),
		)

		e := view.NewEditor(tmp)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		km := command.NewKeymaps()
		m := resize(ui.New(e, km), 100, 20)
		t.Cleanup(m.Close)
		bindNormalTestAction(
			km, "file_picker", m.PickerAction(files.NewFilePickerInCWD),
			[]command.KeyEvent{char('p')},
		)
		assert.NoError(t, e.SplitFocused(view.LayoutVertical))
		m = sendKey(m, 'p')
		m = sendSpecial(m, tea.KeyEscape)
		e.CloseCurrentView()
		m2, _ := m.Update(tea.FocusMsg{})
		m = m2.(ui.Model)

		assert.NoError(t,
			os.WriteFile(path, []byte("package after\n"), 0o644),
		)
		m = drainFileWatch(t, m)

		assert.Contains(t, stripANSI(m.View().Content), "package after")
	})

	t.Run("nested file appears", func(t *testing.T) {
		tmp := resolvedTempDir(t)
		alpha := filepath.Join(tmp, "alpha.go")
		assert.NoError(t, os.WriteFile(alpha, []byte("package alpha\n"), 0o644))

		e := view.NewEditor(tmp)
		_, err := e.OpenFile(alpha)
		assert.NoError(t, err)

		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "file_picker", m.PickerAction(files.NewFilePickerInCWD),
			[]command.KeyEvent{char('p')},
		)
		m = resize(m, 100, 20)
		m = sendKey(m, 'p')

		out := stripANSI(m.View().Content)
		assert.NotContains(t, out, "deep.go")

		nested := filepath.Join(tmp, "pkg", "sub")
		assert.NoError(t, os.MkdirAll(nested, 0o755))
		deep := filepath.Join(nested, "deep.go")
		assert.NoError(t, os.WriteFile(deep, []byte("package sub\n"), 0o644))
		m = drainFileWatch(t, m)

		out = stripANSI(m.View().Content)
		assert.Contains(t, out, "deep.go")
	})

	t.Run("changed-files adds new file", func(t *testing.T) {
		testutil.RequireGit(t)
		repo := testutil.GitRepo(t)
		testutil.GitCommitFile(t, repo, "committed.txt", "one\n")

		m := changedFilePicker(t, repo)
		out := stripANSI(m.View().Content)
		assert.NotContains(t, out, "untracked.txt")

		testutil.WriteFile(t, filepath.Join(repo, "untracked.txt"), "new\n")
		m = drainFileWatch(t, m)

		out = stripANSI(m.View().Content)
		assert.Contains(t, out, "untracked.txt")
	})

	t.Run("selection survives changed-files update", func(t *testing.T) {
		testutil.RequireGit(t)
		repo := testutil.GitRepo(t)
		testutil.GitCommitFile(t, repo, "alpha.txt", "one\n")
		testutil.GitCommitFile(t, repo, "beta.txt", "one\n")
		testutil.GitCommitFile(t, repo, "charlie.txt", "one\n")
		testutil.WriteFile(t, filepath.Join(repo, "alpha.txt"), "two\n")
		testutil.WriteFile(t, filepath.Join(repo, "beta.txt"), "two\n")
		testutil.WriteFile(t, filepath.Join(repo, "charlie.txt"), "two\n")

		m := changedFilePicker(t, repo)
		m = sendSpecial(m, tea.KeyDown)
		before := selectedPickerLine(m)
		assert.Contains(t, before, "beta.txt")

		testutil.WriteFile(t, filepath.Join(repo, "aardvark.txt"), "new\n")
		m = drainFileWatch(t, m)

		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "aardvark.txt")
		assert.Contains(t, selectedPickerLine(m), "beta.txt")
	})

	t.Run("staging externally regroups the row", func(t *testing.T) {
		testutil.RequireGit(t)
		repo := testutil.GitRepo(t)
		testutil.GitCommitFile(t, repo, "alpha.txt", "one\n")
		testutil.WriteFile(t, filepath.Join(repo, "alpha.txt"), "two\n")

		m := changedFilePicker(t, repo)
		out := stripANSI(m.View().Content)
		assert.NotContains(t, out, "Staged Changes")

		// staging rewrites .git/index and leaves the working file alone
		testutil.RunGit(t, repo, "add", "alpha.txt")
		m = drainFileWatch(t, m)

		out = stripANSI(m.View().Content)
		assert.Contains(t, out, "Staged Changes")
		assert.Contains(t, out, "alpha.txt")
	})

	t.Run("a write keeps both stages of a file", func(t *testing.T) {
		testutil.RequireGit(t)
		repo := testutil.GitRepo(t)
		path := testutil.GitCommitFile(t, repo, "both.txt", "one\n")
		testutil.WriteFile(t, path, "two\n")
		testutil.RunGit(t, repo, "add", "both.txt")
		testutil.WriteFile(t, path, "three\n")

		m := changedFilePicker(t, repo)
		assert.Equal(
			t, 2, strings.Count(stripANSI(m.View().Content), "both.txt"),
		)

		testutil.WriteFile(t, path, "four\n")
		m = drainFileWatch(t, m)

		out := stripANSI(m.View().Content)
		assert.Equal(t, 2, strings.Count(out, "both.txt"))
		// the staged row holds the selection, so its preview is unmoved
		assert.Contains(t, out, "+ two")

		out = stripANSI(sendSpecial(m, tea.KeyDown).View().Content)
		assert.Contains(t, out, "+ four")
	})

	t.Run("discarding a row drops it from the list", func(t *testing.T) {
		testutil.RequireGit(t)
		repo := testutil.GitRepo(t)
		path := testutil.GitCommitFile(t, repo, "alpha.txt", "one\n")
		testutil.WriteFile(t, path, "two\n")

		m := changedFilePicker(t, repo)
		m = sendKeyAndFeed(sendCtrl(m, 'r'), 'y')
		m = drainFileWatch(t, m)

		out := stripANSI(m.View().Content)
		assert.NotContains(t, out, "alpha.txt")
	})

	t.Run("staging another file keeps selection", func(t *testing.T) {
		testutil.RequireGit(t)
		repo := testutil.GitRepo(t)
		for _, name := range []string{"alpha.txt", "beta.txt", "gamma.txt"} {
			testutil.GitCommitFile(t, repo, name, "one\n")
			testutil.WriteFile(t, filepath.Join(repo, name), "two\n")
		}

		m := changedFilePicker(t, repo)
		m = sendSpecial(m, tea.KeyDown)
		assert.Contains(t, selectedPickerLine(m), "beta.txt")

		testutil.RunGit(t, repo, "add", "gamma.txt")
		m = drainFileWatch(t, m)

		assert.Contains(t, stripANSI(m.View().Content), "Staged Changes")
		assert.Contains(t, selectedPickerLine(m), "beta.txt")
	})

	t.Run("diff preview updates live", func(t *testing.T) {
		testutil.RequireGit(t)
		repo := testutil.GitRepo(t)
		path := testutil.GitCommitFile(t, repo, "a.txt", "one\n")
		testutil.WriteFile(t, path, "two\n")

		m := changedFilePicker(t, repo)
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "two")

		testutil.WriteFile(t, path, "three\n")
		m = drainFileWatch(t, m)

		out = stripANSI(m.View().Content)
		assert.Contains(t, out, "three")
	})

	t.Run("new file avoids source reload", func(t *testing.T) {
		tmp := resolvedTempDir(t)
		alpha := filepath.Join(tmp, "alpha.txt")
		assert.NoError(t, os.WriteFile(alpha, []byte("a\n"), 0o644))

		e := view.NewEditor(tmp)
		src := &countingPathSource{dir: tmp}
		km := command.NewKeymaps()
		m := ui.New(e, km)
		bindNormalTestAction(
			km, "counting_picker",
			m.PickerAction(func(e *view.Editor) *ui.Picker {
				return ui.NewPicker(e, src)
			}),
			[]command.KeyEvent{char('p')},
		)
		m = resize(m, 100, 20)
		m = sendKey(m, 'p')

		assert.Equal(t, 1, src.loadCalls)
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "alpha.txt")
		assert.NotContains(t, out, "beta.txt")

		beta := filepath.Join(tmp, "beta.txt")
		assert.NoError(t, os.WriteFile(beta, []byte("b\n"), 0o644))
		m = drainFileWatch(t, m)

		out = stripANSI(m.View().Content)
		assert.Contains(t, out, "beta.txt")
		// FSEvents can emit one spurious root-dir event right after a fresh
		// recursive watch registers, forcing one harmless fallback reload
		assert.LessOrEqual(t, src.loadCalls, 2)
	})
}

func (s *countingPathSource) ID() string {
	return "counting"
}

func (*countingPathSource) Title() string {
	return "Counting"
}

func (*countingPathSource) Columns() []string {
	return []string{"name"}
}

func (*countingPathSource) MatchColumn() int {
	return 0
}

func (*countingPathSource) ColumnProportions() []int {
	return []int{1}
}

func (*countingPathSource) Accept(
	*view.Editor, *ui.PickerItem, ui.PickerAcceptAction,
) {
}

func (s *countingPathSource) Load(*view.Editor) ui.PickerLoad {
	s.loadCalls++
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return ui.PickerLoad{Stop: func() {}}
	}
	items := make([]*ui.PickerItem, 0, len(entries))
	for _, entry := range entries {
		path := filepath.Join(s.dir, entry.Name())
		items = append(items, &ui.PickerItem{
			Display:  entry.Name(),
			Location: ui.PickerLocation{Target: ui.PickerTarget{Path: path}},
		})
	}
	return ui.PickerLoad{Items: items, Stop: func() {}}
}

func (*countingPathSource) ItemsForPath(
	_ *view.Editor, path string,
) []*ui.PickerItem {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil
	}
	return []*ui.PickerItem{{
		Display:  filepath.Base(path),
		Location: ui.PickerLocation{Target: ui.PickerTarget{Path: path}},
	}}
}

func selectedPickerLine(m ui.Model) string {
	for line := range strings.SplitSeq(stripANSI(m.View().Content), "\n") {
		if strings.Contains(line, " > ") {
			return line
		}
	}
	return ""
}

func drainFileWatch(t *testing.T, m ui.Model) ui.Model {
	t.Helper()
	batch, ok := m.Init()().(tea.BatchMsg)
	assert.True(t, ok)
	for _, cmd := range batch {
		m = drainCmdWithTimeout(m, cmd, fileWatchTestTimeout)
	}
	return m
}

func resolvedTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	resolved, err := filepath.EvalSymlinks(dir)
	assert.NoError(t, err)
	return resolved
}

func drainCmdWithTimeout(m ui.Model, cmd tea.Cmd, d time.Duration) ui.Model {
	for cmd != nil {
		msg, fired := runWithTimeout(cmd, d)
		if !fired {
			return m
		}
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, next := range batch {
				m = drainCmdWithTimeout(m, next, d)
			}
			return m
		}
		m2, next := m.Update(msg)
		m = m2.(ui.Model)
		cmd = next
	}
	return m
}
