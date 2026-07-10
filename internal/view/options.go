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
	AutoSession          bool
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

// StatusLineLeft returns the left status line items with defaults
func (o *Options) StatusLineLeft() []StatusLineItem {
	if len(o.StatusLine.Left) > 0 {
		return slices.Clone(o.StatusLine.Left)
	}
	return []StatusLineItem{
		{Element: StatusLineMode, Pinned: true},
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
		{Element: StatusLinePosition, Pinned: true},
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
		Theme: DefaultTheme,
		CursorShape: CursorShape{
			Normal: CursorKindBlock,
			Insert: CursorKindBar,
			Select: CursorKindUnderline,
		},
		ScrollOff:            DefaultScrollOff,
		ScrollLines:          DefaultScrollLines,
		Mouse:                true,
		MiddleClickPaste:     true,
		Shell:                DefaultShell(),
		AtomicSave:           true,
		InsertFinalNewline:   true,
		EditorConfig:         true,
		AutoSession:          true,
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
