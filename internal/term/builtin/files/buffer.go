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
	errorNoSuchBufferKey          i18n.Key = "error.noSuchBuffer"
)

var (
	errUnsavedBufferClose    = i18n.NewError(errorUnsavedBufferCloseKey)
	errUnsavedBufferCloseAll = i18n.NewError(errorUnsavedBufferCloseAllKey)
	errNoSuchBuffer          = i18n.NewError(errorNoSuchBufferKey)
)

//go:embed i18n/buffer.*.json
var bufferFS embed.FS

// BufferModule returns the buffer navigation and close commands
func BufferModule() command.Module {
	g := kit.Prefixed(kit.Char('g'))

	return command.Module{
		Translations: i18n.LoadTranslations(bufferFS),
		Commands: []command.Command{
			{
				Name:      actBufferClose,
				DocString: "Close the current buffer",
				Run: func(e *view.Editor, args *command.Args) command.Result {
					return closeBuffers(e, args, false)
				},
				Modes:     command.PaneModes,
				Aliases:   []string{"bc", "bclose"},
				Signature: kit.FileSig(kit.MinArgs(0)),
			},
			{
				Name: actBufferCloseForce,
				DocString: "Close the current buffer forcefully, ignoring " +
					"unsaved changes",
				Run: func(e *view.Editor, args *command.Args) command.Result {
					return closeBuffers(e, args, true)
				},
				Modes:     command.PaneModes,
				Aliases:   []string{"buffer-close!", "bc!", "bclose!"},
				Signature: kit.FileSig(kit.MinArgs(0)),
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
				Modes:   command.PaneModes,
				Aliases: []string{"bca", "bcloseall"},
			},
			{
				Name:      actBufferNext,
				DocString: "Goto next buffer",
				Run:       kit.Runner((*view.Editor).FocusNextView),
				Modes:     command.PaneModes,
				Keys:      kit.Keys(g(kit.Char('n'))),
				Aliases:   []string{"bn", "bnext"},
			},
			{
				Name:      actBufferPrevious,
				DocString: "Goto previous buffer",
				Run:       kit.Runner((*view.Editor).FocusPrevView),
				Modes:     command.PaneModes,
				Keys:      kit.Keys(g(kit.Char('p'))),
				Aliases:   []string{"bp", "bprev"},
			},
		},
	}
}

func closeBuffers(
	e *view.Editor, args *command.Args, force bool,
) command.Result {
	views, err := buffersToClose(e, args)
	if err != nil {
		return command.Result{Error: err}
	}
	if !force {
		for _, v := range views {
			if doc := e.Document(v.DocID()); doc != nil && doc.Modified() {
				return command.Result{Error: errUnsavedBufferClose}
			}
		}
	}
	for _, v := range views {
		e.ClosePane(v.ID())
	}
	return command.Result{Message: "buffer closed"}
}

func buffersToClose(e *view.Editor, args *command.Args) ([]*view.View, error) {
	if args == nil || args.Empty() {
		if v := e.FocusedView(); v != nil {
			return []*view.View{v}, nil
		}
		return nil, nil
	}
	var out []*view.View
	for _, name := range args.Positionals() {
		found := false
		for _, v := range e.AllViews() {
			doc := e.Document(v.DocID())
			if doc == nil ||
				(doc.Path() != name && doc.RelativeName(e.Cwd()) != name) {
				continue
			}
			out = append(out, v)
			found = true
		}
		if !found {
			return nil, errNoSuchBuffer.WithVars(i18n.Vars{"name": name})
		}
	}
	return out, nil
}
