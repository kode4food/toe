package ui_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/ui"
)

func newTerminalPane(t *testing.T) (ui.Model, *ui.TerminalPane) {
	t.Helper()
	e := editorWithText(t, "hello toe")
	m := renderedModel(e)
	cont := m.TerminalAction()(e)
	assert.Nil(t, cont)
	tp, ok := e.Tree().Get(e.Tree().Focus()).(*ui.TerminalPane)
	assert.True(t, ok)
	t.Cleanup(func() { _ = tp.Stop() })
	waitForResize(t, tp)
	return m, tp
}

// TestTerminalPaneTitleUTF8 guards a charmbracelet/x/ansi parser bug (fixed
// via the go.mod replace to kode4food/x/ansi): C1 bytes ended OSC/DCS
// strings unconditionally, even mid-UTF-8, corrupting title glyphs like ✳
func TestTerminalPaneTitleUTF8(t *testing.T) {
	t.Run("ascii output renders normally", func(t *testing.T) {
		m, tp := newTerminalPane(t)
		tp.IngestOutput([]byte("hello\r\n\x1b[31m"))
		assert.Contains(t, stripANSI(m.View().Content), "hello")
	})

	t.Run("ground-state UTF-8 passes through", func(t *testing.T) {
		m, tp := newTerminalPane(t)
		// 日/語 contain C1-range bytes, but ground-state text is unaffected
		tp.IngestOutput([]byte("cafe umbrella 日本語\r\n"))
		assert.Contains(t, stripANSI(m.View().Content), "日本語")
	})

	t.Run("OSC 0 title with dangerous glyph", func(t *testing.T) {
		m, tp := newTerminalPane(t)
		// ✳ = E2 9C B3; 0x9C is also the 8-bit ST control code
		tp.IngestOutput(
			[]byte("\x1b]0;\xe2\x9c\xb3 Pick a choice\x07hello\r\n"),
		)
		assert.Equal(t, "✳ Pick a choice", tp.Title())
		assert.Contains(t, stripANSI(m.View().Content), "hello")
	})

	t.Run("OSC 2 title behaves the same as OSC 0", func(t *testing.T) {
		_, tp := newTerminalPane(t)
		tp.IngestOutput([]byte("\x1b]2;\xe2\x9c\xb3 title\x07hello"))
		assert.Equal(t, "✳ title", tp.Title())
	})

	t.Run("OSC title terminated with 8-bit ST", func(t *testing.T) {
		_, tp := newTerminalPane(t)
		tp.IngestOutput([]byte("\x1b]0;\xe2\x9c\xb3 title\x9chello"))
		assert.Equal(t, "✳ title", tp.Title())
	})

	t.Run("8-bit OSC introducer is also captured", func(t *testing.T) {
		m, tp := newTerminalPane(t)
		tp.IngestOutput(
			[]byte("\x9d0;\xe2\x9c\xb3 title\x9c\xe2\x9c\xb3 after\r\n"),
		)
		assert.Equal(t, "✳ title", tp.Title())
		assert.Contains(t, stripANSI(m.View().Content), "✳ after")
	})

	t.Run("glyph outside a string payload is text", func(t *testing.T) {
		m, tp := newTerminalPane(t)
		tp.IngestOutput([]byte("\xe2\x9c\xb3 done\r\n"))
		assert.Contains(t, stripANSI(m.View().Content), "✳ done")
	})

	t.Run("non-title OSC doesn't corrupt output", func(t *testing.T) {
		m, tp := newTerminalPane(t)
		tp.IngestOutput([]byte("\x1b]9;4;0;\xe2\x9c\xb3\x07hello\r\n"))
		assert.Equal(t, "", tp.Title())
		assert.Contains(t, stripANSI(m.View().Content), "hello")
	})

	t.Run("DCS payload doesn't corrupt output", func(t *testing.T) {
		m, tp := newTerminalPane(t)
		tp.IngestOutput([]byte("\x1bP\xe2\x9c\xb3\x1b\\hello\r\n"))
		assert.Contains(t, stripANSI(m.View().Content), "hello")
	})

	t.Run("standalone C1 byte is consumed silently", func(t *testing.T) {
		m, tp := newTerminalPane(t)
		tp.IngestOutput([]byte{'a', 0x9c, 'b'})
		assert.Contains(t, stripANSI(m.View().Content), "ab")
	})

	t.Run("invalid OSC byte doesn't corrupt output", func(t *testing.T) {
		m, tp := newTerminalPane(t)
		tp.IngestOutput([]byte("\x1b]9;a\xffb\x07hello\r\n"))
		assert.Contains(t, stripANSI(m.View().Content), "hello")
	})

	t.Run("title glyph split across writes", func(t *testing.T) {
		full := "\x1b]0;\xe2\x9c\xb3 title\x07after"
		splits := []int{1, 2, 5, len(full) - 2, len(full) - 1}
		for _, split := range splits {
			t.Run(splitName(split), func(t *testing.T) {
				_, tp := newTerminalPane(t)
				tp.IngestOutput([]byte(full[:split]))
				tp.IngestOutput([]byte(full[split:]))
				assert.Equal(t, "✳ title", tp.Title())
			})
		}
	})

	t.Run("clean title split across writes", func(t *testing.T) {
		// e-acute (C3 A9) has no byte in the C1 range
		full := "\x1b]0;caf\xc3\xa9 title\x07after"
		splits := []int{1, 2, 5, len(full) - 2, len(full) - 1}
		for _, split := range splits {
			t.Run(splitName(split), func(t *testing.T) {
				_, tp := newTerminalPane(t)
				tp.IngestOutput([]byte(full[:split]))
				tp.IngestOutput([]byte(full[split:]))
				assert.Equal(t, "café title", tp.Title())
			})
		}
	})

	t.Run("CJK title with C1 byte has no loss", func(t *testing.T) {
		_, tp := newTerminalPane(t)
		// 日/本 have C1-range bytes, but titles are decoded, not forwarded
		tp.IngestOutput(
			[]byte("\x1b]0;\xe6\x97\xa5\xe6\x9c\xac title\x07after"),
		)
		assert.Equal(t, "日本 title", tp.Title())
	})
}

func splitName(n int) string {
	return fmt.Sprintf("at byte %d", n)
}
