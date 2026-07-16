package tui_test

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/tui"
)

func TestBuffer(t *testing.T) {
	t.Run("NewBuffer initialises to spaces", func(t *testing.T) {
		b := tui.NewBuffer(3, 2)
		for y := range 2 {
			for x := range 3 {
				c := b.Get(x, y)
				assert.Equal(t, " ", c.Symbol)
				assert.Equal(t, tui.Style{}, c.Style)
			}
		}
	})

	t.Run("Set and Get round-trip", func(t *testing.T) {
		b := tui.NewBuffer(5, 5)
		cell := tui.Cell{
			Symbol: "X",
			Style:  tui.Style{}.Fg(tui.ColorRed),
		}
		b.Set(2, 3, cell)
		assert.Equal(t, cell, b.Get(2, 3))
	})

	t.Run("Set ignores out-of-bounds", func(t *testing.T) {
		b := tui.NewBuffer(3, 3)
		b.Set(5, 0, tui.Cell{Symbol: "X"})
		b.Set(-1, 0, tui.Cell{Symbol: "X"})
		assert.Equal(t, " ", b.Get(0, 0).Symbol)
	})

	t.Run("Get out-of-bounds returns default", func(t *testing.T) {
		b := tui.NewBuffer(3, 3)
		assert.Equal(t, " ", b.Get(-1, 0).Symbol)
		assert.Equal(t, " ", b.Get(0, -1).Symbol)
		assert.Equal(t, " ", b.Get(3, 0).Symbol)
		assert.Equal(t, " ", b.Get(0, 3).Symbol)
	})

	t.Run("Clear resets all cells", func(t *testing.T) {
		b := tui.NewBuffer(2, 2)
		b.Set(0, 0, tui.Cell{Symbol: "A"})
		b.Clear()
		assert.Equal(t, " ", b.Get(0, 0).Symbol)
	})

	t.Run("SetString writes graphemes", func(t *testing.T) {
		b := tui.NewBuffer(5, 1)
		b.SetString(0, 0, "hello", tui.Style{})
		assert.Equal(t, "h", b.Get(0, 0).Symbol)
		assert.Equal(t, "e", b.Get(1, 0).Symbol)
		assert.Equal(t, "o", b.Get(4, 0).Symbol)
	})

	t.Run("SetString clips at buffer width", func(t *testing.T) {
		b := tui.NewBuffer(3, 1)
		b.SetString(0, 0, "hello", tui.Style{})
		assert.Equal(t, "h", b.Get(0, 0).Symbol)
		assert.Equal(t, "e", b.Get(1, 0).Symbol)
		assert.Equal(t, "l", b.Get(2, 0).Symbol)
	})

	t.Run("SetString handles wide graphemes", func(t *testing.T) {
		b := tui.NewBuffer(2, 1)
		st := tui.Style{}.Bg(tui.ColorBlue)
		b.SetString(0, 0, "コ", st)
		// first cell has the grapheme; second is space with same style
		assert.Equal(t, "コ", b.Get(0, 0).Symbol)
		assert.True(t, b.Get(1, 0).Skip)
		assert.Equal(t, st, b.Get(1, 0).Style)
		assert.Equal(t, 2, ansi.StringWidth(b.RenderToANSI()))
	})

	t.Run("clips wide graphemes at right edge", func(t *testing.T) {
		b := tui.NewBuffer(1, 1)
		b.SetString(0, 0, "コ", tui.Style{})
		assert.Equal(t, " ", b.Get(0, 0).Symbol)
		assert.Equal(t, 1, ansi.StringWidth(b.RenderToANSI()))
	})

	t.Run("SetString ignores out-of-bounds row", func(t *testing.T) {
		b := tui.NewBuffer(3, 1)
		b.SetString(0, 5, "abc", tui.Style{})
		assert.Equal(t, " ", b.Get(0, 0).Symbol)
	})

	t.Run("ignores start past right edge", func(t *testing.T) {
		b := tui.NewBuffer(3, 1)
		b.SetString(3, 0, "abc", tui.Style{})
		assert.Equal(t, " ", b.Get(0, 0).Symbol)
	})
}

func TestPatchBg(t *testing.T) {
	t.Run("patches bg of existing cell", func(t *testing.T) {
		b := tui.NewBuffer(3, 1)
		b.SetString(0, 0, "abc", tui.Style{}.Fg(tui.ColorRed))
		b.PatchBg(1, 0, tui.ColorBlue)
		assert.Equal(t, tui.Style{}.Fg(tui.ColorRed).Bg(tui.ColorBlue),
			b.Get(1, 0).Style)
	})

	t.Run("ignores out-of-bounds patch", func(t *testing.T) {
		b := tui.NewBuffer(2, 1)
		b.PatchBg(5, 0, tui.ColorRed)
		b.PatchBg(-1, 0, tui.ColorRed)
		assert.Equal(t, tui.Style{}, b.Get(0, 0).Style)
	})
}

func TestPatchBgRange(t *testing.T) {
	t.Run("patches a range of cells", func(t *testing.T) {
		b := tui.NewBuffer(5, 1)
		b.SetString(0, 0, "abcde", tui.Style{})
		b.PatchBgRange(1, 0, 3, tui.ColorGreen)
		assert.Equal(t, tui.ColorGreen, b.Get(1, 0).Style.BgColor())
		assert.Equal(t, tui.ColorGreen, b.Get(2, 0).Style.BgColor())
		assert.Equal(t, tui.ColorGreen, b.Get(3, 0).Style.BgColor())
		assert.Equal(t, tui.Style{}, b.Get(0, 0).Style)
	})
}

func TestSetRightAlignedInt(t *testing.T) {
	t.Run("writes integer right-aligned", func(t *testing.T) {
		b := tui.NewBuffer(5, 1)
		b.SetRightAlignedInt(0, 0, 5, 42, tui.Style{})
		out := b.RenderToANSI()
		assert.Contains(t, out, "42")
	})

	t.Run("writes zero", func(t *testing.T) {
		b := tui.NewBuffer(3, 1)
		b.SetRightAlignedInt(0, 0, 3, 0, tui.Style{})
		out := b.RenderToANSI()
		assert.Contains(t, out, "0")
	})

	t.Run("clips left padding", func(t *testing.T) {
		b := tui.NewBuffer(3, 1)
		b.SetRightAlignedInt(-2, 0, 5, 42, tui.Style{})
		assert.Contains(t, b.RenderToANSI(), "42")
	})

	t.Run("ignores out-of-bounds row", func(t *testing.T) {
		b := tui.NewBuffer(3, 1)
		b.SetRightAlignedInt(0, 5, 3, 42, tui.Style{})
		assert.Equal(t, " ", b.Get(0, 0).Symbol)
	})
}

func TestBlit(t *testing.T) {
	t.Run("copies src cells at offset", func(t *testing.T) {
		dst := tui.NewBuffer(5, 5)
		src := tui.NewBuffer(2, 2)
		src.SetString(0, 0, "AB", tui.Style{}.Fg(tui.ColorRed))
		src.SetString(0, 1, "CD", tui.Style{}.Fg(tui.ColorRed))
		dst.Blit(src, 1, 1)
		assert.Equal(t, "A", dst.Get(1, 1).Symbol)
		assert.Equal(t, "B", dst.Get(2, 1).Symbol)
		assert.Equal(t, "C", dst.Get(1, 2).Symbol)
		assert.Equal(t, "D", dst.Get(2, 2).Symbol)
	})

	t.Run("clips src overflowing bottom-right edge", func(t *testing.T) {
		dst := tui.NewBuffer(3, 3)
		src := tui.NewBuffer(3, 3)
		src.Fill(tui.Style{}.Fg(tui.ColorBlue))
		dst.Blit(src, 1, 1)
		assert.Equal(t, tui.ColorBlue, dst.Get(2, 2).Style.FgColor())
		assert.Equal(t, tui.Style{}, dst.Get(0, 0).Style)
	})

	t.Run("clips src overflowing negative offset", func(t *testing.T) {
		dst := tui.NewBuffer(3, 3)
		src := tui.NewBuffer(3, 3)
		src.Fill(tui.Style{}.Fg(tui.ColorGreen))
		dst.Blit(src, -1, -1)
		assert.Equal(t, tui.ColorGreen, dst.Get(0, 0).Style.FgColor())
		assert.Equal(t, tui.Style{}, dst.Get(2, 2).Style)
	})

	t.Run("transparent bg keeps dst background", func(t *testing.T) {
		dst := tui.NewBuffer(2, 1)
		dst.PatchBg(0, 0, tui.ColorYellow)
		src := tui.NewBuffer(2, 1)
		src.SetString(0, 0, "X", tui.Style{})
		dst.Blit(src, 0, 0)
		assert.Equal(t, "X", dst.Get(0, 0).Symbol)
		assert.Equal(t, tui.ColorYellow, dst.Get(0, 0).Style.BgColor())
	})

	t.Run("opaque bg overwrites dst background", func(t *testing.T) {
		dst := tui.NewBuffer(2, 1)
		dst.PatchBg(0, 0, tui.ColorYellow)
		src := tui.NewBuffer(2, 1)
		src.SetString(0, 0, "X", tui.Style{}.Bg(tui.ColorRed))
		dst.Blit(src, 0, 0)
		assert.Equal(t, tui.ColorRed, dst.Get(0, 0).Style.BgColor())
	})

	t.Run("wide glyph clipped at right edge blanks", func(t *testing.T) {
		dst := tui.NewBuffer(2, 1)
		src := tui.NewBuffer(2, 1)
		src.SetString(0, 0, "コ", tui.Style{})
		dst.Blit(src, 1, 0)
		assert.Equal(t, " ", dst.Get(0, 0).Symbol)
		assert.Equal(t, " ", dst.Get(1, 0).Symbol)
		assert.Equal(t, 2, ansi.StringWidth(dst.RenderToANSI()))
	})

	t.Run("wide glyph fits at right edge", func(t *testing.T) {
		dst := tui.NewBuffer(2, 1)
		src := tui.NewBuffer(2, 1)
		src.SetString(0, 0, "コ", tui.Style{})
		dst.Blit(src, 0, 0)
		assert.Equal(t, "コ", dst.Get(0, 0).Symbol)
		assert.True(t, dst.Get(1, 0).Skip)
		assert.Equal(t, 2, ansi.StringWidth(dst.RenderToANSI()))
	})
}

func TestFill(t *testing.T) {
	t.Run("fills all cells with style", func(t *testing.T) {
		b := tui.NewBuffer(3, 2)
		st := tui.Style{}.Bg(tui.ColorRed)
		b.Fill(st)
		for y := range 2 {
			for x := range 3 {
				c := b.Get(x, y)
				assert.Equal(t, " ", c.Symbol)
				assert.Equal(t, st, c.Style)
			}
		}
	})

	t.Run("fill overwrites existing content", func(t *testing.T) {
		b := tui.NewBuffer(2, 2)
		b.Set(0, 0, tui.Cell{Symbol: "X"})
		b.Fill(tui.Style{}.Fg(tui.ColorBlue))
		assert.Equal(t, " ", b.Get(0, 0).Symbol)
	})

	t.Run("fill on zero-size buffer is no-op", func(t *testing.T) {
		b := tui.NewBuffer(0, 0)
		b.Fill(tui.Style{}.Fg(tui.ColorRed))
		assert.Equal(t, 0, b.Width)
	})

	t.Run("fill range clips to row", func(t *testing.T) {
		b := tui.NewBuffer(3, 1)
		st := tui.Style{}.Bg(tui.ColorRed)
		b.FillRange(-1, 0, 3, st)

		assert.Equal(t, " ", b.Get(0, 0).Symbol)
		assert.Equal(t, st, b.Get(0, 0).Style)
		assert.Equal(t, " ", b.Get(1, 0).Symbol)
		assert.Equal(t, st, b.Get(1, 0).Style)
		assert.Equal(t, tui.Style{}, b.Get(2, 0).Style)
	})

	t.Run("render shows filled style", func(t *testing.T) {
		b := tui.NewBuffer(2, 1)
		b.Fill(tui.Style{}.Bg(tui.ColorGreen))
		out := b.RenderToANSI()
		assert.Contains(t, out, "\x1b[42m")
	})
}
