package action_test

import (
	"strings"
	"testing"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

// Worst-case wheel tick over a 1MB single-line document. The offset deepens
// every iteration, so ns/op averages ticks at increasing scroll depths
func BenchmarkScrollViewColumns(b *testing.B) {
	line := strings.Repeat("x", 1_000_000)
	e := view.NewEditor(b.TempDir())
	doc, ok := e.FocusedDocument()
	if !ok {
		b.Fatal("missing focused document")
	}
	rope := doc.Text()
	cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
		core.TextChange(0, 0, line),
	})
	if err != nil {
		b.Fatal(err)
	}
	tx := core.NewTransaction(rope).
		WithChanges(cs).
		WithSelection(core.PointSelection(0))
	if err := e.Apply(tx); err != nil {
		b.Fatal(err)
	}
	v, ok := e.FocusedView()
	if !ok {
		b.Fatal("missing focused view")
	}
	b.ReportAllocs()
	for b.Loop() {
		action.ScrollViewColumns(e, v, 3, false)
	}
}
