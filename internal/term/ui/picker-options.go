package ui

import "maps"

// PickerLayoutOptions holds the per-overlay list/preview split ratios and
// size scales, keyed by picker or prompt id
type PickerLayoutOptions struct {
	SplitRatios map[string]float64 `toml:"split"`
	Scales      map[string]float64 `toml:"scales"`
}

const (
	DefaultPickerSplitRatio = 0.5
	MinPickerSplitRatio     = 0.2
	MaxPickerSplitRatio     = 0.8
)

// overlay size scales are stored one entry per axis, keyed by overlay id
const (
	widthScaleSuffix  = ".width"
	heightScaleSuffix = ".height"
)

// SplitRatioFor returns the saved split ratio for a picker key
func (o PickerLayoutOptions) SplitRatioFor(key string) float64 {
	if ratio, ok := o.SplitRatios[key]; ok {
		return clampPickerSplitRatio(ratio)
	}
	return DefaultPickerSplitRatio
}

func (o PickerLayoutOptions) widthScale(id string, def float64) float64 {
	return o.scaleFor(id+widthScaleSuffix, def)
}

func (o PickerLayoutOptions) heightScale(id string, def float64) float64 {
	return o.scaleFor(id+heightScaleSuffix, def)
}

func (o PickerLayoutOptions) withWidthScale(
	id string, scale float64,
) PickerLayoutOptions {
	return o.withScale(id+widthScaleSuffix, scale)
}

func (o PickerLayoutOptions) withHeightScale(
	id string, scale float64,
) PickerLayoutOptions {
	return o.withScale(id+heightScaleSuffix, scale)
}

func (o PickerLayoutOptions) scaleFor(key string, def float64) float64 {
	if scale, ok := o.Scales[key]; ok {
		return clampOverlayScale(scale)
	}
	return def
}

func (o PickerLayoutOptions) withScale(
	key string, scale float64,
) PickerLayoutOptions {
	out := o.clone()
	if out.Scales == nil {
		out.Scales = map[string]float64{}
	}
	out.Scales[key] = clampOverlayScale(scale)
	return out
}

func (o PickerLayoutOptions) clone() PickerLayoutOptions {
	o.SplitRatios = maps.Clone(o.SplitRatios)
	o.Scales = maps.Clone(o.Scales)
	return o
}

func clampPickerSplitRatio(ratio float64) float64 {
	return min(max(ratio, MinPickerSplitRatio), MaxPickerSplitRatio)
}
