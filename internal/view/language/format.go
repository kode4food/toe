package language

type TextFormat struct {
	ViewportWidth       int
	TabWidth            int
	SoftWrap            bool
	MaxWrap             int
	MaxIndentRetain     int
	WrapIndicator       string
	SoftWrapAtTextWidth bool
}

const (
	DefaultTextWidth       = 80
	DefaultTabWidth        = 4
	DefaultMaxWrap         = 20
	DefaultMaxIndentRetain = 40
	DefaultWrapIndicator   = "\u21aa" // '↪' - rightwards arrow with hook
)

// TextFormatForConfig builds the render format for a language and width
func TextFormatForConfig(
	lang *Language, textWidth *int, softWrap SoftWrap, w int,
) *TextFormat {
	tw := intValue(nil, textWidth, DefaultTextWidth)
	if lang.TextWidth != nil {
		tw = *lang.TextWidth
	}

	wrapAt := boolValue(
		lang.SoftWrap.WrapAtTextWidth, softWrap.WrapAtTextWidth, false,
	)
	if wrapAt {
		if tw >= w {
			wrapAt = false
		} else {
			w = tw
		}
	}

	enabled := boolValue(lang.SoftWrap.Enable, softWrap.Enable, false)
	format := DefaultTextFormat(w)
	format.SoftWrap = enabled && w > MinSoftWrapWidth
	format.WrapIndicator = stringValue(
		lang.SoftWrap.WrapIndicator, softWrap.WrapIndicator,
		DefaultWrapIndicator,
	)
	format.SoftWrapAtTextWidth = wrapAt
	return format
}

// DefaultTextFormat builds the render format used when no language matches
func DefaultTextFormat(w int) *TextFormat {
	return &TextFormat{
		ViewportWidth:   w,
		TabWidth:        DefaultTabWidth,
		SoftWrap:        false,
		MaxWrap:         min(DefaultMaxWrap, w/4),
		MaxIndentRetain: min(DefaultMaxIndentRetain, w*2/5),
		WrapIndicator:   DefaultWrapIndicator,
	}
}

// WrapIndicatorPrefix is the indicator plus its gap, as drawn at the start of
// a soft-wrapped continuation row
func (t *TextFormat) WrapIndicatorPrefix() string {
	if t.WrapIndicator == "" {
		return ""
	}
	return t.WrapIndicator + " "
}
