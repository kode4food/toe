package ui

type PickerLayoutOptions struct {
	SplitRatio float64 `toml:"split-ratio"`
}

const (
	DefaultPickerSplitRatio = 0.5
	MinPickerSplitRatio     = 0.2
	MaxPickerSplitRatio     = 0.8
)

func DefaultPickerLayoutOptions() PickerLayoutOptions {
	return PickerLayoutOptions{SplitRatio: DefaultPickerSplitRatio}
}

func (o PickerLayoutOptions) WithDefaults() PickerLayoutOptions {
	if o.SplitRatio == 0 {
		o.SplitRatio = DefaultPickerSplitRatio
	}
	if o.SplitRatio < MinPickerSplitRatio {
		o.SplitRatio = MinPickerSplitRatio
	}
	if o.SplitRatio > MaxPickerSplitRatio {
		o.SplitRatio = MaxPickerSplitRatio
	}
	return o
}
