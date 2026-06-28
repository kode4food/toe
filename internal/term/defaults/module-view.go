package defaults

import (
	"cmp"
	"fmt"
	"strconv"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
	"github.com/kode4food/toe/internal/view/config"
	"github.com/kode4food/toe/internal/view/language"
)

type viewSection struct {
	Editor struct {
		LineNumber   view.LineNumber   `toml:"line-number"`
		Cursorline   *bool             `toml:"cursorline"`
		Cursorcolumn *bool             `toml:"cursorcolumn"`
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
	actVSplitView             = "vsplit"
	actHSplitView             = "split"
	actVSplitNew              = "vsplit_new"
	actHSplitNew              = "hsplit_new"
	actTransposeView          = "transpose_view"
	actCloseCurrentView       = "wclose"
	actCloseCurrentViewForce  = "wclose!"
	actCloseOtherViews        = "wonly"
	actJumpViewLeft           = "jump_view_left"
	actJumpViewDown           = "jump_view_down"
	actJumpViewUp             = "jump_view_up"
	actJumpViewRight          = "jump_view_right"
	actSwapViewLeft           = "swap_view_left"
	actSwapViewDown           = "swap_view_down"
	actSwapViewUp             = "swap_view_up"
	actSwapViewRight          = "swap_view_right"
	actRotateView             = "rotate_view"
)

func viewModule() command.Module {
	cfg := new(viewSection)
	z := prefixed(char('z'))
	Z := prefixed(char('Z'))
	Spc := prefixed(char(' '))
	Spcw := prefixed(Spc(char('w')))
	Spcwn := prefixed(Spcw(char('n')))
	Cw := prefixed(ctrl('w'))
	Cwn := prefixed(Cw(char('n')))

	return command.Module{
		Commands: map[string]command.Command{
			actPageUp: {
				DocString: "Move page up",
				Run:       Runner(action.PageUp),
				Modes:     []string{"NOR", "SEL", "INS"},
				Keys: map[string][]command.KeyBinding{"*": {
					{ctrl('b'), special("pageup")},
					{z(ctrl('b')), z(special("pageup"))},
					{Z(ctrl('b')), Z(special("pageup"))},
				}},
			},
			actPageDown: {
				DocString: "Move page down",
				Run:       Runner(action.PageDown),
				Modes:     []string{"NOR", "SEL", "INS"},
				Keys: map[string][]command.KeyBinding{"*": {
					{ctrl('f'), special("pagedown")},
					{z(ctrl('f')), z(special("pagedown"))},
					{Z(ctrl('f')), Z(special("pagedown"))},
				}},
			},
			actPageCursorHalfUp: {
				DocString: "Move page and cursor half up",
				Run:       Runner(action.PageCursorHalfUp),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					{ctrl('u')},
					{z(ctrl('u')), z(special("backspace"))},
					{Z(ctrl('u')), Z(special("backspace"))},
				}},
			},
			actPageCursorHalfDown: {
				DocString: "Move page and cursor half down",
				Run:       Runner(action.PageCursorHalfDown),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					{ctrl('d')},
					{z(ctrl('d')), z(char(' '))},
					{Z(ctrl('d')), Z(char(' '))},
				}},
			},
			actHalfPageUp: {
				DocString: "Move half page up",
				Run:       Runner(action.HalfPageUp),
				Signature: sig(),
			},
			actHalfPageDown: {
				DocString: "Move half page down",
				Run:       Runner(action.HalfPageDown),
				Signature: sig(),
			},
			actPageCursorUp: {
				DocString: "Move page and cursor up",
				Run:       Runner(action.PageUp),
			},
			actPageCursorDown: {
				DocString: "Move page and cursor down",
				Run:       Runner(action.PageDown),
			},
			actCenterCursorLine: {
				DocString: "Align view center",
				Run:       Runner(action.AlignViewCenter),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					{z(char('z')), z(char('c'))},
					{Z(char('z')), Z(char('c'))},
				}},
			},
			actCenterCursorLineTop: {
				DocString: "Align view top",
				Run:       Runner(action.AlignViewTop),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					{z(char('.')), z(char('t'))},
					{Z(char('.')), Z(char('t'))},
				}},
			},
			actCenterCursorLineBottom: {
				DocString: "Align view bottom",
				Run:       Runner(action.AlignViewBottom),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					{z(char('b'))},
					{Z(char('b'))},
				}},
			},
			actScrollUp: {
				DocString: "Scroll view up",
				Run:       Runner(action.ScrollUp),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					{z(char('k')), z(special("up"))},
					{Z(char('k')), Z(special("up"))},
				}},
			},
			actScrollDown: {
				DocString: "Scroll view down",
				Run:       Runner(action.ScrollDown),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					{z(char('j')), z(special("down"))},
					{Z(char('j')), Z(special("down"))},
				}},
			},
			actVSplitView: {
				DocString: "Vertical right split",
				Run:       Runner(action.VSplit),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					{Cw(char('v')), Cw(ctrl('v'))},
					{Spcw(char('v')), Spcw(ctrl('v'))},
				}},
				Aliases:   []string{"vs"},
				Signature: sig(),
			},
			actHSplitView: {
				DocString: "Horizontal bottom split",
				Run:       Runner(action.HSplit),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					{Cw(char('s')), Cw(ctrl('s'))},
					{Spcw(char('s')), Spcw(ctrl('s'))},
				}},
				Aliases:   []string{"hs", "sp"},
				Signature: sig(),
			},
			actVSplitNew: {
				DocString: "Vertical right split scratch buffer",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					e.VSplitNew()
					return command.Result{}
				},
				Modes: []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					{Cwn(char('v')), Cwn(ctrl('v'))},
					{Spcwn(char('v')), Spcwn(ctrl('v'))},
				}},
				Aliases:   []string{"vnew"},
				Signature: sig(),
			},
			actHSplitNew: {
				DocString: "Horizontal bottom split scratch buffer",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					e.HSplitNew()
					return command.Result{}
				},
				Modes: []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					{Cwn(char('s')), Cwn(ctrl('s'))},
					{Spcwn(char('s')), Spcwn(ctrl('s'))},
				}},
				Aliases:   []string{"hnew"},
				Signature: sig(),
			},
			actTransposeView: {
				DocString: "Transpose splits",
				Run:       Runner(action.TransposeView),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					{Cw(char('t')), Cw(ctrl('t'))},
					{Spcw(char('t')), Spcw(ctrl('t'))},
				}},
			},
			actCloseCurrentView: {
				DocString: "Close window",
				Run:       Runner(action.CloseCurrentView),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					{Cw(char('q')), Cw(ctrl('q'))},
					{Spcw(char('q')), Spcw(ctrl('q'))},
				}},
				Aliases:   []string{"wc"},
				Signature: sig(),
			},
			actCloseCurrentViewForce: {
				DocString: "Force close window",
				Run:       Runner(action.CloseCurrentViewForce),
				Aliases:   []string{"wc!"},
				Signature: sig(),
			},
			actCloseOtherViews: {
				DocString: "Close windows except current",
				Run:       Runner(action.CloseOtherViews),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					{Cw(char('o')), Cw(ctrl('o'))},
					{Spcw(char('o')), Spcw(ctrl('o'))},
				}},
				Aliases:   []string{"wo"},
				Signature: sig(),
			},
			actRotateView: {
				DocString: "Goto next window",
				Run:       Runner(action.RotateView),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					{Cw(char('w')), Cw(ctrl('w'))},
					{Spcw(char('w')), Spcw(ctrl('w'))},
				}},
			},
			actJumpViewLeft: {
				DocString: "Jump to left split",
				Run:       Runner(action.JumpViewLeft),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					{Cw(char('h')), Cw(ctrl('h')), Cw(special("left"))},
					{Spcw(char('h')), Spcw(ctrl('h')), Spcw(special("left"))},
				}},
			},
			actJumpViewDown: {
				DocString: "Jump to split below",
				Run:       Runner(action.JumpViewDown),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					{Cw(char('j')), Cw(ctrl('j')), Cw(special("down"))},
					{Spcw(char('j')), Spcw(ctrl('j')), Spcw(special("down"))},
				}},
			},
			actJumpViewUp: {
				DocString: "Jump to split above",
				Run:       Runner(action.JumpViewUp),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					{Cw(char('k')), Cw(ctrl('k')), Cw(special("up"))},
					{Spcw(char('k')), Spcw(ctrl('k')), Spcw(special("up"))},
				}},
			},
			actJumpViewRight: {
				DocString: "Jump to right split",
				Run:       Runner(action.JumpViewRight),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					{Cw(char('l')), Cw(ctrl('l')), Cw(special("right"))},
					{Spcw(char('l')), Spcw(ctrl('l')), Spcw(special("right"))},
				}},
			},
			actSwapViewLeft: {
				DocString: "Swap with left split",
				Run:       Runner(action.SwapViewLeft),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					{Cw(char('H'))},
					{Spcw(char('H'))},
				}},
			},
			actSwapViewDown: {
				DocString: "Swap with split below",
				Run:       Runner(action.SwapViewDown),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					{Cw(char('J'))},
					{Spcw(char('J'))},
				}},
			},
			actSwapViewUp: {
				DocString: "Swap with split above",
				Run:       Runner(action.SwapViewUp),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					{Cw(char('K'))},
					{Spcw(char('K'))},
				}},
			},
			actSwapViewRight: {
				DocString: "Swap with right split",
				Run:       Runner(action.SwapViewRight),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{"*": {
					{Cw(char('L'))},
					{Spcw(char('L'))},
				}},
			},
		},
		Options: []command.Option{
			{
				Key: "editor.line-number",
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
			},
			editorBoolOption("editor.cursorline",
				func(e *view.Editor) bool {
					return e.Options().Cursorline
				},
				func(e *view.Editor, v bool) {
					e.Options().Cursorline = v
				},
			),
			editorBoolOption("editor.cursorcolumn",
				func(e *view.Editor) bool {
					return e.Options().Cursorcolumn
				},
				func(e *view.Editor, v bool) {
					e.Options().Cursorcolumn = v
				},
			),
			editorNullableIntOption("editor.text-width",
				language.DefaultTextWidth,
				func(e *view.Editor) *int {
					return e.Options().TextWidth
				},
				func(e *view.Editor, v *int) {
					e.Options().TextWidth = v
				},
			),
			editorBoolOption("editor.soft-wrap.enable",
				func(e *view.Editor) bool {
					sw := e.Options().SoftWrap.Enable
					return sw != nil && *sw
				},
				func(e *view.Editor, v bool) {
					e.Options().SoftWrap.Enable = &v
				},
			),
			editorNullableIntOption("editor.soft-wrap.max-wrap",
				language.DefaultMaxWrap,
				func(e *view.Editor) *int {
					return e.Options().SoftWrap.MaxWrap
				},
				func(e *view.Editor, v *int) {
					e.Options().SoftWrap.MaxWrap = v
				},
			),
			editorNullableIntOption("editor.soft-wrap.max-indent-retain",
				language.DefaultMaxIndentRetain,
				func(e *view.Editor) *int {
					return e.Options().SoftWrap.MaxIndentRetain
				},
				func(e *view.Editor, v *int) {
					e.Options().SoftWrap.MaxIndentRetain = v
				},
			),
			{
				Key: "editor.soft-wrap.wrap-indicator",
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
			editorBoolOption("editor.soft-wrap.wrap-at-text-width",
				func(e *view.Editor) bool {
					v := e.Options().SoftWrap.WrapAtTextWidth
					return v != nil && *v
				},
				func(e *view.Editor, v bool) {
					e.Options().SoftWrap.WrapAtTextWidth = &v
				},
			),
			{
				Key: "editor.rulers",
				Get: func(e *view.Editor) (string, error) {
					return config.FormatIntSlice(e.Options().Rulers), nil
				},
				Set: func(e *view.Editor, s string) error {
					v, err := config.ParseIntSlice(s)
					if err != nil {
						return err
					}
					e.Options().Rulers = v
					return nil
				},
			},
			{
				Key: "editor.bufferline",
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
			},
			{
				Key: "editor.whitespace.render",
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
					ws.Render.Nnbsp = nil
					ws.Render.Tab = nil
					ws.Render.Newline = nil
					return nil
				},
			},
			editorBoolOption("editor.indent-guides.render",
				func(e *view.Editor) bool {
					return e.Options().IndentGuides.Render
				},
				func(e *view.Editor, v bool) {
					e.Options().IndentGuides.Render = v
				},
			),
			{
				Key: "editor.indent-guides.skip-levels",
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
			{
				Key: "editor.indent-guides.character",
				Get: func(e *view.Editor) (string, error) {
					return string(e.Options().IndentGuides.CharRune()), nil
				},
				Set: func(e *view.Editor, s string) error {
					v, err := config.ParseStringLiteral(s)
					if err != nil {
						return err
					}
					if len([]rune(v)) != 1 {
						return fmt.Errorf("%w: %s", config.ErrInvalidOption, v)
					}
					e.Options().IndentGuides.Character = v
					return nil
				},
			},
			{
				Key: "editor.gutters.line-numbers.min-width",
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
				opts.Cursorline = boolOr(cfg.Editor.Cursorline, false)
				opts.Cursorcolumn = boolOr(cfg.Editor.Cursorcolumn, false)
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
	}
}
