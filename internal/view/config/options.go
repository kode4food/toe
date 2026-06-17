package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/kode4food/toe/internal/core"
)

var (
	ErrUnknownOption = errors.New("unknown option")
	ErrInvalidOption = errors.New("invalid option")
)

const (
	optKeyTheme = "theme"

	optKeyScrolloff    = "editor.scrolloff"
	optKeyScrollLines  = "editor.scroll-lines"
	optKeyLineNumber   = "editor.line-number"
	optKeyCursorline   = "editor.cursorline"
	optKeyCursorcolumn = "editor.cursorcolumn"
	optKeyMouse        = "editor.mouse"
	optKeyMiddlePaste  = "editor.middle-click-paste"
	optKeyTextWidth    = "editor.text-width"

	optKeyInsertFinalNewline = "editor.insert-final-newline"
	optKeyTrimFinalNewlines  = "editor.trim-final-newlines"
	optKeyTrimTrailingWS     = "editor.trim-trailing-whitespace"

	optKeyAutoPairs                 = "editor.auto-pairs"
	optKeyAutoSave                  = "editor.auto-save"
	optKeyAutoSaveFocusLost         = "editor.auto-save.focus-lost"
	optKeyAutoSaveAfterDelayEnable  = "editor.auto-save.after-delay.enable"
	optKeyAutoSaveAfterDelayTimeout = "editor.auto-save.after-delay.timeout"
	optKeyAtomicSave                = "editor.atomic-save"

	optKeyInsecure         = "editor.insecure"
	optKeyEditorConfig     = "editor.editor-config"
	optKeyContinueComments = "editor.continue-comments"

	optKeyCursorShapeNormal = "editor.cursor-shape.normal"
	optKeyCursorShapeSelect = "editor.cursor-shape.select"
	optKeyCursorShapeInsert = "editor.cursor-shape.insert"

	optKeyStatusLineSeparator  = "editor.statusline.separator"
	optKeyStatusLineModeNormal = "editor.statusline.mode.normal"
	optKeyStatusLineModeInsert = "editor.statusline.mode.insert"
	optKeyStatusLineModeSelect = "editor.statusline.mode.select"

	optKeyDefaultLineEnding = "editor.default-line-ending"

	optKeySearchSmartCase  = "editor.search.smart-case"
	optKeySearchWrapAround = "editor.search.wrap-around"

	optKeySoftWrapEnable          = "editor.soft-wrap.enable"
	optKeySoftWrapMaxWrap         = "editor.soft-wrap.max-wrap"
	optKeySoftWrapMaxIndentRetain = "editor.soft-wrap.max-indent-retain"
	optKeySoftWrapWrapIndicator   = "editor.soft-wrap.wrap-indicator"
	optKeySoftWrapWrapAtTextWidth = "editor.soft-wrap.wrap-at-text-width"

	optKeyWhitespaceRender = "editor.whitespace.render"

	optKeyGuttersLineNumbersMinWidth = "editor.gutters.line-numbers.min-width"

	optKeyIndentGuidesRender     = "editor.indent-guides.render"
	optKeyIndentGuidesSkipLevels = "editor.indent-guides.skip-levels"
	optKeyIndentGuidesCharacter  = "editor.indent-guides.character"

	optKeyRulers     = "editor.rulers"
	optKeyShell      = "editor.shell"
	optKeyBufferline = "editor.bufferline"
)

// OptionKeys returns all settable option key names in sorted order
func OptionKeys() []string {
	return []string{
		optKeyAtomicSave,
		optKeyAutoPairs,
		optKeyAutoSave,
		optKeyAutoSaveAfterDelayEnable,
		optKeyAutoSaveAfterDelayTimeout,
		optKeyAutoSaveFocusLost,
		optKeyBufferline,
		optKeyContinueComments,
		optKeyCursorShapeInsert,
		optKeyCursorShapeNormal,
		optKeyCursorShapeSelect,
		optKeyCursorcolumn,
		optKeyCursorline,
		optKeyDefaultLineEnding,
		optKeyEditorConfig,
		optKeyGuttersLineNumbersMinWidth,
		optKeyIndentGuidesCharacter,
		optKeyIndentGuidesRender,
		optKeyIndentGuidesSkipLevels,
		optKeyInsecure,
		optKeyInsertFinalNewline,
		optKeyLineNumber,
		optKeyMiddlePaste,
		optKeyMouse,
		optKeyRulers,
		optKeyScrollLines,
		optKeyScrolloff,
		optKeySearchSmartCase,
		optKeySearchWrapAround,
		optKeyShell,
		optKeySoftWrapEnable,
		optKeySoftWrapMaxIndentRetain,
		optKeySoftWrapMaxWrap,
		optKeySoftWrapWrapAtTextWidth,
		optKeySoftWrapWrapIndicator,
		optKeyStatusLineModeInsert,
		optKeyStatusLineModeNormal,
		optKeyStatusLineModeSelect,
		optKeyStatusLineSeparator,
		optKeyTextWidth,
		optKeyTheme,
		optKeyTrimFinalNewlines,
		optKeyTrimTrailingWS,
		optKeyWhitespaceRender,
	}
}

// BoolOptionKeys returns option keys that accept boolean values
func BoolOptionKeys() []string {
	return []string{
		optKeyAtomicSave,
		optKeyAutoPairs,
		optKeyAutoSave,
		optKeyAutoSaveAfterDelayEnable,
		optKeyAutoSaveFocusLost,
		optKeyContinueComments,
		optKeyCursorcolumn,
		optKeyCursorline,
		optKeyEditorConfig,
		optKeyIndentGuidesRender,
		optKeyInsecure,
		optKeyInsertFinalNewline,
		optKeyMiddlePaste,
		optKeyMouse,
		optKeySearchSmartCase,
		optKeySearchWrapAround,
		optKeySoftWrapEnable,
		optKeySoftWrapWrapAtTextWidth,
		optKeyTrimFinalNewlines,
		optKeyTrimTrailingWS,
	}
}

func GetOption(cfg *Config, key string) (string, error) {
	switch normalizeKey(key) {
	case optKeyScrolloff:
		return strconv.Itoa(cfg.Scrolloff()), nil
	case optKeyScrollLines:
		return strconv.Itoa(cfg.ScrollLines()), nil
	case optKeyLineNumber:
		return string(cfg.LineNumber()), nil
	case optKeyCursorline:
		return strconv.FormatBool(cfg.Cursorline()), nil
	case optKeyCursorcolumn:
		return strconv.FormatBool(cfg.Cursorcolumn()), nil
	case optKeyMouse:
		return strconv.FormatBool(cfg.Mouse()), nil
	case optKeyMiddlePaste:
		return strconv.FormatBool(cfg.MiddleClickPaste()), nil
	case optKeyRulers:
		return formatIntSlice(cfg.Rulers()), nil
	case optKeyShell:
		return formatStringSlice(cfg.Shell()), nil
	case optKeyBufferline:
		return string(cfg.GetBufferLine()), nil
	case optKeyTheme:
		if cfg.Theme.Adaptive {
			return cfg.Theme.Choose(false), nil
		}
		return cfg.Theme.Name, nil
	case optKeyTextWidth:
		return strconv.Itoa(
			intValue(nil, cfg.Editor.TextWidth, DefaultTextWidth),
		), nil
	case optKeyInsertFinalNewline:
		return strconv.FormatBool(cfg.InsertFinalNewline()), nil
	case optKeyTrimFinalNewlines:
		return strconv.FormatBool(cfg.TrimFinalNewlines()), nil
	case optKeyTrimTrailingWS:
		return strconv.FormatBool(cfg.TrimTrailingWhitespace()), nil
	case optKeyAutoPairs:
		_, ok := cfg.AutoPairs()
		return strconv.FormatBool(ok), nil
	case optKeyAutoSave, optKeyAutoSaveFocusLost:
		return strconv.FormatBool(cfg.AutoSaveFocusLost()), nil
	case optKeyAutoSaveAfterDelayEnable:
		return strconv.FormatBool(cfg.AutoSaveAfterDelay()), nil
	case optKeyAutoSaveAfterDelayTimeout:
		return strconv.Itoa(cfg.AutoSaveDelayTimeout()), nil
	case optKeyAtomicSave:
		return strconv.FormatBool(cfg.AtomicSave()), nil
	case optKeyInsecure:
		return strconv.FormatBool(cfg.Insecure()), nil
	case optKeyEditorConfig:
		return strconv.FormatBool(cfg.EditorConfig()), nil
	case optKeyContinueComments:
		return strconv.FormatBool(cfg.ContinueComments()), nil
	case optKeyCursorShapeNormal:
		return string(cfg.CursorShapeForMode("normal")), nil
	case optKeyCursorShapeSelect:
		return string(cfg.CursorShapeForMode("select")), nil
	case optKeyCursorShapeInsert:
		return string(cfg.CursorShapeForMode("insert")), nil
	case optKeyStatusLineSeparator:
		return cfg.StatusLineSeparator(), nil
	case optKeyStatusLineModeNormal:
		return cfg.ModeNameForMode("normal"), nil
	case optKeyStatusLineModeInsert:
		return cfg.ModeNameForMode("insert"), nil
	case optKeyStatusLineModeSelect:
		return cfg.ModeNameForMode("select"), nil
	case optKeyDefaultLineEnding:
		return lineEndingConfigString(cfg.Editor.DefaultLineEnding), nil
	case optKeySearchSmartCase:
		return strconv.FormatBool(cfg.SearchSmartCase()), nil
	case optKeySearchWrapAround:
		return strconv.FormatBool(cfg.SearchWrapAround()), nil
	case optKeySoftWrapEnable:
		return strconv.FormatBool(softWrapEnable(cfg)), nil
	case optKeySoftWrapMaxWrap:
		return strconv.Itoa(intValue(nil, cfg.Editor.SoftWrap.MaxWrap,
			DefaultMaxWrap,
		)), nil
	case optKeySoftWrapMaxIndentRetain:
		return strconv.Itoa(
			intValue(nil, cfg.Editor.SoftWrap.MaxIndentRetain,
				DefaultMaxIndentRetain,
			),
		), nil
	case optKeySoftWrapWrapIndicator:
		return stringValue(nil, cfg.Editor.SoftWrap.WrapIndicator,
			DefaultWrapIndicator,
		), nil
	case optKeySoftWrapWrapAtTextWidth:
		return strconv.FormatBool(softWrapAtTextWidth(cfg)), nil
	case optKeyWhitespaceRender:
		return string(whitespaceRenderFor(
			nil, cfg.Editor.Whitespace.Render.Default,
		)), nil
	case optKeyGuttersLineNumbersMinWidth:
		g := cfg.Gutters()
		return strconv.Itoa(g.LineNumberMinWidth()), nil
	case optKeyIndentGuidesRender:
		return strconv.FormatBool(cfg.Editor.IndentGuides.Render), nil
	case optKeyIndentGuidesSkipLevels:
		return strconv.Itoa(cfg.IndentGuides().GetSkipLevels()), nil
	case optKeyIndentGuidesCharacter:
		return string(cfg.IndentGuides().CharRune()), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnknownOption, key)
	}
}

func SetOption(cfg *Config, key, value string) error {
	switch normalizeKey(key) {
	case optKeyScrolloff:
		v, err := parseNonNegInt(value)
		if err != nil {
			return err
		}
		cfg.Editor.Scrolloff = &v
	case optKeyScrollLines:
		v, err := parsePositiveInt(value)
		if err != nil {
			return err
		}
		cfg.Editor.ScrollLines = &v
	case optKeyLineNumber:
		v, err := parseLineNumber(value)
		if err != nil {
			return err
		}
		cfg.Editor.LineNumber = v
	case optKeyCursorline:
		v, err := parseBool(value)
		if err != nil {
			return err
		}
		cfg.Editor.Cursorline = &v
	case optKeyCursorcolumn:
		v, err := parseBool(value)
		if err != nil {
			return err
		}
		cfg.Editor.Cursorcolumn = &v
	case optKeyMouse:
		v, err := parseBool(value)
		if err != nil {
			return err
		}
		cfg.Editor.Mouse = &v
	case optKeyMiddlePaste:
		v, err := parseBool(value)
		if err != nil {
			return err
		}
		cfg.Editor.MiddleClickPaste = &v
	case optKeyRulers:
		v, err := parseIntSlice(value)
		if err != nil {
			return err
		}
		cfg.Editor.Rulers = v
	case optKeyShell:
		v, err := parseStringSlice(value)
		if err != nil {
			return err
		}
		cfg.Editor.Shell = v
	case optKeyBufferline:
		v, err := parseBufferLine(value)
		if err != nil {
			return err
		}
		cfg.Editor.BufferLine = v
	case optKeyTheme:
		cfg.Theme = Theme{Name: value}
	case optKeyTextWidth:
		v, err := parsePositiveInt(value)
		if err != nil {
			return err
		}
		cfg.Editor.TextWidth = &v
	case optKeyInsertFinalNewline:
		v, err := parseBool(value)
		if err != nil {
			return err
		}
		cfg.Editor.InsertFinalNewline = &v
	case optKeyTrimFinalNewlines:
		v, err := parseBool(value)
		if err != nil {
			return err
		}
		cfg.Editor.TrimFinalNewlines = &v
	case optKeyTrimTrailingWS:
		v, err := parseBool(value)
		if err != nil {
			return err
		}
		cfg.Editor.TrimTrailingWS = &v
	case optKeyAutoPairs:
		v, err := parseBool(value)
		if err != nil {
			return err
		}
		cfg.Editor.AutoPairs = AutoPairConfig{Present: true, Enable: &v}
	case optKeyAutoSave, optKeyAutoSaveFocusLost:
		v, err := parseBool(value)
		if err != nil {
			return err
		}
		cfg.Editor.AutoSave.FocusLost = &v
	case optKeyAutoSaveAfterDelayEnable:
		v, err := parseBool(value)
		if err != nil {
			return err
		}
		cfg.Editor.AutoSave.AfterDelay.Enable = &v
	case optKeyAutoSaveAfterDelayTimeout:
		v, err := parsePositiveInt(value)
		if err != nil {
			return err
		}
		cfg.Editor.AutoSave.AfterDelay.Timeout = &v
	case optKeyAtomicSave:
		v, err := parseBool(value)
		if err != nil {
			return err
		}
		cfg.Editor.AtomicSave = &v
	case optKeyInsecure:
		v, err := parseBool(value)
		if err != nil {
			return err
		}
		cfg.Editor.Insecure = &v
	case optKeyEditorConfig:
		v, err := parseBool(value)
		if err != nil {
			return err
		}
		cfg.Editor.EditorConfig = &v
	case optKeyContinueComments:
		v, err := parseBool(value)
		if err != nil {
			return err
		}
		cfg.Editor.ContinueComments = &v
	case optKeyCursorShapeNormal:
		v, err := parseCursorKind(value)
		if err != nil {
			return err
		}
		cfg.Editor.CursorShape.Normal = v
	case optKeyCursorShapeSelect:
		v, err := parseCursorKind(value)
		if err != nil {
			return err
		}
		cfg.Editor.CursorShape.Select = v
	case optKeyCursorShapeInsert:
		v, err := parseCursorKind(value)
		if err != nil {
			return err
		}
		cfg.Editor.CursorShape.Insert = v
	case optKeyStatusLineSeparator:
		cfg.Editor.StatusLine.Separator = value
	case optKeyStatusLineModeNormal:
		cfg.Editor.StatusLine.Mode.Normal = value
	case optKeyStatusLineModeInsert:
		cfg.Editor.StatusLine.Mode.Insert = value
	case optKeyStatusLineModeSelect:
		cfg.Editor.StatusLine.Mode.Select = value
	case optKeyDefaultLineEnding:
		var le core.LineEnding
		if err := le.UnmarshalText([]byte(value)); err != nil {
			return err
		}
		cfg.Editor.DefaultLineEnding = le
	case optKeySearchSmartCase:
		v, err := parseBool(value)
		if err != nil {
			return err
		}
		cfg.Editor.Search.SmartCase = &v
	case optKeySearchWrapAround:
		v, err := parseBool(value)
		if err != nil {
			return err
		}
		cfg.Editor.Search.WrapAround = &v
	case optKeySoftWrapEnable:
		v, err := parseBool(value)
		if err != nil {
			return err
		}
		cfg.Editor.SoftWrap.Enable = &v
	case optKeySoftWrapMaxWrap:
		v, err := parsePositiveInt(value)
		if err != nil {
			return err
		}
		cfg.Editor.SoftWrap.MaxWrap = &v
	case optKeySoftWrapMaxIndentRetain:
		v, err := parsePositiveInt(value)
		if err != nil {
			return err
		}
		cfg.Editor.SoftWrap.MaxIndentRetain = &v
	case optKeySoftWrapWrapIndicator:
		v, err := parseStringLiteral(value)
		if err != nil {
			return err
		}
		cfg.Editor.SoftWrap.WrapIndicator = &v
	case optKeySoftWrapWrapAtTextWidth:
		v, err := parseBool(value)
		if err != nil {
			return err
		}
		cfg.Editor.SoftWrap.WrapAtTextWidth = &v
	case optKeyWhitespaceRender:
		rv, err := parseWhitespaceRenderValue(value)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidOption, value)
		}
		cfg.Editor.Whitespace.Render.Default = &rv
		cfg.Editor.Whitespace.Render.Space = nil
		cfg.Editor.Whitespace.Render.Nbsp = nil
		cfg.Editor.Whitespace.Render.Nnbsp = nil
		cfg.Editor.Whitespace.Render.Tab = nil
		cfg.Editor.Whitespace.Render.Newline = nil
	case optKeyGuttersLineNumbersMinWidth:
		v, err := parsePositiveInt(value)
		if err != nil {
			return err
		}
		cfg.Editor.Gutters.LineNumbers.MinWidth = &v
	case optKeyIndentGuidesRender:
		v, err := parseBool(value)
		if err != nil {
			return err
		}
		cfg.Editor.IndentGuides.Render = v
	case optKeyIndentGuidesSkipLevels:
		v, err := parseNonNegInt(value)
		if err != nil {
			return err
		}
		cfg.Editor.IndentGuides.SkipLevels = &v
	case optKeyIndentGuidesCharacter:
		v, err := parseStringLiteral(value)
		if err != nil {
			return err
		}
		if len([]rune(v)) != 1 {
			return fmt.Errorf("%w: %s", ErrInvalidOption, v)
		}
		cfg.Editor.IndentGuides.Character = v
	default:
		return fmt.Errorf("%w: %s", ErrUnknownOption, key)
	}
	return nil
}

func ToggleOption(cfg *Config, key string) (string, error) {
	value, ok := boolOptionValue(cfg, key)
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrInvalidOption, key)
	}
	if err := SetOption(cfg, key, strconv.FormatBool(!value)); err != nil {
		return "", err
	}
	return strconv.FormatBool(!value), nil
}

func boolOptionValue(cfg *Config, key string) (bool, bool) {
	switch normalizeKey(key) {
	case optKeyInsertFinalNewline:
		return cfg.InsertFinalNewline(), true
	case optKeyTrimFinalNewlines:
		return cfg.TrimFinalNewlines(), true
	case optKeyTrimTrailingWS:
		return cfg.TrimTrailingWhitespace(), true
	case optKeyAutoPairs:
		_, ok := cfg.AutoPairs()
		return ok, true
	case optKeyAutoSave, optKeyAutoSaveFocusLost:
		return cfg.AutoSaveFocusLost(), true
	case optKeyAutoSaveAfterDelayEnable:
		return cfg.AutoSaveAfterDelay(), true
	case optKeyAtomicSave:
		return cfg.AtomicSave(), true
	case optKeyInsecure:
		return cfg.Insecure(), true
	case optKeyEditorConfig:
		return cfg.EditorConfig(), true
	case optKeyContinueComments:
		return cfg.ContinueComments(), true
	case optKeyCursorline:
		return cfg.Cursorline(), true
	case optKeyIndentGuidesRender:
		return cfg.Editor.IndentGuides.Render, true
	case optKeySearchSmartCase:
		return cfg.SearchSmartCase(), true
	case optKeySearchWrapAround:
		return cfg.SearchWrapAround(), true
	case optKeySoftWrapEnable:
		return softWrapEnable(cfg), true
	case optKeySoftWrapWrapAtTextWidth:
		return softWrapAtTextWidth(cfg), true
	case optKeyCursorcolumn:
		return cfg.Cursorcolumn(), true
	case optKeyMouse:
		return cfg.Mouse(), true
	case optKeyMiddlePaste:
		return cfg.MiddleClickPaste(), true
	default:
		return false, false
	}
}

func lineEndingConfigString(le core.LineEnding) string {
	switch le {
	case core.LineEndingLF:
		return "lf"
	case core.LineEndingCRLF:
		return "crlf"
	default:
		return ""
	}
}

func normalizeKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}

func parseBool(value string) (bool, error) {
	v, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%w: %s", ErrInvalidOption, value)
	}
	return v, nil
}

func parseCursorKind(value string) (CursorKind, error) {
	var c CursorKind
	if err := c.UnmarshalText([]byte(value)); err != nil {
		return "", fmt.Errorf("%w: %s", ErrInvalidOption, value)
	}
	return c, nil
}

func parsePositiveInt(value string) (int, error) {
	v, err := strconv.Atoi(value)
	if err != nil || v < 1 {
		return 0, fmt.Errorf("%w: %s", ErrInvalidOption, value)
	}
	return v, nil
}

func parseNonNegInt(value string) (int, error) {
	v, err := strconv.Atoi(value)
	if err != nil || v < 0 {
		return 0, fmt.Errorf("%w: %s", ErrInvalidOption, value)
	}
	return v, nil
}

func parseIntSlice(value string) ([]int, error) {
	var raw struct {
		Value []int `toml:"value"`
	}
	if _, err := toml.Decode("value = "+value, &raw); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidOption, value)
	}
	return raw.Value, nil
}

func parseStringSlice(value string) ([]string, error) {
	var raw struct {
		Value []string `toml:"value"`
	}
	if _, err := toml.Decode("value = "+value, &raw); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidOption, value)
	}
	return raw.Value, nil
}

func parseStringLiteral(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	switch value[0] {
	case '"', '\'':
		var raw struct {
			Value string `toml:"value"`
		}
		if _, err := toml.Decode("value = "+value, &raw); err != nil {
			return "", fmt.Errorf("%w: %s", ErrInvalidOption, value)
		}
		return raw.Value, nil
	default:
		return value, nil
	}
}

func formatIntSlice(values []int) string {
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, strconv.Itoa(v))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func formatStringSlice(values []string) string {
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, strconv.Quote(v))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func parseLineNumber(value string) (LineNumber, error) {
	var l LineNumber
	if err := l.UnmarshalText([]byte(value)); err != nil {
		return "", fmt.Errorf("%w: %s", ErrInvalidOption, value)
	}
	return l, nil
}

func softWrapEnable(cfg *Config) bool {
	return boolValue(nil, cfg.Editor.SoftWrap.Enable, false)
}

func softWrapAtTextWidth(cfg *Config) bool {
	return boolValue(nil, cfg.Editor.SoftWrap.WrapAtTextWidth, false)
}

func parseBufferLine(value string) (BufferLine, error) {
	var b BufferLine
	if err := b.UnmarshalText([]byte(value)); err != nil {
		return "", fmt.Errorf("%w: %s", ErrInvalidOption, value)
	}
	return b, nil
}
