package defaults

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/config"
)

type (
	pickerSection struct {
		Editor struct {
			BufferPicker ui.BufferPickerOptions `toml:"buffer-picker"`
			FileExplorer fileExplorerConfig     `toml:"file-explorer"`
			Picker       ui.PickerLayoutOptions `toml:"picker"`
		} `toml:"editor"`
	}

	fileExplorerConfig struct {
		Hidden         *bool `toml:"hidden"`
		FollowSymlinks *bool `toml:"follow-symlinks"`
		Parents        *bool `toml:"parents"`
		Ignore         *bool `toml:"ignore"`
		GitIgnore      *bool `toml:"git-ignore"`
		GitGlobal      *bool `toml:"git-global"`
		GitExclude     *bool `toml:"git-exclude"`
		FlattenDirs    *bool `toml:"flatten-dirs"`
	}
)

const (
	actFilePicker      = "file_picker"
	actFilePickerInCWD = "file_picker_in_current_dir"

	actFileExplorer         = "file_explorer"
	actFileExplorerInBufDir = "file_explorer_in_current_buffer_dir"
	actBufferPicker         = "buffer_picker"
	actJumplistPicker       = "jumplist_picker"
	actDiagnosticPicker     = "diagnostic_picker"
	actWorkspaceDiagnostics = "workspace_diagnostics_picker"

	actGlobalSearch   = "global_search"
	actCommandPalette = "command_palette"
	actLastPicker     = "last_picker"
)

func pickerModule(model ui.Model) command.Module {
	spc := prefixed(char(' '))
	cfg := new(pickerSection)

	return command.Module{
		Commands: []command.Command{
			{
				Name:      actFilePicker,
				DocString: "Open file picker",
				Run:       Continuation(model.PickerAction(ui.FilePicker)),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(char('f'))),
			},
			{
				Name:      actFilePickerInCWD,
				DocString: "Open file picker at current working directory",
				Run:       Continuation(model.PickerAction(ui.FilePickerInCWD)),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(char('F'))),
			},
			{
				Name:      actFileExplorer,
				DocString: "Open file explorer at workspace root",
				Run: Continuation(model.PickerAction(
					func(e *view.Editor) *ui.Picker {
						return ui.NewFileExplorer(
							e, fileExplorerOptions(cfg.Editor.FileExplorer),
						)
					},
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  keys(spc(char('e'))),
			},
			{
				Name:      actFileExplorerInBufDir,
				DocString: "Open file explorer at current buffer's directory",
				Run: Continuation(model.PickerAction(
					func(e *view.Editor) *ui.Picker {
						return ui.NewBufferDirExplorer(
							e, fileExplorerOptions(cfg.Editor.FileExplorer),
						)
					},
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  keys(spc(char('.'))),
			},
			{
				Name:      actBufferPicker,
				DocString: "Open buffer picker",
				Run: Continuation(model.PickerAction(
					func(e *view.Editor) *ui.Picker {
						return ui.NewBufferPicker(
							e, bufferPickerOptions(cfg.Editor.BufferPicker),
						)
					},
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  keys(spc(char('b'))),
			},
			{
				Name:      actJumplistPicker,
				DocString: "Open jumplist picker",
				Run:       Continuation(model.PickerAction(ui.JumplistPicker)),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(char('j'))),
			},
			{
				Name:      actDiagnosticPicker,
				DocString: "Open diagnostic picker",
				Run: Continuation(model.PickerAction(
					ui.NewDiagnosticPicker,
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  keys(spc(char('d'))),
			},
			{
				Name:      actWorkspaceDiagnostics,
				DocString: "Open workspace diagnostic picker",
				Run: Continuation(model.PickerAction(
					ui.NewWorkspaceDiagnosticPicker,
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  keys(spc(char('D'))),
			},
			{
				Name:      actGlobalSearch,
				DocString: "Global search in workspace folder",
				Run:       Continuation(model.GlobalSearchAction()),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(char('/'))),
			},
			{
				Name:      actCommandPalette,
				DocString: "Open command palette",
				Run:       Continuation(model.CommandPaletteAction()),
				Aliases:   []string{"command-palette"},
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(char('?'))),
			},
			{
				Name:      actLastPicker,
				DocString: "Reopen the last picker",
				Run:       Continuation(model.LastPickerAction()),
				Aliases:   []string{"last-picker"},
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(char('\''))),
			},
		},
		Section: &command.Section{
			Config: cfg,
			Reset:  func() { *cfg = pickerSection{} },
			Apply: func(*view.Editor) {
				model.SetPickerLayoutOptions(cfg.Editor.Picker)
			},
		},
		Options: []command.Option{
			pickerSplitRatiosOption(model),
		},
	}
}

func pickerSplitRatiosOption(model ui.Model) command.Option {
	const prefix = "editor.picker.split-ratios."
	return command.Option{
		Key: prefix,
		KeyGet: func(*view.Editor) (map[string]string, error) {
			ratios := model.PickerLayoutOptions().SplitRatios
			out := make(map[string]string, len(ratios))
			for key, ratio := range ratios {
				out[prefix+key] = strconv.FormatFloat(ratio, 'f', -1, 64)
			}
			return out, nil
		},
		KeySet: func(_ *view.Editor, key, s string) error {
			name := strings.TrimSpace(key)
			if len(name) <= len(prefix) {
				return fmt.Errorf("%w: %s", config.ErrInvalidOption, key)
			}
			name = name[len(prefix):]
			ratio, err := strconv.ParseFloat(s, 64)
			if err != nil || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
				return fmt.Errorf("%w: %s", config.ErrInvalidOption, s)
			}
			opts := model.PickerLayoutOptions()
			if opts.SplitRatios == nil {
				opts.SplitRatios = map[string]float64{}
			}
			opts.SplitRatios[name] = ratio
			model.SetPickerLayoutOptions(opts)
			return nil
		},
	}
}

func bufferPickerOptions(cfg ui.BufferPickerOptions) ui.BufferPickerOptions {
	if cfg.StartPosition == "" {
		cfg.StartPosition = ui.PickerStartTop
	}
	return cfg
}

func fileExplorerOptions(cfg fileExplorerConfig) ui.FileExplorerOptions {
	opts := ui.DefaultFileExplorerOptions()
	if cfg.Hidden != nil {
		opts.Hidden = *cfg.Hidden
	}
	if cfg.FollowSymlinks != nil {
		opts.FollowSymlinks = *cfg.FollowSymlinks
	}
	if cfg.Parents != nil {
		opts.Parents = *cfg.Parents
	}
	if cfg.Ignore != nil {
		opts.Ignore = *cfg.Ignore
	}
	if cfg.GitIgnore != nil {
		opts.GitIgnore = *cfg.GitIgnore
	}
	if cfg.GitGlobal != nil {
		opts.GitGlobal = *cfg.GitGlobal
	}
	if cfg.GitExclude != nil {
		opts.GitExclude = *cfg.GitExclude
	}
	if cfg.FlattenDirs != nil {
		opts.FlattenDirs = *cfg.FlattenDirs
	}
	return opts
}
