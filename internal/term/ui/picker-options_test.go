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
		{"clamps low ratio", ui.PickerLayoutOptions{SplitRatio: -1}, 0.2},
		{"clamps high ratio", ui.PickerLayoutOptions{SplitRatio: 2}, 0.8},
		{"keeps custom ratio", ui.PickerLayoutOptions{SplitRatio: 0.4}, 0.4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.opts.WithDefaults().SplitRatio)
		})
	}
}
