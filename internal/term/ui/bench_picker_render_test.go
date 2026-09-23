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
	"github.com/kode4food/toe/internal/testutil"
	"github.com/kode4food/toe/internal/vcs"
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

// BenchmarkDiffPreview redraws long diff lines while scrolling
func BenchmarkDiffPreview(b *testing.B) {
	testutil.RequireGit(b)
	b.Setenv("XDG_CONFIG_HOME", b.TempDir())
	root := testutil.GitRepo(b)
	committed := strings.Repeat(strings.Repeat("abcd ", 4000)+"\n", 80)
	edited := strings.Repeat(strings.Repeat("abcd ", 4000)+"z\n", 80)
	path := testutil.GitCommitFile(b, root, "long.txt", []byte(committed))
	testutil.WriteFile(b, path, []byte(edited))

	e := view.NewEditor(root)
	s := vcs.Attach(e)
	b.Cleanup(s.Close)
	m := ui.New(e, command.NewKeymaps()).
		WithInitialPicker(ui.NewChangedFilePicker)
	m = updateAndFeed(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	if !strings.Contains(stripANSI(m.View().Content), "- abcd") {
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
