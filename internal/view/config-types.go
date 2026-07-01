package view

import (
	"errors"
	"fmt"
	"slices"
)

type (
	CursorShape struct {
		Normal CursorKind `toml:"normal"`
		Select CursorKind `toml:"select"`
		Insert CursorKind `toml:"insert"`
	}

	CursorKind string

	StatusLine struct {
		Left      []StatusLineElement `toml:"left"`
		Center    []StatusLineElement `toml:"center"`
		Right     []StatusLineElement `toml:"right"`
		Separator string              `toml:"separator"`
		Mode      StatusLineModeNames `toml:"mode"`
	}

	StatusLineElement string

	StatusLineModeNames struct {
		Normal string `toml:"normal"`
		Insert string `toml:"insert"`
		Select string `toml:"select"`
	}

	LineNumber string

	BufferLine string

	Whitespace struct {
		Render     WhitespaceRender     `toml:"render"`
		Characters WhitespaceCharacters `toml:"characters"`
	}

	// WhitespaceRender holds per-character render settings. When set from a
	// plain string in TOML, Default carries the value and all per-character
	// fields remain nil. When set as a table, each field is independently
	// nullable and falls back to Default then "none"
	WhitespaceRender struct {
		Default *WhitespaceRenderValue
		Space   *WhitespaceRenderValue
		Nbsp    *WhitespaceRenderValue
		Nnbsp   *WhitespaceRenderValue
		Tab     *WhitespaceRenderValue
		Newline *WhitespaceRenderValue
	}

	WhitespaceRenderValue string

	WhitespaceCharacters struct {
		Space   string `toml:"space"`
		Nbsp    string `toml:"nbsp"`
		Nnbsp   string `toml:"nnbsp"`
		Tab     string `toml:"tab"`
		Tabpad  string `toml:"tabpad"`
		Newline string `toml:"newline"`
	}

	IndentGuides struct {
		Render     bool   `toml:"render"`
		Character  string `toml:"character"`
		SkipLevels *int   `toml:"skip-levels"`
	}

	// Gutter controls which gutters are shown and in what order. In TOML it can
	// be an array of type strings or a table with layout/line-numbers. Present
	// tracks whether the config was explicitly set
	Gutter struct {
		Present     bool
		Layout      []GutterType      `toml:"layout"`
		LineNumbers GutterLineNumbers `toml:"line-numbers"`
	}

	GutterType string

	GutterLineNumbers struct {
		MinWidth *int `toml:"min-width"`
	}
)

const (
	DefaultTheme = "mocha"

	BufferLineNever    BufferLine = "never"
	BufferLineAlways   BufferLine = "always"
	BufferLineMultiple BufferLine = "multiple"

	StatusLineMode             StatusLineElement = "mode"
	StatusLineSpinner          StatusLineElement = "spinner"
	StatusLineFileBaseName     StatusLineElement = "file-base-name"
	StatusLineFileName         StatusLineElement = "file-name"
	StatusLineFileAbsolutePath StatusLineElement = "file-absolute-path"
	StatusLineReadOnly         StatusLineElement = "read-only-indicator"
	StatusLineFileEncoding     StatusLineElement = "file-encoding"
	StatusLineFileLineEnding   StatusLineElement = "file-line-ending"
	StatusLineFileIndentStyle  StatusLineElement = "file-indent-style"
	StatusLineFileType         StatusLineElement = "file-type"
	StatusLineDiagnostics      StatusLineElement = "diagnostics"
	StatusLineWorkspaceDiag    StatusLineElement = "workspace-diagnostics"
	StatusLineSelections       StatusLineElement = "selections"
	StatusLinePrimaryLen       StatusLineElement = "primary-selection-length"
	StatusLinePosition         StatusLineElement = "position"
	StatusLineSeparator        StatusLineElement = "separator"
	StatusLinePercent          StatusLineElement = "position-percentage"
	StatusLineTotalLines       StatusLineElement = "total-line-numbers"
	StatusLineSpacer           StatusLineElement = "spacer"
	StatusLineVersionControl   StatusLineElement = "version-control"
	StatusLineRegister         StatusLineElement = "register"
	StatusLineModified         StatusLineElement = "file-modified-indicator"

	CursorKindBlock     CursorKind = "block"
	CursorKindBar       CursorKind = "bar"
	CursorKindUnderline CursorKind = "underline"
	CursorKindHidden    CursorKind = "hidden"

	LineNumberAbsolute LineNumber = "absolute"
	LineNumberRelative LineNumber = "relative"

	DefaultScrollOff     = 5
	DefaultScrollLines   = 3
	DefaultAutoSaveDelay = 3000

	WhitespaceRenderNone WhitespaceRenderValue = "none"
	WhitespaceRenderAll  WhitespaceRenderValue = "all"

	DefaultWSSpace        = '\u00b7' // '·' - middle dot (·)
	DefaultWSNbsp         = '\u237d' // '⍽' - shouldered open box (⍽)
	DefaultWSNnbsp        = '\u2423' // '␣' - open box (␣)
	DefaultWSTab          = '\u2192' // '→' - rightwards arrow (→)
	DefaultWSNewline      = '\u23ce' // '⏎' - return symbol (⏎)
	DefaultWSTabpad  rune = ' '

	DefaultIndentGuideChar = '\u2502' // U+2502 box drawings light vertical

	DefaultStatusLineSeparator = "\u2502" // '│' box drawings light vertical

	GutterTypeDiagnostics GutterType = "diagnostics"
	GutterTypeLineNumbers GutterType = "line-numbers"
	GutterTypeSpacer      GutterType = "spacer"
	GutterTypeDiff        GutterType = "diff"

	DefaultGutterLineNumberMinWidth = 3
)

var (
	ErrInvalidCursorKind       = errors.New("invalid cursor kind")
	ErrInvalidLineNumber       = errors.New("invalid line-number value")
	ErrInvalidStatusLine       = errors.New("invalid statusline element")
	ErrInvalidWhitespaceRender = errors.New("invalid whitespace render value")
	ErrInvalidGutterType       = errors.New("invalid gutter type")
	ErrInvalidBufferLine       = errors.New("invalid bufferline value")
)

func ParseCursorKind(value string) (CursorKind, error) {
	var c CursorKind
	if err := c.UnmarshalText([]byte(value)); err != nil {
		return "", fmt.Errorf("%w: %s", ErrInvalidCursorKind, value)
	}
	return c, nil
}

func ParseLineNumber(value string) (LineNumber, error) {
	var l LineNumber
	if err := l.UnmarshalText([]byte(value)); err != nil {
		return "", fmt.Errorf("%w: %s", ErrInvalidLineNumber, value)
	}
	return l, nil
}

func ParseBufferLine(value string) (BufferLine, error) {
	var b BufferLine
	if err := b.UnmarshalText([]byte(value)); err != nil {
		return "", fmt.Errorf("%w: %s", ErrInvalidBufferLine, value)
	}
	return b, nil
}

func ParseWhitespaceRenderValue(s string) (WhitespaceRenderValue, error) {
	switch WhitespaceRenderValue(s) {
	case WhitespaceRenderNone, WhitespaceRenderAll:
		return WhitespaceRenderValue(s), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidWhitespaceRender, s)
	}
}

func (i IndentGuides) CharRune() rune {
	return runeOrDefault(i.Character, DefaultIndentGuideChar)
}

func (i IndentGuides) GetSkipLevels() int {
	if i.SkipLevels != nil {
		return *i.SkipLevels
	}
	return 0
}

func (g *Gutter) GutterLayout() []GutterType {
	if g.Present {
		return g.Layout
	}
	return []GutterType{
		GutterTypeDiagnostics,
		GutterTypeSpacer,
		GutterTypeLineNumbers,
		GutterTypeSpacer,
		GutterTypeDiff,
	}
}

func (g *Gutter) HasGutterType(gt GutterType) bool {
	return slices.Contains(g.GutterLayout(), gt)
}

func (g *Gutter) LineNumberMinWidth() int {
	return intOr(g.LineNumbers.MinWidth, DefaultGutterLineNumberMinWidth)
}

func (c *CursorKind) UnmarshalText(text []byte) error {
	switch CursorKind(text) {
	case CursorKindBlock, CursorKindBar, CursorKindUnderline, CursorKindHidden:
		*c = CursorKind(text)
	default:
		return fmt.Errorf("%w: %s", ErrInvalidCursorKind, text)
	}
	return nil
}

func (l *LineNumber) UnmarshalText(text []byte) error {
	switch LineNumber(text) {
	case LineNumberAbsolute, LineNumberRelative:
		*l = LineNumber(text)
	default:
		return fmt.Errorf("%w: %s", ErrInvalidLineNumber, text)
	}
	return nil
}

func (s *StatusLineElement) UnmarshalText(text []byte) error {
	switch StatusLineElement(text) {
	case StatusLineMode, StatusLineSpinner, StatusLineFileBaseName,
		StatusLineFileName, StatusLineFileAbsolutePath,
		StatusLineModified, StatusLineReadOnly,
		StatusLineFileEncoding, StatusLineFileLineEnding,
		StatusLineFileIndentStyle, StatusLineFileType, StatusLineDiagnostics,
		StatusLineWorkspaceDiag, StatusLineSelections,
		StatusLinePrimaryLen, StatusLinePosition,
		StatusLineSeparator, StatusLinePercent,
		StatusLineTotalLines, StatusLineSpacer, StatusLineVersionControl,
		StatusLineRegister:
		*s = StatusLineElement(text)
	default:
		return fmt.Errorf("%w: %s", ErrInvalidStatusLine, text)
	}
	return nil
}

func (b *BufferLine) UnmarshalText(text []byte) error {
	switch BufferLine(text) {
	case BufferLineNever, BufferLineAlways, BufferLineMultiple:
		*b = BufferLine(text)
	default:
		return fmt.Errorf("%w: %s", ErrInvalidBufferLine, text)
	}
	return nil
}

func (g *GutterType) UnmarshalText(text []byte) error {
	switch GutterType(text) {
	case GutterTypeDiagnostics, GutterTypeLineNumbers,
		GutterTypeSpacer, GutterTypeDiff:
		*g = GutterType(text)
	default:
		return fmt.Errorf("%w: %s", ErrInvalidGutterType, text)
	}
	return nil
}

func (g *Gutter) UnmarshalTOML(value any) error {
	switch v := value.(type) {
	case []any:
		layout, err := decodeGutterLayout(v)
		if err != nil {
			return err
		}
		g.Present = true
		g.Layout = layout
	case map[string]any:
		g.Present = true
		if raw, ok := v["layout"]; ok {
			items, ok := raw.([]any)
			if !ok {
				return fmt.Errorf("%w: layout must be an array",
					ErrInvalidGutterType)
			}
			layout, err := decodeGutterLayout(items)
			if err != nil {
				return err
			}
			g.Layout = layout
		}
		if raw, ok := v["line-numbers"].(map[string]any); ok {
			g.LineNumbers.MinWidth = intPtr(raw["min-width"])
		}
	case nil:
	default:
		return fmt.Errorf("%w: %v", ErrInvalidGutterType, value)
	}
	return nil
}

func (w *WhitespaceRender) SpaceRender() WhitespaceRenderValue {
	return whitespaceRenderFor(w.Space, w.Default)
}

func (w *WhitespaceRender) NbspRender() WhitespaceRenderValue {
	return whitespaceRenderFor(w.Nbsp, w.Default)
}

func (w *WhitespaceRender) NnbspRender() WhitespaceRenderValue {
	return whitespaceRenderFor(w.Nnbsp, w.Default)
}

func (w *WhitespaceRender) TabRender() WhitespaceRenderValue {
	return whitespaceRenderFor(w.Tab, w.Default)
}

func (w *WhitespaceRender) NewlineRender() WhitespaceRenderValue {
	return whitespaceRenderFor(w.Newline, w.Default)
}

func (w *WhitespaceRender) UnmarshalTOML(value any) error {
	switch v := value.(type) {
	case string:
		rv, err := ParseWhitespaceRenderValue(v)
		if err != nil {
			return err
		}
		w.Default = &rv
	case map[string]any:
		for key, raw := range v {
			s, ok := raw.(string)
			if !ok {
				return fmt.Errorf(
					"%w: %v", ErrInvalidWhitespaceRender, raw,
				)
			}
			rv, err := ParseWhitespaceRenderValue(s)
			if err != nil {
				return err
			}
			switch key {
			case "default":
				w.Default = &rv
			case "space":
				w.Space = &rv
			case "nbsp":
				w.Nbsp = &rv
			case "nnbsp":
				w.Nnbsp = &rv
			case "tab":
				w.Tab = &rv
			case "newline":
				w.Newline = &rv
			}
		}
	case nil:
	default:
		return fmt.Errorf("%w: %v", ErrInvalidWhitespaceRender, value)
	}
	return nil
}

func (w *WhitespaceCharacters) SpaceRune() rune {
	return runeOrDefault(w.Space, DefaultWSSpace)
}

func (w *WhitespaceCharacters) NbspRune() rune {
	return runeOrDefault(w.Nbsp, DefaultWSNbsp)
}

func (w *WhitespaceCharacters) NnbspRune() rune {
	return runeOrDefault(w.Nnbsp, DefaultWSNnbsp)
}

func (w *WhitespaceCharacters) TabRune() rune {
	return runeOrDefault(w.Tab, DefaultWSTab)
}

func (w *WhitespaceCharacters) TabpadRune() rune {
	return runeOrDefault(w.Tabpad, DefaultWSTabpad)
}

func (w *WhitespaceCharacters) NewlineRune() rune {
	return runeOrDefault(w.Newline, DefaultWSNewline)
}

func intOr(p *int, fallback int) int {
	if p != nil {
		return *p
	}
	return fallback
}

func intPtr(value any) *int {
	switch v := value.(type) {
	case int:
		return &v
	case int64:
		return new(int(v))
	default:
		return nil
	}
}

func decodeGutterLayout(items []any) ([]GutterType, error) {
	layout := make([]GutterType, 0, len(items))
	for _, item := range items {
		s, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("%w: %v", ErrInvalidGutterType, item)
		}
		var gt GutterType
		if err := gt.UnmarshalText([]byte(s)); err != nil {
			return nil, err
		}
		layout = append(layout, gt)
	}
	return layout, nil
}

func whitespaceRenderFor(
	specific, def *WhitespaceRenderValue,
) WhitespaceRenderValue {
	if specific != nil {
		return *specific
	}
	if def != nil {
		return *def
	}
	return WhitespaceRenderNone
}

func runeOrDefault(s string, fallback rune) rune {
	if s == "" {
		return fallback
	}
	for _, r := range s {
		return r
	}
	return fallback
}
