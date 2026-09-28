package ui_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/builtin/files"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/vcs"
	"github.com/kode4food/toe/internal/view"
)

type (
	countingPathSource struct {
		dir       string
		loadCalls int
	}

	watchPump struct {
		model ui.Model
		msgs  chan tea.Msg
	}
)

const (
	fileWatchTestTimeout    = 2 * time.Second
	fileWatchBurstWrites    = 8
	msgQueueSize            = 64
	gutterPollPause         = 10 * time.Millisecond
	fileWatchBurstPause     = 5 * time.Millisecond
	pickerReopenWidth       = 120
	pickerReopenHeight      = 24
	pickerReopenMiddleLines = 40
)

func (c *countingPathSource) ID() string {
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

func (*countingPathSource) Accept(*ui.PickerItem, ui.PickerAcceptAction) {
}

func (c *countingPathSource) Load() ui.PickerLoad {
	c.loadCalls++
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return ui.PickerLoad{Stop: func() {}}
	}
	items := make([]*ui.PickerItem, 0, len(entries))
	for _, entry := range entries {
		path := filepath.Join(c.dir, entry.Name())
		items = append(items, &ui.PickerItem{
			Display:  entry.Name(),
			Location: ui.PickerLocation{Target: ui.PickerTarget{Path: path}},
		})
	}
	return ui.PickerLoad{Items: items, Stop: func() {}}
}

func (*countingPathSource) ItemsForPath(path string) []*ui.PickerItem {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil
	}
	return []*ui.PickerItem{{
		Display:  filepath.Base(path),
		Location: ui.PickerLocation{Target: ui.PickerTarget{Path: path}},
	}}
}

func TestPickerReopen(t *testing.T) {
	t.Run("file picker rescans unwatched", func(t *testing.T) {
		tmp := resolvedTempDir(t)
		testutil.WriteFile(t, filepath.Join(tmp, "alpha.go"), []byte("a\n"))

		e := view.NewEditor(tmp)
		e.Options().FileWatch = false
		km := command.NewKeymaps()
		m := ui.New(e, km)
		t.Cleanup(m.Close)
		bindNormalTestAction(
			km, "file_picker", m.PickerAction(files.NewFilePickerInCWD),
			[]command.KeyEvent{char('p')},
		)
		m = resize(m, 100, 20)
		m = sendKeyAndFeed(m, 'p')
		m = sendSpecial(m, tea.KeyEscape)

		testutil.WriteFile(t, filepath.Join(tmp, "beta.go"), []byte("b\n"))
		m = sendKeyAndFeed(m, 'p')
		assert.Contains(t, stripANSI(m.View().Content), "beta.go")
	})

	t.Run("changed-files rescans unwatched", func(t *testing.T) {
		if testing.Short() {
			t.Skip("slow: shells out to git")
		}
		testutil.RequireGit(t)
		repo := testutil.GitRepo(t)
		testutil.GitCommitFile(t, repo, "committed.txt", []byte("one\n"))

		e := view.NewEditor(repo)
		e.Options().FileWatch = false
		s := vcs.Attach(e)
		t.Cleanup(s.Close)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		t.Cleanup(m.Close)
		bindNormalTestAction(
			km, "changed_file_picker", m.PickerAction(ui.NewChangedFilePicker),
			[]command.KeyEvent{char('g')},
		)
		m = resize(m, 120, 24)
		m = sendKeyAndFeed(m, 'g')
		m = sendSpecial(m, tea.KeyEscape)

		testutil.WriteFile(t,
			filepath.Join(repo, "untracked.txt"), []byte("new\n"),
		)
		m = sendKeyAndFeed(m, 'g')
		assert.Contains(t, stripANSI(m.View().Content), "untracked.txt")
	})

	t.Run("unsaved edit", func(t *testing.T) {
		if testing.Short() {
			t.Skip("slow: shells out to git")
		}
		testutil.RequireGit(t)
		repo := testutil.GitRepo(t)
		middle := strings.Repeat("line\n", pickerReopenMiddleLines)
		base := "line\n" + middle + "line\n"
		path := testutil.GitCommitFile(t,
			repo, "changed.txt", []byte(base),
		)
		first := "first\n" + middle + "line\n"
		testutil.WriteFile(t, path, []byte(first))
		other := testutil.GitCommitFile(t,
			repo, "alpha.txt", []byte("one\n"),
		)
		testutil.WriteFile(t, other, []byte("two\n"))

		e := view.NewEditor(repo)
		s := vcs.Attach(e)
		t.Cleanup(s.Close)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		t.Cleanup(m.Close)
		bindNormalTestAction(
			km, "changed_file_picker", m.PickerAction(ui.NewChangedFilePicker),
			[]command.KeyEvent{char('g')},
		)
		m = updateAndFeed(m, tea.WindowSizeMsg{
			Width:  pickerReopenWidth,
			Height: pickerReopenHeight,
		})
		m = sendKeyAndFeed(m, 'g')
		m = sendSpecial(m, tea.KeyDown)
		assert.Contains(t, selectedPickerLine(m), "changed.txt")
		assert.Contains(t, stripANSI(m.View().Content), "+ first")
		m = sendSpecial(m, tea.KeyEnter)

		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		rope := doc.Text()
		second := "line\n" + middle + "second\n"
		changes, err := core.NewChangeSetFromChanges(rope, []core.Change{
			core.TextChange(
				core.Span{From: 0, To: rope.LenChars()}, second,
			),
		})
		assert.NoError(t, err)
		assert.NoError(t,
			e.Apply(core.NewTransaction(rope).WithChanges(changes)),
		)
		m = sendKey(m, 'g')
		assert.Contains(t, selectedPickerLine(m), "changed.txt")
		assert.Contains(t, stripANSI(m.View().Content), "+ second")
	})
}

// staged, saved-but-unstaged, and unsaved-in-buffer are independent, so every
// combination of the three is pinned as the picker renders it
func TestChangedFileRows(t *testing.T) {
	if testing.Short() {
		t.Skip("slow: shells out to git")
	}
	testutil.RequireGit(t)
	for _, tc := range []struct {
		name         string
		staged       bool
		onDisk       bool
		inBuffer     bool
		wantRows     int
		wantStaged   string
		wantUnstaged string
	}{
		{name: "clean"},
		{
			name: "buffer only", inBuffer: true,
			wantRows: 1, wantUnstaged: "+ buffer",
		},
		{
			name: "disk only", onDisk: true,
			wantRows: 1, wantUnstaged: "+ disk",
		},
		{
			name: "disk and buffer", onDisk: true, inBuffer: true,
			wantRows: 1, wantUnstaged: "+ buffer",
		},
		{
			name: "staged only", staged: true,
			wantRows: 1, wantStaged: "+ staged",
		},
		{
			name: "staged and buffer", staged: true, inBuffer: true,
			wantRows: 2, wantStaged: "+ staged", wantUnstaged: "+ buffer",
		},
		{
			name: "staged and disk", staged: true, onDisk: true,
			wantRows: 2, wantStaged: "+ staged", wantUnstaged: "+ disk",
		},
		{
			name:     "staged, disk and buffer",
			staged:   true,
			onDisk:   true,
			inBuffer: true,
			wantRows: 2, wantStaged: "+ staged", wantUnstaged: "+ buffer",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := testutil.GitRepo(t)
			path := testutil.GitCommitFile(t, repo, "a.txt", []byte("base\n"))
			if tc.staged {
				testutil.WriteFile(t, path, []byte("staged\n"))
				testutil.RunGit(t, repo, "add", "a.txt")
			}
			if tc.onDisk {
				testutil.WriteFile(t, path, []byte("disk\n"))
			}

			m, e := changedFilesModel(t, repo)
			_, err := e.OpenFile(path)
			assert.NoError(t, err)

			if tc.inBuffer {
				doc := e.FocusedDocument()
				assert.NotNil(t, doc)
				rope := doc.Text()
				changes, err := core.NewChangeSetFromChanges(
					rope, []core.Change{core.TextChange(
						core.Span{From: 0, To: rope.LenChars()}, "buffer\n",
					)},
				)
				assert.NoError(t, err)
				assert.NoError(t,
					e.Apply(core.NewTransaction(rope).WithChanges(changes)),
				)
			}

			m = sendKeyAndFeed(m, 'g')
			assert.Equal(t, tc.wantRows, pickerRowCount(m, "a.txt"))
			if tc.wantStaged != "" {
				assert.Contains(t, stripANSI(m.View().Content), tc.wantStaged)
			}
			if tc.wantUnstaged == "" {
				return
			}
			if tc.wantStaged != "" {
				m = sendSpecial(m, tea.KeyDown)
			}
			out := stripANSI(m.View().Content)
			assert.Contains(t, out, tc.wantUnstaged)
			if tc.onDisk && tc.inBuffer {
				assert.NotContains(t, out, "+ disk")
			}
		})
	}
}

func TestPickerFileWatch(t *testing.T) {
	if testing.Short() {
		t.Skip("slow: real filesystem watches with multi-second timeouts")
	}
	t.Run("closed picker catches file changes", func(t *testing.T) {
		tmp := resolvedTempDir(t)
		alpha := filepath.Join(tmp, "alpha.go")
		assert.NoError(t, os.WriteFile(alpha, []byte("alpha\n"), 0o644))

		e := view.NewEditor(tmp)
		km := command.NewKeymaps()
		m := ui.New(e, km)
		t.Cleanup(m.Close)
		bindNormalTestAction(
			km, "file_picker", m.PickerAction(files.NewFilePickerInCWD),
			[]command.KeyEvent{char('p')},
		)
		m = resize(m, 100, 20)
		m = sendKeyAndFeed(m, 'p')
		assert.Contains(t, stripANSI(m.View().Content), "alpha.go")
		m = sendSpecial(m, tea.KeyEscape)

		beta := filepath.Join(tmp, "beta.go")
		assert.NoError(t, os.WriteFile(beta, []byte("beta\n"), 0o644))
		assert.NoError(t, os.Remove(alpha))
		m = drainFileWatch(t, m)
		m = sendKey(m, 'p')
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "beta.go")
		assert.NotContains(t, out, "alpha.go")
	})

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
		m = newWatchPump(m).until(t, "changed")

		out = stripANSI(m.View().Content)
		assert.NotContains(t, out, "original")
	})

	t.Run("burst of writes settles", func(t *testing.T) {
		tmp := resolvedTempDir(t)
		alpha := filepath.Join(tmp, "alpha.go")
		assert.NoError(t, os.WriteFile(alpha, []byte("line 0\n"), 0o644))

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
		for _, ch := range "alpha" {
			m = sendKey(m, ch)
		}

		var body strings.Builder
		for i := range fileWatchBurstWrites {
			_, _ = fmt.Fprintf(&body, "line %d\n", i)
			assert.NoError(t,
				os.WriteFile(alpha, []byte(body.String()), 0o644),
			)
			time.Sleep(fileWatchBurstPause)
		}
		last := fmt.Sprintf("line %d", fileWatchBurstWrites-1)
		newWatchPump(m).until(t, last)
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
		newWatchPump(m).until(t, "gamma.go")
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

		pump := newWatchPump(m)
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
			pump.until(t, name)
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
		newWatchPump(m).until(t, "package after")
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
		newWatchPump(m).until(t, "deep.go")
	})

	t.Run("changed-files adds new file", func(t *testing.T) {
		testutil.RequireGit(t)
		repo := testutil.GitRepo(t)
		testutil.GitCommitFile(t, repo, "committed.txt", []byte("one\n"))

		m := changedFilePicker(t, repo)
		out := stripANSI(m.View().Content)
		assert.NotContains(t, out, "untracked.txt")

		testutil.WriteFile(t,
			filepath.Join(repo, "untracked.txt"), []byte("new\n"),
		)
		newWatchPump(m).until(t, "untracked.txt")
	})

	t.Run("open explorer shows new file", func(t *testing.T) {
		tmp := resolvedTempDir(t)
		assert.NoError(t, os.Mkdir(filepath.Join(tmp, "sub"), 0o755))
		testutil.WriteFile(t, filepath.Join(tmp, "alpha.txt"), []byte("a"))
		m := explorerModel(t, tmp)
		t.Cleanup(m.Close)
		assert.NotContains(t, stripANSI(m.View().Content), "DELETE.txt")

		testutil.WriteFile(t, filepath.Join(tmp, "DELETE.txt"), []byte("d"))
		m = newWatchPump(m).until(t, "DELETE.txt")
		out := stripANSI(m.View().Content)
		sub := strings.Index(out, "sub/")
		del := strings.Index(out, "DELETE.txt")
		assert.Less(t, strings.Index(out, "../"), sub)
		assert.Less(t, sub, del)
	})

	t.Run("closed changed-files catches edits", func(t *testing.T) {
		testutil.RequireGit(t)
		repo := testutil.GitRepo(t)
		testutil.GitCommitFile(t, repo, "committed.txt", []byte("one\n"))

		m, _ := changedFilesModel(t, repo)
		m = sendKeyAndFeed(m, 'g')
		assert.NotContains(t, stripANSI(m.View().Content), "untracked.txt")
		m = sendSpecial(m, tea.KeyEscape)

		testutil.WriteFile(t,
			filepath.Join(repo, "untracked.txt"), []byte("new\n"),
		)
		m = drainFileWatch(t, m)
		m = sendKey(m, 'g')
		assert.Contains(t, stripANSI(m.View().Content), "untracked.txt")
	})

	t.Run("selection survives changed-files update", func(t *testing.T) {
		testutil.RequireGit(t)
		repo := testutil.GitRepo(t)
		testutil.GitCommitFile(t, repo, "alpha.txt", []byte("one\n"))
		testutil.GitCommitFile(t, repo, "beta.txt", []byte("one\n"))
		testutil.GitCommitFile(t, repo, "charlie.txt", []byte("one\n"))
		testutil.WriteFile(t, filepath.Join(repo, "alpha.txt"), []byte("two\n"))
		testutil.WriteFile(t, filepath.Join(repo, "beta.txt"), []byte("two\n"))
		testutil.WriteFile(t,
			filepath.Join(repo, "charlie.txt"), []byte("two\n"),
		)

		m := changedFilePicker(t, repo)
		m = sendSpecial(m, tea.KeyDown)
		before := selectedPickerLine(m)
		assert.Contains(t, before, "beta.txt")

		testutil.WriteFile(t,
			filepath.Join(repo, "aardvark.txt"), []byte("new\n"),
		)
		m = newWatchPump(m).until(t, "aardvark.txt")

		assert.Contains(t, selectedPickerLine(m), "beta.txt")
	})

	t.Run("staging externally regroups the row", func(t *testing.T) {
		testutil.RequireGit(t)
		repo := testutil.GitRepo(t)
		testutil.GitCommitFile(t, repo, "alpha.txt", []byte("one\n"))
		testutil.WriteFile(t, filepath.Join(repo, "alpha.txt"), []byte("two\n"))

		m := changedFilePicker(t, repo)
		out := stripANSI(m.View().Content)
		assert.NotContains(t, out, "Staged Changes")

		// staging rewrites .git/index and leaves the working file alone
		testutil.RunGit(t, repo, "add", "alpha.txt")
		newWatchPump(m).until(t, "Staged Changes")

		assert.Contains(t, stripANSI(m.View().Content), "alpha.txt")
	})

	t.Run("git state watched when disabled", func(t *testing.T) {
		testutil.RequireGit(t)
		repo := testutil.GitRepo(t)
		testutil.GitCommitFile(t, repo, "alpha.txt", []byte("one\n"))
		testutil.WriteFile(t, filepath.Join(repo, "alpha.txt"), []byte("two\n"))

		e := view.NewEditor(repo)
		e.Options().FileWatch = false
		s := vcs.Attach(e)
		t.Cleanup(s.Close)
		m := ui.New(e, command.NewKeymaps()).
			WithInitialPicker(ui.NewChangedFilePicker)
		t.Cleanup(m.Close)
		m = updateAndFeed(m, tea.WindowSizeMsg{Width: 120, Height: 24})
		assert.NotContains(t, stripANSI(m.View().Content), "Staged Changes")

		testutil.RunGit(t, repo, "add", "alpha.txt")
		newWatchPump(m).until(t, "Staged Changes")
	})

	t.Run("a write keeps both stages of a file", func(t *testing.T) {
		testutil.RequireGit(t)
		repo := testutil.GitRepo(t)
		path := testutil.GitCommitFile(t, repo, "both.txt", []byte("one\n"))
		testutil.WriteFile(t, path, []byte("two\n"))
		testutil.RunGit(t, repo, "add", "both.txt")
		testutil.WriteFile(t, path, []byte("three\n"))

		m := changedFilePicker(t, repo)
		assert.Equal(t,
			2, strings.Count(stripANSI(m.View().Content), "both.txt"),
		)

		testutil.WriteFile(t, path, []byte("four\n"))
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
		path := testutil.GitCommitFile(t, repo, "alpha.txt", []byte("one\n"))
		testutil.WriteFile(t, path, []byte("two\n"))

		m := changedFilePicker(t, repo)
		m = sendKeyAndFeed(sendCtrl(m, 'r'), 'y')
		m = drainFileWatch(t, m)

		out := stripANSI(m.View().Content)
		assert.NotContains(t, out, "alpha.txt")
	})

	t.Run("discarding reverts an unsaved buffer", func(t *testing.T) {
		testutil.RequireGit(t)
		repo := testutil.GitRepo(t)
		path := testutil.GitCommitFile(t, repo, "alpha.txt", []byte("one\n"))

		m, e := changedFilesModel(t, repo)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)

		doc := e.FocusedDocument()
		assert.NotNil(t, doc)
		waitDiffBase(t, e, doc)
		rope := doc.Text()
		changes, err := core.NewChangeSetFromChanges(
			rope, []core.Change{core.TextChange(
				core.Span{From: 0, To: rope.LenChars()}, "unsaved\n",
			)},
		)
		assert.NoError(t, err)
		assert.NoError(t,
			e.Apply(core.NewTransaction(rope).WithChanges(changes)),
		)

		assert.NotEmpty(t, waitGutterHunks(t, e, doc))
		m = sendKeyAndFeed(m, 'g')
		assert.Equal(t, 1, pickerRowCount(m, "alpha.txt"))
		m = sendKeyAndFeed(sendCtrl(m, 'r'), 'y')

		assert.Equal(t, "one\n", e.FocusedDocument().Text().String())
		assert.Empty(t, waitGutterCleared(t, e, doc))
		m = sendKeyAndFeed(m, 'g')
		assert.Equal(t, 0, pickerRowCount(m, "alpha.txt"))
	})

	t.Run("discarding an open untracked file", func(t *testing.T) {
		testutil.RequireGit(t)
		repo := testutil.GitRepo(t)
		testutil.GitCommitFile(t, repo, "committed.txt", []byte("one\n"))
		path := filepath.Join(repo, "fresh.txt")
		testutil.WriteFile(t, path, []byte("fresh\n"))

		m, e := changedFilesModel(t, repo)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)

		m = sendKeyAndFeed(m, 'g')
		assert.Equal(t, 1, pickerRowCount(m, "fresh.txt"))
		sendKeyAndFeed(sendCtrl(m, 'r'), 'y')

		assert.NoFileExists(t, path)
		assert.Empty(t, e.TakeStatusMsgs())
	})

	t.Run("staging another file keeps selection", func(t *testing.T) {
		testutil.RequireGit(t)
		repo := testutil.GitRepo(t)
		for _, name := range []string{"alpha.txt", "beta.txt", "gamma.txt"} {
			testutil.GitCommitFile(t,
				repo, testutil.GitName(name), []byte("one\n"),
			)
			testutil.WriteFile(t, filepath.Join(repo, name), []byte("two\n"))
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
		path := testutil.GitCommitFile(t,
			repo, "a.txt", []byte("one\nkeep\nend\n"),
		)
		testutil.WriteFile(t, path, []byte("two\nkeep\nend\n"))

		m := changedFilePicker(t, repo)
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "+ two")

		pump := newWatchPump(m)
		for _, tc := range []struct {
			name string
			text string
			want string
		}{
			{name: "moves down", text: "one\nkeep\nthree\n", want: "+ three"},
			{name: "moves up", text: "four\nkeep\nend\n", want: "+ four"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				testutil.WriteFile(t, path, []byte(tc.text))
				m = pump.until(t, tc.want)
				assert.Contains(t, selectedPickerLine(m), "a.txt")
			})
		}
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

func pickerRowCount(m ui.Model, name string) int {
	rows := 0
	for line := range strings.SplitSeq(stripANSI(m.View().Content), "\n") {
		if strings.Contains(line, "│") && strings.Contains(line, name) {
			rows++
		}
	}
	return rows
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

func newWatchPump(m ui.Model) *watchPump {
	p := &watchPump{model: m, msgs: make(chan tea.Msg, msgQueueSize)}
	p.run(m.Init())
	return p
}

func (p *watchPump) until(t *testing.T, want string) ui.Model {
	t.Helper()
	deadline := time.After(fileWatchTestTimeout)
	for !strings.Contains(stripANSI(p.model.View().Content), want) {
		select {
		case msg := <-p.msgs:
			m, cmd := p.model.Update(msg)
			p.model = m.(ui.Model)
			_ = p.model.View()
			p.run(cmd)
		case <-deadline:
			t.Fatalf("timed out waiting for %q", want)
		}
	}
	return p.model
}

func (p *watchPump) run(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	// a batch runs concurrently, so serializing it starves the rest
	go func() {
		switch msg := cmd().(type) {
		case nil:
		case tea.BatchMsg:
			for _, next := range msg {
				p.run(next)
			}
		default:
			p.msgs <- msg
		}
	}()
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
		_ = m.View()
		cmd = next
	}
	return m
}

func waitGutterHunks(
	t *testing.T, e *view.Editor, doc *view.Document,
) []view.DiffHunk {
	t.Helper()
	return waitGutter(t, e, doc, func(n int) bool { return n > 0 })
}

func waitGutterCleared(
	t *testing.T, e *view.Editor, doc *view.Document,
) []view.DiffHunk {
	t.Helper()
	return waitGutter(t, e, doc, func(n int) bool { return n == 0 })
}

func waitGutter(
	t *testing.T, e *view.Editor, doc *view.Document, done func(int) bool,
) []view.DiffHunk {
	t.Helper()
	deadline := time.Now().Add(fileWatchTestTimeout)
	hunks := e.VersionControl().DiffHunks(doc)
	for time.Now().Before(deadline) && !done(len(hunks)) {
		time.Sleep(gutterPollPause)
		hunks = e.VersionControl().DiffHunks(doc)
	}
	return hunks
}

func waitDiffBase(t *testing.T, e *view.Editor, doc *view.Document) {
	t.Helper()
	deadline := time.Now().Add(fileWatchTestTimeout)
	for time.Now().Before(deadline) {
		if _, ok := e.VersionControl().DiffBase(doc); ok {
			return
		}
		time.Sleep(gutterPollPause)
	}
	t.Fatal("timed out waiting for the diff base")
}

func changedFilesModel(t *testing.T, repo string) (ui.Model, *view.Editor) {
	t.Helper()
	e := view.NewEditor(repo)
	s := vcs.Attach(e)
	t.Cleanup(s.Close)
	km := command.NewKeymaps()
	m := ui.New(e, km)
	t.Cleanup(m.Close)
	bindNormalTestAction(
		km, "changed_file_picker", m.PickerAction(ui.NewChangedFilePicker),
		[]command.KeyEvent{char('g')},
	)
	return resize(m, pickerReopenWidth, pickerReopenHeight), e
}
