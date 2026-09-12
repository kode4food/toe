package ui_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/vcs"
	"github.com/kode4food/toe/internal/view"
)

type benchmarkDiffSource struct{ pathPickerSource }

const benchmarkWorkingVariant = 1

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

// BenchmarkDiffPreview redraws long diff lines while scrolling
func BenchmarkDiffPreview(b *testing.B) {
	b.Setenv("XDG_CONFIG_HOME", b.TempDir())
	root := b.TempDir()
	path := filepath.Join(root, "long.txt")
	text := strings.Repeat(strings.Repeat("abcd ", 4000)+"\n", 80)
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		b.Fatal(err)
	}
	e := view.NewEditor(root)
	s := vcs.Attach(e)
	b.Cleanup(s.Close)
	src := &benchmarkDiffSource{path: path}
	m := ui.New(e, command.NewKeymaps()).
		WithInitialPicker(func(e *view.Editor) *ui.Picker {
			return ui.NewPicker(e, src)
		})
	m = updateAndFeed(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	if !strings.Contains(stripANSI(m.View().Content), "+ abcd") {
		b.Fatal("diff preview not rendered")
	}
	i := 0
	b.ReportAllocs()
	for b.Loop() {
		button := tea.MouseWheelDown
		if i%20 >= 10 {
			button = tea.MouseWheelUp
		}
		m = mouse(m, tea.MouseWheelMsg{X: 100, Y: 10, Button: button})
		_ = m.View().Content
		i++
	}
}

func (b *benchmarkDiffSource) Load() ui.PickerLoad {
	load := b.pathPickerSource.Load()
	load.Items[0].DiffPreview = true
	load.Items[0].DiffKind = view.FileChangeAdded
	load.Items[0].Location.Target.Variant = benchmarkWorkingVariant
	return load
}
