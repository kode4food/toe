package defaults

import (
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
)

const (
	actFilePicker      = "file_picker"
	actFilePickerInCWD = "file_picker_in_current_dir"

	actFileExplorer         = "file_explorer"
	actFileExplorerInBufDir = "file_explorer_in_current_buffer_directory"
	actBufferPicker         = "buffer_picker"
	actJumplistPicker       = "jumplist_picker"

	actGlobalSearch   = "global_search"
	actCommandPalette = "command_palette"
	actLastPicker     = "last_picker"
)

func pickerModule(model ui.Model) command.Module {
	spc := prefixed(char(' '))

	return command.Module{
		Commands: map[string]command.Command{
			actFilePicker: {
				DocString: "Open file picker",
				Run:       Continuation(model.PickerAction(ui.FilePicker)),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(char('f'))),
			},
			actFilePickerInCWD: {
				DocString: "Open file picker at current working directory",
				Run:       Continuation(model.PickerAction(ui.FilePickerInCWD)),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(char('F'))),
			},
			actFileExplorer: {
				DocString: "Open file explorer at workspace root",
				Run:       Continuation(model.PickerAction(ui.FileExplorer)),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(char('e'))),
			},
			actFileExplorerInBufDir: {
				DocString: "Open file explorer at current buffer's directory",
				Run: Continuation(model.PickerAction(
					ui.FileExplorerInBufferDir,
				)),
				Modes: []string{"NOR", "SEL"},
				Keys:  keys(spc(char('.'))),
			},
			actBufferPicker: {
				DocString: "Open buffer picker",
				Run:       Continuation(model.PickerAction(ui.BufferPicker)),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(char('b'))),
			},
			actJumplistPicker: {
				DocString: "Open jumplist picker",
				Run:       Continuation(model.PickerAction(ui.JumplistPicker)),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(char('j'))),
			},
			actGlobalSearch: {
				DocString: "Global search in workspace folder",
				Run:       Continuation(model.GlobalSearchAction()),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(char('/'))),
			},
			actCommandPalette: {
				DocString: "Open command palette",
				Run:       Continuation(model.CommandPaletteAction()),
				Aliases:   []string{"command-palette"},
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(char('?'))),
			},
			actLastPicker: {
				DocString: "Reopen the last picker",
				Run:       Continuation(model.LastPickerAction()),
				Aliases:   []string{"last-picker"},
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(spc(char('\''))),
			},
		},
	}
}
