package defaults

import (
	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/language"
)

type pathProvider func() (string, bool)

func configSystemCmds() []command.Command {
	return []command.Command{
		{
			Name:      actConfigOpen,
			DocString: "Open the user config.toml file",
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				return openFromPath(
					e, loader.ConfigFile, "config path unavailable",
				)
			},
			Aliases:   []string{"config-open"},
			Signature: sig(),
		},
		{
			Name:      actConfigOpenWorkspace,
			DocString: "Open the workspace config.toml file",
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				if !loader.QueryWorkspaceTrust(
					e.Cwd(), e.Options().Insecure,
				) {
					return command.Result{
						Message: "workspace untrusted; " +
							"run :workspace_trust to enable",
					}
				}
				path := loader.WorkspaceConfigFile(e.Cwd())
				if _, err := e.SwitchFile(path); err != nil {
					return command.Result{Message: "error: " + err.Error()}
				}
				return command.Result{}
			},
			Aliases:   []string{"config-open-workspace"},
			Signature: sig(),
		},
		{
			Name:      actConfigReload,
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
		{
			Name:      actLogOpen,
			DocString: "Open the editor log file",
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				return openFromPath(
					e, loader.LogFile, "log path unavailable",
				)
			},
			Aliases:   []string{"log-open"},
			Signature: sig(),
		},
		{
			Name: actWorkspaceTrust,
			DocString: "Add current workspace to the list of trusted " +
				"workspaces",
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				if err := loader.TrustWorkspace(e.Cwd()); err != nil {
					return command.Result{Message: "error: " + err.Error()}
				}
				return command.Result{Message: "workspace trusted"}
			},
			Aliases:   []string{"workspace-trust"},
			Signature: sig(),
		},
		{
			Name: actWorkspaceUntrust,
			DocString: "Remove current workspace from the list of " +
				"trusted workspaces",
			Run: func(e *view.Editor, _ *command.Args) command.Result {
				if err := loader.UntrustWorkspace(e.Cwd()); err != nil {
					return command.Result{Message: "error: " + err.Error()}
				}
				return command.Result{Message: "workspace untrusted"}
			},
			Aliases:   []string{"workspace-untrust"},
			Signature: sig(),
		},
	}
}

func configThemeCmds() []command.Command {
	return []command.Command{
		{
			Name: actTheme,
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
	e *view.Editor, pathFn pathProvider, unavailMsg string,
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
