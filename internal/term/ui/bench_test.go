package ui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

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

// BenchmarkRenderPickerPreview renders a buffer-picker frame whose preview is a
// large syntax-highlighted in-memory document with a highlighted cursor line.
// The buffer picker is the worst case: an in-memory doc is not cached, so the
// preview is re-rendered (and re-tokenized) every frame
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
		km, "buffer_picker", m.PickerAction(ui.BufferPicker),
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
