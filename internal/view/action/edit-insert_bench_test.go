package action_test

import (
	"path/filepath"
	"testing"

	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

// One typed character in a Go document, the per-keystroke insert path that
// consults the document's language for auto-pairs
func BenchmarkInsertChar(b *testing.B) {
	b.Setenv("XDG_CONFIG_HOME", b.TempDir())
	dir := b.TempDir()
	path := filepath.Join(dir, "a.go")
	testutil.WriteFile(b, path, []byte("package a\n"))
	e := view.NewEditor(dir)
	if _, err := e.OpenFile(path); err != nil {
		b.Fatal(err)
	}
	e.SetMode(view.ModeInsert)
	b.ReportAllocs()
	for b.Loop() {
		action.InsertChar(e, 'x')
	}
}
