// Package command defines the command registry types for the editor
package command

import "github.com/kode4food/toe/internal/view"

type (
	// Module groups a set of commands, runtime options, and an optional
	// config section. Options are registered into the editor option registry
	// when the module is installed
	Module struct {
		Commands []Command
		Options  []Option
		Section  *Section
		Labels   []PrefixLabel
	}

	// PrefixLabel names an intermediate key-sequence node for the pending-key
	// hint popup, letting a module label the prefixes it owns
	PrefixLabel struct {
		Modes view.Mode
		Seq   KeyBinding
		Label string
	}

	// Option describes a runtime editor option owned by a module. Toggle is
	// nil for options that are not boolean-toggleable
	Option struct {
		Key      string
		Get      OptionGetter
		Set      OptionSetter
		KeyGet   OptionKeyGetter
		KeySet   OptionKeySetter
		Toggle   OptionGetter
		Complete CompletionFunc
	}

	// OptionGetter reads an option's current value from the editor
	OptionGetter func(*view.Editor) (string, error)

	// OptionSetter applies a new option value to the editor
	OptionSetter func(*view.Editor, string) error

	// OptionKeyGetter reads concrete values owned by an option key prefix
	OptionKeyGetter func(*view.Editor) (map[string]string, error)

	// OptionKeySetter applies a concrete option key owned by a key prefix
	OptionKeySetter func(*view.Editor, string, string) error

	// Section declares a module's live config pointer and Apply hook
	Section struct {
		Config any
		Reset  func()
		Apply  func(*view.Editor)
	}

	// Command describes one registered command: its runner, key bindings, mode
	// applicability, typeable aliases, and argument signature
	Command struct {
		Name      string
		Run       Run
		DocString string
		Modes     view.Mode
		Keys      map[view.Mode]KeyBinding
		Aliases   []string
		Signature Signature
	}

	// KeyHint is a (key-string, label) pair used by the pending-key info popup
	KeyHint struct {
		Key   string
		Label string
	}

	// Run executes a registered command, optionally with parsed arguments
	Run func(*view.Editor, *Args) Result

	// Result is returned by a Run function
	Result struct {
		Signal       Signal
		Message      string
		Error        error
		Continuation Continuation
	}

	// Action is a function that performs an operation on an editor
	Action func(*view.Editor)

	// Signal is a post-execution application-level effect
	Signal int
)

const (
	_ Signal = iota
	SignalQuit
	SignalClearScreen
)

const (
	// AllModes is every editing and pane mode
	AllModes = view.ModeNormal |
		view.ModeSelect |
		view.ModeInsert |
		view.ModeTerminal |
		view.ModeImage |
		view.ModeBinary

	// DocNormalModes is the non-insert document modes
	DocNormalModes = view.ModeNormal |
		view.ModeSelect

	// DocModes is every mode backed by an editable document view
	DocModes = view.ModeNormal |
		view.ModeSelect |
		view.ModeInsert

	// PaneModes is every mode for commands that apply to every pane kind
	PaneModes = view.ModeNormal |
		view.ModeSelect |
		view.ModeTerminal |
		view.ModeImage |
		view.ModeBinary

	// CmdKeyModes is every pane mode except terminal, where keystrokes
	// belong to the shell
	CmdKeyModes = view.ModeNormal |
		view.ModeSelect |
		view.ModeImage |
		view.ModeBinary
)
