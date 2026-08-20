package language

// TextFormat is the resolved layout of a document's text: the width to wrap at,
// how tabs measure, and how wrapped rows are indicated
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
	tw := settingValue(settingValueArgs[int]{
		editor: textWidth,
	}, DefaultTextWidth)
	if lang.TextWidth != nil {
		tw = *lang.TextWidth
	}

	wrapAt := settingValue(settingValueArgs[bool]{
		lang:   lang.SoftWrap.WrapAtTextWidth,
		editor: softWrap.WrapAtTextWidth,
	}, false)
	if wrapAt {
		if tw >= w {
			wrapAt = false
		} else {
			w = tw
		}
	}

	enabled := settingValue(settingValueArgs[bool]{
		lang:   lang.SoftWrap.Enable,
		editor: softWrap.Enable,
	}, false)
	format := DefaultTextFormat(w)
	format.SoftWrap = enabled && w > MinSoftWrapWidth
	format.WrapIndicator = settingValue(settingValueArgs[string]{
		lang:   lang.SoftWrap.WrapIndicator,
		editor: softWrap.WrapIndicator,
	}, DefaultWrapIndicator)
	format.SoftWrapAtTextWidth = wrapAt
	return format
}

// DefaultTextFormat builds the render format used when no language matches
func DefaultTextFormat(w int) *TextFormat {
	return &TextFormat{
		ViewportWidth:   w,
		TabWidth:        DefaultTabWidth,
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
