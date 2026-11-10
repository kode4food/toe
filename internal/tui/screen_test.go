package tui_test

import (
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/vt"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
)

func TestScreen(t *testing.T) {
	size := geom.Size{
		Width:  12,
		Height: 4,
	}

	t.Run("first frame writes every row", func(t *testing.T) {
		var s tui.Screen
		buf := tui.NewBuffer(size)
		buf.SetString(geom.Point{}, "hi", tui.Style{})
		var out strings.Builder
		assert.NoError(t, s.Snapshot(buf).WriteFrame(&out))
		for y := 1; y <= size.Height; y++ {
			assert.Contains(t, out.String(), "\x1b["+string(rune('0'+y))+";1H")
		}
	})

	t.Run("writes only changed cells", func(t *testing.T) {
		var s tui.Screen
		buf := tui.NewBuffer(size)
		assert.NoError(t, s.Snapshot(buf).WriteFrame(&strings.Builder{}))
		buf.SetString(geom.Point{
			X: 3,
			Y: 2,
		}, "ab", tui.Style{})
		var out strings.Builder
		assert.NoError(t, s.Snapshot(buf).WriteFrame(&out))
		assert.Equal(t, "\x1b[3;4Hab", out.String())
	})

	t.Run("unchanged frame writes nothing", func(t *testing.T) {
		var s tui.Screen
		buf := tui.NewBuffer(size)
		assert.NoError(t, s.Snapshot(buf).WriteFrame(&strings.Builder{}))
		var out strings.Builder
		assert.NoError(t, s.Snapshot(buf).WriteFrame(&out))
		assert.Empty(t, out.String())
	})

	t.Run("invalidate writes every row", func(t *testing.T) {
		var s tui.Screen
		buf := tui.NewBuffer(size)
		f := s.Snapshot(buf)
		assert.NoError(t, f.WriteFrame(&strings.Builder{}))
		f.Invalidate()
		var out strings.Builder
		assert.NoError(t, f.WriteFrame(&out))
		assert.Equal(t, size.Height, strings.Count(out.String(), "H"))
	})

	t.Run("resize repaints", func(t *testing.T) {
		var s tui.Screen
		first := s.Snapshot(tui.NewBuffer(size))
		assert.NoError(t, first.WriteFrame(&strings.Builder{}))
		var out strings.Builder
		small := tui.NewBuffer(geom.Size{
			Width:  3,
			Height: 2,
		})
		assert.NoError(t, s.Snapshot(small).WriteFrame(&out))
		assert.Equal(t, "\x1b[1;1H   \x1b[2;1H   ", out.String())
	})

	t.Run("resets style at frame end", func(t *testing.T) {
		var s tui.Screen
		buf := tui.NewBuffer(size)
		last := geom.Point{
			X: size.Width - 1,
			Y: size.Height - 1,
		}
		buf.SetString(last, "b", tui.Style{}.Mod(tui.ModifierBold))
		var out strings.Builder
		assert.NoError(t, s.Snapshot(buf).WriteFrame(&out))
		assert.True(t, strings.HasSuffix(out.String(), "\x1b[m"))
	})

	t.Run("snapshot ignores later edits", func(t *testing.T) {
		var s tui.Screen
		buf := tui.NewBuffer(size)
		buf.SetString(geom.Point{}, "old", tui.Style{})
		f := s.Snapshot(buf)
		buf.SetString(geom.Point{}, "new", tui.Style{})
		assert.Contains(t, f.String(), "old")
	})

	t.Run("diffs match a full repaint", func(t *testing.T) {
		var s tui.Screen
		em := vt.NewEmulator(size.Width, size.Height)
		buf := tui.NewBuffer(size)
		for i := range 8 {
			screenPaint(buf, i)
			assert.NoError(t, s.Snapshot(buf).WriteFrame(em))
		}
		var fresh tui.Screen
		want := vt.NewEmulator(size.Width, size.Height)
		assert.NoError(t, fresh.Snapshot(buf).WriteFrame(want))
		assert.Equal(t, want.Render(), em.Render())
		row := strings.Split(want.String(), "\n")[3]
		assert.Equal(t, "a日本b", strings.TrimSpace(row))
	})
}

func TestScreenScroll(t *testing.T) {
	size := geom.Size{
		Width:  16,
		Height: 10,
	}

	t.Run("scrolls up", func(t *testing.T) {
		out := scrollFrames(t, size, []int{0, 3})
		assert.Contains(t, out[1], "\x1b[1;9r\x1b[3S\x1b[r")
		assert.Less(t, len(out[1]), len(out[0])/2)
	})

	t.Run("scrolls down", func(t *testing.T) {
		out := scrollFrames(t, size, []int{5, 3})
		assert.Contains(t, out[1], "\x1b[1;9r\x1b[2T\x1b[r")
		assert.Less(t, len(out[1]), len(out[0])/2)
	})

	t.Run("matches repaint", func(t *testing.T) {
		var s tui.Screen
		em := vt.NewEmulator(size.Width, size.Height)
		buf := tui.NewBuffer(size)
		rng := rand.New(rand.NewPCG(1, 2))
		top := 0
		for i := range 200 {
			top += rng.IntN(13) - 6
			scrollPaint(buf, top)
			if i%3 == 0 {
				at := geom.Point{
					X: rng.IntN(8),
					Y: rng.IntN(size.Height),
				}
				st := tui.Style{}.Bg(tui.ColorIndexed(uint8(i)))
				buf.SetString(at, "日z", st)
			}
			assert.NoError(t, s.Snapshot(buf).WriteFrame(em))
			var fresh tui.Screen
			want := vt.NewEmulator(size.Width, size.Height)
			assert.NoError(t, fresh.Snapshot(buf).WriteFrame(want))
			if !assert.Equal(t, want.Render(), em.Render()) {
				return
			}
		}
	})
}

func BenchmarkScreen(b *testing.B) {
	size := geom.Size{
		Width:  120,
		Height: 40,
	}
	b.Run("scroll", func(b *testing.B) {
		benchmarkScreen(b, size, scrollPaint)
	})
	b.Run("rewrite", func(b *testing.B) {
		benchmarkScreen(b, size, screenPaint)
	})
}

func scrollFrames(t *testing.T, size geom.Size, tops []int) []string {
	t.Helper()
	var s tui.Screen
	buf := tui.NewBuffer(size)
	res := make([]string, 0, len(tops))
	for _, top := range tops {
		scrollPaint(buf, top)
		var out strings.Builder
		assert.NoError(t, s.Snapshot(buf).WriteFrame(&out))
		res = append(res, out.String())
	}
	return res
}

func scrollPaint(buf *tui.Buffer, top int) {
	text := tui.Style{}.Fg(tui.ColorRGB(220, 220, 220))
	buf.Fill(tui.Style{})
	for y := range buf.Height - 1 {
		line := "line " + strconv.Itoa(top+y)
		buf.SetString(geom.Point{Y: y}, line, text)
	}
	status := tui.Style{}.Bg(tui.ColorIndexed(8))
	buf.FillRange(geom.Point{Y: buf.Height - 1}, buf.Width, status)
	buf.SetString(geom.Point{Y: buf.Height - 1}, "NOR", status)
}

func screenPaint(buf *tui.Buffer, i int) {
	plain := tui.Style{}.Fg(tui.ColorRGB(200, 200, 200))
	hl := tui.Style{}.Bg(tui.ColorIndexed(4)).Mod(tui.ModifierBold)
	buf.Fill(plain)
	for y := range buf.Height - 1 {
		st := plain
		if y == i%(buf.Height-1) {
			st = hl
		}
		buf.SetString(
			geom.Point{
				X: i % 3,
				Y: y,
			}, strings.Repeat("x", y+i%4), st,
		)
	}
	wide := "a日本b"
	if i%2 == 0 {
		wide = "日本ab"
	}
	buf.SetString(geom.Point{Y: buf.Height - 1}, wide, plain)
}

func benchmarkScreen(
	b *testing.B, size geom.Size, paint func(*tui.Buffer, int),
) {
	var s tui.Screen
	buf := tui.NewBuffer(size)
	frames := make([]*tui.Buffer, 20)
	for i := range frames {
		frames[i] = tui.NewBuffer(size)
		top := i * 3
		if i >= 10 {
			top = (20 - i) * 3
		}
		paint(frames[i], top)
	}
	var out strings.Builder
	i := 0
	b.ReportAllocs()
	for b.Loop() {
		buf.Blit(frames[i%len(frames)], geom.Point{})
		if err := s.Snapshot(buf).WriteFrame(&out); err != nil {
			b.Fatal(err)
		}
		i++
	}
	b.ReportMetric(float64(out.Len())/float64(b.N), "ansi_B/op")
}
