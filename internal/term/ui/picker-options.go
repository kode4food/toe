package ui

import "maps"

type PickerLayoutOptions struct {
	SplitRatios map[string]float64 `toml:"split-ratios"`
}

const (
	DefaultPickerSplitRatio = 0.5
	MinPickerSplitRatio     = 0.2
	MaxPickerSplitRatio     = 0.8
)

func (o PickerLayoutOptions) clone() PickerLayoutOptions {
	if len(o.SplitRatios) > 0 {
		ratios := make(map[string]float64, len(o.SplitRatios))
		maps.Copy(ratios, o.SplitRatios)
		o.SplitRatios = ratios
	}
	return o
}

// SplitRatioFor returns the saved split ratio for a picker key
func (o PickerLayoutOptions) SplitRatioFor(key string) float64 {
	ratio, ok := o.SplitRatios[key]
	if !ok {
		return DefaultPickerSplitRatio
	}
	return clampPickerSplitRatio(ratio)
}

func clampPickerSplitRatio(ratio float64) float64 {
	return min(max(ratio, MinPickerSplitRatio), MaxPickerSplitRatio)
}
