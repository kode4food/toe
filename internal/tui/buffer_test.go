package tui_test

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
)

func TestBuffer(t *testing.T) {
	t.Run("NewBuffer initialises to spaces", func(t *testing.T) {
		b := tui.NewBuffer(geom.Size{Width: 3, Height: 2})
		for y := range 2 {
			for x := range 3 {
				c := b.Get(geom.Point{X: x, Y: y})
				assert.Equal(t, " ", c.Symbol)
				assert.Equal(t, tui.Style{}, c.Style)
			}
		}
	})

	t.Run("Set and Get round-trip", func(t *testing.T) {
		b := tui.NewBuffer(geom.Size{Width: 5, Height: 5})
		cell := tui.Cell{
			Symbol: "X",
			Style:  tui.Style{}.Fg(tui.ColorRed),
		}
		b.Set(geom.Point{X: 2, Y: 3}, cell)
		assert.Equal(t, cell, b.Get(geom.Point{X: 2, Y: 3}))
	})

	t.Run("Set ignores out-of-bounds", func(t *testing.T) {
		b := tui.NewBuffer(geom.Size{Width: 3, Height: 3})
		b.Set(geom.Point{X: 5, Y: 0}, tui.Cell{Symbol: "X"})
		b.Set(geom.Point{X: -1, Y: 0}, tui.Cell{Symbol: "X"})
		assert.Equal(t, " ", b.Get(geom.Point{X: 0, Y: 0}).Symbol)
	})

	t.Run("Get out-of-bounds returns default", func(t *testing.T) {
		b := tui.NewBuffer(geom.Size{Width: 3, Height: 3})
		assert.Equal(t, " ", b.Get(geom.Point{X: -1, Y: 0}).Symbol)
		assert.Equal(t, " ", b.Get(geom.Point{X: 0, Y: -1}).Symbol)
		assert.Equal(t, " ", b.Get(geom.Point{X: 3, Y: 0}).Symbol)
		assert.Equal(t, " ", b.Get(geom.Point{X: 0, Y: 3}).Symbol)
	})

	t.Run("Clear resets all cells", func(t *testing.T) {
		b := tui.NewBuffer(geom.Size{Width: 2, Height: 2})
		b.Set(geom.Point{X: 0, Y: 0}, tui.Cell{Symbol: "A"})
		b.Clear()
		assert.Equal(t, " ", b.Get(geom.Point{X: 0, Y: 0}).Symbol)
	})

	t.Run("SetString writes graphemes", func(t *testing.T) {
		b := tui.NewBuffer(geom.Size{Width: 5, Height: 1})
		b.SetString(geom.Point{X: 0, Y: 0}, "hello", tui.Style{})
		assert.Equal(t, "h", b.Get(geom.Point{X: 0, Y: 0}).Symbol)
		assert.Equal(t, "e", b.Get(geom.Point{X: 1, Y: 0}).Symbol)
		assert.Equal(t, "o", b.Get(geom.Point{X: 4, Y: 0}).Symbol)
	})

	t.Run("SetString clips at buffer width", func(t *testing.T) {
		b := tui.NewBuffer(geom.Size{Width: 3, Height: 1})
		b.SetString(geom.Point{X: 0, Y: 0}, "hello", tui.Style{})
		assert.Equal(t, "h", b.Get(geom.Point{X: 0, Y: 0}).Symbol)
		assert.Equal(t, "e", b.Get(geom.Point{X: 1, Y: 0}).Symbol)
		assert.Equal(t, "l", b.Get(geom.Point{X: 2, Y: 0}).Symbol)
	})

	t.Run("SetString handles wide graphemes", func(t *testing.T) {
		b := tui.NewBuffer(geom.Size{Width: 2, Height: 1})
		st := tui.Style{}.Bg(tui.ColorBlue)
		b.SetString(geom.Point{X: 0, Y: 0}, "コ", st)
		// first cell has the grapheme; second is space with same style
		assert.Equal(t, "コ", b.Get(geom.Point{X: 0, Y: 0}).Symbol)
		assert.True(t, b.Get(geom.Point{X: 1, Y: 0}).Skip)
		assert.Equal(t, st, b.Get(geom.Point{X: 1, Y: 0}).Style)
		assert.Equal(t, 2, ansi.StringWidth(b.RenderToANSI()))
	})

	t.Run("clips wide graphemes at right edge", func(t *testing.T) {
		b := tui.NewBuffer(geom.Size{Width: 1, Height: 1})
		b.SetString(geom.Point{X: 0, Y: 0}, "コ", tui.Style{})
		assert.Equal(t, " ", b.Get(geom.Point{X: 0, Y: 0}).Symbol)
		assert.Equal(t, 1, ansi.StringWidth(b.RenderToANSI()))
	})

	t.Run("SetString ignores out-of-bounds row", func(t *testing.T) {
		b := tui.NewBuffer(geom.Size{Width: 3, Height: 1})
		b.SetString(geom.Point{X: 0, Y: 5}, "abc", tui.Style{})
		assert.Equal(t, " ", b.Get(geom.Point{X: 0, Y: 0}).Symbol)
	})

	t.Run("ignores start past right edge", func(t *testing.T) {
		b := tui.NewBuffer(geom.Size{Width: 3, Height: 1})
		b.SetString(geom.Point{X: 3, Y: 0}, "abc", tui.Style{})
		assert.Equal(t, " ", b.Get(geom.Point{X: 0, Y: 0}).Symbol)
	})
}

func TestPatchBg(t *testing.T) {
	t.Run("patches bg of existing cell", func(t *testing.T) {
		b := tui.NewBuffer(geom.Size{Width: 3, Height: 1})
		b.SetString(geom.Point{X: 0, Y: 0}, "abc", tui.Style{}.Fg(tui.ColorRed))
		b.PatchBg(geom.Point{X: 1, Y: 0}, tui.ColorBlue)
		assert.Equal(t, tui.Style{}.Fg(tui.ColorRed).Bg(tui.ColorBlue),
			b.Get(geom.Point{X: 1, Y: 0}).Style)
	})

	t.Run("ignores out-of-bounds patch", func(t *testing.T) {
		b := tui.NewBuffer(geom.Size{Width: 2, Height: 1})
		b.PatchBg(geom.Point{X: 5, Y: 0}, tui.ColorRed)
		b.PatchBg(geom.Point{X: -1, Y: 0}, tui.ColorRed)
		assert.Equal(t, tui.Style{}, b.Get(geom.Point{X: 0, Y: 0}).Style)
	})
}

func TestPatchBgRange(t *testing.T) {
	t.Run("patches a range of cells", func(t *testing.T) {
		b := tui.NewBuffer(geom.Size{Width: 5, Height: 1})
		b.SetString(geom.Point{X: 0, Y: 0}, "abcde", tui.Style{})
		b.PatchBgRange(geom.Point{X: 1, Y: 0}, 3, tui.ColorGreen)
		assert.Equal(t,
			tui.ColorGreen, b.Get(geom.Point{X: 1, Y: 0}).Style.BgColor(),
		)
		assert.Equal(t,
			tui.ColorGreen, b.Get(geom.Point{X: 2, Y: 0}).Style.BgColor(),
		)
		assert.Equal(t,
			tui.ColorGreen, b.Get(geom.Point{X: 3, Y: 0}).Style.BgColor(),
		)
		assert.Equal(t, tui.Style{}, b.Get(geom.Point{X: 0, Y: 0}).Style)
	})
}

func TestSetRightAlignedInt(t *testing.T) {
	t.Run("writes integer right-aligned", func(t *testing.T) {
		b := tui.NewBuffer(geom.Size{Width: 5, Height: 1})
		b.SetRightAlignedInt(geom.Point{X: 0, Y: 0}, 5, 42, tui.Style{})
		out := b.RenderToANSI()
		assert.Contains(t, out, "42")
	})

	t.Run("writes zero", func(t *testing.T) {
		b := tui.NewBuffer(geom.Size{Width: 3, Height: 1})
		b.SetRightAlignedInt(geom.Point{X: 0, Y: 0}, 3, 0, tui.Style{})
		out := b.RenderToANSI()
		assert.Contains(t, out, "0")
	})

	t.Run("clips left padding", func(t *testing.T) {
		b := tui.NewBuffer(geom.Size{Width: 3, Height: 1})
		b.SetRightAlignedInt(geom.Point{X: -2, Y: 0}, 5, 42, tui.Style{})
		assert.Contains(t, b.RenderToANSI(), "42")
	})

	t.Run("ignores out-of-bounds row", func(t *testing.T) {
		b := tui.NewBuffer(geom.Size{Width: 3, Height: 1})
		b.SetRightAlignedInt(geom.Point{X: 0, Y: 5}, 3, 42, tui.Style{})
		assert.Equal(t, " ", b.Get(geom.Point{X: 0, Y: 0}).Symbol)
	})
}

func TestBlit(t *testing.T) {
	t.Run("copies src cells at offset", func(t *testing.T) {
		dst := tui.NewBuffer(geom.Size{Width: 5, Height: 5})
		src := tui.NewBuffer(geom.Size{Width: 2, Height: 2})
		src.SetString(geom.Point{X: 0, Y: 0},
			"AB", tui.Style{}.Fg(tui.ColorRed))
		src.SetString(geom.Point{X: 0, Y: 1},
			"CD", tui.Style{}.Fg(tui.ColorRed))
		dst.Blit(src, geom.Point{X: 1, Y: 1})
		assert.Equal(t, "A", dst.Get(geom.Point{X: 1, Y: 1}).Symbol)
		assert.Equal(t, "B", dst.Get(geom.Point{X: 2, Y: 1}).Symbol)
		assert.Equal(t, "C", dst.Get(geom.Point{X: 1, Y: 2}).Symbol)
		assert.Equal(t, "D", dst.Get(geom.Point{X: 2, Y: 2}).Symbol)
	})

	t.Run("clips src overflowing bottom-right edge", func(t *testing.T) {
		dst := tui.NewBuffer(geom.Size{Width: 3, Height: 3})
		src := tui.NewBuffer(geom.Size{Width: 3, Height: 3})
		src.Fill(tui.Style{}.Fg(tui.ColorBlue))
		dst.Blit(src, geom.Point{X: 1, Y: 1})
		assert.Equal(t,
			tui.ColorBlue, dst.Get(geom.Point{X: 2, Y: 2}).Style.FgColor(),
		)
		assert.Equal(t, tui.Style{}, dst.Get(geom.Point{X: 0, Y: 0}).Style)
	})

	t.Run("clips src overflowing negative offset", func(t *testing.T) {
		dst := tui.NewBuffer(geom.Size{Width: 3, Height: 3})
		src := tui.NewBuffer(geom.Size{Width: 3, Height: 3})
		src.Fill(tui.Style{}.Fg(tui.ColorGreen))
		dst.Blit(src, geom.Point{X: -1, Y: -1})
		assert.Equal(t,
			tui.ColorGreen, dst.Get(geom.Point{X: 0, Y: 0}).Style.FgColor(),
		)
		assert.Equal(t, tui.Style{}, dst.Get(geom.Point{X: 2, Y: 2}).Style)
	})

	t.Run("transparent bg keeps dst background", func(t *testing.T) {
		dst := tui.NewBuffer(geom.Size{Width: 2, Height: 1})
		dst.PatchBg(geom.Point{X: 0, Y: 0}, tui.ColorYellow)
		src := tui.NewBuffer(geom.Size{Width: 2, Height: 1})
		src.SetString(geom.Point{X: 0, Y: 0}, "X", tui.Style{})
		dst.Blit(src, geom.Point{X: 0, Y: 0})
		assert.Equal(t, "X", dst.Get(geom.Point{X: 0, Y: 0}).Symbol)
		assert.Equal(t,
			tui.ColorYellow, dst.Get(geom.Point{X: 0, Y: 0}).Style.BgColor(),
		)
	})

	t.Run("opaque bg overwrites dst background", func(t *testing.T) {
		dst := tui.NewBuffer(geom.Size{Width: 2, Height: 1})
		dst.PatchBg(geom.Point{X: 0, Y: 0}, tui.ColorYellow)
		src := tui.NewBuffer(geom.Size{Width: 2, Height: 1})
		src.SetString(geom.Point{X: 0, Y: 0}, "X", tui.Style{}.Bg(tui.ColorRed))
		dst.Blit(src, geom.Point{X: 0, Y: 0})
		assert.Equal(t,
			tui.ColorRed, dst.Get(geom.Point{X: 0, Y: 0}).Style.BgColor(),
		)
	})

	t.Run("wide glyph clipped at right edge blanks", func(t *testing.T) {
		dst := tui.NewBuffer(geom.Size{Width: 2, Height: 1})
		src := tui.NewBuffer(geom.Size{Width: 2, Height: 1})
		src.SetString(geom.Point{X: 0, Y: 0}, "コ", tui.Style{})
		dst.Blit(src, geom.Point{X: 1, Y: 0})
		assert.Equal(t, " ", dst.Get(geom.Point{X: 0, Y: 0}).Symbol)
		assert.Equal(t, " ", dst.Get(geom.Point{X: 1, Y: 0}).Symbol)
		assert.Equal(t, 2, ansi.StringWidth(dst.RenderToANSI()))
	})

	t.Run("wide glyph fits at right edge", func(t *testing.T) {
		dst := tui.NewBuffer(geom.Size{Width: 2, Height: 1})
		src := tui.NewBuffer(geom.Size{Width: 2, Height: 1})
		src.SetString(geom.Point{X: 0, Y: 0}, "コ", tui.Style{})
		dst.Blit(src, geom.Point{X: 0, Y: 0})
		assert.Equal(t, "コ", dst.Get(geom.Point{X: 0, Y: 0}).Symbol)
		assert.True(t, dst.Get(geom.Point{X: 1, Y: 0}).Skip)
		assert.Equal(t, 2, ansi.StringWidth(dst.RenderToANSI()))
	})
}

func TestFill(t *testing.T) {
	t.Run("fills all cells with style", func(t *testing.T) {
		b := tui.NewBuffer(geom.Size{Width: 3, Height: 2})
		st := tui.Style{}.Bg(tui.ColorRed)
		b.Fill(st)
		for y := range 2 {
			for x := range 3 {
				c := b.Get(geom.Point{X: x, Y: y})
				assert.Equal(t, " ", c.Symbol)
				assert.Equal(t, st, c.Style)
			}
		}
	})

	t.Run("fill overwrites existing content", func(t *testing.T) {
		b := tui.NewBuffer(geom.Size{Width: 2, Height: 2})
		b.Set(geom.Point{X: 0, Y: 0}, tui.Cell{Symbol: "X"})
		b.Fill(tui.Style{}.Fg(tui.ColorBlue))
		assert.Equal(t, " ", b.Get(geom.Point{X: 0, Y: 0}).Symbol)
	})

	t.Run("fill on zero-size buffer is no-op", func(t *testing.T) {
		b := tui.NewBuffer(geom.Size{Width: 0, Height: 0})
		b.Fill(tui.Style{}.Fg(tui.ColorRed))
		assert.Equal(t, 0, b.Width)
	})

	t.Run("fill range clips to row", func(t *testing.T) {
		b := tui.NewBuffer(geom.Size{Width: 3, Height: 1})
		st := tui.Style{}.Bg(tui.ColorRed)
		b.FillRange(geom.Point{X: -1, Y: 0}, 3, st)

		assert.Equal(t, " ", b.Get(geom.Point{X: 0, Y: 0}).Symbol)
		assert.Equal(t, st, b.Get(geom.Point{X: 0, Y: 0}).Style)
		assert.Equal(t, " ", b.Get(geom.Point{X: 1, Y: 0}).Symbol)
		assert.Equal(t, st, b.Get(geom.Point{X: 1, Y: 0}).Style)
		assert.Equal(t, tui.Style{}, b.Get(geom.Point{X: 2, Y: 0}).Style)
	})

	t.Run("render shows filled style", func(t *testing.T) {
		b := tui.NewBuffer(geom.Size{Width: 2, Height: 1})
		b.Fill(tui.Style{}.Bg(tui.ColorGreen))
		out := b.RenderToANSI()
		assert.Contains(t, out, "\x1b[42m")
	})
}
