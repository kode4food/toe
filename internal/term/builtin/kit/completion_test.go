package kit_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/builtin/kit"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

func TestFileCompleter(t *testing.T) {
	t.Run("completes files in cwd by prefix", func(t *testing.T) {
		dir := t.TempDir()
		for _, name := range []string{"alpha.txt", "almond.txt", "beta.txt"} {
			assert.NoError(t, os.WriteFile(
				filepath.Join(dir, name), []byte("x"), 0o644,
			))
		}
		e := view.NewEditor(dir)

		var texts []string
		for _, c := range kit.FileCompleter(e, "al") {
			texts = append(texts, c.Text)
		}
		assert.Contains(t, texts, "alpha.txt")
		assert.Contains(t, texts, "almond.txt")
		assert.NotContains(t, texts, "beta.txt")
	})

	t.Run("unreadable base yields nothing", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		assert.Empty(t, kit.FileCompleter(e, "no_such_dir/x"))
	})
}

func TestStaticSig(t *testing.T) {
	sig := kit.StaticSig(kit.Sig(), "one", "two")
	e := view.NewEditor(t.TempDir())

	var texts []string
	for _, c := range sig.Completer.Complete(e, sig, "t") {
		texts = append(texts, c.Text)
	}
	assert.Equal(t, []string{"two"}, texts)
}

func TestFileSig(t *testing.T) {
	dir := t.TempDir()
	assert.NoError(t, os.WriteFile(filepath.Join(dir, "x.go"), []byte("x"), 0o644))
	sig := kit.FileSig(kit.Sig())
	e := view.NewEditor(dir)

	got := sig.Completer.Complete(e, sig, "x")
	assert.Contains(t, completionTexts(got), "x.go")
}

func completionTexts(cs []command.Completion) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.Text)
	}
	return out
}
