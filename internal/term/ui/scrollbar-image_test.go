package ui_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"

	"github.com/stretchr/testify/assert"
)

func TestScrollbarImage(t *testing.T) {
	t.Run("falls back to glyphs without graphics", func(t *testing.T) {
		for _, key := range []string{
			"KITTY_WINDOW_ID", "TERM", "TERM_PROGRAM",
		} {
			t.Setenv(key, "")
		}
		m := scrollbarImageModel(t)

		content := m.View().Content

		assert.False(t, strings.ContainsRune(content, tui.PlaceholderRune))
		assert.True(t, strings.ContainsAny(content, scrollbarThinGlyphs))
	})

	t.Run("paints placeholders when ready", func(t *testing.T) {
		t.Setenv("KITTY_WINDOW_ID", "1")
		m := scrollbarImageModel(t)

		content := m.View().Content

		assert.True(t, strings.ContainsRune(content, tui.PlaceholderRune))
	})

	t.Run("marks land where their line falls", func(t *testing.T) {
		t.Setenv("KITTY_WINDOW_ID", "1")
		e := scrollbarImageEditor(t)
		doc := e.FocusedDocument()
		at, err := doc.Text().LineToChar(95)
		assert.NoError(t, err)
		doc.ReplaceDiagnostics("test", []view.Diagnostic{{
			Range:    core.Span{From: at, To: at + 1},
			Severity: view.DiagnosticSeverityError,
		}})

		img := decodeScrollbarImage(t, scrollbarImageTransmit(t, e))

		var r, g, b uint8
		_, err = fmt.Sscanf(scrollbarErrorFg, "38;2;%d;%d;%d", &r, &g, &b)
		assert.NoError(t, err)
		rows := scrollbarMarkedRows(img, color.RGBA{
			R: r, G: g, B: b, A: 255,
		})
		assert.NotEmpty(t, rows)
		height := img.Bounds().Dy()
		for _, y := range rows {
			assert.Greater(t, y, height*3/4)
		}
	})

	t.Run("renders at the reported cell size", func(t *testing.T) {
		t.Setenv("KITTY_WINDOW_ID", "1")
		e := scrollbarImageEditor(t)
		raws := scrollbarImageTransmit(t, e)

		img := decodeScrollbarImage(t, raws)

		assert.Equal(t, scrollbarTestCell().Width, img.Bounds().Dx())
		assert.Equal(t,
			scrollbarTestCell().Height*scrollbarRows, img.Bounds().Dy(),
		)
	})

	t.Run("one line scroll retransmits", func(t *testing.T) {
		t.Setenv("KITTY_WINDOW_ID", "1")
		e := scrollbarImageEditor(t)
		m, _ := scrollbarImageModelFor(t, e, scrollbarTestCell())

		before := scrollbarRowOf(t, m.View().Content)
		m, raw := scrollbarScroll(m, e)
		after := scrollbarRowOf(t, m.View().Content)

		// the thumb moved less than a cell, so only pixels can carry it
		assert.Equal(t, before, after)
		assert.NotEmpty(t, raw)
	})
}

// a height no whole number of slots divides, exercising the mapping
func scrollbarTestCell() uv.CellSizeEvent {
	return uv.CellSizeEvent{Width: 9, Height: 19}
}

func scrollbarImageEditor(t *testing.T) *view.Editor {
	t.Helper()
	// a line is worth several pixel rows but a fraction of a cell
	e := editorWithText(t, numberedLines(100))
	e.Options().Scrollbar = true
	return e
}

func scrollbarImageModel(t *testing.T) ui.Model {
	t.Helper()
	m, _ := scrollbarImageModelFor(
		t, scrollbarImageEditor(t), scrollbarTestCell(),
	)
	return m
}

func scrollbarImageTransmit(t *testing.T, e *view.Editor) []string {
	t.Helper()
	_, raws := scrollbarImageModelFor(t, e, scrollbarTestCell())
	return raws
}

func scrollbarImageModelFor(
	t *testing.T, e *view.Editor, cell uv.CellSizeEvent,
) (ui.Model, []string) {
	t.Helper()
	m := ui.New(e, command.NewKeymaps())
	m2, _ := m.Update(tea.WindowSizeMsg{
		Width: scrollbarWidth, Height: scrollbarHeight,
	})
	m = m2.(ui.Model)
	// nothing is drawn as an image until a cell size is known
	m2, _ = m.Update(cell)
	m = m2.(ui.Model)
	// the transmit reads the bar this pass caches
	_ = m.View().Content
	m2, cmd := m.Update(tea.FocusMsg{})
	m = m2.(ui.Model)
	m, raws := collectModelRawMsgs(m, cmd)
	return m, raws
}

func scrollbarScroll(m ui.Model, e *view.Editor) (ui.Model, []string) {
	action.MoveDown(e)
	e.FocusedView().MarkDirty()
	m2, cmd := m.Update(tea.FocusMsg{})
	m = m2.(ui.Model)
	m, _ = collectModelRawMsgs(m, cmd)
	_ = m.View().Content
	m2, cmd = m.Update(tea.FocusMsg{})
	m = m2.(ui.Model)
	return collectModelRawMsgs(m, cmd)
}

func scrollbarRowOf(t *testing.T, content string) string {
	t.Helper()
	cells := scrollbarCells(t, content, scrollbarRows)
	var out strings.Builder
	for _, cell := range cells {
		out.WriteRune(cell.glyph)
	}
	return out.String()
}

func decodeScrollbarImage(t *testing.T, raws []string) image.Image {
	t.Helper()
	for _, raw := range raws {
		_, payload, ok := strings.Cut(raw, ";")
		if !ok {
			continue
		}
		data, err := base64.StdEncoding.DecodeString(
			strings.TrimSuffix(payload, "\x1b\\"),
		)
		assert.NoError(t, err)
		img, err := png.Decode(bytes.NewReader(data))
		assert.NoError(t, err)
		return img
	}
	t.Fatal("no image payload was transmitted")
	return nil
}

func scrollbarMarkedRows(img image.Image, want color.RGBA) []int {
	var out []int
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		r, g, bl, _ := img.At(b.Min.X, y).RGBA()
		got := color.RGBA{
			R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(bl >> 8), A: 255,
		}
		if got == want {
			out = append(out, y-b.Min.Y)
		}
	}
	return out
}
