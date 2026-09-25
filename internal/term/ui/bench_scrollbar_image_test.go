package ui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

type scrollbarBenchState struct {
	model  ui.Model
	editor *view.Editor
	view   *view.View
	doc    *view.Document
}

const (
	scrollbarBenchLines      = 100
	scrollbarBenchThumbLines = 300
	scrollbarBenchBurstMoves = 4
	scrollbarBenchThumbTop   = 23
	scrollbarBenchThumbNext  = 24
	kittyPNGFormatField      = "f=100"
)

func newScrollbarBenchState(b *testing.B, lines int) *scrollbarBenchState {
	b.Helper()
	b.Setenv("KITTY_WINDOW_ID", "1")
	root := b.TempDir()
	path := filepath.Join(root, "lines.txt")
	if err := os.WriteFile(
		path, []byte(numberedLines(lines)), 0o644,
	); err != nil {
		b.Fatal(err)
	}
	e := view.NewEditor(root)
	v, err := e.OpenFile(path)
	if err != nil {
		b.Fatal(err)
	}
	e.Options().Scrollbar = true
	m := ui.New(e, command.NewKeymaps())
	m2, _ := m.Update(tea.WindowSizeMsg{
		Width:  scrollbarWidth,
		Height: scrollbarHeight,
	})
	m = m2.(ui.Model)
	m2, _ = m.Update(scrollbarTestCell())
	m = m2.(ui.Model)
	_ = m.View().Content
	m2, cmd := m.Update(tea.FocusMsg{})
	m, _ = collectModelRawMsgs(m2.(ui.Model), cmd)
	return &scrollbarBenchState{
		model:  m,
		editor: e,
		view:   v,
		doc:    e.FocusedDocument(),
	}
}

// BenchmarkScrollbarKitty measures single moves and queued four-move bursts
func BenchmarkScrollbarKitty(b *testing.B) {
	for _, tc := range []struct {
		name  string
		moves int
	}{
		{name: "single", moves: 1},
		{name: "burst4", moves: scrollbarBenchBurstMoves},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.StopTimer()
			s := newScrollbarBenchState(b, scrollbarBenchLines)
			b.StartTimer()
			b.ReportAllocs()
			var pending [scrollbarBenchBurstMoves]tea.Cmd
			var transmits, kittyBytes int
			i := 0
			for b.Loop() {
				for j := range tc.moves {
					if i%2 == 0 {
						action.MoveDown(s.editor)
					} else {
						action.MoveUp(s.editor)
					}
					s.view.MarkDirty()
					_ = s.model.View().Content
					m, cmd := s.model.Update(tea.FocusMsg{})
					s.model = m.(ui.Model)
					pending[j] = cmd
				}
				for j := range tc.moves {
					var raws []string
					s.model, raws = collectModelRawMsgs(s.model, pending[j])
					for _, raw := range raws {
						kittyBytes += len(raw)
						if strings.Contains(raw, kittyPNGFormatField) {
							transmits++
						}
					}
				}
				i++
			}
			b.ReportMetric(float64(transmits)/float64(b.N), "transmits/op")
			b.ReportMetric(float64(kittyBytes)/float64(b.N), "kitty_B/op")
		})
	}
}

// BenchmarkScrollbarKittyThumb measures pixel-identical free scrolling
func BenchmarkScrollbarKittyThumb(b *testing.B) {
	b.StopTimer()
	s := newScrollbarBenchState(b, scrollbarBenchThumbLines)
	var anchors [2]int
	for i, line := range [...]int{
		scrollbarBenchThumbTop, scrollbarBenchThumbNext,
	} {
		at, err := s.doc.Text().LineToChar(line)
		if err != nil {
			b.Fatal(err)
		}
		anchors[i] = at
	}
	s.view.BeginFreeScroll(s.doc.Revision(), s.doc.SelectionFor(s.view.ID()))
	s.view.SetOffset(view.Position{Anchor: anchors[0]})
	_ = s.model.View().Content
	m, cmd := s.model.Update(tea.FocusMsg{})
	s.model, _ = collectModelRawMsgs(m.(ui.Model), cmd)
	b.StartTimer()
	b.ReportAllocs()
	var transmits, kittyBytes int
	i := 0
	for b.Loop() {
		s.view.SetOffset(view.Position{Anchor: anchors[(i+1)%2]})
		_ = s.model.View().Content
		m, cmd := s.model.Update(tea.FocusMsg{})
		var raws []string
		s.model, raws = collectModelRawMsgs(m.(ui.Model), cmd)
		for _, raw := range raws {
			kittyBytes += len(raw)
			if strings.Contains(raw, kittyPNGFormatField) {
				transmits++
			}
		}
		i++
	}
	b.ReportMetric(float64(transmits)/float64(b.N), "transmits/op")
	b.ReportMetric(float64(kittyBytes)/float64(b.N), "kitty_B/op")
}
