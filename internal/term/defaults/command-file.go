package defaults

import (
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

const (
	actWrite            = "write"
	actWriteAll         = "write_all"
	actWriteQuit        = "write_quit"
	actWriteQuitAll     = "write_quit_all"
	actWriteBufferClose = "write_buffer_close"
	actUpdate           = "update"
	actOpen             = "open"
	actNew              = "new"
	actReload           = "reload"
	actReloadAll        = "reload_all"
	actMove             = "move"
	actRead             = "read"
)

func registerFileCommands(r *registry) {
	r.RegisterCommand(actWrite, command.Command{
		DocString: "Write changes to disk. " +
			"Accepts an optional path (:write some/path.txt)",
		Run: func(e *view.Editor, args *command.Args) command.Result {
			setPathFromArgs(e, args)
			if err := e.Save(); err != nil {
				return command.Result{Message: "error: " + err.Error()}
			}
			if doc, ok := e.FocusedDocument(); ok {
				return command.Result{
					Message: "'" + doc.RelativeName(e.Cwd()) + "' written",
				}
			}
			return command.Result{Message: "written"}
		},
		Aliases:   []string{"w"},
		Signature: sig(),
	})
	r.RegisterCommand("write!", command.Command{
		DocString: "Force write changes to disk creating necessary " +
			"subdirectories. Accepts an optional path (:write! some/path.txt)",
		Run: func(e *view.Editor, args *command.Args) command.Result {
			setPathFromArgs(e, args)
			_ = e.Save()
			if doc, ok := e.FocusedDocument(); ok {
				return command.Result{
					Message: "'" + doc.RelativeName(e.Cwd()) + "' written",
				}
			}
			return command.Result{Message: "written"}
		},
		Aliases:   []string{"w!"},
		Signature: sig(),
	})
	r.RegisterCommand(actWriteAll, command.Command{
		DocString: "Write changes from all buffers to disk",
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			if errs := e.SaveAll(); len(errs) > 0 {
				return command.Result{Message: "error: " + errs[0].Error()}
			}
			return command.Result{Message: "all documents written"}
		},
		Aliases:   []string{"write-all", "wa"},
		Signature: sig(),
	})
	r.RegisterCommand("write-all!", command.Command{
		DocString: "Forcefully write changes from all buffers to disk " +
			"creating necessary subdirectories",
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			for _, doc := range e.AllDocuments() {
				_ = doc.Save()
			}
			return command.Result{Message: "all documents written"}
		},
		Aliases:   []string{"wa!"},
		Signature: sig(),
	})
	r.RegisterCommand(actWriteQuit, command.Command{
		DocString: "Write changes to disk and close the current view. " +
			"Accepts an optional path (:wq some/path.txt)",
		Run: func(e *view.Editor, args *command.Args) command.Result {
			setPathFromArgs(e, args)
			if err := e.Save(); err != nil {
				return command.Result{Message: "error: " + err.Error()}
			}
			return command.Result{Signal: command.SignalQuit}
		},
		Aliases:   []string{"write-quit", "wq", "exit", "x", "xit"},
		Signature: fileSig(sig()),
	})
	r.RegisterCommand("write-quit!", command.Command{
		DocString: "Write changes to disk and close the current view " +
			"forcefully. Accepts an optional path (:wq! some/path.txt)",
		Run: func(e *view.Editor, args *command.Args) command.Result {
			setPathFromArgs(e, args)
			_ = e.Save()
			return command.Result{Signal: command.SignalQuit}
		},
		Aliases:   []string{"wq!", "exit!", "x!", "xit!"},
		Signature: fileSig(sig()),
	})
	r.RegisterCommand(actWriteQuitAll, command.Command{
		DocString: "Write changes from all buffers to disk and close all views",
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			if errs := e.SaveAll(); len(errs) > 0 {
				return command.Result{Message: "error: " + errs[0].Error()}
			}
			return command.Result{Signal: command.SignalQuit}
		},
		Aliases:   []string{"write-quit-all", "wqa", "xa"},
		Signature: sig(),
	})
	r.RegisterCommand("write-quit-all!", command.Command{
		DocString: "Forcefully write changes from all buffers to disk, " +
			"creating necessary subdirectories, and close all views " +
			"(ignoring unsaved changes)",
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			for _, doc := range e.AllDocuments() {
				_ = doc.Save()
			}
			return command.Result{Signal: command.SignalQuit}
		},
		Aliases:   []string{"wqa!", "xa!"},
		Signature: sig(),
	})
	r.RegisterCommand(actWriteBufferClose, command.Command{
		DocString: "Write changes to disk and closes the buffer. " +
			"Accepts an optional path (:write-buffer-close some/path.txt)",
		Run: func(e *view.Editor, args *command.Args) command.Result {
			setPathFromArgs(e, args)
			if err := e.Save(); err != nil {
				return command.Result{Message: "error: " + err.Error()}
			}
			e.CloseCurrentView()
			return command.Result{Message: "written and closed"}
		},
		Aliases:   []string{"write-buffer-close", "wbc"},
		Signature: sig(),
	})
	r.RegisterCommand("write-buffer-close!", command.Command{
		DocString: "Force write changes to disk creating necessary " +
			"subdirectories and closes the buffer. Accepts an optional " +
			"path (:write-buffer-close! some/path.txt)",
		Run: func(e *view.Editor, args *command.Args) command.Result {
			setPathFromArgs(e, args)
			_ = e.Save()
			e.CloseCurrentView()
			return command.Result{Message: "written and closed"}
		},
		Aliases:   []string{"wbc!"},
		Signature: sig(),
	})
	r.RegisterCommand(actUpdate, command.Command{
		DocString: "Write changes only if the file has been modified",
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			doc, ok := e.FocusedDocument()
			if !ok || !doc.Modified() {
				return command.Result{Message: "no changes to write"}
			}
			if err := e.Save(); err != nil {
				return command.Result{Message: "error: " + err.Error()}
			}
			return command.Result{
				Message: "'" + doc.RelativeName(e.Cwd()) + "' written",
			}
		},
		Aliases:   []string{"u"},
		Signature: sig(),
	})
	r.RegisterCommand(actOpen, command.Command{
		DocString: "Open a file from disk into the current view",
		Run: func(e *view.Editor, args *command.Args) command.Result {
			if args == nil || args.Empty() {
				return command.Result{Message: "error: no filename given"}
			}
			path, _ := args.First()
			if _, err := e.SwitchFile(path); err != nil {
				return command.Result{Message: "error: " + err.Error()}
			}
			if doc, ok := e.FocusedDocument(); ok {
				return command.Result{
					Message: "'" + doc.RelativeName(e.Cwd()) + "' opened",
				}
			}
			return command.Result{Message: "opened"}
		},
		Aliases:   []string{"o", "edit", "e"},
		Signature: fileSig(minArgs(1)),
	})
	r.RegisterCommand(actNew, command.Command{
		DocString: "Create a new scratch buffer",
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			e.NewDocument()
			return command.Result{Message: "[scratch]"}
		},
		Aliases:   []string{"n"},
		Signature: sig(),
	})
	r.RegisterCommand(actReload, command.Command{
		DocString: "Discard changes and reload from the source file",
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			if err := e.Reload(); err != nil {
				return command.Result{Message: "error: " + err.Error()}
			}
			if doc, ok := e.FocusedDocument(); ok {
				return command.Result{
					Message: "'" + doc.RelativeName(e.Cwd()) + "' reloaded",
				}
			}
			return command.Result{Message: "reloaded"}
		},
		Aliases:   []string{"rl"},
		Signature: sig(),
	})
	r.RegisterCommand(actReloadAll, command.Command{
		DocString: "Discard changes and reload all documents from the source files",
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			if errs := e.ReloadAll(); len(errs) > 0 {
				return command.Result{Message: "error: " + errs[0].Error()}
			}
			return command.Result{Message: "all documents reloaded"}
		},
		Aliases:   []string{"reload-all", "rla"},
		Signature: sig(),
	})
	r.RegisterCommand(actMove, command.Command{
		DocString: "Move the current buffer and its corresponding file " +
			"to a different path",
		Run: func(e *view.Editor, args *command.Args) command.Result {
			if args == nil || args.Empty() {
				return command.Result{Message: "error: no filename given"}
			}
			doc, ok := e.FocusedDocument()
			if !ok {
				return command.Result{Message: "error: no document"}
			}
			if doc.Modified() {
				return command.Result{
					Message: "error: unsaved changes (use move! to force)",
				}
			}
			path, _ := args.First()
			doc.SetPath(path)
			if err := e.Save(); err != nil {
				return command.Result{Message: "error: " + err.Error()}
			}
			return command.Result{Message: "moved to '" + path + "'"}
		},
		Aliases:   []string{"mv"},
		Signature: fileSig(minArgs(1)),
	})
	r.RegisterCommand("move!", command.Command{
		DocString: "Move the current buffer and its corresponding file " +
			"to a different path creating necessary subdirectories",
		Run: func(e *view.Editor, args *command.Args) command.Result {
			if args == nil || args.Empty() {
				return command.Result{Message: "error: no filename given"}
			}
			doc, ok := e.FocusedDocument()
			if !ok {
				return command.Result{Message: "error: no document"}
			}
			path, _ := args.First()
			doc.SetPath(path)
			_ = e.Save()
			return command.Result{Message: "moved to '" + path + "'"}
		},
		Aliases:   []string{"mv!"},
		Signature: fileSig(minArgs(1)),
	})
	r.RegisterCommand(actRead, command.Command{
		DocString: "Load a file into buffer",
		Run: func(e *view.Editor, args *command.Args) command.Result {
			if args == nil || args.Empty() {
				return command.Result{Message: "error: no filename given"}
			}
			path, _ := args.First()
			if err := action.ReadFile(e, path); err != nil {
				return command.Result{Message: "error: " + err.Error()}
			}
			return command.Result{Message: "'" + path + "' inserted"}
		},
		Aliases:   []string{"r"},
		Signature: fileSig(minArgs(1)),
	})
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
