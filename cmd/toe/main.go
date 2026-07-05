package main

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
	"github.com/kode4food/toe/internal/view/config"
)

var ErrDirectoryArgument = errors.New(
	"expected a path to file, but found a directory",
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	if len(args) == 1 && args[0] == "--health" {
		return health.Run(out)
	}

	var configPath string
	args = parseConfigFlag(args, &configPath)

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	var pickerDir string
	if len(args) > 0 {
		if fi, err := os.Stat(args[0]); err == nil && fi.IsDir() {
			pickerDir, args = args[0], args[1:]
		}
	}
	sessionRoot := cwd
	if pickerDir != "" {
		sessionRoot, err = filepath.Abs(pickerDir)
		if err != nil {
			return err
		}
	}

	editor := view.NewEditor(sessionRoot)
	for _, path := range args {
		if fi, err := os.Stat(path); err == nil && fi.IsDir() {
			return fmt.Errorf(
				"%w: %q"+
					" (to open a directory pass it as first argument)",
				ErrDirectoryArgument, path,
			)
		}
		if _, err := editor.OpenFile(path); err != nil {
			return err
		}
	}

	vcsSession := vcs.Attach(editor)
	defer vcsSession.Close()

	km := command.NewKeymaps()
	model := ui.New(editor, km)
	reg, err := defaults.RegisterDefaults(model, km)
	if err != nil {
		return err
	}
	raw, _ := config.LoadRawConfigForDir(sessionRoot)
	if raw == nil {
		raw = map[string]any{}
	}
	if err := reg.ApplyTOML(editor, raw); err != nil {
		return err
	}
	if configPath != "" {
		if raw, ok := config.LoadRawConfig(configPath); ok {
			if err := reg.ApplyTOML(editor, raw); err != nil {
				return err
			}
		}
	}
	baseValues, err := reg.OptionValues(editor)
	if err != nil {
		return err
	}
	sessionPath := view.WorkspaceSessionFile(sessionRoot)
	workspaceTrusted := func() bool {
		return loader.QueryWorkspaceTrust(
			sessionRoot, editor.Options().Insecure,
		)
	}
	if editor.Options().AutoSession && len(args) == 0 && workspaceTrusted() {
		values, ok, err := editor.RestoreSession(sessionPath)
		if err != nil && !errors.Is(err, view.ErrSessionEmpty) {
			return err
		}
		if ok {
			if err := reg.ApplyOptionValues(editor, values); err != nil {
				return err
			}
			pickerDir = ""
		}
	}
	lspSession := lsp.Attach(context.Background(), editor)
	defer func() { _ = lspSession.Close() }()
	editor.SetConfigReload(func() error {
		raw, _ := config.LoadRawConfigForDir(editor.Cwd())
		if raw == nil {
			raw = map[string]any{}
		}
		if err := reg.ApplyTOML(editor, raw); err != nil {
			return err
		}
		return lspSession.ReloadConfig()
	})
	if pickerDir != "" {
		abs, err := filepath.Abs(pickerDir)
		if err != nil {
			return err
		}
		model = model.WithInitialPicker(ui.FilePickerInDir(abs))
	}
	if !editor.Options().Insecure && !workspaceTrusted() {
		model = model.WithStartupMessage(
			"Workspace untrusted — session and workspace config disabled. " +
				":workspace_trust to enable",
		)
	}

	p := tea.NewProgram(model)
	_, err = p.Run()
	if err == nil && editor.Options().AutoSession && workspaceTrusted() {
		values, valueErr := reg.OptionValues(editor)
		if valueErr != nil {
			return valueErr
		}
		err = editor.SaveSession(
			sessionPath, changedOptionValues(baseValues, values),
		)
	}
	return err
}

func changedOptionValues(base, values map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range values {
		if base[key] != value {
			out[key] = value
		}
	}
	return out
}

// parseConfigFlag strips --config <path> from args and sets path
func parseConfigFlag(args []string, path *string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--config":
			if i+1 < len(args) {
				*path = args[i+1]
				i++
			}
		default:
			out = append(out, args[i])
		}
	}
	return out
}
