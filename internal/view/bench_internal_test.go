package view

import (
	"strings"
	"testing"

	"github.com/kode4food/toe/internal/core"
)

func BenchmarkVisualColumn(b *testing.B) {
	line := strings.Repeat("abcd", 1250) // 5000 ASCII columns
	doc := core.NewRope(line)
	b.ReportAllocs()
	for b.Loop() {
		_ = visualColumn(doc, 0, 5000, 4)
	}
}
