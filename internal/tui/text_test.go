package tui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/tui"
)

func TestText(t *testing.T) {
	t.Run("StyledGrapheme equality", func(t *testing.T) {
		st := tui.Style{}.Fg(tui.ColorRed)
		a := tui.StyledGrapheme{Symbol: "a", Style: st}
		b := tui.StyledGrapheme{Symbol: "a", Style: st}
		assert.Equal(t, a, b)
	})

	t.Run("Spans", func(t *testing.T) {
		spans := tui.Spans{
			{Content: "hello", Style: tui.Style{}},
			{Content: " world", Style: tui.Style{}.Fg(tui.ColorGreen)},
		}
		assert.Len(t, spans, 2)
		assert.Equal(t, "hello", spans[0].Content)
		assert.Equal(t, " world", spans[1].Content)
	})
}
