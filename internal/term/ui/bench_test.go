package ui_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

// BenchmarkRenderImage redraws a full image pane
func BenchmarkRenderImage(b *testing.B) {
	root := b.TempDir()
	path := writeRenderImage(b, root, 1600, 900, nil)
	e := view.NewEditor(root)
	pane, err := ui.NewImagePane(e, path)
	if err != nil {
		b.Fatal(err)
	}
	e.ReplacePane(e.Tree().Focus(), pane)
	m := ui.New(e, command.NewKeymaps())
	m2, cmd := m.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	m = feedImageMsgs(m2.(ui.Model), cmd)

	b.ReportAllocs()
	for b.Loop() {
		pane.MarkDirty()
		_ = m.View().Content
	}
}

// BenchmarkRenderLongLine renders a single very long line with the cursor
// scrolled to the far right, exercising the horizontal-scroll render path
func BenchmarkRenderLongLine(b *testing.B) {
	root := b.TempDir()
	path := filepath.Join(root, "long.txt")
	line := strings.Repeat("abcd ", 4000) // 20000 columns, one logical line
	if err := os.WriteFile(path, []byte(line), 0o644); err != nil {
		b.Fatal(err)
	}
	e := view.NewEditor(root)
	v, err := e.OpenFile(path)
	if err != nil {
		b.Fatal(err)
	}
	doc, _ := e.FocusedDocument()
	doc.SetSelectionFor(v.ID(), core.PointSelection(len(line)))

	m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

	b.ReportAllocs()
	for b.Loop() {
		_ = m.View().Content
	}
}

// BenchmarkRenderPickerPreview renders a large syntax-highlighted in-memory
// buffer preview with a highlighted cursor line. This uncached worst case
// re-renders and re-tokenizes the preview every frame
func BenchmarkRenderPickerPreview(b *testing.B) {
	root := b.TempDir()
	path := filepath.Join(root, "big.go")
	var sb strings.Builder
	sb.WriteString("package main\n\nimport \"fmt\"\n\n")
	for range 600 {
		sb.WriteString("func f() { fmt.Println(\"x\") }\n")
	}
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		b.Fatal(err)
	}
	e := view.NewEditor(root)
	if _, err := e.OpenFile(path); err != nil {
		b.Fatal(err)
	}
	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(
		km, "buffer_picker", m.PickerAction(bufferPicker),
		[]command.KeyEvent{char('b')},
	)
	m = resize(m, 120, 40)
	m = sendKey(m, 'b')
	for _, ch := range "big" {
		m = sendKey(m, ch)
	}
	_ = m.View().Content // prime

	b.ReportAllocs()
	for b.Loop() {
		_ = m.View().Content
	}
}

// BenchmarkRenderEmptyPicker renders the picker no-results state
func BenchmarkRenderEmptyPicker(b *testing.B) {
	e := view.NewEditor(b.TempDir())
	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(
		km, "command_picker",
		m.PickerAction(func(e *view.Editor) *ui.Picker {
			return ui.CommandPalettePicker(e, km)
		}),
		[]command.KeyEvent{char('p')},
	)
	m = resize(m, 80, 24)
	m = sendKey(m, 'p')
	for _, ch := range "no-match" {
		m = sendKey(m, ch)
	}
	_ = m.View().Content

	b.ReportAllocs()
	for b.Loop() {
		_ = m.View().Content
	}
}

// BenchmarkRenderLongLineCursorStart renders a long line with the cursor at the
// start (no horizontal scroll) — the common case where only the first columns
// are visible yet the whole line is otherwise processed
func BenchmarkRenderLongLineCursorStart(b *testing.B) {
	root := b.TempDir()
	path := filepath.Join(root, "long.txt")
	line := strings.Repeat("abcd ", 4000)
	if err := os.WriteFile(path, []byte(line), 0o644); err != nil {
		b.Fatal(err)
	}
	e := view.NewEditor(root)
	v, err := e.OpenFile(path)
	if err != nil {
		b.Fatal(err)
	}
	doc, _ := e.FocusedDocument()
	doc.SetSelectionFor(v.ID(), core.PointSelection(0))

	m := resize(ui.New(e, command.NewKeymaps()), 80, 24)

	b.ReportAllocs()
	for b.Loop() {
		_ = m.View().Content
	}
}

func benchTerminal(b *testing.B, fill string) {
	e := view.NewEditor(b.TempDir())
	e.Options().Theme = view.DefaultTheme
	m := resize(ui.New(e, command.NewKeymaps()), 80, 24)
	_ = m.TerminalAction()(e)
	tp := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
	b.Cleanup(func() { _ = tp.Stop() })
	tp.IngestOutput([]byte(fill))
	_ = m.View().Content // prime

	b.ReportAllocs()
	for b.Loop() {
		tp.MarkDirty()
		_ = m.View().Content
	}
}

// BenchmarkRenderTerminal redraws a full terminal screen every frame, the worst
// case for the emulator-to-buffer cell copy and per-cell style conversion
func BenchmarkRenderTerminal(b *testing.B) {
	var sb strings.Builder
	for i := range 22 {
		_, _ = fmt.Fprintf(&sb, "line %2d %s\r\n", i, strings.Repeat("x", 60))
	}
	benchTerminal(b, sb.String())
}

// BenchmarkRenderTerminalColored exercises the realistic case of many short
// same-style runs (coloured ls/grep/build output), where the style memo has a
// lower but still winning hit rate
func BenchmarkRenderTerminalColored(b *testing.B) {
	colors := []string{"31", "32", "33", "34", "36"}
	var sb strings.Builder
	for range 22 {
		for c := range 9 {
			_, _ = fmt.Fprintf(&sb, "\x1b[%sm%s", colors[c%len(colors)],
				strings.Repeat("w", 8))
		}
		sb.WriteString("\x1b[0m\r\n")
	}
	benchTerminal(b, sb.String())
}
