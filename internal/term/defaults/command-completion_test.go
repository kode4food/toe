package defaults_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/defaults"
	"github.com/kode4food/toe/internal/term/ui"
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
		km := command.NewKeymaps()
		e := view.NewEditor(dir)
		_, _ = defaults.RegisterDefaults(ui.New(e, km), km)

		cmd, ok := km.ResolveCommand("open")
		assert.True(t, ok)
		got := cmd.Signature.Completer.Complete(e, cmd.Signature, "al")

		var texts []string
		for _, c := range got {
			texts = append(texts, c.Text)
		}
		assert.Contains(t, texts, "alpha.txt")
		assert.Contains(t, texts, "almond.txt")
		assert.NotContains(t, texts, "beta.txt")
	})

	t.Run("unreadable base yields nothing", func(t *testing.T) {
		km := command.NewKeymaps()
		e := view.NewEditor(t.TempDir())
		_, _ = defaults.RegisterDefaults(ui.New(e, km), km)

		cmd, ok := km.ResolveCommand("open")
		assert.True(t, ok)
		got := cmd.Signature.Completer.Complete(
			e, cmd.Signature, "no_such_dir/x",
		)
		assert.Empty(t, got)
	})
}
