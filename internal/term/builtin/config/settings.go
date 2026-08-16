package config

import (
	"cmp"
	"embed"
	"slices"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/term/builtin/kit"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/config"
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
		FileWatch         *bool            `toml:"file-watch"`
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
)

const (
	errorUsageGetKey      i18n.Key = "error.usageGet"
	errorUsageSetKey      i18n.Key = "error.usageSet"
	errorUsageToggleKey   i18n.Key = "error.usageToggle"
	errorUnknownOptionKey i18n.Key = "error.unknownOptionKey"
	errorInvalidOptionKey i18n.Key = "error.invalidOptionKey"
)

var (
	errUsageGet      = i18n.NewError(errorUsageGetKey)
	errUsageSet      = i18n.NewError(errorUsageSetKey)
	errUsageToggle   = i18n.NewError(errorUsageToggleKey)
	errUnknownOption = i18n.NewError(errorUnknownOptionKey)
	errInvalidOption = i18n.NewError(errorInvalidOptionKey)
)

//go:embed i18n/settings.*.json
var settingsFS embed.FS

// SettingsModule returns the option and config commands
func SettingsModule(r *command.Registry) command.Module {
	cfg := new(uiSection)
	cmds := optionCmds(r)
	cmds = append(cmds, systemCmds()...)
	cmds = append(cmds, themeCmds(r)...)
	cmds = append(cmds, formatCmds()...)
	return command.Module{
		Translations: i18n.LoadTranslations(settingsFS),
		Commands:     cmds,
		Options: []command.Option{
			{
				Key:       "theme",
				DocString: "Color theme applied to the editor",
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
			).WithDoc("Enable mouse mode"),
			kit.EditorBoolOption("middle-click-paste",
				func(e *view.Editor) bool {
					return e.Options().MiddleClickPaste
				},
				func(e *view.Editor, v bool) {
					e.Options().MiddleClickPaste = v
				},
			).WithDoc("Middle click paste support"),
			kit.EditorBoolOption("nerd-fonts",
				func(e *view.Editor) bool {
					return e.Options().NerdFonts
				},
				func(e *view.Editor, v bool) {
					e.Options().NerdFonts = v
				},
			).WithDoc("Use Nerd Font glyphs for icons"),
			kit.EditorBoolOption("insecure",
				func(e *view.Editor) bool {
					return e.Options().Insecure
				},
				func(e *view.Editor, v bool) {
					e.Options().Insecure = v
				},
			).WithDoc("Disable workspace trust checks"),
			kit.EditorBoolOption("editor-config",
				func(e *view.Editor) bool {
					return e.Options().EditorConfig
				},
				func(e *view.Editor, v bool) {
					e.Options().EditorConfig = v
				},
			).WithDoc("Read settings from EditorConfig files"),
			kit.EditorBoolOption("auto-session",
				func(e *view.Editor) bool {
					return e.Options().AutoSession
				},
				func(e *view.Editor, v bool) {
					e.Options().AutoSession = v
				},
			).WithDoc("Save and restore sessions automatically"),
			kit.EditorBoolOption("file-watch",
				func(e *view.Editor) bool {
					return e.Options().FileWatch
				},
				func(e *view.Editor, v bool) {
					e.Options().FileWatch = v
				},
			).WithDoc("Detect external file changes"),
			{
				Key:       "default-line-ending",
				DocString: "Line ending for new documents",
				Get: func(e *view.Editor) (string, error) {
					switch e.Options().DefaultLineEnding {
					case core.LineEndingLF:
						return core.LineEndingNameLF, nil
					case core.LineEndingCRLF:
						return core.LineEndingNameCRLF, nil
					default:
						return core.LineEndingNameNative, nil
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
			cursorShapeOption("cursor-shape.normal", view.ModeNormal,
				func(o *view.Options, v view.CursorKind) {
					o.CursorShape.Normal = v
				},
			).WithDoc("Cursor shape in normal mode"),
			cursorShapeOption("cursor-shape.select", view.ModeSelect,
				func(o *view.Options, v view.CursorKind) {
					o.CursorShape.Select = v
				},
			).WithDoc("Cursor shape in select mode"),
			cursorShapeOption("cursor-shape.insert", view.ModeInsert,
				func(o *view.Options, v view.CursorKind) {
					o.CursorShape.Insert = v
				},
			).WithDoc("Cursor shape in insert mode"),
			statuslineItemsOption("statusline.left",
				func(o *view.Options) []view.StatusLineItem {
					return o.StatusLineLeft()
				},
				func(o *view.Options, v []view.StatusLineItem) {
					o.StatusLine.Left = v
				},
			).WithDoc("Elements aligned left on the statusline"),
			statuslineItemsOption("statusline.right",
				func(o *view.Options) []view.StatusLineItem {
					return o.StatusLineRight()
				},
				func(o *view.Options, v []view.StatusLineItem) {
					o.StatusLine.Right = v
				},
			).WithDoc("Elements aligned right on the statusline"),
			{
				Key:       "statusline.separator",
				DocString: "Character separating statusline elements",
				Get: func(e *view.Editor) (string, error) {
					return e.Options().StatusLineSeparator(), nil
				},
				Set: func(e *view.Editor, s string) error {
					v, err := config.ParseStringLiteral(s)
					if err != nil {
						return err
					}
					e.Options().StatusLine.Separator = v
					return nil
				},
			},
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
				opts.FileWatch = kit.BoolOr(cfg.Editor.FileWatch, true)
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

func optionCmds(r *command.Registry) []command.Command {
	return []command.Command{
		{
			Name:      actGetOption,
			DocString: "Get the current value of a config option",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				if args == nil || args.Empty() {
					return command.Result{Error: errUsageGet}
				}
				key, _ := args.First()
				o := r.LookupOption(key)
				if o == nil {
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
			Modes:   command.PaneModes,
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
				o := r.LookupOption(key)
				if o == nil {
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
			Modes:   command.PaneModes,
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
				return toggleOption(e, r, args)
			},
			Modes:   command.PaneModes,
			Aliases: []string{"toggle"},
			Signature: command.Signature{
				Positionals: command.Positionals{Min: 1, Max: -1},
				Completer: command.PositionalCompleter(
					r.BoolOptionCompleter(), nil,
				),
			},
		},
	}
}

func toggleOption(
	e *view.Editor, r *command.Registry, args *command.Args,
) command.Result {
	if args == nil || args.Empty() {
		return command.Result{Error: errUsageToggle}
	}
	key, _ := args.First()
	o := r.LookupOption(key)
	if o == nil {
		return command.Result{
			Error: errInvalidOption.WithVars(i18n.Vars{"key": key}),
		}
	}
	if values := args.Positionals()[1:]; len(values) > 0 {
		return cycleOption(e, o, key, values)
	}
	if o.Toggle == nil {
		return command.Result{Error: errUsageToggle}
	}
	value, err := o.Toggle(e)
	if err != nil {
		return command.Result{Error: err}
	}
	return command.Result{Message: "'" + key + "' is now set to " + value}
}

func cycleOption(
	e *view.Editor, o *command.Option, key string, values []string,
) command.Result {
	if o.Get == nil || o.Set == nil {
		return command.Result{
			Error: errInvalidOption.WithVars(i18n.Vars{"key": key}),
		}
	}
	current, err := o.Get(e)
	if err != nil {
		return command.Result{Error: err}
	}
	next := values[0]
	if i := slices.Index(values, current); i >= 0 {
		next = values[(i+1)%len(values)]
	}
	if err := o.Set(e, next); err != nil {
		return command.Result{Error: err}
	}
	return command.Result{Message: "'" + key + "' is now set to " + next}
}
