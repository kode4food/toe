package tui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/tui"
)

func TestColor(t *testing.T) {
	t.Run("ColorReset is reset", func(t *testing.T) {
		assert.True(t, tui.ColorReset.IsReset())
	})

	t.Run("named colors are not reset", func(t *testing.T) {
		assert.False(t, tui.ColorRed.IsReset())
		assert.False(t, tui.ColorBlue.IsReset())
	})

	t.Run("RGB color equality", func(t *testing.T) {
		a := tui.ColorRGB(1, 2, 3)
		b := tui.ColorRGB(1, 2, 3)
		c := tui.ColorRGB(1, 2, 4)
		assert.Equal(t, a, b)
		assert.NotEqual(t, a, c)
	})

	t.Run("Indexed color equality", func(t *testing.T) {
		a := tui.ColorIndexed(240)
		b := tui.ColorIndexed(240)
		c := tui.ColorIndexed(241)
		assert.Equal(t, a, b)
		assert.NotEqual(t, a, c)
	})
}

func TestStyle(t *testing.T) {
	t.Run("zero style has reset colors", func(t *testing.T) {
		var s tui.Style
		assert.Equal(t, tui.Style{}, s)
	})

	t.Run("builder methods return new style", func(t *testing.T) {
		s := tui.Style{}.
			Fg(tui.ColorRGB(1, 2, 3)).
			Bg(tui.ColorIndexed(240)).
			Mod(tui.ModifierBold | tui.ModifierItalic)
		assert.Equal(t, tui.ColorRGB(1, 2, 3), s.FgColor())
		assert.Equal(t, tui.ColorIndexed(240), s.BgColor())
		assert.True(t, s.HasMod(tui.ModifierBold))
		assert.True(t, s.HasMod(tui.ModifierItalic))
		assert.False(t, s.HasMod(tui.ModifierReversed))
	})

	t.Run("UlColor round-trips via render", func(t *testing.T) {
		b := tui.NewBuffer(1, 1)
		st := tui.Style{}.UlColor(tui.ColorRGB(10, 20, 30))
		b.SetString(0, 0, "x", st)
		out := b.RenderToANSI()
		assert.Contains(t, out, "\x1b[58:2::10:20:30m")
	})

	t.Run("UlStyle round-trips via render", func(t *testing.T) {
		b := tui.NewBuffer(1, 1)
		st := tui.Style{}.UlStyle(tui.UnderlineLine)
		b.SetString(0, 0, "x", st)
		out := b.RenderToANSI()
		assert.Contains(t, out, "\x1b[4m")
	})

	t.Run("HasMod returns false for zero modifier", func(t *testing.T) {
		s := tui.Style{}
		assert.False(t, s.HasMod(tui.ModifierBold))
	})

	t.Run("HasMod matches exact bit", func(t *testing.T) {
		s := tui.Style{}.Mod(tui.ModifierDim)
		assert.True(t, s.HasMod(tui.ModifierDim))
		assert.False(t, s.HasMod(tui.ModifierBold))
	})

	t.Run("Mod accumulates bits", func(t *testing.T) {
		s := tui.Style{}.
			Mod(tui.ModifierBold).
			Mod(tui.ModifierItalic)
		assert.True(t, s.HasMod(tui.ModifierBold))
		assert.True(t, s.HasMod(tui.ModifierItalic))
	})
}
