package ui_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

type (
	terminalOutputBench struct {
		name      string
		split     bool
		relative  bool
		scrollOpt bool
	}

	terminalOutputRenderer struct {
		out bytes.Buffer
		r   *uv.TerminalRenderer
		w   int
		h   int
	}
)

func BenchmarkTerminalScrollOutput(b *testing.B) {
	cases := []terminalOutputBench{
		{name: "single-pane", scrollOpt: true},
		{name: "single-pane-relative", relative: true, scrollOpt: true},
		{name: "single-pane-no-scroll-opt", scrollOpt: false},
		{name: "vertical-split", split: true, scrollOpt: true},
		{
			name:      "vertical-split-relative",
			split:     true,
			relative:  true,
			scrollOpt: true,
		},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			benchmarkTerminalScrollOutput(b, tc)
		})
	}
}

func benchmarkTerminalScrollOutput(b *testing.B, tc terminalOutputBench) {
	m := terminalScrollModel(b, tc)
	tr := newTerminalOutputRenderer(100, 40, tc.scrollOpt)

	if err := tr.Render(m.View().Content); err != nil {
		b.Fatal(err)
	}
	tr.ResetOutput()

	down := tea.MouseWheelMsg{Button: tea.MouseWheelDown}
	up := tea.MouseWheelMsg{Button: tea.MouseWheelUp}

	var bytesOut int
	var scrollSeqs int
	i := 0
	b.ReportAllocs()
	for b.Loop() {
		msg := tea.Msg(down)
		if i%20 >= 10 {
			msg = up
		}
		next, _ := tea.Model(m).Update(msg)
		m = next.(ui.Model)

		if err := tr.Render(m.View().Content); err != nil {
			b.Fatal(err)
		}
		out := tr.ResetOutput()
		bytesOut += len(out)
		scrollSeqs += countScrollSequences(out)
		i++
	}
	b.ReportMetric(float64(bytesOut)/float64(b.N), "ansi_B/op")
	b.ReportMetric(float64(scrollSeqs)/float64(b.N), "scrollseq/op")
}

func terminalScrollModel(b *testing.B, tc terminalOutputBench) ui.Model {
	b.Helper()
	root := b.TempDir()
	pathA := filepath.Join(root, "a.go")
	pathB := filepath.Join(root, "b.go")
	if err := os.WriteFile(
		pathA, []byte(largeGoSource(2000)), 0o644,
	); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(
		pathB, []byte(largeGoSource(2000)), 0o644,
	); err != nil {
		b.Fatal(err)
	}

	e := view.NewEditor(root)
	if tc.relative {
		e.Options().LineNumber = view.LineNumberRelative
	}
	if _, err := e.OpenFile(pathA); err != nil {
		b.Fatal(err)
	}
	if tc.split {
		docB, err := e.SwitchOrOpenDoc(pathB)
		if err != nil {
			b.Fatal(err)
		}
		e.ResizeTree(100, 40)
		if _, ok := e.VSplit(docB.ID()); !ok {
			b.Fatal("vsplit failed")
		}
	}
	return resize(ui.New(e, command.NewKeymaps()), 100, 40)
}

func newTerminalOutputRenderer(
	w, h int, scrollOpt bool,
) *terminalOutputRenderer {
	tr := &terminalOutputRenderer{w: w, h: h}
	tr.r = uv.NewTerminalRenderer(&tr.out, []string{"TERM=xterm-256color"})
	tr.r.SetFullscreen(true)
	tr.r.SetScrollOptim(scrollOpt)
	tr.r.Resize(w, h)
	return tr
}

func (t *terminalOutputRenderer) Render(content string) error {
	buf := uv.NewScreenBuffer(t.w, t.h)
	uv.NewStyledString(content).Draw(buf, buf.Bounds())
	t.r.Render(buf.RenderBuffer)
	return t.r.Flush()
}

func (t *terminalOutputRenderer) ResetOutput() string {
	out := t.out.String()
	t.out.Reset()
	return out
}

func countScrollSequences(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] != '\x1b' {
			continue
		}
		if i+1 < len(s) && s[i+1] == 'M' {
			n++
			i++
			continue
		}
		if i+1 >= len(s) || s[i+1] != '[' {
			continue
		}
		j := i + 2
		for ; j < len(s); j++ {
			ch := s[j]
			if ch < '@' || ch > '~' {
				continue
			}
			if ch == 'L' || ch == 'M' || ch == 'S' ||
				ch == 'T' || ch == 'r' {
				n++
			}
			i = j
			break
		}
	}
	return n
}
