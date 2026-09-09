package ui_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

// BenchmarkBufferPickerRender renders a multi-column picker over many open
// buffers, the per-frame cost paid on every cursor move
func BenchmarkBufferPickerRender(b *testing.B) {
	root := b.TempDir()
	e := view.NewEditor(root)
	for i := range 500 {
		path := filepath.Join(root, fmt.Sprintf("file%d.go", i))
		src := fmt.Sprintf("package p\n\nfunc fn%d() {}\n", i)
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			b.Fatal(err)
		}
		if _, err := e.OpenFile(path); err != nil {
			b.Fatal(err)
		}
	}
	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(
		km, "buffer_picker", m.PickerAction(bufferPicker),
		[]command.KeyEvent{char('p')},
	)
	m = resize(m, 120, 40)
	m = sendKey(m, 'p')
	_ = m.View().Content

	b.ReportAllocs()
	for b.Loop() {
		m = sendSpecial(m, tea.KeyDown)
		_ = m.View().Content
	}
}
