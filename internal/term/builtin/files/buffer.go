package files

import (
	"embed"

	"github.com/kode4food/toe/internal/i18n"
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

const (
	errorUnsavedBufferCloseKey    i18n.Key = "error.unsavedBufferClose"
	errorUnsavedBufferCloseAllKey i18n.Key = "error.unsavedBufferCloseAll"
)

var (
	//go:embed i18n/buffer.*.json
	bufferFS embed.FS

	errUnsavedBufferClose    = i18n.NewError(errorUnsavedBufferCloseKey)
	errUnsavedBufferCloseAll = i18n.NewError(errorUnsavedBufferCloseAllKey)
)

// BufferModule returns the buffer navigation and close commands
func BufferModule() command.Module {
	g := kit.Prefixed(kit.Char('g'))

	return command.Module{
		Translations: i18n.LoadTranslations(bufferFS),
		Commands: []command.Command{
			{
				Name:      actBufferClose,
				DocString: "Close the current buffer",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					doc := e.FocusedDocument()
					if doc != nil && doc.Modified() {
						return command.Result{Error: errUnsavedBufferClose}
					}
					e.CloseCurrentView()
					return command.Result{Message: "buffer closed"}
				},
				Modes:     command.PaneModes,
				Aliases:   []string{"bc", "bclose"},
				Signature: command.DefaultSignature(),
			},
			{
				Name: actBufferCloseForce,
				DocString: "Close the current buffer forcefully, ignoring " +
					"unsaved changes",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					e.CloseCurrentView()
					return command.Result{Message: "buffer closed"}
				},
				Modes:     command.PaneModes,
				Aliases:   []string{"buffer-close!", "bc!", "bclose!"},
				Signature: command.DefaultSignature(),
			},
			{
				Name:      actBufferCloseOthers,
				DocString: "Close all buffers but the currently focused one",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					focused := e.FocusedView()
					for _, v := range e.AllViews() {
						if focused == nil || v.ID() != focused.ID() {
							e.ClosePane(v.ID())
						}
					}
					return command.Result{Message: "other buffers closed"}
				},
				Modes: command.PaneModes,
				Aliases: []string{
					"bco", "bcloseother",
				},
				Signature: command.DefaultSignature(),
			},
			{
				Name:      actBufferCloseAll,
				DocString: "Close all buffers without quitting",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					for _, doc := range e.AllDocuments() {
						if doc.Modified() {
							return command.Result{
								Error: errUnsavedBufferCloseAll,
							}
						}
					}
					for _, v := range e.AllViews() {
						e.ClosePane(v.ID())
					}
					return command.Result{Message: "all buffers closed"}
				},
				Modes:     command.PaneModes,
				Aliases:   []string{"bca", "bcloseall"},
				Signature: command.DefaultSignature(),
			},
			{
				Name:      actBufferNext,
				DocString: "Goto next buffer",
				Run:       kit.Runner((*view.Editor).FocusNextView),
				Modes:     command.PaneModes,
				Keys:      kit.Keys(g(kit.Char('n'))),
				Aliases:   []string{"bn", "bnext"},
				Signature: command.DefaultSignature(),
			},
			{
				Name:      actBufferPrevious,
				DocString: "Goto previous buffer",
				Run:       kit.Runner((*view.Editor).FocusPrevView),
				Modes:     command.PaneModes,
				Keys:      kit.Keys(g(kit.Char('p'))),
				Aliases:   []string{"bp", "bprev"},
				Signature: command.DefaultSignature(),
			},
		},
	}
}
