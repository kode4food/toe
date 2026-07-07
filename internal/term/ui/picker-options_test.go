package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/ui"
)

func TestPickerLayoutOptions(t *testing.T) {
	tests := []struct {
		name string
		opts ui.PickerLayoutOptions
		want float64
	}{
		{"defaults zero ratio", ui.PickerLayoutOptions{}, 0.5},
		{
			"clamps low ratio",
			ui.PickerLayoutOptions{
				SplitRatios: map[string]float64{"fixed": -1},
			},
			0.2,
		},
		{
			"clamps high ratio",
			ui.PickerLayoutOptions{
				SplitRatios: map[string]float64{"fixed": 2},
			},
			0.8,
		},
		{
			"keeps custom ratio",
			ui.PickerLayoutOptions{
				SplitRatios: map[string]float64{"fixed": 0.4},
			},
			0.4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.opts.SplitRatioFor("fixed"))
		})
	}

	t.Run("uses picker-specific ratio", func(t *testing.T) {
		opts := ui.PickerLayoutOptions{
			SplitRatios: map[string]float64{
				"Diagnostics": 0.7,
			},
		}

		assert.Equal(t, 0.7, opts.SplitRatioFor("Diagnostics"))
		assert.Equal(t, 0.5, opts.SplitRatioFor("Global search"))
	})

	t.Run("clamps picker-specific ratio", func(t *testing.T) {
		opts := ui.PickerLayoutOptions{
			SplitRatios: map[string]float64{
				"Diagnostics": 0.95,
			},
		}

		assert.Equal(
			t, ui.MaxPickerSplitRatio,
			opts.SplitRatioFor("Diagnostics"),
		)
	})
}
