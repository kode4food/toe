package language

const (
	DefaultTextWidth       = 80
	DefaultTabWidth        = 4
	DefaultMaxWrap         = 20
	DefaultMaxIndentRetain = 40
	DefaultWrapIndicator   = "↪ "
)

type TextFormat struct {
	ViewportWidth       int
	TabWidth            int
	SoftWrap            bool
	MaxWrap             int
	MaxIndentRetain     int
	WrapIndicator       string
	SoftWrapAtTextWidth bool
}

func TextFormatForLanguageWithConfig(
	lang string, textWidth *int, softWrap SoftWrap, w int,
) *TextFormat {
	return TextFormatForConfig(LoadLanguage(lang), textWidth, softWrap, w)
}

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
	format.MaxWrap = min(
		intValue(lang.SoftWrap.MaxWrap, softWrap.MaxWrap, DefaultMaxWrap),
		w/4,
	)
	retain := intValue(
		lang.SoftWrap.MaxIndentRetain,
		softWrap.MaxIndentRetain,
		DefaultMaxIndentRetain,
	)
	format.MaxIndentRetain = min(retain, w*2/5)
	format.WrapIndicator = stringValue(
		lang.SoftWrap.WrapIndicator, softWrap.WrapIndicator,
		DefaultWrapIndicator,
	)
	format.SoftWrapAtTextWidth = wrapAt
	return format
}

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
