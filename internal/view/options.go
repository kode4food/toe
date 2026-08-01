package view

import (
	"slices"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view/language"
)

// Options holds the editor's typed runtime config values. Fields are exported
// so module Apply functions can write to them directly
type Options struct {
	Theme                string
	ScrollOff            int
	ScrollLines          int
	InactiveDim          int
	Mouse                bool
	MiddleClickPaste     bool
	NerdFonts            bool
	Shell                []string
	AutoSaveFocusLost    bool
	AutoSaveAfterDelay   bool
	AutoSaveDelayTimeout int
	AtomicSave           bool
	InsertFinalNewline   bool
	TrimFinalNewlines    bool
	TrimTrailingWS       bool
	EditorConfig         bool
	AutoSession          bool
	FileWatch            bool
	Insecure             bool
	ContinueComments     bool
	SearchSmartCase      bool
	SearchWrapAround     bool
	CursorLine           bool
	CursorColumn         bool
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
	Gen                  int
}

// StatusLineSeparator returns the status line separator string with default
func (o *Options) StatusLineSeparator() string {
	if o.StatusLine.Separator != "" {
		return o.StatusLine.Separator
	}
	return DefaultStatusLineSeparator
}

// StatusLineLeft returns the left status line items with defaults
func (o *Options) StatusLineLeft() []StatusLineItem {
	if len(o.StatusLine.Left) > 0 {
		return slices.Clone(o.StatusLine.Left)
	}
	return []StatusLineItem{
		{Element: StatusLineMode, Pinned: true},
		{Element: StatusLineSpinner},
		{Element: StatusLineFileName},
		{Element: StatusLineReadOnly},
		{Element: StatusLineModified},
	}
}

// StatusLineRight returns the right status line items with defaults
func (o *Options) StatusLineRight() []StatusLineItem {
	if len(o.StatusLine.Right) > 0 {
		return slices.Clone(o.StatusLine.Right)
	}
	return []StatusLineItem{
		{Element: StatusLineDiagnostics},
		{Element: StatusLineSelections},
		{Element: StatusLineRegister},
		{Element: StatusLineFileEncoding},
		{Element: StatusLinePosition, Pinned: true},
	}
}

// AutoPairs returns the auto-pair map and whether auto-pairs are enabled
func (o *Options) AutoPairs() (core.AutoPairs, bool) {
	return o.AutoPairMap, o.HasAutoPairs
}

// CursorShapeForMode returns the cursor shape for the given mode
func (o *Options) CursorShapeForMode(mode Mode) CursorKind {
	var k CursorKind
	switch mode {
	case ModeInsert:
		k = o.CursorShape.Insert
	case ModeSelect:
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
		Theme: DefaultTheme,
		CursorShape: CursorShape{
			Normal: CursorKindBlock,
			Insert: CursorKindBar,
			Select: CursorKindUnderline,
		},
		ScrollOff:            DefaultScrollOff,
		ScrollLines:          DefaultScrollLines,
		InactiveDim:          DefaultInactiveDim,
		Mouse:                true,
		MiddleClickPaste:     true,
		NerdFonts:            true,
		Shell:                DefaultShell(),
		AtomicSave:           true,
		InsertFinalNewline:   true,
		EditorConfig:         true,
		AutoSession:          true,
		FileWatch:            true,
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
