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
	viewcfg "github.com/kode4food/toe/internal/view/config"
	"github.com/kode4food/toe/internal/view/language"
)

type viewSection struct {
	Editor struct {
		LineNumber   view.LineNumber   `toml:"line-number"`
		CursorLine   *bool             `toml:"cursorline"`
		CursorColumn *bool             `toml:"cursorcolumn"`
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
				Modes:     []string{"IMG"},
				Keys:      kit.Keys(kit.Char('+'), kit.Char('=')),
				Aliases:   []string{"zoom-in"},
				Signature: kit.Sig(),
			},
			{
				Name:      actImageZoomOut,
				DocString: "Zoom image out",
				Run:       kit.Runner(imageZoomOut),
				Modes:     []string{"IMG"},
				Keys:      kit.Keys(kit.Char('-')),
				Aliases:   []string{"zoom-out"},
				Signature: kit.Sig(),
			},
			{
				Name:      actImageZoomReset,
				DocString: "Fit image to pane",
				Run:       kit.Runner(imageZoomReset),
				Modes:     []string{"IMG"},
				Keys:      kit.Keys(kit.Char('0')),
				Aliases:   []string{"zoom-reset"},
				Signature: kit.Sig(),
			},
			{
				Name:      actPageUp,
				DocString: "Move page up",
				Run:       kit.Runner(action.PageUp),
				Modes:     []string{"NOR", "SEL", "INS"},
				Keys: map[string][]command.KeyBinding{"*": {
					kit.Or(kit.Ctrl('b'), kit.PgUp),
					kit.Or(z(kit.Ctrl('b')), z(kit.PgUp)),
					kit.Or(Z(kit.Ctrl('b')), Z(kit.PgUp)),
				}},
			},
			{
				Name:      actPageDown,
				DocString: "Move page down",
				Run:       kit.Runner(action.PageDown),
				Modes:     []string{"NOR", "SEL", "INS"},
				Keys: map[string][]command.KeyBinding{"*": {
					kit.Or(kit.Ctrl('f'), kit.PgDn),
					kit.Or(z(kit.Ctrl('f')), z(kit.PgDn)),
					kit.Or(Z(kit.Ctrl('f')), Z(kit.PgDn)),
				}},
			},
			{
				Name:      actPageCursorHalfUp,
				DocString: "Move page and cursor half up",
				Run:       kit.Runner(action.PageCursorHalfUp),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					kit.Or(kit.Ctrl('u')),
					kit.Or(z(kit.Ctrl('u')), z(kit.Bksp)),
					kit.Or(Z(kit.Ctrl('u')), Z(kit.Bksp)),
				}},
			},
			{
				Name:      actPageCursorHalfDown,
				DocString: "Move page and cursor half down",
				Run:       kit.Runner(action.PageCursorHalfDown),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					kit.Or(kit.Ctrl('d')),
					kit.Or(z(kit.Ctrl('d')), z(kit.Char(' '))),
					kit.Or(Z(kit.Ctrl('d')), Z(kit.Char(' '))),
				}},
			},
			{
				Name:      actHalfPageUp,
				DocString: "Move half page up",
				Run:       kit.Runner(action.HalfPageUp),
				Modes:     command.DocumentModes(),
				Signature: kit.Sig(),
			},
			{
				Name:      actHalfPageDown,
				DocString: "Move half page down",
				Run:       kit.Runner(action.HalfPageDown),
				Modes:     command.DocumentModes(),
				Signature: kit.Sig(),
			},
			{
				Name:      actPageCursorUp,
				DocString: "Move page and cursor up",
				Run:       kit.Runner(action.PageUp),
				Modes:     command.DocumentModes(),
			},
			{
				Name:      actPageCursorDown,
				DocString: "Move page and cursor down",
				Run:       kit.Runner(action.PageDown),
				Modes:     command.DocumentModes(),
			},
			{
				Name:      actCenterCursorLine,
				DocString: "Align view center",
				Run:       kit.Runner(action.AlignViewCenter),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					kit.Or(z(kit.Char('z')), z(kit.Char('c'))),
					kit.Or(Z(kit.Char('z')), Z(kit.Char('c'))),
				}},
			},
			{
				Name:      actCenterCursorLineTop,
				DocString: "Align view top",
				Run:       kit.Runner(action.AlignViewTop),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					kit.Or(z(kit.Char('.')), z(kit.Char('t'))),
					kit.Or(Z(kit.Char('.')), Z(kit.Char('t'))),
				}},
			},
			{
				Name:      actCenterCursorLineBottom,
				DocString: "Align view bottom",
				Run:       kit.Runner(action.AlignViewBottom),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					kit.Or(z(kit.Char('b'))),
					kit.Or(Z(kit.Char('b'))),
				}},
			},
			{
				Name:      actScrollUp,
				DocString: "Scroll view up",
				Run:       kit.Runner(action.ScrollUp),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					kit.Or(z(kit.Char('k')), z(kit.Up)),
					kit.Or(Z(kit.Char('k')), Z(kit.Up)),
				}},
			},
			{
				Name:      actScrollDown,
				DocString: "Scroll view down",
				Run:       kit.Runner(action.ScrollDown),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					kit.Or(z(kit.Char('j')), z(kit.Down)),
					kit.Or(Z(kit.Char('j')), Z(kit.Down)),
				}},
			},
			{
				Name:      actTerminal,
				DocString: "Open a new terminal",
				Run:       kit.Continuation(model.TerminalAction()),
				Modes:     []string{"NOR", "SEL", "IMG"},
				Keys: map[string][]command.KeyBinding{"*": {
					kit.Or(Cw(kit.Char('x'))),
					kit.Or(Spcw(kit.Char('x'))),
				}},
				Signature: kit.Sig(),
			},
			{
				Name:      actTerminalSearch,
				DocString: "Search focused terminal's scrollback",
				Run:       kit.Continuation(model.TerminalSearchAction()),
				Modes:     []string{"TRM"},
				Keys:      kit.Window(kit.Char('/')),
				Signature: kit.Sig(),
			},
			{
				Name:      actVSplitView,
				DocString: "Vertical right split",
				Run:       kit.Runner(action.VSplit),
				Modes:     []string{"NOR", "SEL", "TRM", "IMG"},
				Keys:      kit.Window(kit.Char('v'), kit.Ctrl('v')),
				Aliases:   []string{"vs"},
				Signature: kit.Sig(),
			},
			{
				Name:      actHSplitView,
				DocString: "Horizontal bottom split",
				Run:       kit.Runner(action.HSplit),
				Modes:     []string{"NOR", "SEL", "TRM", "IMG"},
				Keys:      kit.Window(kit.Char('s'), kit.Ctrl('s')),
				Aliases:   []string{"hs", "sp"},
				Signature: kit.Sig(),
			},
			{
				Name:      actVSplitNew,
				DocString: "Vertical right split scratch buffer",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					e.VSplitNew()
					return command.Result{}
				},
				Modes:     []string{"NOR", "SEL", "TRM", "IMG"},
				Aliases:   []string{"vnew"},
				Signature: kit.Sig(),
			},
			{
				Name:      actHSplitNew,
				DocString: "Horizontal bottom split scratch buffer",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					e.HSplitNew()
					return command.Result{}
				},
				Modes:     []string{"NOR", "SEL", "TRM", "IMG"},
				Aliases:   []string{"hnew"},
				Signature: kit.Sig(),
			},
			{
				Name:      actTransposeView,
				DocString: "Transpose splits",
				Run:       kit.Runner(action.TransposeView),
				Modes:     []string{"NOR", "SEL", "TRM", "IMG"},
				Keys:      kit.Window(kit.Char('t'), kit.Ctrl('t')),
			},
			{
				Name:      actCloseCurrentView,
				DocString: "Close window",
				Run:       kit.Runner(action.CloseCurrentView),
				Modes:     []string{"NOR", "SEL", "TRM", "IMG"},
				Keys:      kit.Window(kit.Char('q'), kit.Ctrl('q')),
				Aliases:   []string{"wc"},
				Signature: kit.Sig(),
			},
			{
				Name:      actCloseCurrentViewForce,
				DocString: "Force close window",
				Run:       kit.Runner(action.CloseCurrentViewForce),
				Modes:     []string{"NOR", "SEL", "TRM", "IMG"},
				Aliases:   []string{"wc!"},
				Signature: kit.Sig(),
			},
			{
				Name:      actCloseOtherViews,
				DocString: "Close windows except current",
				Run:       kit.Runner(action.CloseOtherViews),
				Modes:     []string{"NOR", "SEL", "TRM", "IMG"},
				Keys:      kit.Window(kit.Char('o'), kit.Ctrl('o')),
				Aliases:   []string{"wo"},
				Signature: kit.Sig(),
			},
			{
				Name:      actRotateView,
				DocString: "Goto next window",
				Run:       kit.Runner(action.RotateView),
				Modes:     []string{"NOR", "SEL", "TRM", "IMG"},
				Keys:      kit.Window(kit.Char('w'), kit.Ctrl('w')),
			},
			{
				Name:      actJumpViewLeft,
				DocString: "Jump to left split",
				Run:       kit.Runner(action.JumpViewLeft),
				Modes:     []string{"NOR", "SEL", "TRM", "IMG"},
				Keys: kit.Window(
					kit.Char('h'), kit.Ctrl('h'), kit.Left,
				),
			},
			{
				Name:      actJumpViewDown,
				DocString: "Jump to split below",
				Run:       kit.Runner(action.JumpViewDown),
				Modes:     []string{"NOR", "SEL", "TRM", "IMG"},
				Keys: kit.Window(
					kit.Char('j'), kit.Ctrl('j'), kit.Down,
				),
			},
			{
				Name:      actJumpViewUp,
				DocString: "Jump to split above",
				Run:       kit.Runner(action.JumpViewUp),
				Modes:     []string{"NOR", "SEL", "TRM", "IMG"},
				Keys: kit.Window(
					kit.Char('k'), kit.Ctrl('k'), kit.Up,
				),
			},
			{
				Name:      actJumpViewRight,
				DocString: "Jump to right split",
				Run:       kit.Runner(action.JumpViewRight),
				Modes:     []string{"NOR", "SEL", "TRM", "IMG"},
				Keys: kit.Window(
					kit.Char('l'), kit.Ctrl('l'), kit.Right,
				),
			},
			{
				Name:      actSwapViewLeft,
				DocString: "Swap with left split",
				Run:       kit.Runner(action.SwapViewLeft),
				Modes:     []string{"NOR", "SEL", "TRM", "IMG"},
				Keys:      kit.Window(kit.Char('H')),
			},
			{
				Name:      actSwapViewDown,
				DocString: "Swap with split below",
				Run:       kit.Runner(action.SwapViewDown),
				Modes:     []string{"NOR", "SEL", "TRM", "IMG"},
				Keys:      kit.Window(kit.Char('J')),
			},
			{
				Name:      actSwapViewUp,
				DocString: "Swap with split above",
				Run:       kit.Runner(action.SwapViewUp),
				Modes:     []string{"NOR", "SEL", "TRM", "IMG"},
				Keys:      kit.Window(kit.Char('K')),
			},
			{
				Name:      actSwapViewRight,
				DocString: "Swap with right split",
				Run:       kit.Runner(action.SwapViewRight),
				Modes:     []string{"NOR", "SEL", "TRM", "IMG"},
				Keys:      kit.Window(kit.Char('L')),
			},
			{
				Name:      actResizeView,
				DocString: "Resize split",
				Run:       kit.Continuation(model.ResizeViewAction),
				Modes:     []string{"NOR", "SEL", "TRM", "IMG"},
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
						return fmt.Errorf("%w: %s", viewcfg.ErrInvalidOption, s)
					}
					e.Options().LineNumber = v
					return nil
				},
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
			kit.EditorNullableIntOption("soft-wrap.max-wrap",
				language.DefaultMaxWrap,
				func(e *view.Editor) *int {
					return e.Options().SoftWrap.MaxWrap
				},
				func(e *view.Editor, v *int) {
					e.Options().SoftWrap.MaxWrap = v
				},
			),
			kit.EditorNullableIntOption("soft-wrap.max-indent-retain",
				language.DefaultMaxIndentRetain,
				func(e *view.Editor) *int {
					return e.Options().SoftWrap.MaxIndentRetain
				},
				func(e *view.Editor, v *int) {
					e.Options().SoftWrap.MaxIndentRetain = v
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
					v, err := viewcfg.ParseStringLiteral(s)
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
				Key: "rulers",
				Get: func(e *view.Editor) (string, error) {
					return viewcfg.FormatIntSlice(e.Options().Rulers), nil
				},
				Set: func(e *view.Editor, s string) error {
					v, err := viewcfg.ParseIntSlice(s)
					if err != nil {
						return err
					}
					e.Options().Rulers = v
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
						return fmt.Errorf("%w: %s", viewcfg.ErrInvalidOption, s)
					}
					e.Options().BufferLine = v
					return nil
				},
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
						return fmt.Errorf("%w: %s", viewcfg.ErrInvalidOption, s)
					}
					ws := &e.Options().Whitespace
					ws.Render.Default = &rv
					ws.Render.Space = nil
					ws.Render.Nbsp = nil
					ws.Render.Nnbsp = nil
					ws.Render.Tab = nil
					ws.Render.Newline = nil
					return nil
				},
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
			whitespaceRenderOption("whitespace.render.nnbsp",
				func(w *view.WhitespaceRender) view.WhitespaceRenderValue {
					return w.NnbspRender()
				},
				func(w *view.WhitespaceRender, v *view.WhitespaceRenderValue) {
					w.Nnbsp = v
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
			runeOption("whitespace.characters.nnbsp",
				func(o *view.Options) rune {
					return o.Whitespace.Characters.NnbspRune()
				},
				func(o *view.Options, s string) {
					o.Whitespace.Characters.Nnbsp = s
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
					v, err := viewcfg.ParseNonNegInt(s)
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
					return viewcfg.FormatStringSlice(values), nil
				},
				Set: func(e *view.Editor, s string) error {
					values, err := viewcfg.ParseStringSlice(s)
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
			},
			{
				Key: "gutters.line-numbers.min-width",
				Get: func(e *view.Editor) (string, error) {
					n := e.Options().Gutters.LineNumberMinWidth()
					return strconv.Itoa(n), nil
				},
				Set: func(e *view.Editor, s string) error {
					v, err := viewcfg.ParsePositiveInt(s)
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
				opts.CursorLine = kit.BoolOr(cfg.Editor.CursorLine, true)
				opts.CursorColumn = kit.BoolOr(cfg.Editor.CursorColumn, false)
				opts.TextWidth = cfg.Editor.TextWidth
				opts.SoftWrap = cfg.Editor.SoftWrap
				opts.Rulers = cfg.Editor.Rulers
				opts.BufferLine = cmp.Or(
					cfg.Editor.BufferLine, view.BufferLineNever,
				)
				opts.Whitespace = cfg.Editor.Whitespace
				opts.IndentGuides = cfg.Editor.IndentGuides
				opts.Gutters = cfg.Editor.Gutters
			},
		},
		Labels: []command.PrefixLabel{
			kit.Label("View", kit.Char('z'), "NOR", "SEL"),
			kit.Label("View", kit.Char('Z'), "NOR", "SEL"),
			kit.Label("Window", kit.Ctrl('w'), "NOR", "SEL", "TRM", "IMG"),
			kit.Label(
				"Window", kit.LeaderPrefix(kit.Char('w')),
				"NOR", "SEL", "TRM", "IMG",
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
				return fmt.Errorf("%w: %s", viewcfg.ErrInvalidOption, s)
			}
			set(&e.Options().Whitespace.Render, &v)
			return nil
		},
		Complete: command.StaticCompleter(
			string(view.WhitespaceRenderNone),
			string(view.WhitespaceRenderAll),
		),
	}
}

func runeOption(
	key string, get optionGetter[rune], set optionSetter[string],
) command.Option {
	return command.Option{
		Key: key,
		Get: func(e *view.Editor) (string, error) {
			return string(get(e.Options())), nil
		},
		Set: func(e *view.Editor, s string) error {
			v, err := viewcfg.ParseStringLiteral(s)
			if err != nil {
				return err
			}
			if utf8.RuneCountInString(v) != 1 {
				return fmt.Errorf("%w: %s", viewcfg.ErrInvalidOption, v)
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
