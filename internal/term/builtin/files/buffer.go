package files

import (
	"github.com/kode4food/toe/internal/term/builtin/kit"
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

// BufferModule returns the buffer navigation and close commands
func BufferModule() command.Module {
	g := kit.Prefixed(kit.Char('g'))

	return command.Module{
		Commands: []command.Command{
			{
				Name:      actBufferClose,
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
				Aliases:   []string{"bc", "bclose"},
				Signature: kit.Sig(),
			},
			{
				Name: actBufferCloseForce,
				DocString: "Close the current buffer forcefully, ignoring " +
					"unsaved changes",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					e.CloseCurrentView()
					return command.Result{Message: "buffer closed"}
				},
				Modes:     []string{"NOR", "SEL"},
				Aliases:   []string{"buffer-close!", "bc!", "bclose!"},
				Signature: kit.Sig(),
			},
			{
				Name:      actBufferCloseOthers,
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
					"bco", "bcloseother",
				},
				Signature: kit.Sig(),
			},
			{
				Name:      actBufferCloseAll,
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
				Aliases:   []string{"bca", "bcloseall"},
				Signature: kit.Sig(),
			},
			{
				Name:      actBufferNext,
				DocString: "Goto next buffer",
				Run:       kit.Runner((*view.Editor).FocusNextView),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(g(kit.Char('n'))),
				Aliases:   []string{"bn", "bnext"},
				Signature: kit.Sig(),
			},
			{
				Name:      actBufferPrevious,
				DocString: "Goto previous buffer",
				Run:       kit.Runner((*view.Editor).FocusPrevView),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(g(kit.Char('p'))),
				Aliases:   []string{"bp", "bprev"},
				Signature: kit.Sig(),
			},
		},
	}
}
