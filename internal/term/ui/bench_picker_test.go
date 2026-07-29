package ui_test

import (
	"fmt"
	"testing"
)

func BenchmarkPickerFeedLoad(b *testing.B) {
	// the feed arrives in batches, and every batch refilters the whole list
	for _, n := range []int{1000, 10000, 50000} {
		paths := benchPickerPaths(n)
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				m := feedPickerModel(b, paths)
				b.StartTimer()
				sendKeyAndFeed(m, 'p')
			}
		})
	}
}

func BenchmarkPickerTyping(b *testing.B) {
	for _, n := range []int{1000, 10000, 50000} {
		paths := benchPickerPaths(n)
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				m := sendKeyAndFeed(feedPickerModel(b, paths), 'p')
				b.StartTimer()
				for _, ch := range "picker" {
					m = sendKey(m, ch)
				}
			}
		})
	}
}

func benchPickerPaths(n int) []string {
	seed := []string{
		"internal/term/ui/picker-fuzzy.go",
		"internal/view/action/selection-lines.go",
		"internal/core/rope-balance.go",
		"internal/lsp/capabilities.go",
		"cmd/toe/internal/app.go",
		"docs/content/docs/architecture.md",
		"internal/term/builtin/files/open.go",
		"internal/vcs/git/diff.go",
	}
	out := make([]string, 0, n)
	for i := 0; len(out) < n; i++ {
		for _, p := range seed {
			if len(out) == n {
				break
			}
			out = append(out, fmt.Sprintf("pkg%d/%s", i, p))
		}
	}
	return out
}
