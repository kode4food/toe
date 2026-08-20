package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/kode4food/toe/internal/health"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/lsp"
	"github.com/kode4food/toe/internal/term/ale"
	"github.com/kode4food/toe/internal/term/builtin"
	"github.com/kode4food/toe/internal/term/builtin/files"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/vcs"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
	"github.com/kode4food/toe/internal/view/config"
)

// App is one editor process: the paths resolved from the command line, the
// editor and its command registry, and the services attached to them. Build it
// with New, bring it up with Start, take it down with Stop
type App struct {
	ConfigPath string
	Root       string
	ShowPicker bool
	Files      []string

	Editor *view.Editor
	Model  ui.Model
	Reg    *command.Registry

	keymaps  *command.Keymaps
	baseOpts map[string]string
	lsp      *lsp.Session
	vcs      *vcs.Session
}

var ErrDirectoryArgument = errors.New(
	"expected a path to file, but found a directory",
)

// New resolves the command line into the workspace root, the config path, and
// the files to open. It touches no editor state. Call Start for that
func New(args []string, cwd string) (*App, error) {
	a := &App{}
	args = a.parseConfigFlag(args)
	if err := a.resolvePaths(args, cwd); err != nil {
		return nil, err
	}
	return a, nil
}

// Run starts the editor, or writes the health report when --health is the only
// argument
func Run(args []string, out io.Writer) error {
	if len(args) == 1 && args[0] == "--health" {
		return health.Run(out)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	a, err := New(args, cwd)
	if err != nil {
		return err
	}
	if err := a.Start(context.Background()); err != nil {
		return errors.Join(err, a.Stop())
	}
	_, err = tea.NewProgram(a.Model, teaOptions()...).Run()
	return errors.Join(err, a.Stop())
}

// Start brings the app up in order: editor and commands, named files, config
// and init files, the base options those establish, a saved session, then the
// attached services and the model
func (a *App) Start(ctx context.Context) error {
	if err := a.initEditor(); err != nil {
		return err
	}
	if err := a.openFiles(); err != nil {
		return err
	}
	if err := a.applyConfigFiles(); err != nil {
		return err
	}
	if err := a.applyInitFile(); err != nil {
		return err
	}
	if err := a.resolveBaseOptions(); err != nil {
		return err
	}
	if err := a.restoreSession(); err != nil {
		return err
	}
	a.vcs = vcs.Attach(a.Editor)
	a.lsp = lsp.Attach(ctx, a.Editor)
	a.configureModel()
	return nil
}

// Stop saves the session and releases everything Start attached
func (a *App) Stop() error {
	err := a.saveSession()
	if a.Editor == nil {
		return err
	}
	ui.CloseAllTerminalPanes(a.Editor)
	a.Model.Close()
	if a.lsp != nil {
		err = errors.Join(err, a.lsp.Close())
	}
	if a.vcs != nil {
		a.vcs.Close()
	}
	return err
}

// WorkspaceTrusted reports whether the current workspace is trusted
func (a *App) WorkspaceTrusted() bool {
	return loader.QueryWorkspaceTrust(a.Root, a.Editor.Options().Insecure)
}

func (a *App) parseConfigFlag(args []string) []string {
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

// resolvePaths populates Root, ShowPicker, and Files from args. A leading
// directory argument becomes the root, and opens the file picker there
func (a *App) resolvePaths(args []string, cwd string) error {
	a.Root = cwd
	a.Files = args
	if len(args) == 0 {
		return nil
	}
	if fi, err := os.Stat(args[0]); err == nil && fi.IsDir() {
		abs, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}
		a.Root = abs
		a.ShowPicker = true
		a.Files = args[1:]
		return nil
	}
	path := args[0]
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	root, _ := loader.FindWorkspace(cwd)
	rel, err := filepath.Rel(root, abs)
	if err != nil || !filepath.IsLocal(rel) {
		a.Root = filepath.Dir(abs)
	}
	return nil
}

func (a *App) initEditor() error {
	a.keymaps = command.NewKeymaps()
	a.Editor = view.NewEditor(a.Root)
	a.Model = ui.New(a.Editor, a.keymaps)
	reg, err := builtin.Register(a.Model, a.keymaps)
	if err != nil {
		return err
	}
	a.Reg = reg
	a.Editor.SetClipboard(
		action.NewOSC52Clipboard(action.NewSystemClipboard()),
	)
	a.Editor.SetBaseOptions(a.baseOptions)
	a.Editor.SetConfigReload(a.reloadConfig)
	return a.Editor.Chdir(a.Root)
}

func (a *App) openFiles() error {
	for _, path := range a.Files {
		if fi, err := os.Stat(path); err == nil && fi.IsDir() {
			return fmt.Errorf(
				"%w: %q (to open a directory pass it as first argument)",
				ErrDirectoryArgument, path,
			)
		}
		_, _, err := ui.OpenPath(a.Editor, path, ui.PickerAcceptReplace)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *App) applyConfigFiles() error {
	raw, _ := config.LoadRawConfigForDir(a.Root)
	if raw == nil {
		raw = map[string]any{}
	}
	if err := a.Reg.ApplyTOML(a.Editor, raw); err != nil {
		return err
	}
	if a.ConfigPath == "" {
		return nil
	}
	raw, ok := config.LoadRawConfig(a.ConfigPath)
	if !ok {
		return nil
	}
	return a.Reg.ApplyTOML(a.Editor, raw)
}

func (a *App) applyInitFile() error {
	rt, err := ale.NewRuntime(a.Editor, a.keymaps)
	if err != nil {
		return err
	}
	if dir, ok := loader.ConfigDir(); ok {
		if err := evalInitFile(rt, filepath.Join(dir, "init.ale")); err != nil {
			return err
		}
	}
	if !a.WorkspaceTrusted() {
		return nil
	}
	return evalInitFile(rt, loader.WorkspaceInitFile(a.Root))
}

// resolveBaseOptions re-reads the option values a saved session is compared
// against, so a session records only what changed after startup
func (a *App) resolveBaseOptions() error {
	values, err := a.Reg.OptionValues(a.Editor)
	if err != nil {
		return err
	}
	a.baseOpts = values
	return nil
}

func (a *App) baseOptions() map[string]string {
	return a.baseOpts
}

func (a *App) reloadConfig() error {
	if err := a.applyConfigFiles(); err != nil {
		return err
	}
	if err := a.resolveBaseOptions(); err != nil {
		return err
	}
	if a.lsp == nil {
		return nil
	}
	return a.lsp.ReloadConfig()
}

func (a *App) restoreSession() error {
	if !a.isSessionEnabled() {
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
	a.ShowPicker = false
	return nil
}

func (a *App) saveSession() error {
	if a.Editor == nil || a.Reg == nil {
		return nil
	}
	if !a.isSessionEnabled() {
		return nil
	}
	values, err := a.Reg.ChangedOptionValues(a.Editor)
	if err != nil {
		return err
	}
	return a.Editor.SaveSession(view.WorkspaceSessionFile(a.Root), values)
}

func (a *App) configureModel() {
	if a.ShowPicker {
		a.Model = a.Model.WithInitialPicker(
			files.NewFilePickerInDir(a.Root),
		)
	}
	_, workspaceFallback := loader.FindWorkspace(a.Root)
	if a.Editor.Options().Insecure ||
		workspaceFallback || a.WorkspaceTrusted() {
		return
	}
	a.Model = a.Model.WithStartupMessage(
		i18n.Text(i18n.ErrorWorkspaceUntrustedHint),
	)
}

func (a *App) isSessionEnabled() bool {
	return len(a.Files) == 0 && a.Editor.Options().AutoSession &&
		a.WorkspaceTrusted()
}

func teaOptions() []tea.ProgramOption {
	if !ui.TrueColorSupported() {
		return nil
	}
	return []tea.ProgramOption{tea.WithColorProfile(colorprofile.TrueColor)}
}

func evalInitFile(rt *ale.Runtime, path string) error {
	src, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := rt.Eval(string(src)); err != nil {
		return fmt.Errorf("%s:\n%s", path, i18n.ErrorText(err))
	}
	return nil
}
