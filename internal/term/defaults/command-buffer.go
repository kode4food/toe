package defaults

import (
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

const (
	actBufferClose       = "buffer_close"
	actBufferCloseForce  = "buffer_close_force"
	actBufferCloseOthers = "buffer_close_others"
	actBufferCloseAll    = "buffer_close_all"
	actBufferNext        = "buffer_next"
	actBufferPrevious    = "buffer_previous"
)

func registerBufferCommands(r *registry) {
	r.RegisterCommand(actBufferClose, command.Command{
		DocString: "Close the current buffer",
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			if doc, ok := e.FocusedDocument(); ok && doc.Modified() {
				return command.Result{
					Message: "document has unsaved changes (use :buffer-close! to force)",
				}
			}
			e.CloseCurrentView()
			return command.Result{Message: "buffer closed"}
		},
		Modes:     []string{"NOR", "SEL"},
		Aliases:   []string{"buffer-close", "bc", "bclose"},
		Signature: sig(),
	})
	r.RegisterCommand(actBufferCloseForce, command.Command{
		DocString: "Close the current buffer forcefully, ignoring unsaved changes",
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			e.CloseCurrentView()
			return command.Result{Message: "buffer closed"}
		},
		Modes:     []string{"NOR", "SEL"},
		Aliases:   []string{"buffer-close!", "bc!", "bclose!"},
		Signature: sig(),
	})
	r.RegisterCommand(actBufferCloseOthers, command.Command{
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
		Modes:     []string{"NOR", "SEL"},
		Aliases:   []string{"buffer-close-others", "bco", "bcloseother"},
		Signature: sig(),
	})
	r.RegisterCommand(actBufferCloseAll, command.Command{
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
	})
	r.RegisterCommand(actBufferNext, command.Command{
		DocString: "Goto next buffer",
		Run:       Runner((*view.Editor).FocusNextView),
		Modes:     []string{"NOR", "SEL"},
		Aliases:   []string{"buffer-next", "bn", "bnext"},
		Signature: sig(),
	})
	r.RegisterCommand(actBufferPrevious, command.Command{
		DocString: "Goto previous buffer",
		Run:       Runner((*view.Editor).FocusPrevView),
		Modes:     []string{"NOR", "SEL"},
		Aliases:   []string{"buffer-previous", "bp", "bprev"},
		Signature: sig(),
	})
}
