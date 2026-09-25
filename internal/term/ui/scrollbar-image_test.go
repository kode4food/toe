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
	"time"

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

// longer than the wait a quiet terminal is given to answer
const scrollbarQuietWait = 400 * time.Millisecond

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

	t.Run("draws nothing until the image is up", func(t *testing.T) {
		t.Setenv("KITTY_WINDOW_ID", "1")
		m := resize(
			ui.New(scrollbarImageEditor(t), command.NewKeymaps()),
			scrollbarWidth, scrollbarHeight,
		)

		assertNoScrollbar(t, m.View().Content)

		m2, _ := m.Update(scrollbarTestCell())
		m = m2.(ui.Model)

		assertNoScrollbar(t, m.View().Content)
	})

	t.Run("gives up on a terminal that stays quiet", func(t *testing.T) {
		if testing.Short() {
			t.Skip("slow: waits out the cell size query")
		}
		t.Setenv("KITTY_WINDOW_ID", "1")
		m := resize(
			ui.New(scrollbarImageEditor(t), command.NewKeymaps()),
			scrollbarWidth, scrollbarHeight,
		)
		time.Sleep(scrollbarQuietWait)

		content := m.View().Content

		assert.True(t, strings.ContainsAny(content, scrollbarThinGlyphs))
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

	t.Run("thumb matches the cell bar", func(t *testing.T) {
		for _, key := range []string{
			"KITTY_WINDOW_ID", "TERM", "TERM_PROGRAM",
		} {
			t.Setenv(key, "")
		}
		glyphs := resize(
			ui.New(scrollbarImageEditor(t), command.NewKeymaps()),
			scrollbarWidth, scrollbarHeight,
		)
		rows := 0
		cells := scrollbarCells(t, glyphs.View().Content, scrollbarRows)
		for _, c := range cells {
			if c.style.bg == scrollbarThumbBg {
				rows++
			}
		}
		assert.NotZero(t, rows)

		t.Setenv("KITTY_WINDOW_ID", "1")
		_, raws := scrollbarImageModelFor(
			t, scrollbarImageEditor(t), scrollbarTestCell(),
		)
		img := decodeScrollbarImage(t, raws)

		// only the cursor marks this document and it sits in the thumb, so the
		// track color resumes exactly where the thumb ends
		assert.Equal(t,
			rows*scrollbarTestCell().Height, scrollbarThumbEnd(img),
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

func scrollbarThumbEnd(img image.Image) int {
	b := img.Bounds()
	track, _, _, _ := img.At(b.Min.X, b.Max.Y-1).RGBA()
	for y := b.Max.Y - 1; y >= b.Min.Y; y-- {
		if r, _, _, _ := img.At(b.Min.X, y).RGBA(); r != track {
			return y - b.Min.Y + 1
		}
	}
	return 0
}

func assertNoScrollbar(t *testing.T, content string) {
	t.Helper()
	assert.False(t, strings.ContainsRune(content, tui.PlaceholderRune))
	assert.False(t, strings.ContainsAny(content, scrollbarThinGlyphs))
}
