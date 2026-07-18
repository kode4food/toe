package files

import (
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
	actWriteQuitAll          = "write_quit_all"
	actWriteQuitAllForce     = "write-quit-all!"
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

// FileModule returns the file open, write, and manage commands
func FileModule() command.Module {
	cfg := new(fileSection)
	cmds := fileWriteCmds()
	cmds = append(cmds, fileManageCmds()...)
	return command.Module{
		Commands: cmds,
		Options: []command.Option{
			kit.EditorBoolOption("insert-final-newline",
				func(e *view.Editor) bool {
					return e.Options().InsertFinalNewline
				},
				func(e *view.Editor, v bool) {
					e.Options().InsertFinalNewline = v
				},
			),
			kit.EditorBoolOption("trim-final-newlines",
				func(e *view.Editor) bool {
					return e.Options().TrimFinalNewlines
				},
				func(e *view.Editor, v bool) {
					e.Options().TrimFinalNewlines = v
				},
			),
			kit.EditorBoolOption("trim-trailing-whitespace",
				func(e *view.Editor) bool {
					return e.Options().TrimTrailingWS
				},
				func(e *view.Editor, v bool) {
					e.Options().TrimTrailingWS = v
				},
			),
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
				opts.TrimTrailingWS = kit.BoolOr(cfg.Editor.TrimTrailingWS, false)
			},
		},
	}
}

func fileWriteCmds() []command.Command {
	return []command.Command{
		{
			Name: actWrite,
			DocString: "Write changes to disk. Accepts an optional path " +
				"(:write some/path.txt)",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				setPathFromArgs(e, args)
				autoFormat(e)
				if err := e.Save(false); err != nil {
					return command.Result{Message: "error: " + err.Error()}
				}
				if doc, ok := e.FocusedDocument(); ok {
					return command.Result{
						Message: "'" + doc.RelativeName(e.Cwd()) +
							"' written",
					}
				}
				return command.Result{
					Message: i18n.Text(i18n.StatusWritten),
				}
			},
			Modes:     command.DocumentModes(),
			Aliases:   []string{"w"},
			Signature: kit.Sig(),
		},
		{
			Name: actWriteForce,
			DocString: "Force write changes to disk creating necessary " +
				"subdirectories. Accepts an optional path (:write! " +
				"some/path.txt)",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				setPathFromArgs(e, args)
				autoFormat(e)
				_ = e.Save(true)
				if doc, ok := e.FocusedDocument(); ok {
					return command.Result{
						Message: "'" + doc.RelativeName(e.Cwd()) +
							"' written",
					}
				}
				return command.Result{
					Message: i18n.Text(i18n.StatusWritten),
				}
			},
			Modes:     command.DocumentModes(),
			Aliases:   []string{"w!"},
			Signature: kit.Sig(),
		},
		{
			Name:      actWriteAll,
			DocString: "Write changes from all buffers to disk",
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				if errs := e.SaveAll(false); len(errs) > 0 {
					return command.Result{
						Message: "error: " + errs[0].Error(),
					}
				}
				return command.Result{
					Message: i18n.Text(i18n.StatusAllWritten),
				}
			},
			Modes:     command.DocumentModes(),
			Aliases:   []string{"wa"},
			Signature: kit.Sig(),
		},
		{
			Name: actWriteAllForce,
			DocString: "Forcefully write changes from all buffers to disk " +
				"creating necessary subdirectories",
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				for _, doc := range e.AllDocuments() {
					_ = doc.Save(e.Options(), true)
				}
				return command.Result{
					Message: i18n.Text(i18n.StatusAllWritten),
				}
			},
			Modes:     command.DocumentModes(),
			Aliases:   []string{"wa!"},
			Signature: kit.Sig(),
		},
		{
			Name: actWriteQuit,
			DocString: "Write changes to disk and close the current " +
				"view. Accepts an optional path (:wq some/path.txt)",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				setPathFromArgs(e, args)
				autoFormat(e)
				if err := e.Save(false); err != nil {
					return command.Result{Message: "error: " + err.Error()}
				}
				return command.Result{Signal: command.SignalQuit}
			},
			Modes:     command.DocumentModes(),
			Aliases:   []string{"wq", "exit", "x", "xit"},
			Signature: kit.FileSig(kit.Sig()),
		},
		{
			Name: actWriteQuitForce,
			DocString: "Write changes to disk and close the current view " +
				"forcefully. Accepts an optional path (:wq! some/path.txt)",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				setPathFromArgs(e, args)
				autoFormat(e)
				_ = e.Save(true)
				return command.Result{Signal: command.SignalQuit}
			},
			Modes:     command.DocumentModes(),
			Aliases:   []string{"wq!", "exit!", "x!", "xit!"},
			Signature: kit.FileSig(kit.Sig()),
		},
		{
			Name: actWriteQuitAll,
			DocString: "Write changes from all buffers to disk and close " +
				"all views",
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				if errs := e.SaveAll(false); len(errs) > 0 {
					return command.Result{
						Message: "error: " + errs[0].Error(),
					}
				}
				return command.Result{Signal: command.SignalQuit}
			},
			Modes:     command.DocumentModes(),
			Aliases:   []string{"wqa", "xa"},
			Signature: kit.Sig(),
		},
		{
			Name: actWriteQuitAllForce,
			DocString: "Forcefully write changes from all buffers to " +
				"disk, creating necessary subdirectories, and close all " +
				"views (ignoring unsaved changes)",
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				for _, doc := range e.AllDocuments() {
					_ = doc.Save(e.Options(), true)
				}
				return command.Result{Signal: command.SignalQuit}
			},
			Modes:     command.DocumentModes(),
			Aliases:   []string{"wqa!", "xa!"},
			Signature: kit.Sig(),
		},
		{
			Name: actWriteBufferClose,
			DocString: "Write changes to disk and closes the buffer. " +
				"Accepts an optional path (:write-buffer-close " +
				"some/path.txt)",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				setPathFromArgs(e, args)
				autoFormat(e)
				if err := e.Save(false); err != nil {
					return command.Result{Message: "error: " + err.Error()}
				}
				e.CloseCurrentView()
				return command.Result{
					Message: i18n.Text(i18n.StatusWrittenAndClosed),
				}
			},
			Modes:     command.DocumentModes(),
			Aliases:   []string{"wbc"},
			Signature: kit.Sig(),
		},
		{
			Name: actWriteBufferCloseForce,
			DocString: "Force write changes to disk creating necessary " +
				"subdirectories and closes the buffer. Accepts an " +
				"optional path (:write-buffer-close! some/path.txt)",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				setPathFromArgs(e, args)
				autoFormat(e)
				_ = e.Save(true)
				e.CloseCurrentView()
				return command.Result{
					Message: i18n.Text(i18n.StatusWrittenAndClosed),
				}
			},
			Modes:     command.DocumentModes(),
			Aliases:   []string{"wbc!"},
			Signature: kit.Sig(),
		},
	}
}

func fileManageCmds() []command.Command {
	Cw := kit.Prefixed(kit.Ctrl('w'))
	Spc := kit.Prefixed(kit.Char(' '))
	Spcw := kit.Prefixed(Spc(kit.Char('w')))
	return []command.Command{
		{
			Name:      actUpdate,
			DocString: "Write changes only if the file has been modified",
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				doc, ok := e.FocusedDocument()
				if !ok || !doc.Modified() {
					return command.Result{Message: "no changes to write"}
				}
				autoFormat(e)
				if err := e.Save(false); err != nil {
					return command.Result{Message: "error: " + err.Error()}
				}
				return command.Result{
					Message: "'" + doc.RelativeName(e.Cwd()) + "' written",
				}
			},
			Modes:     command.DocumentModes(),
			Aliases:   []string{"u"},
			Signature: kit.Sig(),
		},
		{
			Name:      actOpen,
			DocString: "Open a file from disk into the current view",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				if args == nil || args.Empty() {
					return command.Result{
						Message: i18n.Text(i18n.ErrorNoFilename),
					}
				}
				path, _ := args.First()
				_, _, err := ui.OpenPath(e, path, ui.PickerAcceptReplace)
				if err != nil {
					return command.Result{Message: "error: " + err.Error()}
				}
				if doc, ok := e.FocusedDocument(); ok {
					return command.Result{
						Message: "'" + doc.RelativeName(e.Cwd()) +
							"' opened",
					}
				}
				return command.Result{Message: "opened"}
			},
			Modes:     command.DocumentModes(),
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
			Modes: []string{"NOR", "SEL", "IMG"},
			Keys: map[string][]command.KeyBinding{"*": {
				{Cw(kit.Char('n'))},
				{Spcw(kit.Char('n'))},
			}},
			Aliases:   []string{"n"},
			Signature: kit.Sig(),
		},
		{
			Name:      actReload,
			DocString: "Discard changes and reload from the source file",
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				if err := e.Reload(); err != nil {
					return command.Result{Message: "error: " + err.Error()}
				}
				if doc, ok := e.FocusedDocument(); ok {
					return command.Result{
						Message: "'" + doc.RelativeName(e.Cwd()) +
							"' reloaded",
					}
				}
				return command.Result{Message: "reloaded"}
			},
			Modes:     command.DocumentModes(),
			Aliases:   []string{"rl"},
			Signature: kit.Sig(),
		},
		{
			Name: actReloadAll,
			DocString: "Discard changes and reload all documents from " +
				"the source files",
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				if errs := e.ReloadAll(); len(errs) > 0 {
					return command.Result{
						Message: "error: " + errs[0].Error(),
					}
				}
				return command.Result{Message: "all documents reloaded"}
			},
			Modes:     command.DocumentModes(),
			Aliases:   []string{"rla"},
			Signature: kit.Sig(),
		},
		{
			Name: actMove,
			DocString: "Move the current buffer and its corresponding " +
				"file to a different path",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				if args == nil || args.Empty() {
					return command.Result{
						Message: i18n.Text(i18n.ErrorNoFilename),
					}
				}
				doc, ok := e.FocusedDocument()
				if !ok {
					return command.Result{
						Message: i18n.Text(i18n.ErrorNoDocument),
					}
				}
				if doc.Modified() {
					return command.Result{
						Message: "error: unsaved changes (use move! " +
							"to force)",
					}
				}
				path, _ := args.First()
				if err := e.MoveFocusedFile(path, false); err != nil {
					return command.Result{Message: "error: " + err.Error()}
				}
				return command.Result{Message: "moved to '" + path + "'"}
			},
			Modes:     command.DocumentModes(),
			Aliases:   []string{"mv"},
			Signature: kit.FileSig(kit.MinArgs(1)),
		},
		{
			Name: actMoveForce,
			DocString: "Move the current buffer and its corresponding " +
				"file to a different path creating necessary " +
				"subdirectories",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				if args == nil || args.Empty() {
					return command.Result{
						Message: i18n.Text(i18n.ErrorNoFilename),
					}
				}
				_, ok := e.FocusedDocument()
				if !ok {
					return command.Result{
						Message: i18n.Text(i18n.ErrorNoDocument),
					}
				}
				path, _ := args.First()
				_ = e.MoveFocusedFile(path, true)
				return command.Result{Message: "moved to '" + path + "'"}
			},
			Modes:     command.DocumentModes(),
			Aliases:   []string{"mv!"},
			Signature: kit.FileSig(kit.MinArgs(1)),
		},
		{
			Name:      actRead,
			DocString: "Load a file into buffer",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				if args == nil || args.Empty() {
					return command.Result{
						Message: i18n.Text(i18n.ErrorNoFilename),
					}
				}
				path, _ := args.First()
				if err := action.ReadFile(e, path); err != nil {
					return command.Result{Message: "error: " + err.Error()}
				}
				return command.Result{Message: "'" + path + "' inserted"}
			},
			Modes:     command.DocumentModes(),
			Aliases:   []string{"r"},
			Signature: kit.FileSig(kit.MinArgs(1)),
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
	if doc, ok := e.FocusedDocument(); ok {
		doc.SetPath(path)
	}
}
