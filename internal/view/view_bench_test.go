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
	v := e.FocusedView()
	if v == nil {
		b.Fatal("missing focused view")
	}
	cs := &view.CursorScroll{
		Doc:       doc,
		Selection: sel,
		Width:     80,
		TabWidth:  4,
		ScrollOff: 5,
	}
	b.ReportAllocs()
	for b.Loop() {
		v.EnsureCursorVisibleHorizontal(cs)
	}
}
