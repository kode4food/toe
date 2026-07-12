package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/health"
	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/lsp"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/defaults"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/vcs"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
	"github.com/kode4food/toe/internal/view/config"
)

type App struct {
	ConfigPath string
	Root       string
	PickerDir  string
	Files      []string
	Editor     *view.Editor
	Model      ui.Model
	Reg        *command.Registry
}

var ErrDirectoryArgument = errors.New(
	"expected a path to file, but found a directory",
)

func Run(args []string, out io.Writer) error {
	if len(args) == 1 && args[0] == "--health" {
		return health.Run(out)
	}
	a := &App{}
	args = a.ParseConfigFlag(args)
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := a.ResolveSession(args, cwd); err != nil {
		return err
	}
	a.Editor = view.NewEditor(a.Root)
	a.Editor.SetClipboard(
		action.NewOSC52Clipboard(action.NewSystemClipboard()),
	)
	if err := a.OpenEditorFiles(); err != nil {
		return err
	}
	defer vcs.Attach(a.Editor).Close()
	if err := a.InitReg(); err != nil {
		return err
	}
	if err := a.ApplyConfigFiles(); err != nil {
		return err
	}
	baseValues, err := a.Reg.OptionValues(a.Editor)
	if err != nil {
		return err
	}
	if err := a.MaybeRestoreSession(a.WorkspaceTrusted); err != nil {
		return err
	}
	defer a.InitLSP(context.Background())()
	if err := a.ConfigureModel(); err != nil {
		return err
	}
	a.Model.RestoreTerminalPanes(a.Editor)
	defer ui.CloseAllTerminalPanes(a.Editor)
	if _, err := tea.NewProgram(a.Model).Run(); err != nil {
		return err
	}
	return a.MaybeSaveSession(baseValues)
}

// ParseConfigFlag strips --config and its value from args into ConfigPath
func (a *App) ParseConfigFlag(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--config":
			if i+1 < len(args) {
				a.ConfigPath = args[i+1]
				i++
			}
		default:
			out = append(out, args[i])
		}
	}
	return out
}

// ResolveSession populates Root, PickerDir, and Files from args
func (a *App) ResolveSession(args []string, cwd string) error {
	a.Root = cwd
	a.Files = args
	if len(args) > 0 {
		if fi, err := os.Stat(args[0]); err == nil && fi.IsDir() {
			a.PickerDir = args[0]
			a.Files = args[1:]
		}
	}
	if a.PickerDir != "" {
		abs, err := filepath.Abs(a.PickerDir)
		if err != nil {
			return err
		}
		a.Root = abs
	}
	return nil
}

// OpenEditorFiles opens each path in Files into the editor
func (a *App) OpenEditorFiles() error {
	for _, path := range a.Files {
		if fi, err := os.Stat(path); err == nil && fi.IsDir() {
			return fmt.Errorf(
				"%w: %q (to open a directory pass it as first argument)",
				ErrDirectoryArgument, path,
			)
		}
		if _, err := a.Editor.OpenFile(path); err != nil {
			return err
		}
	}
	return nil
}

// InitReg wires up keymaps, model, and command registry
func (a *App) InitReg() error {
	km := command.NewKeymaps()
	a.Model = ui.New(a.Editor, km)
	var err error
	a.Reg, err = defaults.RegisterDefaults(a.Model, km)
	return err
}

// ApplyConfigFiles merges workspace and explicit config TOML into the editor
func (a *App) ApplyConfigFiles() error {
	raw, _ := config.LoadRawConfigForDir(a.Root)
	if raw == nil {
		raw = map[string]any{}
	}
	if err := a.Reg.ApplyTOML(a.Editor, raw); err != nil {
		return err
	}
	if a.ConfigPath != "" {
		if raw, ok := config.LoadRawConfig(a.ConfigPath); ok {
			if err := a.Reg.ApplyTOML(a.Editor, raw); err != nil {
				return err
			}
		}
	}
	return nil
}

// WorkspaceTrusted reports whether the current workspace is trusted
func (a *App) WorkspaceTrusted() bool {
	return loader.QueryWorkspaceTrust(a.Root, a.Editor.Options().Insecure)
}

// MaybeRestoreSession restores a saved session when no files were given and
// the workspace is trusted; clears PickerDir when a session is restored
func (a *App) MaybeRestoreSession(trusted func() bool) error {
	if !a.Editor.Options().AutoSession || len(a.Files) != 0 || !trusted() {
		return nil
	}
	sessionPath := view.WorkspaceSessionFile(a.Root)
	values, ok, err := a.Editor.RestoreSession(sessionPath)
	if err != nil && !errors.Is(err, view.ErrSessionEmpty) {
		return err
	}
	if !ok {
		return nil
	}
	if err := a.Reg.ApplyOptionValues(a.Editor, values); err != nil {
		return err
	}
	a.PickerDir = ""
	return nil
}

// InitLSP attaches the LSP session and wires the config-reload callback
func (a *App) InitLSP(ctx context.Context) func() {
	session := lsp.Attach(ctx, a.Editor)
	a.Editor.SetConfigReload(func() error {
		raw, _ := config.LoadRawConfigForDir(a.Editor.Cwd())
		if raw == nil {
			raw = map[string]any{}
		}
		if err := a.Reg.ApplyTOML(a.Editor, raw); err != nil {
			return err
		}
		return session.ReloadConfig()
	})
	return func() { _ = session.Close() }
}

// ConfigureModel sets the initial picker and any startup message on the model
func (a *App) ConfigureModel() error {
	if a.PickerDir != "" {
		abs, err := filepath.Abs(a.PickerDir)
		if err != nil {
			return err
		}
		a.Model = a.Model.WithInitialPicker(ui.FilePickerInDir(abs))
	}
	_, workspaceFallback := loader.FindWorkspace(a.Root)
	trusted := loader.QueryWorkspaceTrust(a.Root, a.Editor.Options().Insecure)
	if !a.Editor.Options().Insecure && !workspaceFallback && !trusted {
		a.Model = a.Model.WithStartupMessage(
			"Workspace untrusted; run :workspace_trust to enable config",
		)
	}
	return nil
}

// MaybeSaveSession saves the session if AutoSession is enabled and trusted
func (a *App) MaybeSaveSession(base map[string]string) error {
	if !a.Editor.Options().AutoSession || !a.WorkspaceTrusted() {
		return nil
	}
	values, err := a.Reg.OptionValues(a.Editor)
	if err != nil {
		return err
	}
	return a.Editor.SaveSession(
		view.WorkspaceSessionFile(a.Root),
		ChangedOptionValues(base, values),
	)
}

// ChangedOptionValues returns only the entries in values that differ from base
func ChangedOptionValues(base, values map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range values {
		if base[key] != value {
			out[key] = value
		}
	}
	return out
}
