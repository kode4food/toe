package config

import (
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/term/builtin/kit"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/language"
)

type pathProvider func() (string, bool)

var (
	errThemeLoad          = i18n.NewError(i18n.ErrorThemeLoad)
	errConfigUnavailable  = i18n.NewError(i18n.ErrorConfigUnavailable)
	errWorkspaceUntrusted = i18n.NewError(i18n.ErrorWorkspaceUntrustedHint)
	errLogUnavailable     = i18n.NewError(i18n.ErrorLogUnavailable)
)

func systemCmds() []command.Command {
	return []command.Command{
		{
			Name:      actConfigOpen,
			DocString: "Open the user config.toml file",
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				return openFromPath(
					e, loader.ConfigFile, errConfigUnavailable,
				)
			},
			Modes:     command.PaneModes,
			Signature: command.DefaultSignature(),
		},
		{
			Name:      actConfigOpenWorkspace,
			DocString: "Open the workspace config.toml file",
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				if !loader.QueryWorkspaceTrust(
					e.Cwd(), e.Options().Insecure,
				) {
					return command.Result{Error: errWorkspaceUntrusted}
				}
				path := loader.WorkspaceConfigFile(e.Cwd())
				if _, err := e.OpenFile(path); err != nil {
					return command.Result{Error: err}
				}
				return command.Result{}
			},
			Modes:     command.PaneModes,
			Signature: command.DefaultSignature(),
		},
		{
			Name:      actConfigReload,
			DocString: "Refresh user config",
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				if err := e.ReloadConfig(); err != nil {
					return command.Result{Error: err}
				}
				return command.Result{Message: "config reloaded"}
			},
			Modes:     command.PaneModes,
			Signature: command.DefaultSignature(),
		},
		{
			Name:      actLogOpen,
			DocString: "Open the editor log file",
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				return openFromPath(e, loader.LogFile, errLogUnavailable)
			},
			Modes:     command.PaneModes,
			Signature: command.DefaultSignature(),
		},
		{
			Name: actWorkspaceTrust,
			DocString: "Add current workspace to the list of trusted " +
				"workspaces",
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				if err := loader.TrustWorkspace(e.Cwd()); err != nil {
					return command.Result{Error: err}
				}
				return command.Result{Message: "workspace trusted"}
			},
			Modes:     command.PaneModes,
			Signature: command.DefaultSignature(),
		},
		{
			Name: actWorkspaceUntrust,
			DocString: "Remove current workspace from the list of trusted " +
				"workspaces",
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				if err := loader.UntrustWorkspace(e.Cwd()); err != nil {
					return command.Result{Error: err}
				}
				return command.Result{Message: "workspace untrusted"}
			},
			Modes:     command.PaneModes,
			Signature: command.DefaultSignature(),
		},
	}
}

func themeCmds() []command.Command {
	return []command.Command{
		{
			Name: actTheme,
			DocString: "Change the editor theme (show current theme if no " +
				"name specified)",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				if args == nil || args.Empty() {
					name := e.Options().Theme
					if _, err := theme.Load(name); err != nil {
						th, _ := theme.Default()
						name = th.Name()
					}
					return command.Result{Message: name}
				}
				name, _ := args.First()
				if name == "default" {
					name = view.DefaultTheme
				}
				if _, err := theme.Load(name); err != nil {
					return command.Result{
						Error: errThemeLoad.WithVars(i18n.Vars{
							"message": err,
						}),
					}
				}
				e.Options().Theme = name
				return command.Result{}
			},
			Modes:     command.PaneModes,
			Signature: kit.StaticSig(kit.OptionalArg(), loader.ThemeNames()...),
		},
	}
}

func languageNames() []string {
	langs, ok := language.LoadBundledLanguages()
	if !ok {
		return []string{view.DefaultLanguage}
	}
	names := make([]string, 0, len(langs.Languages)+1)
	for _, l := range langs.Languages {
		if l.Name != "" {
			names = append(names, l.Name)
		}
	}
	names = append(names, view.DefaultLanguage)
	return names
}

func openFromPath(
	e *view.Editor, pathFn pathProvider, unavailable error,
) command.Result {
	path, ok := pathFn()
	if !ok {
		return command.Result{Error: unavailable}
	}
	if _, err := e.OpenFile(path); err != nil {
		return command.Result{Error: err}
	}
	return command.Result{}
}
