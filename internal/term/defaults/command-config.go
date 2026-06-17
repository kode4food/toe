package defaults

import (
	"os"
	"strconv"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
	"github.com/kode4food/toe/internal/view/config"
)

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

func registerConfigCommands(r *registry) {
	r.RegisterCommand(actGetOption, command.Command{
		DocString: "Get the current value of a config option",
		Run: func(e *view.Editor, args *command.Args) command.Result {
			if args == nil || args.Empty() {
				return command.Result{Message: "error: usage: get <key>"}
			}
			key, _ := args.First()
			value, err := config.GetOption(e.Config(), key)
			if err != nil {
				return command.Result{Message: "error: " + err.Error()}
			}
			return command.Result{Message: value}
		},
		Aliases: []string{"get-option", "get"},
		Signature: staticSig(
			exactArgs(1),
			config.OptionKeys()...,
		),
	})
	r.RegisterCommand(actSetOption, command.Command{
		DocString: "Set a config option at runtime",
		Run: func(e *view.Editor, args *command.Args) command.Result {
			if args == nil || args.Len() < 2 {
				return command.Result{
					Message: "error: usage: set <key> <value>",
				}
			}
			key, _ := args.Get(0)
			val, _ := args.Get(1)
			if err := config.SetOption(e.Config(), key, val); err != nil {
				return command.Result{Message: "error: " + err.Error()}
			}
			return command.Result{}
		},
		Aliases: []string{"set-option", "set"},
		Signature: staticSig(
			rawAfter(1, 2, 2, nil),
			config.OptionKeys()...,
		),
	})
	r.RegisterCommand(actToggleOption, command.Command{
		DocString: "Toggle a config option at runtime",
		Run: func(e *view.Editor, args *command.Args) command.Result {
			if args == nil || args.Empty() {
				return command.Result{Message: "error: usage: toggle <key>"}
			}
			key, _ := args.First()
			value, err := config.ToggleOption(e.Config(), key)
			if err != nil {
				return command.Result{Message: "error: " + err.Error()}
			}
			return command.Result{
				Message: "'" + key + "' is now set to " + value,
			}
		},
		Aliases: []string{"toggle-option", "toggle"},
		Signature: staticSig(
			exactArgs(1),
			config.BoolOptionKeys()...,
		),
	})
	r.RegisterCommand(actConfigOpen, command.Command{
		DocString: "Open the user config.toml file",
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			path, ok := config.UserConfigPath()
			if !ok {
				return command.Result{Message: "error: config path unavailable"}
			}
			if _, err := e.SwitchFile(path); err != nil {
				return command.Result{Message: "error: " + err.Error()}
			}
			return command.Result{}
		},
		Aliases:   []string{"config-open"},
		Signature: sig(),
	})
	r.RegisterCommand(actConfigOpenWorkspace, command.Command{
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
	})
	r.RegisterCommand(actConfigReload, command.Command{
		DocString: "Refresh user config",
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			cfg, ok := config.LoadUserConfig()
			if !ok {
				return command.Result{Message: "error: config path unavailable"}
			}
			e.SetConfig(cfg)
			return command.Result{Message: "config reloaded"}
		},
		Aliases:   []string{"config-reload"},
		Signature: sig(),
	})
	r.RegisterCommand(actLogOpen, command.Command{
		DocString: "Open the editor log file",
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			path, ok := config.LogFilePath()
			if !ok {
				return command.Result{Message: "error: log path unavailable"}
			}
			if _, err := e.SwitchFile(path); err != nil {
				return command.Result{Message: "error: " + err.Error()}
			}
			return command.Result{}
		},
		Aliases:   []string{"log-open"},
		Signature: sig(),
	})
	r.RegisterCommand(actWorkspaceTrust, command.Command{
		DocString: "Add current workspace to the list of trusted workspaces",
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			if err := config.TrustWorkspace(e.Cwd()); err != nil {
				return command.Result{Message: "error: " + err.Error()}
			}
			return command.Result{Message: "workspace trusted"}
		},
		Aliases:   []string{"workspace-trust"},
		Signature: sig(),
	})
	r.RegisterCommand(actWorkspaceUntrust, command.Command{
		DocString: "Remove current workspace from the list of trusted workspaces",
		Run: func(e *view.Editor, _ *command.Args) command.Result {
			if err := config.UntrustWorkspace(e.Cwd()); err != nil {
				return command.Result{Message: "error: " + err.Error()}
			}
			return command.Result{Message: "workspace untrusted"}
		},
		Aliases:   []string{"workspace-untrust"},
		Signature: sig(),
	})
	r.RegisterCommand(actTheme, command.Command{
		DocString: "Change the editor theme " +
			"(show current theme if no name specified)",
		Run: func(e *view.Editor, args *command.Args) command.Result {
			cfg := e.Config()
			if args == nil || args.Empty() {
				name := cfg.Theme.Choose(false)
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
					Message: "error: could not load theme: " + err.Error(),
				}
			}
			if !(terminalTrueColor() || th.Is16Color()) {
				return command.Result{
					Message: "error: theme requires true color support",
				}
			}
			cfg.Theme = config.Theme{Name: name}
			e.SetConfig(cfg)
			return command.Result{}
		},

		Signature: staticSig(optionalArg(), loader.ThemeNames()...),
	})
	r.RegisterCommand(actSetLanguage, command.Command{
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
	})
	r.RegisterCommand(actSetLineEnding, command.Command{
		DocString: "Set the document's default line ending. Options: crlf, lf",
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
	})
	r.RegisterCommand(actIndentStyle, command.Command{
		DocString: "Set the indentation style for editing. " +
			"('t' for tabs or 1-16 for number of spaces)",
		Run: func(e *view.Editor, args *command.Args) command.Result {
			doc, ok := e.FocusedDocument()
			if !ok {
				return command.Result{Message: "error: no document"}
			}
			if args == nil || args.Empty() {
				return command.Result{Message: doc.IndentStyle().AsStr()}
			}
			arg, _ := args.First()
			switch arg {
			case "tabs", "tab", "t":
				doc.SetIndentStyle(core.Tabs())
			default:
				n, err := strconv.Atoi(arg)
				if err != nil || n < 1 || n > core.MaxIndent {
					return command.Result{
						Message: "error: expected 'tab' or spaces count (1-16)",
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
	})
	r.RegisterCommand(actEncoding, command.Command{
		DocString: "Set encoding",
		Run: func(_ *view.Editor, _ *command.Args) command.Result {
			return command.Result{Message: "utf-8"}
		},

		Signature: sig(),
	})
}

// languageNames returns sorted language names from the bundled language config,
// with "text" appended as the plain-text fallback option
func languageNames() []string {
	langs, ok := config.LoadBundledLanguages()
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
