package ui

type PickerLayoutOptions struct {
	SplitRatios map[string]float64 `toml:"split-ratios"`
}

const (
	DefaultPickerSplitRatio = 0.5
	MinPickerSplitRatio     = 0.2
	MaxPickerSplitRatio     = 0.8
)

func (o PickerLayoutOptions) WithDefaults() PickerLayoutOptions {
	if len(o.SplitRatios) > 0 {
		ratios := make(map[string]float64, len(o.SplitRatios))
		for key, ratio := range o.SplitRatios {
			ratios[key] = clampPickerSplitRatio(ratio)
		}
		o.SplitRatios = ratios
	}
	return o
}

// SplitRatioFor returns the saved split ratio for a picker key
func (o PickerLayoutOptions) SplitRatioFor(key string) float64 {
	if ratio, ok := o.WithDefaults().SplitRatios[key]; ok {
		return ratio
	}
	return DefaultPickerSplitRatio
}

func clampPickerSplitRatio(ratio float64) float64 {
	if ratio < MinPickerSplitRatio {
		return MinPickerSplitRatio
	}
	if ratio > MaxPickerSplitRatio {
		return MaxPickerSplitRatio
	}
	return ratio
}
