package view

import (
	"slices"
	"strings"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view/language"
)

// Options holds the editor's typed runtime config values. Fields are exported
// so module Apply functions can write to them directly
type Options struct {
	Theme                string
	ScrollOff            int
	ScrollLines          int
	Mouse                bool
	MiddleClickPaste     bool
	Shell                []string
	AutoSaveFocusLost    bool
	AutoSaveAfterDelay   bool
	AutoSaveDelayTimeout int
	AtomicSave           bool
	InsertFinalNewline   bool
	TrimFinalNewlines    bool
	TrimTrailingWS       bool
	EditorConfig         bool
	Insecure             bool
	ContinueComments     bool
	SearchSmartCase      bool
	SearchWrapAround     bool
	Cursorline           bool
	Cursorcolumn         bool
	TextWidth            *int
	SoftWrap             language.SoftWrap
	DefaultLineEnding    core.LineEnding
	Rulers               []int
	LineNumber           LineNumber
	Gutters              Gutter
	Whitespace           Whitespace
	IndentGuides         IndentGuides
	CursorShape          CursorShape
	StatusLine           StatusLine
	AutoPairMap          core.AutoPairs
	HasAutoPairs         bool
	BufferLine           BufferLine
}

// StatusLineSeparator returns the status line separator string with default
func (o *Options) StatusLineSeparator() string {
	if o.StatusLine.Separator != "" {
		return o.StatusLine.Separator
	}
	return DefaultStatusLineSeparator
}

// ModeNameForMode returns the display name for the given mode string
func (o *Options) ModeNameForMode(mode string) string {
	switch strings.ToLower(mode) {
	case "ins", "insert":
		if o.StatusLine.Mode.Insert != "" {
			return o.StatusLine.Mode.Insert
		}
		return "INS"
	case "sel", "select":
		if o.StatusLine.Mode.Select != "" {
			return o.StatusLine.Mode.Select
		}
		return "SEL"
	default:
		if o.StatusLine.Mode.Normal != "" {
			return o.StatusLine.Mode.Normal
		}
		return "NOR"
	}
}

// StatusLineLeft returns the left status line elements with defaults
func (o *Options) StatusLineLeft() []StatusLineElement {
	if len(o.StatusLine.Left) > 0 {
		return slices.Clone(o.StatusLine.Left)
	}
	return []StatusLineElement{
		StatusLineMode,
		StatusLineSpinner,
		StatusLineFileName,
		StatusLineReadOnly,
		StatusLineModified,
	}
}

// StatusLineCenter returns the center status line elements
func (o *Options) StatusLineCenter() []StatusLineElement {
	return slices.Clone(o.StatusLine.Center)
}

// StatusLineRight returns the right status line elements with defaults
func (o *Options) StatusLineRight() []StatusLineElement {
	if len(o.StatusLine.Right) > 0 {
		return slices.Clone(o.StatusLine.Right)
	}
	return []StatusLineElement{
		StatusLineDiagnostics,
		StatusLineSelections,
		StatusLineRegister,
		StatusLinePosition,
		StatusLineFileEncoding,
	}
}

// AutoPairs returns the auto-pair map and whether auto-pairs are enabled
func (o *Options) AutoPairs() (core.AutoPairs, bool) {
	return o.AutoPairMap, o.HasAutoPairs
}

// CursorShapeForMode returns the cursor shape for the given mode string
// ("NOR", "INS", "SEL")
func (o *Options) CursorShapeForMode(mode string) CursorKind {
	var k CursorKind
	switch mode {
	case "INS":
		k = o.CursorShape.Insert
	case "SEL":
		k = o.CursorShape.Select
	default:
		k = o.CursorShape.Normal
	}
	if k == "" {
		return CursorKindBlock
	}
	return k
}

func defaultOptions() Options {
	return Options{
		Theme:                DefaultTheme,
		ScrollOff:            DefaultScrollOff,
		ScrollLines:          DefaultScrollLines,
		Mouse:                true,
		MiddleClickPaste:     true,
		Shell:                DefaultShell(),
		AtomicSave:           true,
		InsertFinalNewline:   true,
		EditorConfig:         true,
		ContinueComments:     true,
		SearchSmartCase:      true,
		SearchWrapAround:     true,
		AutoSaveDelayTimeout: DefaultAutoSaveDelay,
		LineNumber:           LineNumberAbsolute,
		BufferLine:           BufferLineNever,
		AutoPairMap:          core.DefaultAutoPairs(),
		HasAutoPairs:         true,
	}
}
