package defaults

import (
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

const (
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

func registerViewCommands(r *registry, model ui.Model) {
	z := prefixed(char('z'))
	Z := prefixed(char('Z'))
	Spc := prefixed(char(' '))
	Spcw := prefixed(Spc(char('w')))
	Spcwn := prefixed(Spcw(char('n')))
	Cw := prefixed(ctrl('w'))
	Cwn := prefixed(Cw(char('n')))

	r.RegisterCommand(actPageUp, command.Command{
		DocString: "Move page up",
		Run:       Runner(action.PageUp),
		Modes:     []string{"NOR", "SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{
				ctrl('b'), special("pageup"),
			},
			[][]command.KeyEvent{
				z(ctrl('b')), z(special("pageup")),
			},
			[][]command.KeyEvent{
				Z(ctrl('b')), Z(special("pageup")),
			},
		},
	})
	r.RegisterCommand(actPageDown, command.Command{
		DocString: "Move page down",
		Run:       Runner(action.PageDown),
		Modes:     []string{"NOR", "SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{
				ctrl('f'), special("pagedown"),
			},
			[][]command.KeyEvent{
				z(ctrl('f')), z(special("pagedown")),
			},
			[][]command.KeyEvent{
				Z(ctrl('f')), Z(special("pagedown")),
			},
		},
	})
	r.RegisterCommand(actPageCursorHalfUp, command.Command{
		DocString: "Move page and cursor half up",
		Run:       Runner(action.PageCursorHalfUp),
		Modes:     []string{"NOR", "SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{ctrl('u')},
			[][]command.KeyEvent{
				z(ctrl('u')), z(special("backspace")),
			},
			[][]command.KeyEvent{
				Z(ctrl('u')), Z(special("backspace")),
			},
		},
	})
	r.RegisterCommand(actPageCursorHalfDown, command.Command{
		DocString: "Move page and cursor half down",
		Run:       Runner(action.PageCursorHalfDown),
		Modes:     []string{"NOR", "SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{ctrl('d')},
			[][]command.KeyEvent{z(ctrl('d')), z(char(' '))},
			[][]command.KeyEvent{Z(ctrl('d')), Z(char(' '))},
		},
	})
	r.RegisterCommand(actHalfPageUp, command.Command{
		DocString: "Move half page up",
		Run:       Runner(action.HalfPageUp),
		Signature: sig(),
	})
	r.RegisterCommand(actHalfPageDown, command.Command{
		DocString: "Move half page down",
		Run:       Runner(action.HalfPageDown),
		Signature: sig(),
	})
	r.RegisterCommand(actPageCursorUp, command.Command{
		DocString: "Move page and cursor up",
		Run:       Runner(action.PageCursorUp),
	})
	r.RegisterCommand(actPageCursorDown, command.Command{
		DocString: "Move page and cursor down",
		Run:       Runner(action.PageCursorDown),
	})
	r.RegisterCommand(actCenterCursorLine, command.Command{
		DocString: "Align view center",
		Run:       Runner(action.AlignViewCenter),
		Modes:     []string{"NOR", "SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{z(char('z')), z(char('c'))},
			[][]command.KeyEvent{Z(char('z')), Z(char('c'))},
		},
	})
	r.RegisterCommand(actCenterCursorLineTop, command.Command{
		DocString: "Align view top",
		Run:       Runner(action.AlignViewTop),
		Modes:     []string{"NOR", "SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{z(char('.')), z(char('t'))},
			[][]command.KeyEvent{Z(char('.')), Z(char('t'))},
		},
	})
	r.RegisterCommand(actCenterCursorLineBottom, command.Command{
		DocString: "Align view bottom",
		Run:       Runner(action.AlignViewBottom),
		Modes:     []string{"NOR", "SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{z(char('b'))},
			[][]command.KeyEvent{Z(char('b'))},
		},
	})
	r.RegisterCommand(actScrollUp, command.Command{
		DocString: "Scroll view up",
		Run:       Runner(action.ScrollUp),
		Modes:     []string{"NOR", "SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{z(char('k')), z(special("up"))},
			[][]command.KeyEvent{Z(char('k')), Z(special("up"))},
		},
	})
	r.RegisterCommand(actScrollDown, command.Command{
		DocString: "Scroll view down",
		Run:       Runner(action.ScrollDown),
		Modes:     []string{"NOR", "SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{z(char('j')), z(special("down"))},
			[][]command.KeyEvent{Z(char('j')), Z(special("down"))},
		},
	})

	r.RegisterCommand(actVSplitView, command.Command{
		DocString: "Vertical right split",
		Run:       Runner(action.VSplit),
		Modes:     []string{"NOR", "SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{Cw(char('v')), Cw(ctrl('v'))},
			[][]command.KeyEvent{Spcw(char('v')), Spcw(ctrl('v'))},
		},
		Aliases:   []string{"vs"},
		Signature: sig(),
	})
	r.RegisterCommand(actHSplitView, command.Command{
		DocString: "Horizontal bottom split",
		Run:       Runner(action.HSplit),
		Modes:     []string{"NOR", "SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{Cw(char('s')), Cw(ctrl('s'))},
			[][]command.KeyEvent{Spcw(char('s')), Spcw(ctrl('s'))},
		},
		Aliases:   []string{"hs", "sp"},
		Signature: sig(),
	})
	r.RegisterCommand(actVSplitNew, command.Command{
		DocString: "Vertical right split scratch buffer",
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			e.VSplitNew()
			return command.Result{}
		},
		Modes: []string{"NOR", "SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{Cwn(char('v')), Cwn(ctrl('v'))},
			[][]command.KeyEvent{Spcwn(char('v')), Spcwn(ctrl('v'))},
		},
		Aliases:   []string{"vnew"},
		Signature: sig(),
	})
	r.RegisterCommand(actHSplitNew, command.Command{
		DocString: "Horizontal bottom split scratch buffer",
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			e.HSplitNew()
			return command.Result{}
		},
		Modes: []string{"NOR", "SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{Cwn(char('s')), Cwn(ctrl('s'))},
			[][]command.KeyEvent{Spcwn(char('s')), Spcwn(ctrl('s'))},
		},
		Aliases:   []string{"hnew"},
		Signature: sig(),
	})
	r.RegisterCommand(actTransposeView, command.Command{
		DocString: "Transpose splits",
		Run:       Runner(action.TransposeView),
		Modes:     []string{"NOR", "SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{Cw(char('t')), Cw(ctrl('t'))},
			[][]command.KeyEvent{Spcw(char('t')), Spcw(ctrl('t'))},
		},
	})
	r.RegisterCommand(actCloseCurrentView, command.Command{
		DocString: "Close window",
		Run:       Runner(action.CloseCurrentView),
		Modes:     []string{"NOR", "SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{Cw(char('q')), Cw(ctrl('q'))},
			[][]command.KeyEvent{Spcw(char('q')), Spcw(ctrl('q'))},
		},
		Aliases:   []string{"wc"},
		Signature: sig(),
	})
	r.RegisterCommand(actCloseCurrentViewForce, command.Command{
		DocString: "Force close window",
		Run:       Runner(action.CloseCurrentViewForce),
		Aliases:   []string{"wc!"},
		Signature: sig(),
	})
	r.RegisterCommand(actCloseOtherViews, command.Command{
		DocString: "Close windows except current",
		Run:       Runner(action.CloseOtherViews),
		Modes:     []string{"NOR", "SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{Cw(char('o')), Cw(ctrl('o'))},
			[][]command.KeyEvent{Spcw(char('o')), Spcw(ctrl('o'))},
		},
		Aliases:   []string{"wo"},
		Signature: sig(),
	})
	r.RegisterCommand(actRotateView, command.Command{
		DocString: "Goto next window",
		Run:       Runner(action.RotateView),
		Modes:     []string{"NOR", "SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{Cw(char('w')), Cw(ctrl('w'))},
			[][]command.KeyEvent{Spcw(char('w')), Spcw(ctrl('w'))},
		},
	})
	r.RegisterCommand(actJumpViewLeft, command.Command{
		DocString: "Jump to left split",
		Run:       Runner(action.JumpViewLeft),
		Modes:     []string{"NOR", "SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{
				Cw(char('h')), Cw(ctrl('h')), Cw(special("left")),
			},
			[][]command.KeyEvent{
				Spcw(char('h')), Spcw(ctrl('h')), Spcw(special("left")),
			},
		},
	})
	r.RegisterCommand(actJumpViewDown, command.Command{
		DocString: "Jump to split below",
		Run:       Runner(action.JumpViewDown),
		Modes:     []string{"NOR", "SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{
				Cw(char('j')), Cw(ctrl('j')), Cw(special("down")),
			},
			[][]command.KeyEvent{
				Spcw(char('j')), Spcw(ctrl('j')), Spcw(special("down")),
			},
		},
	})
	r.RegisterCommand(actJumpViewUp, command.Command{
		DocString: "Jump to split above",
		Run:       Runner(action.JumpViewUp),
		Modes:     []string{"NOR", "SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{
				Cw(char('k')), Cw(ctrl('k')), Cw(special("up")),
			},
			[][]command.KeyEvent{
				Spcw(char('k')), Spcw(ctrl('k')), Spcw(special("up")),
			},
		},
	})
	r.RegisterCommand(actJumpViewRight, command.Command{
		DocString: "Jump to right split",
		Run:       Runner(action.JumpViewRight),
		Modes:     []string{"NOR", "SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{
				Cw(char('l')), Cw(ctrl('l')), Cw(special("right")),
			},
			[][]command.KeyEvent{
				Spcw(char('l')), Spcw(ctrl('l')), Spcw(special("right")),
			},
		},
	})
	r.RegisterCommand(actSwapViewLeft, command.Command{
		DocString: "Swap with left split",
		Run:       Runner(action.SwapViewLeft),
		Modes:     []string{"NOR", "SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{Cw(char('H'))},
			[][]command.KeyEvent{Spcw(char('H'))},
		},
	})
	r.RegisterCommand(actSwapViewDown, command.Command{
		DocString: "Swap with split below",
		Run:       Runner(action.SwapViewDown),
		Modes:     []string{"NOR", "SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{Cw(char('J'))},
			[][]command.KeyEvent{Spcw(char('J'))},
		},
	})
	r.RegisterCommand(actSwapViewUp, command.Command{
		DocString: "Swap with split above",
		Run:       Runner(action.SwapViewUp),
		Modes:     []string{"NOR", "SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{Cw(char('K'))},
			[][]command.KeyEvent{Spcw(char('K'))},
		},
	})
	r.RegisterCommand(actSwapViewRight, command.Command{
		DocString: "Swap with right split",
		Run:       Runner(action.SwapViewRight),
		Modes:     []string{"NOR", "SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{Cw(char('L'))},
			[][]command.KeyEvent{Spcw(char('L'))},
		},
	})
	// z/Z sub-mode: search aliases
	r.RegisterCommand(actSearch, command.Command{
		DocString: "Search for regex pattern",
		Run:       Continuation(model.SearchAction(true)),
		Modes:     []string{"NOR", "SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{z(char('/'))},
			[][]command.KeyEvent{Z(char('/'))},
		},
	})
	r.RegisterCommand(actSearchReverse, command.Command{
		DocString: "Reverse search for regex pattern",
		Run:       Continuation(model.SearchAction(false)),
		Modes:     []string{"NOR", "SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{z(char('?'))},
			[][]command.KeyEvent{Z(char('?'))},
		},
	})
	r.RegisterCommand(actSearchNext, command.Command{
		DocString: "Select next search match",
		Run:       Runner(action.SearchNext),
		Modes:     []string{"NOR"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{z(char('n'))},
			[][]command.KeyEvent{Z(char('n'))},
		},
	})
	r.RegisterCommand(actSearchPrev, command.Command{
		DocString: "Select previous search match",
		Run:       Runner(action.SearchPrev),
		Modes:     []string{"NOR"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{z(char('N'))},
			[][]command.KeyEvent{Z(char('N'))},
		},
	})
	r.RegisterCommand(actExtendSearchNext, command.Command{
		DocString: "Add next search match to selection",
		Run:       Runner(action.ExtendSearchNext),
		Modes:     []string{"SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{z(char('n'))},
			[][]command.KeyEvent{Z(char('n'))},
		},
	})
	r.RegisterCommand(actExtendSearchPrev, command.Command{
		DocString: "Add previous search match to selection",
		Run:       Runner(action.ExtendSearchPrev),
		Modes:     []string{"SEL"},
		Keys: []command.KeyBinding{
			[][]command.KeyEvent{z(char('N'))},
			[][]command.KeyEvent{Z(char('N'))},
		},
	})
}
