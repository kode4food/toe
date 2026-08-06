package ui_test

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/term/builtin"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

// BenchmarkCompletionTyping re-filters an open completion popup on every
// keystroke, the same shape as BenchmarkPickerTyping
func BenchmarkCompletionTyping(b *testing.B) {
	for _, n := range []int{1000, 10000, 50000} {
		items := benchCompletionItems(n)
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				m := openCompletionPopup(b, items)
				b.StartTimer()
				for _, ch := range "println" {
					m2, cmd := m.Update(tea.KeyPressMsg{
						Code: ch, Text: string(ch),
					})
					m = m2.(ui.Model)
					m = feedCmds(m, cmd)
				}
			}
		})
	}
}

func openCompletionPopup(b testing.TB, items []*view.CompletionItem) ui.Model {
	b.Helper()
	e := view.NewEditor(b.TempDir())
	e.SetMode(view.ModeInsert)
	ctl := &completionController{editor: e, items: items}
	e.SetLanguageServerController(ctl)
	km := command.NewKeymaps()
	m := ui.New(e, km)
	if _, err := builtin.Register(m, km); err != nil {
		b.Fatal(err)
	}
	m = resize(m, 80, 24)
	return sendModifiedAndFeed(m, 'x', tea.ModCtrl)
}

func benchCompletionItems(n int) []*view.CompletionItem {
	kinds := []string{"function", "method", "variable", "field", "constant"}
	out := make([]*view.CompletionItem, n)
	for i := range n {
		label := fmt.Sprintf("printItem%d", i)
		out[i] = &view.CompletionItem{
			Label:  label,
			Insert: label,
			Kind:   kinds[i%len(kinds)],
			Sort:   fmt.Sprintf("%06d", i),
		}
	}
	return out
}
