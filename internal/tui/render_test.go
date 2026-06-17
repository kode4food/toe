package tui_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/tui"
)

func render(s tui.Style) string {
	b := tui.NewBuffer(1, 1)
	b.SetString(0, 0, "x", s)
	return b.RenderToANSI()
}

func renderTwo(s0, s1 tui.Style) string {
	b := tui.NewBuffer(2, 1)
	b.SetString(0, 0, "a", s0)
	b.SetString(1, 0, "b", s1)
	return b.RenderToANSI()
}

func TestRenderEmpty(t *testing.T) {
	t.Run("zero-width returns empty", func(t *testing.T) {
		b := tui.NewBuffer(0, 5)
		assert.Equal(t, "", b.RenderToANSI())
	})

	t.Run("zero-height returns empty", func(t *testing.T) {
		b := tui.NewBuffer(5, 0)
		assert.Equal(t, "", b.RenderToANSI())
	})

	t.Run("multi-row separated by newline", func(t *testing.T) {
		b := tui.NewBuffer(2, 2)
		b.SetString(0, 0, "ab", tui.Style{})
		b.SetString(0, 1, "cd", tui.Style{})
		out := b.RenderToANSI()
		rows := strings.Split(out, "\n")
		assert.Len(t, rows, 2)
	})
}

func TestEmitFgColor(t *testing.T) {
	cases := []struct {
		name  string
		color tui.Color
		seq   string
	}{
		{"black", tui.ColorBlack, "\x1b[30m"},
		{"red", tui.ColorRed, "\x1b[31m"},
		{"green", tui.ColorGreen, "\x1b[32m"},
		{"yellow", tui.ColorYellow, "\x1b[33m"},
		{"blue", tui.ColorBlue, "\x1b[34m"},
		{"magenta", tui.ColorMagenta, "\x1b[35m"},
		{"cyan", tui.ColorCyan, "\x1b[36m"},
		{"gray", tui.ColorGray, "\x1b[90m"},
		{"light red", tui.ColorLightRed, "\x1b[91m"},
		{"light green", tui.ColorLightGreen, "\x1b[92m"},
		{"light yellow", tui.ColorLightYellow, "\x1b[93m"},
		{"light blue", tui.ColorLightBlue, "\x1b[94m"},
		{"light magenta", tui.ColorLightMagenta, "\x1b[95m"},
		{"light cyan", tui.ColorLightCyan, "\x1b[96m"},
		{"light gray", tui.ColorLightGray, "\x1b[37m"},
		{"white", tui.ColorWhite, "\x1b[97m"},
		{"indexed", tui.ColorIndexed(200), "\x1b[38;5;200m"},
		{"rgb", tui.ColorRGB(1, 2, 3), "\x1b[38;2;1;2;3m"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := render(tui.Style{}.Fg(tc.color))
			assert.Contains(t, out, tc.seq)
		})
	}

	t.Run("reset", func(t *testing.T) {
		out := renderTwo(
			tui.Style{}.Fg(tui.ColorRed),
			tui.Style{}.Fg(tui.ColorReset),
		)
		assert.Contains(t, out, "\x1b[39m")
	})
}

func TestEmitBgColor(t *testing.T) {
	cases := []struct {
		name  string
		color tui.Color
		seq   string
	}{
		{"black", tui.ColorBlack, "\x1b[40m"},
		{"red", tui.ColorRed, "\x1b[41m"},
		{"green", tui.ColorGreen, "\x1b[42m"},
		{"yellow", tui.ColorYellow, "\x1b[43m"},
		{"blue", tui.ColorBlue, "\x1b[44m"},
		{"magenta", tui.ColorMagenta, "\x1b[45m"},
		{"cyan", tui.ColorCyan, "\x1b[46m"},
		{"gray", tui.ColorGray, "\x1b[100m"},
		{"light red", tui.ColorLightRed, "\x1b[101m"},
		{"light green", tui.ColorLightGreen, "\x1b[102m"},
		{"light yellow", tui.ColorLightYellow, "\x1b[103m"},
		{"light blue", tui.ColorLightBlue, "\x1b[104m"},
		{"light magenta", tui.ColorLightMagenta, "\x1b[105m"},
		{"light cyan", tui.ColorLightCyan, "\x1b[106m"},
		{"light gray", tui.ColorLightGray, "\x1b[47m"},
		{"white", tui.ColorWhite, "\x1b[107m"},
		{"indexed", tui.ColorIndexed(5), "\x1b[48;5;5m"},
		{"rgb", tui.ColorRGB(10, 20, 30), "\x1b[48;2;10;20;30m"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := render(tui.Style{}.Bg(tc.color))
			assert.Contains(t, out, tc.seq)
		})
	}

	t.Run("reset", func(t *testing.T) {
		out := renderTwo(
			tui.Style{}.Bg(tui.ColorRed),
			tui.Style{}.Bg(tui.ColorReset),
		)
		assert.Contains(t, out, "\x1b[49m")
	})
}

func TestEmitUlColor(t *testing.T) {
	t.Run("reset emits 59m", func(t *testing.T) {
		// Change from a non-reset to reset to force emission.
		out := renderTwo(
			tui.Style{}.UlColor(tui.ColorRGB(1, 2, 3)),
			tui.Style{}.UlColor(tui.ColorReset),
		)
		assert.Contains(t, out, "\x1b[59m")
	})

	t.Run("indexed", func(t *testing.T) {
		out := render(tui.Style{}.UlColor(tui.ColorIndexed(42)))
		assert.Contains(t, out, "\x1b[58:5:42m")
	})

	t.Run("rgb", func(t *testing.T) {
		out := render(tui.Style{}.UlColor(tui.ColorRGB(1, 2, 3)))
		assert.Contains(t, out, "\x1b[58:2::1:2:3m")
	})

	t.Run("named color falls back to reset seq", func(t *testing.T) {
		out := renderTwo(
			tui.Style{}.UlColor(tui.ColorRGB(1, 2, 3)),
			tui.Style{}.UlColor(tui.ColorRed),
		)
		assert.Contains(t, out, "\x1b[59m")
	})
}

func TestEmitUnderline(t *testing.T) {
	cases := []struct {
		name string
		ul   tui.UnderlineStyle
		seq  string
	}{
		{"line", tui.UnderlineLine, "\x1b[4m"},
		{"curl", tui.UnderlineCurl, "\x1b[4:3m"},
		{"dotted", tui.UnderlineDotted, "\x1b[4:4m"},
		{"dashed", tui.UnderlineDashed, "\x1b[4:5m"},
		{"double", tui.UnderlineDoubleLine, "\x1b[21m"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := render(tui.Style{}.UlStyle(tc.ul))
			assert.Contains(t, out, tc.seq)
		})
	}

	t.Run("reset emits 24m", func(t *testing.T) {
		out := renderTwo(
			tui.Style{}.UlStyle(tui.UnderlineLine),
			tui.Style{}.UlStyle(tui.UnderlineReset),
		)
		assert.Contains(t, out, "\x1b[24m")
	})
}

func TestEmitModifiers(t *testing.T) {
	t.Run("bold", func(t *testing.T) {
		out := render(tui.Style{}.Mod(tui.ModifierBold))
		assert.Contains(t, out, "\x1b[1m")
	})

	t.Run("italic", func(t *testing.T) {
		out := render(tui.Style{}.Mod(tui.ModifierItalic))
		assert.Contains(t, out, "\x1b[3m")
	})

	t.Run("dim", func(t *testing.T) {
		out := render(tui.Style{}.Mod(tui.ModifierDim))
		assert.Contains(t, out, "\x1b[2m")
	})

	t.Run("crossed out", func(t *testing.T) {
		out := render(tui.Style{}.Mod(tui.ModifierCrossedOut))
		assert.Contains(t, out, "\x1b[9m")
	})

	t.Run("slow blink", func(t *testing.T) {
		out := render(tui.Style{}.Mod(tui.ModifierSlowBlink))
		assert.Contains(t, out, "\x1b[5m")
	})

	t.Run("rapid blink", func(t *testing.T) {
		out := render(tui.Style{}.Mod(tui.ModifierRapidBlink))
		assert.Contains(t, out, "\x1b[6m")
	})

	t.Run("reversed", func(t *testing.T) {
		out := render(tui.Style{}.Mod(tui.ModifierReversed))
		assert.Contains(t, out, "\x1b[7m")
	})

	t.Run("hidden", func(t *testing.T) {
		out := render(tui.Style{}.Mod(tui.ModifierHidden))
		assert.Contains(t, out, "\x1b[8m")
	})

	t.Run("same modifier not re-emitted", func(t *testing.T) {
		out := renderTwo(
			tui.Style{}.Mod(tui.ModifierBold),
			tui.Style{}.Mod(tui.ModifierBold),
		)
		assert.Equal(t, 1, strings.Count(out, "\x1b[1m"))
	})

	t.Run("remove reversed emits 27m", func(t *testing.T) {
		out := renderTwo(
			tui.Style{}.Mod(tui.ModifierReversed),
			tui.Style{},
		)
		assert.Contains(t, out, "\x1b[27m")
	})

	t.Run("remove italic emits 23m", func(t *testing.T) {
		out := renderTwo(
			tui.Style{}.Mod(tui.ModifierItalic),
			tui.Style{},
		)
		assert.Contains(t, out, "\x1b[23m")
	})

	t.Run("remove dim emits 22m", func(t *testing.T) {
		out := renderTwo(
			tui.Style{}.Mod(tui.ModifierDim),
			tui.Style{},
		)
		assert.Contains(t, out, "\x1b[22m")
	})

	t.Run("remove crossed out emits 29m", func(t *testing.T) {
		out := renderTwo(
			tui.Style{}.Mod(tui.ModifierCrossedOut),
			tui.Style{},
		)
		assert.Contains(t, out, "\x1b[29m")
	})

	t.Run("remove slow blink emits 25m", func(t *testing.T) {
		out := renderTwo(
			tui.Style{}.Mod(tui.ModifierSlowBlink),
			tui.Style{},
		)
		assert.Contains(t, out, "\x1b[25m")
	})

	t.Run("remove rapid blink emits 25m", func(t *testing.T) {
		out := renderTwo(
			tui.Style{}.Mod(tui.ModifierRapidBlink),
			tui.Style{},
		)
		assert.Contains(t, out, "\x1b[25m")
	})

	t.Run("remove hidden emits 28m", func(t *testing.T) {
		out := renderTwo(
			tui.Style{}.Mod(tui.ModifierHidden),
			tui.Style{},
		)
		assert.Contains(t, out, "\x1b[28m")
	})

	t.Run("remove bold then add dim emits dim", func(t *testing.T) {
		// Removing bold emits 22m (normal intensity); dim must be re-added
		// because 22m also clears dim
		out := renderTwo(
			tui.Style{}.Mod(tui.ModifierBold),
			tui.Style{}.Mod(tui.ModifierDim),
		)
		assert.Contains(t, out, "\x1b[22m")
		assert.Contains(t, out, "\x1b[2m")
	})
}
