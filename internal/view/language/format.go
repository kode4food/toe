package language

import "github.com/kode4food/toe/internal/loader"

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

func TextFormatForLanguage(lang string, w int) *TextFormat {
	textWidth, sw := loadUserEditorFormat()
	return TextFormatForLanguageWithConfig(lang, textWidth, sw, w)
}

// loadUserEditorFormat reads the user's config.toml for editor text-width and
// soft-wrap settings. Returns nil/zero values when the file is absent or unparseable.
func loadUserEditorFormat() (*int, SoftWrap) {
	path, ok := loader.ConfigFile()
	if !ok {
		return nil, SoftWrap{}
	}
	data, ok := loader.LoadMergedTOML([]string{path}, 3)
	if !ok {
		return nil, SoftWrap{}
	}
	editor, _ := data["editor"].(map[string]any)
	textWidth := intPtrOrNil(editor["text-width"])
	sw, _ := editor["soft-wrap"].(map[string]any)
	return textWidth, SoftWrap{
		Enable:          boolPtr(sw["enable"]),
		MaxWrap:         intPtrOrNil(sw["max-wrap"]),
		MaxIndentRetain: intPtrOrNil(sw["max-indent-retain"]),
		WrapIndicator:   stringPtr(sw["wrap-indicator"]),
		WrapAtTextWidth: boolPtr(sw["wrap-at-text-width"]),
	}
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

	wrapAt := boolValue(lang.SoftWrap.WrapAtTextWidth, softWrap.WrapAtTextWidth, false)
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
	format.MaxIndentRetain = min(
		intValue(lang.SoftWrap.MaxIndentRetain, softWrap.MaxIndentRetain, DefaultMaxIndentRetain),
		w*2/5,
	)
	format.WrapIndicator = stringValue(
		lang.SoftWrap.WrapIndicator, softWrap.WrapIndicator, DefaultWrapIndicator,
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
