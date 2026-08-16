package files

import (
	"embed"
	"errors"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/builtin/kit"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

type fileSection struct {
	Editor struct {
		InsertFinalNewline *bool `toml:"insert-final-newline"`
		TrimFinalNewlines  *bool `toml:"trim-final-newlines"`
		TrimTrailingWS     *bool `toml:"trim-trailing-whitespace"`
	} `toml:"editor"`
}

const (
	actWrite                 = "write"
	actWriteForce            = "write!"
	actWriteAll              = "write_all"
	actWriteAllForce         = "write-all!"
	actWriteQuit             = "write_quit"
	actWriteQuitForce        = "write-quit!"
	actWriteBufferClose      = "write_buffer_close"
	actWriteBufferCloseForce = "write-buffer-close!"
	actUpdate                = "update"
	actOpen                  = "open"
	actNew                   = "new"
	actReload                = "reload"
	actReloadAll             = "reload_all"
	actMove                  = "move"
	actMoveForce             = "move!"
	actRead                  = "read"
)

const (
	errorNoFilenameKey        i18n.Key = "error.noFilename"
	errorUnsavedMoveKey       i18n.Key = "error.unsavedMove"
	statusWrittenKey          i18n.Key = "status.written"
	statusAllWrittenKey       i18n.Key = "status.allWritten"
	statusWrittenAndClosedKey i18n.Key = "status.writtenAndClosed"
)

var (
	errNoFilename  = i18n.NewError(errorNoFilenameKey)
	errNoDocument  = i18n.NewError(i18n.ErrorNoDocument)
	errUnsavedMove = i18n.NewError(errorUnsavedMoveKey)
)

//go:embed i18n/file.*.json
var fileFS embed.FS

// FileModule returns the file open, write, and manage commands
func FileModule() command.Module {
	cfg := new(fileSection)
	cmds := fileWriteCmds()
	cmds = append(cmds, fileManageCmds()...)
	return command.Module{
		Translations: i18n.LoadTranslations(fileFS),
		Commands:     cmds,
		Options: []command.Option{
			kit.EditorBoolOption("insert-final-newline",
				func(e *view.Editor) bool {
					return e.Options().InsertFinalNewline
				},
				func(e *view.Editor, v bool) {
					e.Options().InsertFinalNewline = v
				},
			).WithDoc("Add a trailing line ending on write if missing"),
			kit.EditorBoolOption("trim-final-newlines",
				func(e *view.Editor) bool {
					return e.Options().TrimFinalNewlines
				},
				func(e *view.Editor, v bool) {
					e.Options().TrimFinalNewlines = v
				},
			).WithDoc("Remove extra trailing line endings on write"),
			kit.EditorBoolOption("trim-trailing-whitespace",
				func(e *view.Editor) bool {
					return e.Options().TrimTrailingWhitespace
				},
				func(e *view.Editor, v bool) {
					e.Options().TrimTrailingWhitespace = v
				},
			).WithDoc("Remove trailing whitespace on write"),
		},
		Section: &command.Section{
			Config: cfg,
			Reset:  func() { *cfg = fileSection{} },
			Apply: func(e *view.Editor) {
				opts := e.Options()
				opts.InsertFinalNewline = kit.BoolOr(
					cfg.Editor.InsertFinalNewline, true,
				)
				opts.TrimFinalNewlines = kit.BoolOr(
					cfg.Editor.TrimFinalNewlines, false,
				)
				opts.TrimTrailingWhitespace = kit.BoolOr(
					cfg.Editor.TrimTrailingWS, false,
				)
			},
		},
	}
}

func fileWriteCmds() []command.Command {
	return []command.Command{
		{
			Name:      actWrite,
			DocString: "Write changes to disk. Accepts an optional path",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				setPathFromArgs(e, args)
				autoFormat(e)
				if err := e.Save(false); err != nil {
					return command.Result{Error: err}
				}
				if doc := e.FocusedDocument(); doc != nil {
					return command.Result{
						Message: "'" + doc.RelativeName(e.Cwd()) +
							"' written",
					}
				}
				return command.Result{
					Message: i18n.Text(statusWrittenKey),
				}
			},
			Modes:     command.DocModes,
			Aliases:   []string{"w"},
			Signature: kit.FileSig(kit.OptionalArg()),
		},
		{
			Name:      actWriteForce,
			DocString: "Force write changes to disk, creating subdirectories",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				setPathFromArgs(e, args)
				autoFormat(e)
				_ = e.Save(true)
				if doc := e.FocusedDocument(); doc != nil {
					return command.Result{
						Message: "'" + doc.RelativeName(e.Cwd()) +
							"' written",
					}
				}
				return command.Result{
					Message: i18n.Text(statusWrittenKey),
				}
			},
			Modes:     command.DocModes,
			Aliases:   []string{"w!"},
			Signature: kit.FileSig(kit.OptionalArg()),
		},
		{
			Name:      actWriteAll,
			DocString: "Write changes from all buffers to disk",
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				if errs := e.SaveAll(false); len(errs) > 0 {
					return command.Result{Error: errs[0]}
				}
				return command.Result{
					Message: i18n.Text(statusAllWrittenKey),
				}
			},
			Modes:   command.PaneModes,
			Aliases: []string{"wa"},
		},
		{
			Name:      actWriteAllForce,
			DocString: "Force write all buffers, creating subdirectories",
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				for _, doc := range e.AllDocuments() {
					_ = doc.Save(e.Options(), true)
				}
				return command.Result{
					Message: i18n.Text(statusAllWrittenKey),
				}
			},
			Modes:   command.PaneModes,
			Aliases: []string{"wa!"},
		},
		{
			Name:      actWriteQuit,
			DocString: "Write all documents and quit",
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				if errs := e.SaveAll(false); len(errs) > 0 {
					return command.Result{Error: errs[0]}
				}
				return command.Result{Signal: command.SignalQuit}
			},
			Modes:   command.PaneModes,
			Aliases: []string{"wq"},
		},
		{
			Name: actWriteQuitForce,
			DocString: "Write all documents and quit, discarding " +
				"scratch buffers",
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				for _, err := range e.SaveAll(true) {
					if !errors.Is(err, view.ErrDocumentNoPath) {
						return command.Result{Error: err}
					}
				}
				return command.Result{Signal: command.SignalQuit}
			},
			Modes:   command.PaneModes,
			Aliases: []string{"wq!"},
		},
		{
			Name:      actWriteBufferClose,
			DocString: "Write changes to disk and close the buffer",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				setPathFromArgs(e, args)
				autoFormat(e)
				if err := e.Save(false); err != nil {
					return command.Result{Error: err}
				}
				e.CloseCurrentView()
				return command.Result{
					Message: i18n.Text(statusWrittenAndClosedKey),
				}
			},
			Modes:     command.DocModes,
			Aliases:   []string{"wbc"},
			Signature: kit.FileSig(kit.OptionalArg()),
		},
		{
			Name: actWriteBufferCloseForce,
			DocString: "Force write and close the buffer, creating " +
				"subdirectories",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				setPathFromArgs(e, args)
				autoFormat(e)
				_ = e.Save(true)
				e.CloseCurrentView()
				return command.Result{
					Message: i18n.Text(statusWrittenAndClosedKey),
				}
			},
			Modes:     command.DocModes,
			Aliases:   []string{"wbc!"},
			Signature: kit.FileSig(kit.OptionalArg()),
		},
	}
}

func fileManageCmds() []command.Command {
	return []command.Command{
		{
			Name:      actUpdate,
			DocString: "Write changes only if the file has been modified",
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				doc := e.FocusedDocument()
				if doc == nil || !doc.Modified() {
					return command.Result{Message: "no changes to write"}
				}
				autoFormat(e)
				if err := e.Save(false); err != nil {
					return command.Result{Error: err}
				}
				return command.Result{
					Message: "'" + doc.RelativeName(e.Cwd()) + "' written",
				}
			},
			Modes:   command.DocModes,
			Aliases: []string{"u"},
		},
		{
			Name:      actOpen,
			DocString: "Open a file from disk into the current view",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				if args == nil || args.Empty() {
					return command.Result{Error: errNoFilename}
				}
				path, _ := args.First()
				_, _, err := ui.OpenPath(e, path, ui.PickerAcceptReplace)
				if err != nil {
					return command.Result{Error: err}
				}
				if doc := e.FocusedDocument(); doc != nil {
					return command.Result{
						Message: "'" + doc.RelativeName(e.Cwd()) +
							"' opened",
					}
				}
				return command.Result{Message: "opened"}
			},
			Modes:     command.PaneModes,
			Aliases:   []string{"o", "edit", "e"},
			Signature: kit.FileSig(kit.MinArgs(1)),
		},
		{
			Name:      actNew,
			DocString: "Create a new scratch buffer",
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				e.NewDocument()
				return command.Result{Message: "[scratch]"}
			},
			Modes:   command.PaneModes,
			Keys:    kit.Window(kit.Char('n')),
			Aliases: []string{"n"},
		},
		{
			Name:      actReload,
			DocString: "Discard changes and reload from the source file",
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				if err := e.Reload(); err != nil {
					return command.Result{Error: err}
				}
				if doc := e.FocusedDocument(); doc != nil {
					return command.Result{
						Message: "'" + doc.RelativeName(e.Cwd()) +
							"' reloaded",
					}
				}
				return command.Result{Message: "reloaded"}
			},
			Modes:   command.DocModes,
			Aliases: []string{"rl"},
		},
		{
			Name:      actReloadAll,
			DocString: "Discard changes and reload all documents from disk",
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				if errs := e.ReloadAll(); len(errs) > 0 {
					return command.Result{Error: errs[0]}
				}
				return command.Result{Message: "all documents reloaded"}
			},
			Modes:   command.PaneModes,
			Aliases: []string{"rla"},
		},
		{
			Name: actMove,
			DocString: "Move the current buffer and its file to a different " +
				"path",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				if args == nil || args.Empty() {
					return command.Result{Error: errNoFilename}
				}
				doc := e.FocusedDocument()
				if doc == nil {
					return command.Result{Error: errNoDocument}
				}
				if doc.Modified() {
					return command.Result{Error: errUnsavedMove}
				}
				path, _ := args.First()
				if err := e.MoveFocusedFile(path, false); err != nil {
					return command.Result{Error: err}
				}
				return command.Result{Message: "moved to '" + path + "'"}
			},
			Modes:     command.DocModes,
			Aliases:   []string{"mv"},
			Signature: kit.FileSig(kit.RequiredArg()),
		},
		{
			Name:      actMoveForce,
			DocString: "Move the buffer and its file, creating subdirectories",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				if args == nil || args.Empty() {
					return command.Result{Error: errNoFilename}
				}
				if e.FocusedDocument() == nil {
					return command.Result{Error: errNoDocument}
				}
				path, _ := args.First()
				_ = e.MoveFocusedFile(path, true)
				return command.Result{Message: "moved to '" + path + "'"}
			},
			Modes:     command.DocModes,
			Aliases:   []string{"mv!"},
			Signature: kit.FileSig(kit.RequiredArg()),
		},
		{
			Name:      actRead,
			DocString: "Load a file into buffer",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				if args == nil || args.Empty() {
					return command.Result{Error: errNoFilename}
				}
				path, _ := args.First()
				if err := action.ReadFile(e, path); err != nil {
					return command.Result{Error: err}
				}
				return command.Result{Message: "'" + path + "' inserted"}
			},
			Modes:     command.DocModes,
			Aliases:   []string{"r"},
			Signature: kit.FileSig(kit.RequiredArg()),
		},
	}
}

func setPathFromArgs(e *view.Editor, args *command.Args) {
	if args == nil {
		return
	}
	path, ok := args.First()
	if !ok {
		return
	}
	if doc := e.FocusedDocument(); doc != nil {
		doc.SetPath(path)
	}
}
