package files

import (
	"github.com/kode4food/toe/internal/term/builtin/kit"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

type (
	filesPickerSection struct {
		Editor struct {
			BufferPicker BufferPickerOptions `toml:"buffer-picker"`
			FileExplorer fileExplorerConfig  `toml:"file-explorer"`
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
	actFilePicker           = "file_picker"
	actFilePickerInCWD      = "file_picker_in_current_dir"
	actFileExplorer         = "file_explorer"
	actFileExplorerInBufDir = "file_explorer_in_current_buffer_dir"
	actBufferPicker         = "buffer_picker"
	actDiagnosticPicker     = "diagnostic_picker"
	actWorkspaceDiagnostics = "workspace_diagnostics_picker"
	actGlobalSearch         = "global_search"
)

// PickerModule returns the file, buffer, and explorer picker commands
func PickerModule(model ui.Model) command.Module {
	spc := kit.Prefixed(kit.Char(' '))
	cfg := new(filesPickerSection)

	return command.Module{
		Commands: []command.Command{
			{
				Name:      actFilePicker,
				DocString: "Open file picker",
				Run:       kit.Continuation(model.PickerAction(NewFilePicker)),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(spc(kit.Char('f'))),
			},
			{
				Name:      actFilePickerInCWD,
				DocString: "Open file picker at current working directory",
				Run: kit.Continuation(
					model.PickerAction(NewFilePickerInCWD),
				),
				Modes: []string{"NOR", "SEL"},
				Keys:  kit.Keys(spc(kit.Char('F'))),
			},
			{
				Name:      actFileExplorer,
				DocString: "Open file explorer at workspace root",
				Run: kit.Continuation(model.PickerAction(
					func(e *view.Editor) *ui.Picker {
						return NewFileExplorer(
							e, fileExplorerOptions(cfg.Editor.FileExplorer),
						)
					},
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  kit.Keys(spc(kit.Char('e'))),
			},
			{
				Name:      actFileExplorerInBufDir,
				DocString: "Open file explorer at current buffer's directory",
				Run: kit.Continuation(model.PickerAction(
					func(e *view.Editor) *ui.Picker {
						return NewBufferDirExplorer(
							e, fileExplorerOptions(cfg.Editor.FileExplorer),
						)
					},
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  kit.Keys(spc(kit.Char('.'))),
			},
			{
				Name:      actBufferPicker,
				DocString: "Open buffer picker",
				Run: kit.Continuation(model.PickerAction(
					func(e *view.Editor) *ui.Picker {
						return NewBufferPicker(
							e, bufferPickerOptions(cfg.Editor.BufferPicker),
						)
					},
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  kit.Keys(spc(kit.Char('b'))),
			},
		},
		Section: &command.Section{
			Config: cfg,
			Reset:  func() { *cfg = filesPickerSection{} },
		},
	}
}

// DiagnosticsModule returns diagnostic and workspace search picker commands
// separately from PickerModule, allowing independent placement in the
// space-leader menu
func DiagnosticsModule(model ui.Model) command.Module {
	spc := kit.Prefixed(kit.Char(' '))

	return command.Module{
		Commands: []command.Command{
			{
				Name:      actDiagnosticPicker,
				DocString: "Open diagnostic picker",
				Run: kit.Continuation(
					model.PickerAction(NewDiagnosticPicker),
				),
				Modes: []string{"NOR", "SEL"},
				Keys:  kit.Keys(spc(kit.Char('d'))),
			},
			{
				Name:      actWorkspaceDiagnostics,
				DocString: "Open workspace diagnostic picker",
				Run: kit.Continuation(
					model.PickerAction(NewWorkspaceDiagnosticPicker),
				),
				Modes: []string{"NOR", "SEL"},
				Keys:  kit.Keys(spc(kit.Char('D'))),
			},
			{
				Name:      actGlobalSearch,
				DocString: "Global search in workspace folder",
				Run: kit.Continuation(
					model.PickerAction(NewGlobalSearchPicker),
				),
				Modes: []string{"NOR", "SEL"},
				Keys:  kit.Keys(spc(kit.Char('/'))),
			},
		},
	}
}

func bufferPickerOptions(cfg BufferPickerOptions) BufferPickerOptions {
	if cfg.StartPosition == "" {
		cfg.StartPosition = PickerStartTop
	}
	return cfg
}

func fileExplorerOptions(cfg fileExplorerConfig) FileExplorerOptions {
	opts := DefaultFileExplorerOptions()
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
