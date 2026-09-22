package ui_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

func largeGoSource(funcs int) string {
	var b strings.Builder
	b.WriteString("package main\n\nimport \"fmt\"\n\n")
	for i := range funcs {
		_, _ = fmt.Fprintf(&b, "// fn%d does something with %d\n", i, i)
		_, _ = fmt.Fprintf(&b, "func fn%d(x int) int {\n", i)
		_, _ = fmt.Fprintf(&b, "\ts := \"value %d\"\n", i)
		_, _ = fmt.Fprintf(&b, "\tfmt.Println(s, x)\n")
		_, _ = fmt.Fprintf(&b, "\treturn x * %d\n}\n\n", i)
	}
	return b.String()
}

// BenchmarkRenderLargeFileSteady renders a large highlighted file without
// edits, the per-frame cost paid on every mouse-scroll tick
func BenchmarkRenderLargeFileSteady(b *testing.B) {
	root := b.TempDir()
	path := filepath.Join(root, "big.go")
	err := os.WriteFile(path, []byte(largeGoSource(2000)), 0o644)
	if err != nil {
		b.Fatal(err)
	}
	e := view.NewEditor(root)
	if _, err := e.OpenFile(path); err != nil {
		b.Fatal(err)
	}
	m := resize(ui.New(e, command.NewKeymaps()), 100, 40)
	// prime caches (first View tokenizes)
	_ = m.View().Content

	b.ReportAllocs()
	for b.Loop() {
		_ = m.View().Content
	}
}

// BenchmarkRenderLargeFileScrollbar renders the same file with the scrollbar
// on and search highlights active, the worst case for mark collection
func BenchmarkRenderLargeFileScrollbar(b *testing.B) {
	root := b.TempDir()
	path := filepath.Join(root, "big.go")
	err := os.WriteFile(path, []byte(largeGoSource(2000)), 0o644)
	if err != nil {
		b.Fatal(err)
	}
	e := view.NewEditor(root)
	v, err := e.OpenFile(path)
	if err != nil {
		b.Fatal(err)
	}
	e.Options().Scrollbar = true
	e.FocusedDocument().ShowSearchHighlights(v.ID())
	e.Registers().Set('/', "fmt")
	m := resize(ui.New(e, command.NewKeymaps()), 100, 40)
	_ = m.View().Content

	b.ReportAllocs()
	for b.Loop() {
		v.MarkDirty()
		_ = m.View().Content
	}
}
