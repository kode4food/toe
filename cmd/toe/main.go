package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/health"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/defaults"
	"github.com/kode4food/toe/internal/term/ui"
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

	editor := view.NewEditor(cwd)
	if configPath != "" {
		if cfg, ok := config.LoadConfig(configPath); ok {
			editor.SetConfig(cfg)
		}
	}

	var pickerDir string
	if len(args) > 0 {
		if fi, err := os.Stat(args[0]); err == nil && fi.IsDir() {
			pickerDir, args = args[0], args[1:]
		}
	}
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

	km := command.NewKeymaps()
	model := ui.New(editor, km)
	defaults.RegisterDefaults(model, km)
	if pickerDir != "" {
		abs, err := filepath.Abs(pickerDir)
		if err != nil {
			return err
		}
		model = model.WithInitialPicker(ui.FilePickerInDir(abs))
	}

	p := tea.NewProgram(model)
	_, err = p.Run()
	return err
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
