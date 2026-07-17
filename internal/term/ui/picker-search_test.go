package ui_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/builtin/files"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

func TestGlobalSearch(t *testing.T) {
	t.Run("finds matching lines across files", func(t *testing.T) {
		m, _ := globalSearchModel(t, "findme")
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "a.txt")
		assert.NotContains(t, out, "b.txt")
	})

	t.Run("preview highlights the matched line", func(t *testing.T) {
		t.Setenv("COLORTERM", "truecolor")
		m, e := globalSearchModel(t, "findme", map[string]string{
			"a.txt": "alpha\nfindme here\nbravo\n",
			"b.txt": "beta\n",
		})
		e.Options().Theme = "mocha"
		m = resize(m, 120, 30)
		out := m.View().Content

		matchLine, otherLine := "", ""
		for line := range strings.SplitSeq(out, "\n") {
			plain := stripANSI(line)
			switch {
			case strings.Contains(plain, "findme here"):
				matchLine = line
			case strings.Contains(plain, "bravo"):
				otherLine = line
			}
		}
		assert.NotEmpty(t, matchLine)
		assert.NotEmpty(t, otherLine)
		matchBg := bgAt(matchLine, "findme here")
		otherBg := bgAt(otherLine, "bravo")
		assert.NotEmpty(t, matchBg)
		assert.NotEqual(t, otherBg, matchBg)
	})

	t.Run("accept opens match", func(t *testing.T) {
		m, e := globalSearchModel(t, "findme")
		_ = sendSpecial(m, tea.KeyEnter)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		assert.True(t, strings.HasSuffix(doc.Path(), "a.txt"))
		v, ok := e.FocusedView()
		assert.True(t, ok)
		line, err := doc.Text().CharToLine(
			doc.SelectionFor(v.ID()).Primary().Cursor(doc.Text()),
		)
		assert.NoError(t, err)
		assert.Equal(t, 1, line) // 0-indexed line 2 holds "findme here"
	})

	t.Run("empty query has no matches", func(t *testing.T) {
		m, _ := globalSearchModel(t, "")
		out := stripANSI(m.View().Content)
		assert.NotContains(t, out, "a.txt")
		assert.NotContains(t, out, "b.txt")
	})

	t.Run("invalid regex has no matches", func(t *testing.T) {
		m, _ := globalSearchModel(t, "[")
		out := stripANSI(m.View().Content)
		assert.NotContains(t, out, "a.txt")
		assert.NotContains(t, out, "b.txt")
	})

	t.Run("lowercase query ignores case", func(t *testing.T) {
		m, _ := globalSearchModel(t, "findme", map[string]string{
			"a.txt": "alpha\nFindMe here\n",
			"b.txt": "beta\n",
		})
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "a.txt")
	})

	t.Run("uppercase query is case sensitive", func(t *testing.T) {
		m, _ := globalSearchModel(t, "FINDME", map[string]string{
			"a.txt": "alpha\nfindme here\n",
			"b.txt": "beta\n",
		})
		out := stripANSI(m.View().Content)
		assert.NotContains(t, out, "a.txt")
	})

	t.Run("skips binary files", func(t *testing.T) {
		m, _ := globalSearchModel(t, "findme", map[string]string{
			"a.bin": "findme\x00here\n",
			"b.txt": "beta\n",
		})
		out := stripANSI(m.View().Content)
		assert.NotContains(t, out, "a.bin")
	})

	t.Run("respects ignore files", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(
			filepath.Join(dir, ".gitignore"),
			[]byte("ignored.txt\nignored-dir/\n"),
			0o644,
		)
		assert.NoError(t, err)
		err = os.WriteFile(
			filepath.Join(dir, "ignored.txt"),
			[]byte("findme hidden\n"),
			0o644,
		)
		assert.NoError(t, err)
		ignoredDir := filepath.Join(dir, "ignored-dir")
		err = os.MkdirAll(ignoredDir, 0o755)
		assert.NoError(t, err)
		err = os.WriteFile(
			filepath.Join(ignoredDir, "nested.txt"),
			[]byte("findme nested\n"),
			0o644,
		)
		assert.NoError(t, err)
		err = os.WriteFile(
			filepath.Join(dir, "visible.txt"),
			[]byte("findme visible\n"),
			0o644,
		)
		assert.NoError(t, err)

		m := openGlobalSearch(t, view.NewEditor(dir), "findme")
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "visible.txt")
		assert.NotContains(t, out, "ignored.txt")
		assert.NotContains(t, out, "ignored-dir")
	})

	t.Run("follows external directory symlink", func(t *testing.T) {
		tmp := t.TempDir()
		root := filepath.Join(tmp, "root")
		external := filepath.Join(tmp, "external")
		assert.NoError(t, os.MkdirAll(root, 0o755))
		assert.NoError(t, os.MkdirAll(external, 0o755))
		assert.NoError(t, os.WriteFile(
			filepath.Join(external, "needle.txt"),
			[]byte("findme outside\n"),
			0o644,
		))
		assert.NoError(t, os.Symlink(external, filepath.Join(root, "linked")))

		m := openGlobalSearch(t, view.NewEditor(root), "findme")
		out := stripANSI(m.View().Content)

		assert.Contains(t, out, "linked/needle.txt")
	})

	t.Run("searches open document text", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "a.txt")
		assert.NoError(t, os.WriteFile(path, []byte("disk\n"), 0o644))

		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		rope := doc.Text()
		cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
			core.TextChange(0, rope.LenChars(), "memory needle\n"),
		})
		assert.NoError(t, err)
		tx := core.NewTransaction(rope).
			WithChanges(cs).
			WithSelection(core.PointSelection(0))
		assert.NoError(t, e.Apply(tx))

		m := openGlobalSearch(t, e, "needle")
		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "a.txt")
	})
}

// globalSearchModel writes two files, opens the global-search picker, and types
// the query. sendKeyAndFeed drains the dynamic source's async feed per key
func globalSearchModel(
	t *testing.T, query string, files ...map[string]string,
) (ui.Model, *view.Editor) {
	t.Helper()
	cfg := map[string]string{
		"a.txt": "alpha\nfindme here\n",
		"b.txt": "beta\n",
	}
	if len(files) > 0 {
		cfg = files[0]
	}
	dir := t.TempDir()
	for name, text := range cfg {
		path := filepath.Join(dir, name)
		err := os.WriteFile(path, []byte(text), 0o644)
		assert.NoError(t, err)
	}
	e := view.NewEditor(dir)
	return openGlobalSearch(t, e, query), e
}

func openGlobalSearch(t *testing.T, e *view.Editor, query string) ui.Model {
	t.Helper()
	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(
		km, "global_search", m.PickerAction(files.NewGlobalSearchPicker),
		[]command.KeyEvent{char('s')},
	)
	m = resize(m, 120, 30)
	m = sendKeyAndFeed(m, 's')
	for _, ch := range query {
		m = sendKeyAndFeed(m, ch)
	}
	return m
}

var previewCellBgRE = regexp.MustCompile(`48;2;\d+;\d+;\d+`)

// bgAt returns the true-color background escape in effect at the first
// occurrence of needle in line, tracking SGR state left to right
func bgAt(line, needle string) string {
	idx := strings.Index(stripANSI(line), needle)
	bg, seen := "", 0
	for len(line) > 0 {
		if strings.HasPrefix(line, "\x1b[") {
			end := strings.IndexByte(line, 'm')
			if end < 0 {
				break
			}
			if m := previewCellBgRE.FindString(line[2:end]); m != "" {
				bg = m
			}
			line = line[end+1:]
			continue
		}
		if seen == idx {
			return bg
		}
		seen++
		line = line[1:]
	}
	return bg
}
