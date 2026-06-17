// Package command defines the command registry types for the editor
package command

import "github.com/kode4food/toe/internal/view"

type (
	// Command describes one registered command: its runner, key bindings,
	// mode applicability, typeable aliases, and argument signature
	Command struct {
		Run       Run
		DocString string
		Modes     []string
		Keys      []KeyBinding
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
