package ui_test

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

// benchmarkScroll measures wheel handling and rendering while alternating
// direction so the anchor moves every frame
func benchmarkScroll(b *testing.B, source string) {
	root := b.TempDir()
	path := filepath.Join(root, "f.go")
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		b.Fatal(err)
	}
	e := view.NewEditor(root)
	if _, err := e.OpenFile(path); err != nil {
		b.Fatal(err)
	}
	m := tea.Model(resize(ui.New(e, command.NewKeymaps()), 100, 40))
	_ = m.(ui.Model).View().Content

	down := tea.MouseWheelMsg{Button: tea.MouseWheelDown}
	up := tea.MouseWheelMsg{Button: tea.MouseWheelUp}

	b.ReportAllocs()
	i := 0
	for b.Loop() {
		msg := tea.Msg(down)
		if i%20 >= 10 {
			msg = up
		}
		m, _ = m.Update(msg)
		_ = m.(ui.Model).View().Content
		i++
	}
}

func BenchmarkScrollLargeFile(b *testing.B) {
	benchmarkScroll(b, largeGoSource(2000))
}

func BenchmarkScrollSmallFile(b *testing.B) {
	benchmarkScroll(b, largeGoSource(40))
}

// BenchmarkScrollTwoPanes scrolls with two panes showing two different
// documents — exercises the per-document render caches under split layout
func BenchmarkScrollTwoPanes(b *testing.B) {
	root := b.TempDir()
	pathA := filepath.Join(root, "a.go")
	pathB := filepath.Join(root, "b.go")
	if err := os.WriteFile(pathA, []byte(largeGoSource(2000)), 0o644); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(pathB, []byte(largeGoSource(2000)), 0o644); err != nil {
		b.Fatal(err)
	}
	e := view.NewEditor(root)
	if _, err := e.OpenFile(pathA); err != nil {
		b.Fatal(err)
	}
	docB, err := e.SwitchOrOpenDoc(pathB)
	if err != nil {
		b.Fatal(err)
	}
	e.ResizeTree(100, 40)
	if _, ok := e.VSplit(docB.ID()); !ok {
		b.Fatal("vsplit failed")
	}
	m := tea.Model(resize(ui.New(e, command.NewKeymaps()), 100, 40))
	_ = m.(ui.Model).View().Content

	down := tea.MouseWheelMsg{Button: tea.MouseWheelDown}
	up := tea.MouseWheelMsg{Button: tea.MouseWheelUp}

	b.ReportAllocs()
	i := 0
	for b.Loop() {
		msg := tea.Msg(down)
		if i%20 >= 10 {
			msg = up
		}
		m, _ = m.Update(msg)
		_ = m.(ui.Model).View().Content
		i++
	}
}
