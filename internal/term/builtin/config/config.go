package config

import (
	"cmp"
	"os"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/term/builtin/kit"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

type uiSection struct {
	Theme  string `toml:"theme"`
	Editor struct {
		Mouse             *bool            `toml:"mouse"`
		MiddleClickPaste  *bool            `toml:"middle-click-paste"`
		NerdFonts         *bool            `toml:"nerd-fonts"`
		Insecure          *bool            `toml:"insecure"`
		EditorConfig      *bool            `toml:"editor-config"`
		AutoSession       *bool            `toml:"auto-session"`
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

var (
	errUsageGet      = i18n.NewError(i18n.ErrorUsageGet)
	errUsageSet      = i18n.NewError(i18n.ErrorUsageSet)
	errUsageToggle   = i18n.NewError(i18n.ErrorUsageToggle)
	errUnknownOption = i18n.NewError(i18n.ErrorUnknownOptionKey)
	errInvalidOption = i18n.NewError(i18n.ErrorInvalidOptionKey)
)

func terminalTrueColor() bool {
	ct := os.Getenv("COLORTERM")
	return ct == "truecolor" || ct == "24bit" ||
		os.Getenv("WSL_DISTRO_NAME") != ""
}

// ConfigurationModule returns the option and config commands
func ConfigurationModule(r *command.Registry) command.Module {
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
				Complete: command.StaticCompleter(loader.ThemeNames()...),
			},
			kit.EditorBoolOption("mouse",
				func(e *view.Editor) bool {
					return e.Options().Mouse
				},
				func(e *view.Editor, v bool) {
					e.Options().Mouse = v
				},
			),
			kit.EditorBoolOption("middle-click-paste",
				func(e *view.Editor) bool {
					return e.Options().MiddleClickPaste
				},
				func(e *view.Editor, v bool) {
					e.Options().MiddleClickPaste = v
				},
			),
			kit.EditorBoolOption("nerd-fonts",
				func(e *view.Editor) bool {
					return e.Options().NerdFonts
				},
				func(e *view.Editor, v bool) {
					e.Options().NerdFonts = v
				},
			),
			kit.EditorBoolOption("insecure",
				func(e *view.Editor) bool {
					return e.Options().Insecure
				},
				func(e *view.Editor, v bool) {
					e.Options().Insecure = v
				},
			),
			kit.EditorBoolOption("editor-config",
				func(e *view.Editor) bool {
					return e.Options().EditorConfig
				},
				func(e *view.Editor, v bool) {
					e.Options().EditorConfig = v
				},
			),
			kit.EditorBoolOption("auto-session",
				func(e *view.Editor) bool {
					return e.Options().AutoSession
				},
				func(e *view.Editor, v bool) {
					e.Options().AutoSession = v
				},
			),
			{
				Key: "default-line-ending",
				Get: func(e *view.Editor) (string, error) {
					switch e.Options().DefaultLineEnding {
					case core.LineEndingLF:
						return core.LineEndingNameLF, nil
					case core.LineEndingCRLF:
						return core.LineEndingNameCRLF, nil
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
				Complete: command.StaticCompleter(core.LineEndingNames()...),
			},
			cursorShapeOption("cursor-shape.normal", "NOR",
				func(o *view.Options, v view.CursorKind) {
					o.CursorShape.Normal = v
				},
			),
			cursorShapeOption("cursor-shape.select", "SEL",
				func(o *view.Options, v view.CursorKind) {
					o.CursorShape.Select = v
				},
			),
			cursorShapeOption("cursor-shape.insert", "INS",
				func(o *view.Options, v view.CursorKind) {
					o.CursorShape.Insert = v
				},
			),
			{
				Key: "statusline.separator",
				Get: func(e *view.Editor) (string, error) {
					return e.Options().StatusLineSeparator(), nil
				},
				Set: func(e *view.Editor, s string) error {
					e.Options().StatusLine.Separator = s
					return nil
				},
			},
			statuslineModeOption("statusline.mode.normal", "normal",
				func(o *view.Options, s string) {
					o.StatusLine.Mode.Normal = s
				},
			),
			statuslineModeOption("statusline.mode.insert", "insert",
				func(o *view.Options, s string) {
					o.StatusLine.Mode.Insert = s
				},
			),
			statuslineModeOption("statusline.mode.select", "select",
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
				opts.Mouse = kit.BoolOr(cfg.Editor.Mouse, true)
				opts.MiddleClickPaste = kit.BoolOr(
					cfg.Editor.MiddleClickPaste, true,
				)
				opts.NerdFonts = kit.BoolOr(cfg.Editor.NerdFonts, true)
				opts.Insecure = kit.BoolOr(cfg.Editor.Insecure, false)
				opts.EditorConfig = kit.BoolOr(cfg.Editor.EditorConfig, true)
				opts.AutoSession = kit.BoolOr(
					cfg.Editor.AutoSession, true,
				)
				opts.DefaultLineEnding = cfg.Editor.DefaultLineEnding
				opts.CursorShape = view.CursorShape{
					Normal: cmp.Or(
						cfg.Editor.CursorShape.Normal, view.CursorKindBlock,
					),
					Insert: cmp.Or(
						cfg.Editor.CursorShape.Insert, view.CursorKindBar,
					),
					Select: cmp.Or(
						cfg.Editor.CursorShape.Select, view.CursorKindUnderline,
					),
				}
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
					return command.Result{Error: errUsageGet}
				}
				key, _ := args.First()
				o, ok := r.LookupOption(key)
				if !ok {
					return command.Result{
						Error: errUnknownOption.WithVars(i18n.Vars{
							"key": key,
						}),
					}
				}
				value, err := o.Get(e)
				if err != nil {
					return command.Result{Error: err}
				}
				return command.Result{Message: value}
			},
			Modes:   command.PaneModes(),
			Aliases: []string{"get"},
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
					return command.Result{Error: errUsageSet}
				}
				key, _ := args.Get(0)
				val, _ := args.Get(1)
				o, ok := r.LookupOption(key)
				if !ok {
					return command.Result{
						Error: errUnknownOption.WithVars(i18n.Vars{
							"key": key,
						}),
					}
				}
				if err := o.Set(e, val); err != nil {
					return command.Result{Error: err}
				}
				return command.Result{}
			},
			Modes:   command.PaneModes(),
			Aliases: []string{"set"},
			Signature: command.Signature{
				Positionals: command.Positionals{Min: 2, Max: 2},
				RawAfter:    1,
				Completer: command.Completer{
					Positionals: []command.CompletionFunc{
						r.OptionCompleter(),
					},
					Raw: r.OptionValueCompleter(),
				},
			},
		},
		{
			Name:      actToggleOption,
			DocString: "Toggle a config option at runtime",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				if args == nil || args.Empty() {
					return command.Result{Error: errUsageToggle}
				}
				key, _ := args.First()
				o, ok := r.LookupOption(key)
				if !ok || o.Toggle == nil {
					return command.Result{
						Error: errInvalidOption.WithVars(i18n.Vars{
							"key": key,
						}),
					}
				}
				value, err := o.Toggle(e)
				if err != nil {
					return command.Result{Error: err}
				}
				return command.Result{
					Message: "'" + key + "' is now set to " + value,
				}
			},
			Modes:   command.PaneModes(),
			Aliases: []string{"toggle"},
			Signature: command.Signature{
				Positionals: command.Positionals{Min: 1, Max: 1},
				Completer: command.PositionalCompleter(
					r.BoolOptionCompleter(),
				),
			},
		},
	}
}
