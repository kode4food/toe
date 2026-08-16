package config

import (
	"cmp"
	"embed"
	"fmt"
	"strconv"
	"unicode/utf8"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/builtin/kit"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
	"github.com/kode4food/toe/internal/view/config"
	"github.com/kode4food/toe/internal/view/language"
)

type viewSection struct {
	Editor struct {
		LineNumber   view.LineNumber   `toml:"line-number"`
		InactiveDim  *int              `toml:"inactive-dim"`
		CursorLine   *bool             `toml:"cursorline"`
		CursorColumn *bool             `toml:"cursorcolumn"`
		Animation    *bool             `toml:"animation"`
		AutoSize     *bool             `toml:"auto-size"`
		TextWidth    *int              `toml:"text-width"`
		SoftWrap     language.SoftWrap `toml:"soft-wrap"`
		Rulers       []int             `toml:"rulers"`
		BufferLine   view.BufferLine   `toml:"bufferline"`
		Whitespace   view.Whitespace   `toml:"whitespace"`
		IndentGuides view.IndentGuides `toml:"indent-guides"`
		Gutters      view.Gutter       `toml:"gutters"`
	} `toml:"editor"`
}

const (
	actPageUp                 = "page_up"
	actPageDown               = "page_down"
	actPageCursorHalfUp       = "page_cursor_half_up"
	actPageCursorHalfDown     = "page_cursor_half_down"
	actHalfPageUp             = "half_page_up"
	actHalfPageDown           = "half_page_down"
	actPageCursorUp           = "page_cursor_up"
	actPageCursorDown         = "page_cursor_down"
	actCenterCursorLine       = "center_cursor_line"
	actCenterCursorLineTop    = "align_view_top"
	actCenterCursorLineBottom = "align_view_bottom"
	actScrollUp               = "scroll_up"
	actScrollDown             = "scroll_down"
	actImageZoomIn            = "image_zoom_in"
	actImageZoomOut           = "image_zoom_out"
	actImageZoomReset         = "image_zoom_reset"
	actImagePanLeft           = "image_pan_left"
	actImagePanDown           = "image_pan_down"
	actImagePanUp             = "image_pan_up"
	actImagePanRight          = "image_pan_right"
	actVSplitView             = "vsplit"
	actHSplitView             = "hsplit"
	actVSplitNew              = "vsplit_new"
	actHSplitNew              = "hsplit_new"
	actTransposeView          = "transpose_view"
	actCloseCurrentView       = "wclose"
	actCloseCurrentViewForce  = "wclose!"
	actCloseOtherViews        = "wonly"
	actTerminal               = "terminal"
	actTerminalSearch         = "terminal_search"
	actJumpViewLeft           = "jump_view_left"
	actJumpViewDown           = "jump_view_down"
	actJumpViewUp             = "jump_view_up"
	actJumpViewRight          = "jump_view_right"
	actSwapViewLeft           = "swap_view_left"
	actSwapViewDown           = "swap_view_down"
	actSwapViewUp             = "swap_view_up"
	actSwapViewRight          = "swap_view_right"
	actRotateView             = "rotate_view"
	actTogglePaneMaximized    = "toggle_pane_maximized"
	actResizeView             = "resize_view"

	hintResizeKey = i18n.Key("hint.resize")
)

//go:embed i18n/view.*.json
var viewFS embed.FS

// ViewModule returns the split, scroll, and view-option commands
func ViewModule(model ui.Model) command.Module {
	cfg := new(viewSection)
	z := kit.Prefixed(kit.Char('z'))
	Z := kit.Prefixed(kit.Char('Z'))
	Spcw := kit.Prefixed(kit.LeaderPrefix(kit.Char('w')))
	Cw := kit.Prefixed(kit.Ctrl('w'))

	return command.Module{
		Translations: i18n.LoadTranslations(viewFS),
		Commands: []command.Command{
			{
				Name:      actImageZoomIn,
				DocString: "Zoom image in",
				Run:       kit.Runner(runImgAction((*ui.ImagePane).ZoomIn)),
				Modes:     view.ModeImage,
				Keys:      kit.Keys(kit.Char('+'), kit.Char('=')),
				Aliases:   []string{"zoom-in"},
			},
			{
				Name:      actImageZoomOut,
				DocString: "Zoom image out",
				Run:       kit.Runner(runImgAction((*ui.ImagePane).ZoomOut)),
				Modes:     view.ModeImage,
				Keys:      kit.Keys(kit.Char('-')),
				Aliases:   []string{"zoom-out"},
			},
			{
				Name:      actImageZoomReset,
				DocString: "Fit image to pane",
				Run:       kit.Runner(runImgAction((*ui.ImagePane).ResetZoom)),
				Modes:     view.ModeImage,
				Keys:      kit.Keys(kit.Char('0')),
				Aliases:   []string{"zoom-reset"},
			},
			{
				Name:      actImagePanLeft,
				DocString: "Pan image left",
				Run:       kit.Runner(runImgAction((*ui.ImagePane).PanLeft)),
				Modes:     view.ModeImage,
				Keys:      kit.Keys(kit.Char('h'), kit.Left),
				Aliases:   []string{"pan-left"},
			},
			{
				Name:      actImagePanDown,
				DocString: "Pan image down",
				Run:       kit.Runner(runImgAction((*ui.ImagePane).PanDown)),
				Modes:     view.ModeImage,
				Keys:      kit.Keys(kit.Char('j'), kit.Down),
				Aliases:   []string{"pan-down"},
			},
			{
				Name:      actImagePanUp,
				DocString: "Pan image up",
				Run:       kit.Runner(runImgAction((*ui.ImagePane).PanUp)),
				Modes:     view.ModeImage,
				Keys:      kit.Keys(kit.Char('k'), kit.Up),
				Aliases:   []string{"pan-up"},
			},
			{
				Name:      actImagePanRight,
				DocString: "Pan image right",
				Run:       kit.Runner(runImgAction((*ui.ImagePane).PanRight)),
				Modes:     view.ModeImage,
				Keys:      kit.Keys(kit.Char('l'), kit.Right),
				Aliases:   []string{"pan-right"},
			},
			{
				Name:      actPageUp,
				DocString: "Move page up",
				Run:       kit.Runner(action.PageUp),
				Modes:     command.DocModes,
				Keys: map[view.Mode]command.KeyBinding{view.ModeAny: kit.Or(
					kit.Or(kit.Ctrl('b'), kit.PgUp),
					kit.Or(z(kit.Ctrl('b')), z(kit.PgUp)),
					kit.Or(Z(kit.Ctrl('b')), Z(kit.PgUp)),
				)},
			},
			{
				Name:      actPageDown,
				DocString: "Move page down",
				Run:       kit.Runner(action.PageDown),
				Modes:     command.DocModes,
				Keys: map[view.Mode]command.KeyBinding{view.ModeAny: kit.Or(
					kit.Or(kit.Ctrl('f'), kit.PgDn),
					kit.Or(z(kit.Ctrl('f')), z(kit.PgDn)),
					kit.Or(Z(kit.Ctrl('f')), Z(kit.PgDn)),
				)},
			},
			{
				Name:      actPageCursorHalfUp,
				DocString: "Move page and cursor half up",
				Run:       kit.Runner(action.PageCursorHalfUp),
				Modes:     command.DocNormalModes,
				Keys: map[view.Mode]command.KeyBinding{view.ModeAny: kit.Or(
					kit.Or(kit.Ctrl('u')),
					kit.Or(z(kit.Ctrl('u')), z(kit.Bksp)),
					kit.Or(Z(kit.Ctrl('u')), Z(kit.Bksp)),
				)},
			},
			{
				Name:      actPageCursorHalfDown,
				DocString: "Move page and cursor half down",
				Run:       kit.Runner(action.PageCursorHalfDown),
				Modes:     command.DocNormalModes,
				Keys: map[view.Mode]command.KeyBinding{view.ModeAny: kit.Or(
					kit.Or(kit.Ctrl('d')),
					kit.Or(z(kit.Ctrl('d')), z(kit.Char(' '))),
					kit.Or(Z(kit.Ctrl('d')), Z(kit.Char(' '))),
				)},
			},
			{
				Name:      actHalfPageUp,
				DocString: "Move half page up",
				Run:       kit.Runner(action.HalfPageUp),
				Modes:     command.DocModes,
			},
			{
				Name:      actHalfPageDown,
				DocString: "Move half page down",
				Run:       kit.Runner(action.HalfPageDown),
				Modes:     command.DocModes,
			},
			{
				Name:      actPageCursorUp,
				DocString: "Move page and cursor up",
				Run:       kit.Runner(action.PageUp),
				Modes:     command.DocModes,
			},
			{
				Name:      actPageCursorDown,
				DocString: "Move page and cursor down",
				Run:       kit.Runner(action.PageDown),
				Modes:     command.DocModes,
			},
			{
				Name:      actCenterCursorLine,
				DocString: "Align view center",
				Run:       kit.Runner(action.AlignViewCenter),
				Modes:     command.DocNormalModes,
				Keys: map[view.Mode]command.KeyBinding{view.ModeAny: kit.Or(
					kit.Or(z(kit.Char('z')), z(kit.Char('c'))),
					kit.Or(Z(kit.Char('z')), Z(kit.Char('c'))),
				)},
			},
			{
				Name:      actCenterCursorLineTop,
				DocString: "Align view top",
				Run:       kit.Runner(action.AlignViewTop),
				Modes:     command.DocNormalModes,
				Keys: map[view.Mode]command.KeyBinding{view.ModeAny: kit.Or(
					kit.Or(z(kit.Char('.')), z(kit.Char('t'))),
					kit.Or(Z(kit.Char('.')), Z(kit.Char('t'))),
				)},
			},
			{
				Name:      actCenterCursorLineBottom,
				DocString: "Align view bottom",
				Run:       kit.Runner(action.AlignViewBottom),
				Modes:     command.DocNormalModes,
				Keys: map[view.Mode]command.KeyBinding{view.ModeAny: kit.Or(
					kit.Or(z(kit.Char('b'))),
					kit.Or(Z(kit.Char('b'))),
				)},
			},
			{
				Name:      actScrollUp,
				DocString: "Scroll view up",
				Run:       kit.Runner(action.ScrollUp),
				Counted:   true,
				Modes:     command.DocNormalModes,
				Keys: map[view.Mode]command.KeyBinding{view.ModeAny: kit.Or(
					kit.Or(z(kit.Char('k')), z(kit.Up)),
					kit.Or(Z(kit.Char('k')), Z(kit.Up)),
				)},
			},
			{
				Name:      actScrollDown,
				DocString: "Scroll view down",
				Run:       kit.Runner(action.ScrollDown),
				Counted:   true,
				Modes:     command.DocNormalModes,
				Keys: map[view.Mode]command.KeyBinding{view.ModeAny: kit.Or(
					kit.Or(z(kit.Char('j')), z(kit.Down)),
					kit.Or(Z(kit.Char('j')), Z(kit.Down)),
				)},
			},
			{
				Name:      actTerminal,
				DocString: "Open a new terminal",
				Run:       kit.Runner(model.TerminalAction),
				Modes:     command.CmdKeyModes,
				Keys: map[view.Mode]command.KeyBinding{view.ModeAny: kit.Or(
					kit.Or(Cw(kit.Char('x'))),
					kit.Or(Spcw(kit.Char('x'))),
				)},
			},
			{
				Name:      actTerminalSearch,
				DocString: "Search focused terminal's scrollback",
				Run:       kit.Runner(model.TerminalSearchAction),
				Modes:     view.ModeTerminal,
				Keys:      kit.Window(kit.Char('/')),
			},
			{
				Name: actVSplitView,
				DocString: "Vertical right split. Opens the given files in " +
					"the split",
				Run:       splitRun(view.LayoutVertical),
				Modes:     command.PaneModes,
				Keys:      kit.Window(kit.Char('v'), kit.Ctrl('v')),
				Aliases:   []string{"vs"},
				Signature: kit.FileSig(kit.MinArgs(0)),
			},
			{
				Name: actHSplitView,
				DocString: "Horizontal bottom split. Opens the given files " +
					"in the split",
				Run:       splitRun(view.LayoutHorizontal),
				Modes:     command.PaneModes,
				Keys:      kit.Window(kit.Char('s'), kit.Ctrl('s')),
				Aliases:   []string{"split", "hs", "sp"},
				Signature: kit.FileSig(kit.MinArgs(0)),
			},
			{
				Name:      actVSplitNew,
				DocString: "Vertical right split scratch buffer",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					e.VSplitNew()
					return command.Result{}
				},
				Modes:   command.PaneModes,
				Aliases: []string{"vnew"},
			},
			{
				Name:      actHSplitNew,
				DocString: "Horizontal bottom split scratch buffer",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					e.HSplitNew()
					return command.Result{}
				},
				Modes:   command.PaneModes,
				Aliases: []string{"hnew"},
			},
			{
				Name:      actTransposeView,
				DocString: "Transpose splits",
				Run:       kit.Runner((*view.Editor).Transpose),
				Modes:     command.PaneModes,
				Keys:      kit.Window(kit.Char('t'), kit.Ctrl('t')),
			},
			{
				Name:      actCloseCurrentView,
				DocString: "Close window",
				Run:       kit.Runner(action.CloseCurrentView),
				Modes:     command.PaneModes,
				Keys:      kit.Window(kit.Char('q'), kit.Ctrl('q')),
				Aliases:   []string{"wc"},
			},
			{
				Name:      actCloseCurrentViewForce,
				DocString: "Force close window",
				Run:       kit.Runner((*view.Editor).CloseCurrentView),
				Modes:     command.PaneModes,
				Aliases:   []string{"wc!"},
			},
			{
				Name:      actCloseOtherViews,
				DocString: "Close windows except current",
				Run:       kit.Runner((*view.Editor).CloseAllOtherViews),
				Modes:     command.PaneModes,
				Keys:      kit.Window(kit.Char('o'), kit.Ctrl('o')),
				Aliases:   []string{"wo"},
			},
			{
				Name:      actRotateView,
				DocString: "Goto next window",
				Run:       kit.Runner((*view.Editor).FocusNextView),
				Modes:     command.PaneModes,
				Keys:      kit.Window(kit.Char('w'), kit.Ctrl('w')),
			},
			{
				Name:      actTogglePaneMaximized,
				DocString: "Toggle focused pane maximized",
				Run:       kit.Runner((*view.Editor).TogglePaneMaximized),
				Modes:     command.PaneModes,
				Keys:      kit.Window(kit.Char('z')),
			},
			{
				Name:      actJumpViewLeft,
				DocString: "Jump to left split",
				Run: kit.Runner(runDirAction(
					(*view.Editor).FocusDirection, view.DirectionLeft,
				)),
				Modes: command.PaneModes,
				Keys: kit.Window(
					kit.Char('h'), kit.Ctrl('h'), kit.Left,
				),
			},
			{
				Name:      actJumpViewDown,
				DocString: "Jump to split below",
				Run: kit.Runner(runDirAction(
					(*view.Editor).FocusDirection, view.DirectionDown,
				)),
				Modes: command.PaneModes,
				Keys: kit.Window(
					kit.Char('j'), kit.Ctrl('j'), kit.Down,
				),
			},
			{
				Name:      actJumpViewUp,
				DocString: "Jump to split above",
				Run: kit.Runner(runDirAction(
					(*view.Editor).FocusDirection, view.DirectionUp,
				)),
				Modes: command.PaneModes,
				Keys: kit.Window(
					kit.Char('k'), kit.Ctrl('k'), kit.Up,
				),
			},
			{
				Name:      actJumpViewRight,
				DocString: "Jump to right split",
				Run: kit.Runner(runDirAction(
					(*view.Editor).FocusDirection, view.DirectionRight,
				)),
				Modes: command.PaneModes,
				Keys: kit.Window(
					kit.Char('l'), kit.Ctrl('l'), kit.Right,
				),
			},
			{
				Name:      actSwapViewLeft,
				DocString: "Swap with left split",
				Run: kit.Runner(runDirAction(
					(*view.Editor).SwapSplitInDirection, view.DirectionLeft,
				)),
				Modes: command.PaneModes,
				Keys:  kit.Window(kit.Char('H')),
			},
			{
				Name:      actSwapViewDown,
				DocString: "Swap with split below",
				Run: kit.Runner(runDirAction(
					(*view.Editor).SwapSplitInDirection, view.DirectionDown,
				)),
				Modes: command.PaneModes,
				Keys:  kit.Window(kit.Char('J')),
			},
			{
				Name:      actSwapViewUp,
				DocString: "Swap with split above",
				Run: kit.Runner(runDirAction(
					(*view.Editor).SwapSplitInDirection, view.DirectionUp,
				)),
				Modes: command.PaneModes,
				Keys:  kit.Window(kit.Char('K')),
			},
			{
				Name:      actSwapViewRight,
				DocString: "Swap with right split",
				Run: kit.Runner(runDirAction(
					(*view.Editor).SwapSplitInDirection, view.DirectionRight,
				)),
				Modes: command.PaneModes,
				Keys:  kit.Window(kit.Char('L')),
			},
			{
				Name:      actResizeView,
				DocString: "Resize split",
				Run:       kit.Continuation(model.ResizeViewAction),
				Hints: func(*view.Editor) []command.KeyHint {
					return []command.KeyHint{{Label: i18n.Text(hintResizeKey)}}
				},
				Modes: command.PaneModes,
				Keys:  kit.Window(kit.Char('r')),
			},
		},
		Options: []command.Option{
			{
				Key:       "line-number",
				DocString: "Line numbers: absolute or relative",
				Get: func(e *view.Editor) (string, error) {
					return string(e.Options().LineNumber), nil
				},
				Set: func(e *view.Editor, s string) error {
					v, err := view.ParseLineNumber(s)
					if err != nil {
						return fmt.Errorf("%w: %s", config.ErrInvalidOption, s)
					}
					e.Options().LineNumber = v
					return nil
				},
				Complete: command.StaticCompleter(
					view.LineNumberAbsolute,
					view.LineNumberRelative,
				),
			},
			kit.EditorBoolOption("cursorline",
				func(e *view.Editor) bool {
					return e.Options().CursorLine
				},
				func(e *view.Editor, v bool) {
					e.Options().CursorLine = v
				},
			).WithDoc("Highlight the cursor's line"),
			kit.EditorBoolOption("cursorcolumn",
				func(e *view.Editor) bool {
					return e.Options().CursorColumn
				},
				func(e *view.Editor, v bool) {
					e.Options().CursorColumn = v
				},
			).WithDoc("Highlight the cursor's column"),
			kit.EditorBoolOption("animation",
				func(*view.Editor) bool {
					return model.Animation()
				},
				func(_ *view.Editor, v bool) {
					model.SetAnimation(v)
				},
			).WithDoc("Animate UI transitions"),
			kit.EditorBoolOption("auto-size",
				func(*view.Editor) bool {
					return model.AutoSize()
				},
				func(_ *view.Editor, v bool) {
					model.SetAutoSize(v)
				},
			).WithDoc("Widen a focused pane to fit its content"),
			kit.EditorNullableIntOption("text-width",
				language.DefaultTextWidth,
				func(e *view.Editor) *int {
					return e.Options().TextWidth
				},
				func(e *view.Editor, v *int) {
					e.Options().TextWidth = v
				},
			).WithDoc("Maximum line length for reflow and wrapping"),
			kit.EditorBoolOption("soft-wrap.enable",
				func(e *view.Editor) bool {
					sw := e.Options().SoftWrap.Enable
					return sw != nil && *sw
				},
				func(e *view.Editor, v bool) {
					e.Options().SoftWrap.Enable = &v
				},
			).WithDoc("Enable soft wrapping"),
			{
				Key:       "soft-wrap.wrap-indicator",
				DocString: "Text shown before soft wrapped lines",
				Get: func(e *view.Editor) (string, error) {
					wi := language.DefaultWrapIndicator
					if e.Options().SoftWrap.WrapIndicator != nil {
						wi = *e.Options().SoftWrap.WrapIndicator
					}
					return wi, nil
				},
				Set: func(e *view.Editor, s string) error {
					v, err := config.ParseStringLiteral(s)
					if err != nil {
						return err
					}
					e.Options().SoftWrap.WrapIndicator = &v
					return nil
				},
			},
			kit.EditorBoolOption("soft-wrap.wrap-at-text-width",
				func(e *view.Editor) bool {
					v := e.Options().SoftWrap.WrapAtTextWidth
					return v != nil && *v
				},
				func(e *view.Editor, v bool) {
					e.Options().SoftWrap.WrapAtTextWidth = &v
				},
			).WithDoc("Soft wrap at text-width, not the viewport"),
			{
				Key:       "inactive-dim",
				DocString: "Percent to darken unfocused panes; 0 disables",
				Get: func(e *view.Editor) (string, error) {
					return strconv.Itoa(e.Options().InactiveDim), nil
				},
				Set: func(e *view.Editor, s string) error {
					v, err := config.ParseNonNegInt(s)
					if err != nil {
						return err
					}
					e.Options().InactiveDim = v
					return nil
				},
			},
			{
				Key:       "rulers",
				DocString: "Columns at which to draw rulers",
				Get: func(e *view.Editor) (string, error) {
					return config.FormatIntSlice(e.Options().Rulers), nil
				},
				Set: func(e *view.Editor, s string) error {
					v, err := config.ParseIntSlice(s)
					if err != nil {
						return err
					}
					e.Options().SetRulers(v)
					return nil
				},
			},
			{
				Key:       "bufferline",
				DocString: "Show buffer tabs: always, never, or multiple",
				Get: func(e *view.Editor) (string, error) {
					return string(e.Options().BufferLine), nil
				},
				Set: func(e *view.Editor, s string) error {
					v, err := view.ParseBufferLine(s)
					if err != nil {
						return fmt.Errorf("%w: %s", config.ErrInvalidOption, s)
					}
					e.Options().BufferLine = v
					return nil
				},
				Complete: command.StaticCompleter(
					view.BufferLineNever,
					view.BufferLineAlways,
					view.BufferLineMultiple,
				),
			},
			{
				Key:       "whitespace.render",
				DocString: "Render whitespace: all, none, or per type",
				Get: func(e *view.Editor) (string, error) {
					rv := view.WhitespaceRenderNone
					if e.Options().Whitespace.Render.Default != nil {
						rv = *e.Options().Whitespace.Render.Default
					}
					return string(rv), nil
				},
				Set: func(e *view.Editor, s string) error {
					rv, err := view.ParseWhitespaceRenderValue(s)
					if err != nil {
						return fmt.Errorf("%w: %s", config.ErrInvalidOption, s)
					}
					ws := &e.Options().Whitespace
					ws.Render.Default = &rv
					ws.Render.Space = nil
					ws.Render.Nbsp = nil
					ws.Render.Tab = nil
					ws.Render.Newline = nil
					return nil
				},
				Complete: command.StaticCompleter(
					view.WhitespaceRenderNone,
					view.WhitespaceRenderAll,
				),
			},
			whitespaceRenderOption("whitespace.render.space",
				func(w *view.WhitespaceRender) view.WhitespaceRenderValue {
					return w.SpaceRender()
				},
				func(w *view.WhitespaceRender, v *view.WhitespaceRenderValue) {
					w.Space = v
				},
			).WithDoc("Render spaces"),
			whitespaceRenderOption("whitespace.render.nbsp",
				func(w *view.WhitespaceRender) view.WhitespaceRenderValue {
					return w.NbspRender()
				},
				func(w *view.WhitespaceRender, v *view.WhitespaceRenderValue) {
					w.Nbsp = v
				},
			).WithDoc("Render non-breaking spaces"),
			whitespaceRenderOption("whitespace.render.tab",
				func(w *view.WhitespaceRender) view.WhitespaceRenderValue {
					return w.TabRender()
				},
				func(w *view.WhitespaceRender, v *view.WhitespaceRenderValue) {
					w.Tab = v
				},
			).WithDoc("Render tabs"),
			whitespaceRenderOption("whitespace.render.newline",
				func(w *view.WhitespaceRender) view.WhitespaceRenderValue {
					return w.NewlineRender()
				},
				func(w *view.WhitespaceRender, v *view.WhitespaceRenderValue) {
					w.Newline = v
				},
			).WithDoc("Render newlines"),
			runeOption("whitespace.characters.space",
				func(o *view.Options) rune {
					return o.Whitespace.Characters.SpaceRune()
				},
				func(o *view.Options, s string) {
					o.Whitespace.Characters.Space = s
				},
			).WithDoc("Character rendered for a space"),
			runeOption("whitespace.characters.nbsp",
				func(o *view.Options) rune {
					return o.Whitespace.Characters.NbspRune()
				},
				func(o *view.Options, s string) {
					o.Whitespace.Characters.Nbsp = s
				},
			).WithDoc("Character rendered for a non-breaking space"),
			runeOption("whitespace.characters.tab",
				func(o *view.Options) rune {
					return o.Whitespace.Characters.TabRune()
				},
				func(o *view.Options, s string) {
					o.Whitespace.Characters.Tab = s
				},
			).WithDoc("Character rendered for a tab"),
			runeOption("whitespace.characters.tabpad",
				func(o *view.Options) rune {
					return o.Whitespace.Characters.TabpadRune()
				},
				func(o *view.Options, s string) {
					o.Whitespace.Characters.Tabpad = s
				},
			).WithDoc("Character padding a rendered tab"),
			runeOption("whitespace.characters.newline",
				func(o *view.Options) rune {
					return o.Whitespace.Characters.NewlineRune()
				},
				func(o *view.Options, s string) {
					o.Whitespace.Characters.Newline = s
				},
			).WithDoc("Character rendered for a newline"),
			kit.EditorBoolOption("indent-guides.render",
				func(e *view.Editor) bool {
					return e.Options().IndentGuides.Render
				},
				func(e *view.Editor, v bool) {
					e.Options().IndentGuides.Render = v
				},
			).WithDoc("Render indent guides"),
			{
				Key:       "indent-guides.skip-levels",
				DocString: "Indent levels to skip",
				Get: func(e *view.Editor) (string, error) {
					n := e.Options().IndentGuides.GetSkipLevels()
					return strconv.Itoa(n), nil
				},
				Set: func(e *view.Editor, s string) error {
					v, err := config.ParseNonNegInt(s)
					if err != nil {
						return err
					}
					e.Options().IndentGuides.SkipLevels = &v
					return nil
				},
			},
			runeOption("indent-guides.character",
				func(o *view.Options) rune {
					return o.IndentGuides.CharRune()
				},
				func(o *view.Options, s string) {
					o.IndentGuides.Character = s
				},
			).WithDoc("Character used to draw indent guides"),
			{
				Key:       "gutters.layout",
				DocString: "Gutters to display, in order",
				Get: func(e *view.Editor) (string, error) {
					layout := e.Options().Gutters.GutterLayout()
					values := make([]string, len(layout))
					for i, gutter := range layout {
						values[i] = string(gutter)
					}
					return config.FormatStringSlice(values), nil
				},
				Set: func(e *view.Editor, s string) error {
					values, err := config.ParseStringSlice(s)
					if err != nil {
						return err
					}
					layout := make([]view.GutterType, len(values))
					for i, value := range values {
						if err := layout[i].UnmarshalText(
							[]byte(value),
						); err != nil {
							return err
						}
					}
					e.Options().Gutters.Present = true
					e.Options().Gutters.Layout = layout
					return nil
				},
				Complete: sliceCompleter(
					view.GutterTypeDiagnostics,
					view.GutterTypeLineNumbers,
					view.GutterTypeSpacer,
					view.GutterTypeDiff,
				),
			},
			{
				Key:       "gutters.line-numbers.min-width",
				DocString: "Minimum line number gutter width",
				Get: func(e *view.Editor) (string, error) {
					n := e.Options().Gutters.LineNumberMinWidth()
					return strconv.Itoa(n), nil
				},
				Set: func(e *view.Editor, s string) error {
					v, err := config.ParsePositiveInt(s)
					if err != nil {
						return err
					}
					e.Options().Gutters.LineNumbers.MinWidth = &v
					return nil
				},
			},
		},
		Section: &command.Section{
			Config: cfg,
			Reset:  func() { *cfg = viewSection{} },
			Apply: func(e *view.Editor) {
				opts := e.Options()
				opts.LineNumber = cmp.Or(
					cfg.Editor.LineNumber, view.LineNumberAbsolute,
				)
				opts.InactiveDim = kit.IntOr(
					cfg.Editor.InactiveDim, view.DefaultInactiveDim,
				)
				opts.CursorLine = kit.BoolOr(cfg.Editor.CursorLine, true)
				opts.CursorColumn = kit.BoolOr(cfg.Editor.CursorColumn, false)
				model.SetAnimation(kit.BoolOr(cfg.Editor.Animation, true))
				model.SetAutoSize(kit.BoolOr(cfg.Editor.AutoSize, false))
				opts.TextWidth = cfg.Editor.TextWidth
				opts.SoftWrap = cfg.Editor.SoftWrap
				opts.SetRulers(cfg.Editor.Rulers)
				opts.BufferLine = cmp.Or(
					cfg.Editor.BufferLine, view.BufferLineNever,
				)
				opts.Whitespace = cfg.Editor.Whitespace
				opts.IndentGuides = cfg.Editor.IndentGuides
				opts.Gutters = cfg.Editor.Gutters
			},
		},
		Labels: []command.PrefixLabel{
			kit.Label(
				"View", kit.Char('z'), command.DocNormalModes,
			),
			kit.Label(
				"View", kit.Char('Z'), command.DocNormalModes,
			),
			kit.Label(
				"Window", kit.Ctrl('w'), command.PaneModes,
			),
			kit.Label(
				"Window", kit.LeaderPrefix(kit.Char('w')), command.PaneModes,
			),
		},
	}
}

func whitespaceRenderOption(
	key string, get wsRenderGetter, set wsRenderSetter,
) command.Option {
	return command.Option{
		Key: key,
		Get: func(e *view.Editor) (string, error) {
			return string(get(&e.Options().Whitespace.Render)), nil
		},
		Set: func(e *view.Editor, s string) error {
			v, err := view.ParseWhitespaceRenderValue(s)
			if err != nil {
				return fmt.Errorf("%w: %s", config.ErrInvalidOption, s)
			}
			set(&e.Options().Whitespace.Render, &v)
			return nil
		},
		Complete: command.StaticCompleter(
			view.WhitespaceRenderNone,
			view.WhitespaceRenderAll,
		),
	}
}

func runeOption(
	key string, get optionGetter[rune], set optionSetter[string],
) command.Option {
	return command.Option{
		Key: key,
		Get: func(e *view.Editor) (string, error) {
			return strconv.Quote(string(get(e.Options()))), nil
		},
		Set: func(e *view.Editor, s string) error {
			v, err := config.ParseStringLiteral(s)
			if err != nil {
				return err
			}
			if utf8.RuneCountInString(v) != 1 {
				return fmt.Errorf("%w: %s", config.ErrInvalidOption, v)
			}
			set(e.Options(), v)
			return nil
		},
	}
}

func runImgAction(fn func(*ui.ImagePane)) command.Action {
	return func(e *view.Editor) {
		if p, ok := e.FocusedPane().(*ui.ImagePane); ok {
			fn(p)
		}
	}
}

func runDirAction(
	fn func(*view.Editor, view.Direction), dir view.Direction,
) command.Action {
	return func(e *view.Editor) { fn(e, dir) }
}

func splitRun(layout view.Layout) command.Run {
	return func(e *view.Editor, args *command.Args) command.Result {
		if args == nil || args.Empty() {
			if err := e.SplitFocused(layout); err != nil {
				return command.Result{Error: err}
			}
			return command.Result{}
		}
		for _, path := range args.Positionals() {
			if err := e.SplitFocused(layout); err != nil {
				return command.Result{Error: err}
			}
			if _, err := e.OpenFile(path); err != nil {
				return command.Result{Error: err}
			}
		}
		return command.Result{}
	}
}
