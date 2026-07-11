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
	}

	// Option describes a runtime editor option owned by a module. Toggle is
	// nil for options that are not boolean-toggleable
	Option struct {
		Key    string
		Get    OptionGetter
		Set    OptionSetter
		KeyGet OptionKeyGetter
		KeySet OptionKeySetter
		Toggle OptionGetter
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
		Config any // *ConcreteConfig, pre-filled with defaults
		Reset  func()
		Apply  func(*view.Editor)
	}

	// Command describes one registered command: its runner, key bindings,
	// mode applicability, typeable aliases, and argument signature
	Command struct {
		Name      string
		Run       Run
		DocString string
		Modes     []string
		Keys      map[string][]KeyBinding
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
		Continuation Continuation
	}

	// Action is a function that performs an operation on an editor
	Action func(*view.Editor)

	// Signal is a post-execution application-level effect
	Signal int
)

const (
	SignalNone Signal = iota
	SignalQuit
	SignalClearScreen
)
