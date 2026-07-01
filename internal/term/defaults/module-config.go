package defaults

import (
	"cmp"
	"os"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/config"
)

type uiSection struct {
	Theme  string `toml:"theme"`
	Editor struct {
		Mouse             *bool            `toml:"mouse"`
		MiddleClickPaste  *bool            `toml:"middle-click-paste"`
		Insecure          *bool            `toml:"insecure"`
		EditorConfig      *bool            `toml:"editor-config"`
		DefaultLineEnding core.LineEnding  `toml:"default-line-ending"`
		CursorShape       view.CursorShape `toml:"cursor-shape"`
		StatusLine        view.StatusLine  `toml:"statusline"`
	} `toml:"editor"`
}

const (
	actGetOption           = "get_option"
	actSetOption           = "set_option"
	actToggleOption        = "toggle_option"
	actConfigOpen          = "config_open"
	actConfigOpenWorkspace = "config_open_workspace"
	actConfigReload        = "config_reload"
	actLogOpen             = "log_open"
	actWorkspaceTrust      = "workspace_trust"
	actWorkspaceUntrust    = "workspace_untrust"
	actTheme               = "theme"
	actSetLanguage         = "set_language"
	actSetLineEnding       = "set_line_ending"
	actIndentStyle         = "indent_style"
	actEncoding            = "encoding"
)

func terminalTrueColor() bool {
	ct := os.Getenv("COLORTERM")
	return ct == "truecolor" || ct == "24bit" ||
		os.Getenv("WSL_DISTRO_NAME") != ""
}

func configModule(r *command.Registry) command.Module {
	cfg := new(uiSection)
	cmds := configOptionCmds(r)
	cmds = append(cmds, configSystemCmds()...)
	cmds = append(cmds, configThemeCmds()...)
	cmds = append(cmds, configFormatCmds()...)
	return command.Module{
		Commands: cmds,
		Options: []command.Option{
			{
				Key: "theme",
				Get: func(e *view.Editor) (string, error) {
					return e.Options().Theme, nil
				},
				Set: func(e *view.Editor, s string) error {
					e.Options().Theme = s
					return nil
				},
			},
			editorBoolOption("editor.mouse",
				func(e *view.Editor) bool {
					return e.Options().Mouse
				},
				func(e *view.Editor, v bool) {
					e.Options().Mouse = v
				},
			),
			editorBoolOption("editor.middle-click-paste",
				func(e *view.Editor) bool {
					return e.Options().MiddleClickPaste
				},
				func(e *view.Editor, v bool) {
					e.Options().MiddleClickPaste = v
				},
			),
			editorBoolOption("editor.insecure",
				func(e *view.Editor) bool {
					return e.Options().Insecure
				},
				func(e *view.Editor, v bool) {
					e.Options().Insecure = v
				},
			),
			editorBoolOption("editor.editor-config",
				func(e *view.Editor) bool {
					return e.Options().EditorConfig
				},
				func(e *view.Editor, v bool) {
					e.Options().EditorConfig = v
				},
			),
			{
				Key: "editor.default-line-ending",
				Get: func(e *view.Editor) (string, error) {
					switch e.Options().DefaultLineEnding {
					case core.LineEndingLF:
						return "lf", nil
					case core.LineEndingCRLF:
						return "crlf", nil
					default:
						return "", nil
					}
				},
				Set: func(e *view.Editor, s string) error {
					var le core.LineEnding
					if err := le.UnmarshalText([]byte(s)); err != nil {
						return err
					}
					e.Options().DefaultLineEnding = le
					return nil
				},
			},
			cursorShapeOption("editor.cursor-shape.normal", "NOR",
				func(o *view.Options, v view.CursorKind) {
					o.CursorShape.Normal = v
				},
			),
			cursorShapeOption("editor.cursor-shape.select", "SEL",
				func(o *view.Options, v view.CursorKind) {
					o.CursorShape.Select = v
				},
			),
			cursorShapeOption("editor.cursor-shape.insert", "INS",
				func(o *view.Options, v view.CursorKind) {
					o.CursorShape.Insert = v
				},
			),
			{
				Key: "editor.statusline.separator",
				Get: func(e *view.Editor) (string, error) {
					return e.Options().StatusLineSeparator(), nil
				},
				Set: func(e *view.Editor, s string) error {
					e.Options().StatusLine.Separator = s
					return nil
				},
			},
			statuslineModeOption("editor.statusline.mode.normal", "normal",
				func(o *view.Options, s string) {
					o.StatusLine.Mode.Normal = s
				},
			),
			statuslineModeOption("editor.statusline.mode.insert", "insert",
				func(o *view.Options, s string) {
					o.StatusLine.Mode.Insert = s
				},
			),
			statuslineModeOption("editor.statusline.mode.select", "select",
				func(o *view.Options, s string) {
					o.StatusLine.Mode.Select = s
				},
			),
		},
		Section: &command.Section{
			Config: cfg,
			Reset:  func() { *cfg = uiSection{} },
			Apply: func(e *view.Editor) {
				opts := e.Options()
				opts.Theme = cmp.Or(cfg.Theme, view.DefaultTheme)
				opts.Mouse = boolOr(cfg.Editor.Mouse, true)
				opts.MiddleClickPaste = boolOr(
					cfg.Editor.MiddleClickPaste, true,
				)
				opts.Insecure = boolOr(cfg.Editor.Insecure, false)
				opts.EditorConfig = boolOr(cfg.Editor.EditorConfig, true)
				opts.DefaultLineEnding = cfg.Editor.DefaultLineEnding
				opts.CursorShape = cfg.Editor.CursorShape
				opts.StatusLine = cfg.Editor.StatusLine
			},
		},
	}
}

func configOptionCmds(r *command.Registry) []command.Command {
	return []command.Command{
		{
			Name:      actGetOption,
			DocString: "Get the current value of a config option",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				if args == nil || args.Empty() {
					return command.Result{
						Message: "error: usage: get <key>",
					}
				}
				key, _ := args.First()
				o, ok := r.LookupOption(key)
				if !ok {
					return command.Result{
						Message: "error: " +
							config.ErrUnknownOption.Error() + ": " + key,
					}
				}
				value, err := o.Get(e)
				if err != nil {
					return command.Result{Message: "error: " + err.Error()}
				}
				return command.Result{Message: value}
			},
			Aliases: []string{"get-option", "get"},
			Signature: command.Signature{
				Positionals: command.Positionals{Min: 1, Max: 1},
				Completer: command.PositionalCompleter(
					r.OptionCompleter(),
				),
			},
		},
		{
			Name:      actSetOption,
			DocString: "Set a config option at runtime",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				if args == nil || args.Len() < 2 {
					return command.Result{
						Message: "error: usage: set <key> <value>",
					}
				}
				key, _ := args.Get(0)
				val, _ := args.Get(1)
				o, ok := r.LookupOption(key)
				if !ok {
					return command.Result{
						Message: "error: " +
							config.ErrUnknownOption.Error() + ": " + key,
					}
				}
				if err := o.Set(e, val); err != nil {
					return command.Result{Message: "error: " + err.Error()}
				}
				return command.Result{}
			},
			Aliases: []string{"set-option", "set"},
			Signature: command.Signature{
				Positionals: command.Positionals{Min: 2, Max: 2},
				RawAfter:    1,
				Completer: command.PositionalCompleter(
					r.OptionCompleter(),
				),
			},
		},
		{
			Name:      actToggleOption,
			DocString: "Toggle a config option at runtime",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				if args == nil || args.Empty() {
					return command.Result{
						Message: "error: usage: toggle <key>",
					}
				}
				key, _ := args.First()
				o, ok := r.LookupOption(key)
				if !ok || o.Toggle == nil {
					return command.Result{
						Message: "error: " +
							config.ErrInvalidOption.Error() + ": " + key,
					}
				}
				value, err := o.Toggle(e)
				if err != nil {
					return command.Result{Message: "error: " + err.Error()}
				}
				return command.Result{
					Message: "'" + key + "' is now set to " + value,
				}
			},
			Aliases: []string{"toggle-option", "toggle"},
			Signature: command.Signature{
				Positionals: command.Positionals{Min: 1, Max: 1},
				Completer: command.PositionalCompleter(
					r.BoolOptionCompleter(),
				),
			},
		},
	}
}
