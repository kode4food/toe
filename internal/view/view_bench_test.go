package view_test

import (
	"strings"
	"testing"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

func BenchmarkVisualColumn(b *testing.B) {
	line := strings.Repeat("abcd", 1250) // 5000 ASCII columns
	doc := core.NewRope(line)
	sel := core.PointSelection(5000)
	e := view.NewEditor(b.TempDir())
	v, ok := e.FocusedView()
	if !ok {
		b.Fatal("missing focused view")
	}
	b.ReportAllocs()
	for b.Loop() {
		v.EnsureCursorVisibleHorizontal(doc, sel, 80, 4, 5)
	}
}
