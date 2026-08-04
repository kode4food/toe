package config

import (
	"cmp"
	"fmt"
	"strconv"
	"unicode/utf8"

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
		LineNumber   view.LineNumber `toml:"line-number"`
		InactiveDim  *int            `toml:"inactive-dim"`
		CursorLine   *bool           `toml:"cursorline"`
		CursorColumn *bool           `toml:"cursorcolumn"`
		Animation    *bool           `toml:"animation"`
		AutoSize     struct {
			Enable      *bool `toml:"enable"`
			VerticalPct *int  `toml:"vertical-percent"`
		} `toml:"auto-size"`
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
	actHSplitView             = "split"
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
)

// ViewModule returns the split, scroll, and view-option commands
func ViewModule(model ui.Model) command.Module {
	cfg := new(viewSection)
	z := kit.Prefixed(kit.Char('z'))
	Z := kit.Prefixed(kit.Char('Z'))
	Spcw := kit.Prefixed(kit.LeaderPrefix(kit.Char('w')))
	Cw := kit.Prefixed(kit.Ctrl('w'))

	return command.Module{
		Commands: []command.Command{
			{
				Name:      actImageZoomIn,
				DocString: "Zoom image in",
				Run:       kit.Runner(imageZoomIn),
				Modes:     view.ModeImage,
				Keys:      kit.Keys(kit.Char('+'), kit.Char('=')),
				Aliases:   []string{"zoom-in"},
				Signature: command.DefaultSignature(),
			},
			{
				Name:      actImageZoomOut,
				DocString: "Zoom image out",
				Run:       kit.Runner(imageZoomOut),
				Modes:     view.ModeImage,
				Keys:      kit.Keys(kit.Char('-')),
				Aliases:   []string{"zoom-out"},
				Signature: command.DefaultSignature(),
			},
			{
				Name:      actImageZoomReset,
				DocString: "Fit image to pane",
				Run:       kit.Runner(imageZoomReset),
				Modes:     view.ModeImage,
				Keys:      kit.Keys(kit.Char('0')),
				Aliases:   []string{"zoom-reset"},
				Signature: command.DefaultSignature(),
			},
			{
				Name:      actImagePanLeft,
				DocString: "Pan image left",
				Run:       kit.Runner(imagePanLeft),
				Modes:     view.ModeImage,
				Keys:      kit.Keys(kit.Char('h'), kit.Left),
				Aliases:   []string{"pan-left"},
				Signature: command.DefaultSignature(),
			},
			{
				Name:      actImagePanDown,
				DocString: "Pan image down",
				Run:       kit.Runner(imagePanDown),
				Modes:     view.ModeImage,
				Keys:      kit.Keys(kit.Char('j'), kit.Down),
				Aliases:   []string{"pan-down"},
				Signature: command.DefaultSignature(),
			},
			{
				Name:      actImagePanUp,
				DocString: "Pan image up",
				Run:       kit.Runner(imagePanUp),
				Modes:     view.ModeImage,
				Keys:      kit.Keys(kit.Char('k'), kit.Up),
				Aliases:   []string{"pan-up"},
				Signature: command.DefaultSignature(),
			},
			{
				Name:      actImagePanRight,
				DocString: "Pan image right",
				Run:       kit.Runner(imagePanRight),
				Modes:     view.ModeImage,
				Keys:      kit.Keys(kit.Char('l'), kit.Right),
				Aliases:   []string{"pan-right"},
				Signature: command.DefaultSignature(),
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
				Signature: command.DefaultSignature(),
			},
			{
				Name:      actHalfPageDown,
				DocString: "Move half page down",
				Run:       kit.Runner(action.HalfPageDown),
				Modes:     command.DocModes,
				Signature: command.DefaultSignature(),
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
				Signature: command.DefaultSignature(),
			},
			{
				Name:      actTerminalSearch,
				DocString: "Search focused terminal's scrollback",
				Run:       kit.Runner(model.TerminalSearchAction),
				Modes:     view.ModeTerminal,
				Keys:      kit.Window(kit.Char('/')),
				Signature: command.DefaultSignature(),
			},
			{
				Name:      actVSplitView,
				DocString: "Vertical right split",
				Run:       kit.Runner(action.VSplit),
				Modes:     command.PaneModes,
				Keys:      kit.Window(kit.Char('v'), kit.Ctrl('v')),
				Aliases:   []string{"vs"},
				Signature: command.DefaultSignature(),
			},
			{
				Name:      actHSplitView,
				DocString: "Horizontal bottom split",
				Run:       kit.Runner(action.HSplit),
				Modes:     command.PaneModes,
				Keys:      kit.Window(kit.Char('s'), kit.Ctrl('s')),
				Aliases:   []string{"hs", "sp"},
				Signature: command.DefaultSignature(),
			},
			{
				Name:      actVSplitNew,
				DocString: "Vertical right split scratch buffer",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					e.VSplitNew()
					return command.Result{}
				},
				Modes:     command.PaneModes,
				Aliases:   []string{"vnew"},
				Signature: command.DefaultSignature(),
			},
			{
				Name:      actHSplitNew,
				DocString: "Horizontal bottom split scratch buffer",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					e.HSplitNew()
					return command.Result{}
				},
				Modes:     command.PaneModes,
				Aliases:   []string{"hnew"},
				Signature: command.DefaultSignature(),
			},
			{
				Name:      actTransposeView,
				DocString: "Transpose splits",
				Run:       kit.Runner(action.TransposeView),
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
				Signature: command.DefaultSignature(),
			},
			{
				Name:      actCloseCurrentViewForce,
				DocString: "Force close window",
				Run:       kit.Runner(action.CloseCurrentViewForce),
				Modes:     command.PaneModes,
				Aliases:   []string{"wc!"},
				Signature: command.DefaultSignature(),
			},
			{
				Name:      actCloseOtherViews,
				DocString: "Close windows except current",
				Run:       kit.Runner(action.CloseOtherViews),
				Modes:     command.PaneModes,
				Keys:      kit.Window(kit.Char('o'), kit.Ctrl('o')),
				Aliases:   []string{"wo"},
				Signature: command.DefaultSignature(),
			},
			{
				Name:      actRotateView,
				DocString: "Goto next window",
				Run:       kit.Runner(action.RotateView),
				Modes:     command.PaneModes,
				Keys:      kit.Window(kit.Char('w'), kit.Ctrl('w')),
			},
			{
				Name:      actTogglePaneMaximized,
				DocString: "Toggle focused pane maximized",
				Run:       kit.Runner(action.TogglePaneMaximized),
				Modes:     command.PaneModes,
				Keys:      kit.Window(kit.Char('z')),
				Signature: command.DefaultSignature(),
			},
			{
				Name:      actJumpViewLeft,
				DocString: "Jump to left split",
				Run:       kit.Runner(action.JumpViewLeft),
				Modes:     command.PaneModes,
				Keys: kit.Window(
					kit.Char('h'), kit.Ctrl('h'), kit.Left,
				),
			},
			{
				Name:      actJumpViewDown,
				DocString: "Jump to split below",
				Run:       kit.Runner(action.JumpViewDown),
				Modes:     command.PaneModes,
				Keys: kit.Window(
					kit.Char('j'), kit.Ctrl('j'), kit.Down,
				),
			},
			{
				Name:      actJumpViewUp,
				DocString: "Jump to split above",
				Run:       kit.Runner(action.JumpViewUp),
				Modes:     command.PaneModes,
				Keys: kit.Window(
					kit.Char('k'), kit.Ctrl('k'), kit.Up,
				),
			},
			{
				Name:      actJumpViewRight,
				DocString: "Jump to right split",
				Run:       kit.Runner(action.JumpViewRight),
				Modes:     command.PaneModes,
				Keys: kit.Window(
					kit.Char('l'), kit.Ctrl('l'), kit.Right,
				),
			},
			{
				Name:      actSwapViewLeft,
				DocString: "Swap with left split",
				Run:       kit.Runner(action.SwapViewLeft),
				Modes:     command.PaneModes,
				Keys:      kit.Window(kit.Char('H')),
			},
			{
				Name:      actSwapViewDown,
				DocString: "Swap with split below",
				Run:       kit.Runner(action.SwapViewDown),
				Modes:     command.PaneModes,
				Keys:      kit.Window(kit.Char('J')),
			},
			{
				Name:      actSwapViewUp,
				DocString: "Swap with split above",
				Run:       kit.Runner(action.SwapViewUp),
				Modes:     command.PaneModes,
				Keys:      kit.Window(kit.Char('K')),
			},
			{
				Name:      actSwapViewRight,
				DocString: "Swap with right split",
				Run:       kit.Runner(action.SwapViewRight),
				Modes:     command.PaneModes,
				Keys:      kit.Window(kit.Char('L')),
			},
			{
				Name:      actResizeView,
				DocString: "Resize split",
				Run:       kit.Continuation(model.ResizeViewAction),
				Modes:     command.PaneModes,
				Keys:      kit.Window(kit.Char('r')),
			},
		},
		Options: []command.Option{
			{
				Key: "line-number",
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
			),
			kit.EditorBoolOption("cursorcolumn",
				func(e *view.Editor) bool {
					return e.Options().CursorColumn
				},
				func(e *view.Editor, v bool) {
					e.Options().CursorColumn = v
				},
			),
			kit.EditorBoolOption("animation",
				func(*view.Editor) bool {
					return model.Animation()
				},
				func(_ *view.Editor, v bool) {
					model.SetAnimation(v)
				},
			),
			kit.EditorBoolOption("auto-size.enable",
				func(*view.Editor) bool {
					return model.AutoSize()
				},
				func(_ *view.Editor, v bool) {
					model.SetAutoSize(v)
				},
			),
			{
				Key: "auto-size.vertical-percent",
				Get: func(*view.Editor) (string, error) {
					return strconv.Itoa(model.AutoSizeVerticalPercent()), nil
				},
				Set: func(_ *view.Editor, s string) error {
					percent, err := config.ParsePercent(s)
					if err != nil {
						return err
					}
					model.SetAutoSizeVerticalPercent(percent)
					return nil
				},
			},
			kit.EditorNullableIntOption("text-width",
				language.DefaultTextWidth,
				func(e *view.Editor) *int {
					return e.Options().TextWidth
				},
				func(e *view.Editor, v *int) {
					e.Options().TextWidth = v
				},
			),
			kit.EditorBoolOption("soft-wrap.enable",
				func(e *view.Editor) bool {
					sw := e.Options().SoftWrap.Enable
					return sw != nil && *sw
				},
				func(e *view.Editor, v bool) {
					e.Options().SoftWrap.Enable = &v
				},
			),
			{
				Key: "soft-wrap.wrap-indicator",
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
			),
			{
				Key: "inactive-dim",
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
				Key: "rulers",
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
				Key: "bufferline",
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
				Key: "whitespace.render",
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
			),
			whitespaceRenderOption("whitespace.render.nbsp",
				func(w *view.WhitespaceRender) view.WhitespaceRenderValue {
					return w.NbspRender()
				},
				func(w *view.WhitespaceRender, v *view.WhitespaceRenderValue) {
					w.Nbsp = v
				},
			),
			whitespaceRenderOption("whitespace.render.tab",
				func(w *view.WhitespaceRender) view.WhitespaceRenderValue {
					return w.TabRender()
				},
				func(w *view.WhitespaceRender, v *view.WhitespaceRenderValue) {
					w.Tab = v
				},
			),
			whitespaceRenderOption("whitespace.render.newline",
				func(w *view.WhitespaceRender) view.WhitespaceRenderValue {
					return w.NewlineRender()
				},
				func(w *view.WhitespaceRender, v *view.WhitespaceRenderValue) {
					w.Newline = v
				},
			),
			runeOption("whitespace.characters.space",
				func(o *view.Options) rune {
					return o.Whitespace.Characters.SpaceRune()
				},
				func(o *view.Options, s string) {
					o.Whitespace.Characters.Space = s
				},
			),
			runeOption("whitespace.characters.nbsp",
				func(o *view.Options) rune {
					return o.Whitespace.Characters.NbspRune()
				},
				func(o *view.Options, s string) {
					o.Whitespace.Characters.Nbsp = s
				},
			),
			runeOption("whitespace.characters.tab",
				func(o *view.Options) rune {
					return o.Whitespace.Characters.TabRune()
				},
				func(o *view.Options, s string) {
					o.Whitespace.Characters.Tab = s
				},
			),
			runeOption("whitespace.characters.tabpad",
				func(o *view.Options) rune {
					return o.Whitespace.Characters.TabpadRune()
				},
				func(o *view.Options, s string) {
					o.Whitespace.Characters.Tabpad = s
				},
			),
			runeOption("whitespace.characters.newline",
				func(o *view.Options) rune {
					return o.Whitespace.Characters.NewlineRune()
				},
				func(o *view.Options, s string) {
					o.Whitespace.Characters.Newline = s
				},
			),
			kit.EditorBoolOption("indent-guides.render",
				func(e *view.Editor) bool {
					return e.Options().IndentGuides.Render
				},
				func(e *view.Editor, v bool) {
					e.Options().IndentGuides.Render = v
				},
			),
			{
				Key: "indent-guides.skip-levels",
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
			),
			{
				Key: "gutters.layout",
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
				Key: "gutters.line-numbers.min-width",
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
				model.SetAutoSize(kit.BoolOr(cfg.Editor.AutoSize.Enable, false))
				model.SetAutoSizeVerticalPercent(kit.IntOr(
					cfg.Editor.AutoSize.VerticalPct,
					ui.DefaultAutoSizeVerticalPct,
				))
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

func imageZoomIn(e *view.Editor) {
	if p, ok := e.FocusedPane().(*ui.ImagePane); ok {
		p.ZoomIn()
	}
}

func imageZoomOut(e *view.Editor) {
	if p, ok := e.FocusedPane().(*ui.ImagePane); ok {
		p.ZoomOut()
	}
}

func imageZoomReset(e *view.Editor) {
	if p, ok := e.FocusedPane().(*ui.ImagePane); ok {
		p.ResetZoom()
	}
}

func imagePanLeft(e *view.Editor) {
	if p, ok := e.FocusedPane().(*ui.ImagePane); ok {
		p.PanLeft()
	}
}

func imagePanDown(e *view.Editor) {
	if p, ok := e.FocusedPane().(*ui.ImagePane); ok {
		p.PanDown()
	}
}

func imagePanUp(e *view.Editor) {
	if p, ok := e.FocusedPane().(*ui.ImagePane); ok {
		p.PanUp()
	}
}

func imagePanRight(e *view.Editor) {
	if p, ok := e.FocusedPane().(*ui.ImagePane); ok {
		p.PanRight()
	}
}
