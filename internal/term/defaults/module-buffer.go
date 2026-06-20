package defaults

import (
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

const (
	actBufferClose        = "buffer_close"
	actBufferCloseForce   = "buffer_close_force"
	actBufferCloseOthers  = "buffer_close_others"
	actBufferCloseAll     = "buffer_close_all"
	actBufferNext         = "buffer_next"
	actBufferPrevious     = "buffer_previous"
	actGotoNextBuffer     = "goto_next_buffer"
	actGotoPreviousBuffer = "goto_previous_buffer"
)

func bufferModule() command.Module {
	g := prefixed(char('g'))

	return command.Module{
		Commands: map[string]command.Command{
			actBufferClose: {
				DocString: "Close the current buffer",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					if doc, ok := e.FocusedDocument(); ok && doc.Modified() {
						return command.Result{
							Message: "document has unsaved changes (use " +
								":buffer-close! to force)",
						}
					}
					e.CloseCurrentView()
					return command.Result{Message: "buffer closed"}
				},
				Modes:     []string{"NOR", "SEL"},
				Aliases:   []string{"buffer-close", "bc", "bclose"},
				Signature: sig(),
			},
			actBufferCloseForce: {
				DocString: "Close the current buffer forcefully, ignoring " +
					"unsaved changes",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					e.CloseCurrentView()
					return command.Result{Message: "buffer closed"}
				},
				Modes:     []string{"NOR", "SEL"},
				Aliases:   []string{"buffer-close!", "bc!", "bclose!"},
				Signature: sig(),
			},
			actBufferCloseOthers: {
				DocString: "Close all buffers but the currently focused one",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					focused, _ := e.FocusedView()
					for _, v := range e.AllViews() {
						if focused == nil || v.ID() != focused.ID() {
							e.CloseView(v.ID())
						}
					}
					return command.Result{Message: "other buffers closed"}
				},
				Modes: []string{"NOR", "SEL"},
				Aliases: []string{
					"buffer-close-others", "bco", "bcloseother",
				},
				Signature: sig(),
			},
			actBufferCloseAll: {
				DocString: "Close all buffers without quitting",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					for _, doc := range e.AllDocuments() {
						if doc.Modified() {
							return command.Result{
								Message: "documents have unsaved changes " +
									"(use :buffer-close-all! to force)",
							}
						}
					}
					for _, v := range e.AllViews() {
						e.CloseView(v.ID())
					}
					return command.Result{Message: "all buffers closed"}
				},
				Modes:     []string{"NOR", "SEL"},
				Aliases:   []string{"buffer-close-all", "bca", "bcloseall"},
				Signature: sig(),
			},
			actBufferNext: {
				DocString: "Goto next buffer",
				Run:       Runner((*view.Editor).FocusNextView),
				Modes:     []string{"NOR", "SEL"},
				Aliases:   []string{"buffer-next", "bn", "bnext"},
				Signature: sig(),
			},
			actBufferPrevious: {
				DocString: "Goto previous buffer",
				Run:       Runner((*view.Editor).FocusPrevView),
				Modes:     []string{"NOR", "SEL"},
				Aliases:   []string{"buffer-previous", "bp", "bprev"},
				Signature: sig(),
			},
			actGotoNextBuffer: {
				DocString: "Goto next buffer",
				Run:       method((*view.Editor).FocusNextView),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(g(char('n'))),
			},
			actGotoPreviousBuffer: {
				DocString: "Goto previous buffer",
				Run:       method((*view.Editor).FocusPrevView),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(g(char('p'))),
			},
		},
	}
}
