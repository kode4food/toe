package defaults

import (
	"fmt"
	"os"
	"strconv"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
	"github.com/kode4food/toe/internal/view/config"
	"github.com/kode4food/toe/internal/view/language"
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
	return command.Module{
		Commands: map[string]command.Command{
			actGetOption: {
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
			actSetOption: {
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
			actToggleOption: {
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
			actConfigOpen: {
				DocString: "Open the user config.toml file",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					return openFromPath(
						e, config.UserConfigPath, "config path unavailable",
					)
				},
				Aliases:   []string{"config-open"},
				Signature: sig(),
			},
			actConfigOpenWorkspace: {
				DocString: "Open the workspace config.toml file",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					path := config.WorkspaceConfigPath(e.Cwd())
					if _, err := e.SwitchFile(path); err != nil {
						return command.Result{Message: "error: " + err.Error()}
					}
					return command.Result{}
				},
				Aliases:   []string{"config-open-workspace"},
				Signature: sig(),
			},
			actConfigReload: {
				DocString: "Refresh user config",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					if err := e.ReloadConfig(); err != nil {
						return command.Result{Message: "error: " + err.Error()}
					}
					return command.Result{Message: "config reloaded"}
				},
				Aliases:   []string{"config-reload"},
				Signature: sig(),
			},
			actLogOpen: {
				DocString: "Open the editor log file",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					return openFromPath(
						e, config.LogFilePath, "log path unavailable",
					)
				},
				Aliases:   []string{"log-open"},
				Signature: sig(),
			},
			actWorkspaceTrust: {
				DocString: "Add current workspace to the list of trusted " +
					"workspaces",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					if err := config.TrustWorkspace(e.Cwd()); err != nil {
						return command.Result{Message: "error: " + err.Error()}
					}
					return command.Result{Message: "workspace trusted"}
				},
				Aliases:   []string{"workspace-trust"},
				Signature: sig(),
			},
			actWorkspaceUntrust: {
				DocString: "Remove current workspace from the list of " +
					"trusted workspaces",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					if err := config.UntrustWorkspace(e.Cwd()); err != nil {
						return command.Result{Message: "error: " + err.Error()}
					}
					return command.Result{Message: "workspace untrusted"}
				},
				Aliases:   []string{"workspace-untrust"},
				Signature: sig(),
			},
			actTheme: {
				DocString: "Change the editor theme " +
					"(show current theme if no name specified)",
				Run: func(e *view.Editor, args *command.Args) command.Result {
					if args == nil || args.Empty() {
						name := e.Options().Theme
						if _, _, err := theme.Load(name); err != nil {
							th, _, _ := theme.Default()
							name = th.Name()
						}
						return command.Result{Message: name}
					}
					name, _ := args.First()
					if name == "default" {
						name = "mocha"
					}
					th, _, err := theme.Load(name)
					if err != nil {
						return command.Result{
							Message: "error: could not load theme: " +
								err.Error(),
						}
					}
					if !(terminalTrueColor() || th.Is16Color()) {
						return command.Result{
							Message: "error: theme requires true color support",
						}
					}
					e.Options().Theme = name
					return command.Result{}
				},
				Signature: staticSig(optionalArg(), loader.ThemeNames()...),
			},
			actSetLanguage: {
				DocString: "Set the language of current buffer " +
					"(show current language if no value specified)",
				Run: func(e *view.Editor, args *command.Args) command.Result {
					doc, ok := e.FocusedDocument()
					if !ok {
						return command.Result{Message: "error: no document"}
					}
					if args == nil || args.Empty() {
						lang := doc.Lang()
						if lang == "" {
							lang = "text"
						}
						return command.Result{Message: lang}
					}
					lang, _ := args.First()
					if lang == "text" {
						lang = ""
					}
					doc.SetLang(lang)
					return command.Result{Message: ""}
				},
				Aliases:   []string{"set-language", "lang"},
				Signature: staticSig(optionalArg(), languageNames()...),
			},
			actSetLineEnding: {
				DocString: "Set the document's default line ending. " +
					"Options: crlf, lf",
				Run: func(e *view.Editor, args *command.Args) command.Result {
					if args == nil || args.Empty() {
						doc, ok := e.FocusedDocument()
						if !ok {
							return command.Result{Message: "error: no document"}
						}
						switch doc.LineEnding() {
						case core.LineEndingCRLF:
							return command.Result{Message: "crlf"}
						default:
							return command.Result{Message: "line feed"}
						}
					}
					name, _ := args.First()
					var le core.LineEnding
					switch name {
					case "lf":
						le = core.LineEndingLF
					case "crlf":
						le = core.LineEndingCRLF
					default:
						return command.Result{
							Message: "error: unknown line ending: " + name,
						}
					}
					if err := action.SetLineEnding(e, le); err != nil {
						return command.Result{Message: "error: " + err.Error()}
					}
					return command.Result{Message: ""}
				},
				Aliases:   []string{"line-ending"},
				Signature: staticSig(optionalArg(), "crlf", "lf"),
			},
			actIndentStyle: {
				DocString: "Set the indentation style for editing. " +
					"('t' for tabs or 1-16 for number of spaces)",
				Run: func(e *view.Editor, args *command.Args) command.Result {
					doc, ok := e.FocusedDocument()
					if !ok {
						return command.Result{Message: "error: no document"}
					}
					if args == nil || args.Empty() {
						return command.Result{
							Message: doc.IndentStyle().AsStr(),
						}
					}
					arg, _ := args.First()
					switch arg {
					case "tabs", "tab", "t":
						doc.SetIndentStyle(core.Tabs())
					default:
						n, err := strconv.Atoi(arg)
						if err != nil || n < 1 || n > core.MaxIndent {
							return command.Result{
								Message: "error: expected 'tab' or spaces " +
									"count (1-16)",
							}
						}
						doc.SetIndentStyle(core.Spaces(uint8(n)))
					}
					return command.Result{Message: "indent style set"}
				},
				Aliases: []string{"indent-style"},
				Signature: staticSig(
					sig(),
					"tabs", "tab", "t", "1", "2", "3", "4", "5", "6", "7", "8",
					"9", "10", "11", "12", "13", "14", "15", "16",
				),
			},
			actEncoding: {
				DocString: "Set encoding",
				Run: func(_ *view.Editor, _ *command.Args) command.Result {
					return command.Result{Message: "utf-8"}
				},
				Signature: sig(),
			},
		},
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
				opts.Theme = stringOr(cfg.Theme, view.DefaultTheme)
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

func cursorShapeOption(
	key, mode string, set func(*view.Options, view.CursorKind),
) command.Option {
	return command.Option{
		Key: key,
		Get: func(e *view.Editor) (string, error) {
			return string(e.Options().CursorShapeForMode(mode)), nil
		},
		Set: func(e *view.Editor, s string) error {
			v, err := view.ParseCursorKind(s)
			if err != nil {
				return fmt.Errorf("%w: %s", config.ErrInvalidOption, s)
			}
			set(e.Options(), v)
			return nil
		},
	}
}

func statuslineModeOption(
	key, mode string, set func(*view.Options, string),
) command.Option {
	return command.Option{
		Key: key,
		Get: func(e *view.Editor) (string, error) {
			return e.Options().ModeNameForMode(mode), nil
		},
		Set: func(e *view.Editor, s string) error {
			set(e.Options(), s)
			return nil
		},
	}
}

func languageNames() []string {
	langs, ok := language.LoadBundledLanguages()
	if !ok {
		return []string{"text"}
	}
	names := make([]string, 0, len(langs.Languages)+1)
	for _, l := range langs.Languages {
		if l.Name != "" {
			names = append(names, l.Name)
		}
	}
	names = append(names, "text")
	return names
}

func openFromPath(
	e *view.Editor, pathFn func() (string, bool), unavailMsg string,
) command.Result {
	path, ok := pathFn()
	if !ok {
		return command.Result{Message: "error: " + unavailMsg}
	}
	if _, err := e.SwitchFile(path); err != nil {
		return command.Result{Message: "error: " + err.Error()}
	}
	return command.Result{}
}
