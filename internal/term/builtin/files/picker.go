package files

import (
	"github.com/kode4food/toe/internal/term/builtin/kit"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

type (
	pickerSection struct {
		Editor struct {
			BufferPicker BufferPickerOptions `toml:"buffer-picker"`
			FileExplorer fileExplorerConfig  `toml:"file-explorer"`
		} `toml:"editor"`
	}

	fileExplorerConfig struct {
		Hidden         bool `toml:"hidden"`
		FollowSymlinks bool `toml:"follow-symlinks"`
		Parents        bool `toml:"parents"`
		IgnoreFiles    bool `toml:"ignore-files"`
		FlattenDirs    bool `toml:"flatten-dirs"`
	}
)

const (
	actFilePicker           = "file_picker"
	actFilePickerInCWD      = "file_picker_in_current_dir"
	actFileExplorer         = "file_explorer"
	actFileExplorerForPane  = "file_explorer_in_current_pane_dir"
	actBufferPicker         = "buffer_picker"
	actDiagnosticPicker     = "diagnostic_picker"
	actWorkspaceDiagnostics = "workspace_diagnostics_picker"
	actGlobalSearch         = "global_search"
)

// PickerModule returns the file, buffer, and explorer picker commands
func PickerModule(model ui.Model) command.Module {
	cfg := new(pickerSection)
	reset := func() {
		*cfg = pickerSection{}
		cfg.Editor.FileExplorer = fileExplorerConfig(
			DefaultFileExplorerOptions(),
		)
	}
	reset()

	return command.Module{
		Commands: []command.Command{
			{
				Name:      actFilePicker,
				DocString: "Open file picker",
				Run:       kit.Runner(model.PickerAction(NewFilePicker)),
				Modes:     command.PaneModes,
				Keys:      kit.Leader('f'),
			},
			{
				Name:      actFilePickerInCWD,
				DocString: "Open file picker at current working directory",
				Run:       kit.Runner(model.PickerAction(NewFilePickerInCWD)),
				Modes:     command.PaneModes,
				Keys:      kit.Leader('F'),
			},
			{
				Name:      actFileExplorer,
				DocString: "Open file explorer at workspace root",
				Run: kit.Runner(model.PickerAction(
					func(e *view.Editor) *ui.Picker {
						return NewFileExplorer(
							e, FileExplorerOptions(cfg.Editor.FileExplorer),
						)
					},
				)),
				Modes: command.PaneModes,
				Keys:  kit.Leader('e'),
			},
			{
				Name:      actFileExplorerForPane,
				DocString: "Open file explorer at current pane's directory",
				Run: kit.Runner(model.PickerAction(
					func(e *view.Editor) *ui.Picker {
						return NewFocusedPaneDirExplorer(
							e, FileExplorerOptions(cfg.Editor.FileExplorer),
						)
					},
				)),
				Modes: command.PaneModes,
				Keys:  kit.Leader('.'),
			},
			{
				Name:      actBufferPicker,
				DocString: "Open buffer picker",
				Run: kit.Runner(model.PickerAction(
					func(e *view.Editor) *ui.Picker {
						return NewBufferPicker(
							e, bufferPickerOptions(cfg.Editor.BufferPicker),
						)
					},
				)),
				Modes: command.PaneModes,
				Keys:  kit.Leader('b'),
			},
		},
		Options: []command.Option{
			bufferPickerStartOption(
				&cfg.Editor.BufferPicker.StartPosition,
			),
			fileExplorerBoolOption(
				"file-explorer.hidden",
				&cfg.Editor.FileExplorer.Hidden,
			),
			fileExplorerBoolOption(
				"file-explorer.follow-symlinks",
				&cfg.Editor.FileExplorer.FollowSymlinks,
			),
			fileExplorerBoolOption(
				"file-explorer.parents",
				&cfg.Editor.FileExplorer.Parents,
			),
			fileExplorerBoolOption(
				"file-explorer.ignore-files",
				&cfg.Editor.FileExplorer.IgnoreFiles,
			),
			fileExplorerBoolOption(
				"file-explorer.flatten-dirs",
				&cfg.Editor.FileExplorer.FlattenDirs,
			),
		},
		Section: &command.Section{
			Config: cfg,
			Reset:  reset,
		},
	}
}

// DiagnosticsModule returns diagnostic and workspace search picker commands
// separately from PickerModule, allowing independent placement in the
// space-leader menu
func DiagnosticsModule(model ui.Model) command.Module {
	return command.Module{
		Commands: []command.Command{
			{
				Name:      actDiagnosticPicker,
				DocString: "Open diagnostic picker",
				Run:       kit.Runner(model.PickerAction(NewDiagnosticPicker)),
				Modes:     command.PaneModes,
				Keys:      kit.Leader('d'),
			},
			{
				Name:      actWorkspaceDiagnostics,
				DocString: "Open workspace diagnostic picker",
				Run: kit.Runner(
					model.PickerAction(NewWorkspaceDiagnosticPicker),
				),
				Modes: command.PaneModes,
				Keys:  kit.Leader('D'),
			},
			{
				Name:      actGlobalSearch,
				DocString: "Global search in workspace folder",
				Run: kit.Runner(
					model.PickerAction(NewGlobalSearchPicker),
				),
				Modes: command.PaneModes,
				Keys:  kit.Leader('/'),
			},
		},
	}
}

func bufferPickerStartOption(value *PickerStartPosition) command.Option {
	return command.Option{
		Key: "buffer-picker.start-position",
		Get: func(*view.Editor) (string, error) {
			if *value == "" {
				return string(PickerStartTop), nil
			}
			return string(*value), nil
		},
		Set: func(_ *view.Editor, s string) error {
			var next PickerStartPosition
			if err := next.UnmarshalText([]byte(s)); err != nil {
				return err
			}
			*value = next
			return nil
		},
		Complete: command.StaticCompleter(
			PickerStartTop, PickerStartPrevious,
		),
	}
}

func fileExplorerBoolOption(key string, value *bool) command.Option {
	return kit.EditorBoolOption(key,
		func(*view.Editor) bool {
			return *value
		},
		func(_ *view.Editor, next bool) {
			*value = next
		},
	)
}

func bufferPickerOptions(cfg BufferPickerOptions) BufferPickerOptions {
	if cfg.StartPosition == "" {
		cfg.StartPosition = PickerStartTop
	}
	return cfg
}
