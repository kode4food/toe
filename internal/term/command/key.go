package command

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/view"
)

type (
	// KeyModifiers is a bitmask of modifier keys
	KeyModifiers uint8

	// Special enumerates the non-printable keys; SpecialNone means the key is
	// a printable [KeyCode.Char] instead
	Special uint8

	// KeyCode represents a single keyboard key
	KeyCode struct {
		// Char holds the rune for printable characters; 0 for special keys
		Char    rune
		Special Special
	}

	// KeyEvent is a key code combined with modifier state
	KeyEvent struct {
		Code KeyCode
		Mods KeyModifiers
	}

	// Continuation is called with subsequent keys while an action is in
	// progress. Its result tells the key-entry UI how the interaction moved.
	// ContinuationStay with nil retains the current callback
	Continuation func(*view.Editor, KeyEvent) (Continuation, Transition)

	// Transition describes how an interaction handled a key
	Transition uint8

	// KeyAction handles a key sequence and may return a continuation
	KeyAction func(*view.Editor) Continuation

	// KeyResultAction handles a key sequence and returns its command result
	KeyResultAction func(*view.Editor) Result

	// KeyBinding describes default key sequences for a command
	KeyBinding [][]KeyEvent
)

const (
	ContinuationDone Transition = iota
	ContinuationStay
	ContinuationPush
	ContinuationPop
)

const (
	ModNone  KeyModifiers = 0
	ModShift KeyModifiers = 1 << iota
	ModCtrl
	ModAlt
)

const (
	SpecialNone Special = iota
	SpecialUnknown
	Enter
	Backspace
	Delete
	Escape
	Tab
	Up
	Down
	Left
	Right
	Home
	End
	PageUp
	PageDown
)

var (
	// ErrInvalidKey reports malformed keycap notation
	ErrInvalidKey = i18n.NewError(i18n.ErrorInvalidKey)
)

// specialNames gives the compact keycap form for each special key, indexed by
// its [Special] value
var specialNames = []string{
	SpecialUnknown: "?",
	Enter:          "ret",
	Backspace:      "bksp",
	Delete:         "del",
	Escape:         "esc",
	Tab:            "tab",
	Up:             "up",
	Down:           "down",
	Left:           "left",
	Right:          "right",
	Home:           "home",
	End:            "end",
	PageUp:         "pgup",
	PageDown:       "pgdn",
}

// PopOnBackspace makes unmodified Backspace pop a continuation
func PopOnBackspace(handle Continuation) Continuation {
	return func(e *view.Editor, k KeyEvent) (Continuation, Transition) {
		if k.Code.Special == Backspace && k.Mods == ModNone {
			return nil, ContinuationPop
		}
		return handle(e, k)
	}
}

// ReadChar handles an unmodified character, or pops on Backspace
func ReadChar(handle func(*view.Editor, rune) Continuation) Continuation {
	return PopOnBackspace(func(
		e *view.Editor, k KeyEvent,
	) (Continuation, Transition) {
		if k.Code.Char == 0 || k.Mods != ModNone {
			return nil, ContinuationStay
		}
		if next := handle(e, k.Code.Char); next != nil {
			return next, ContinuationPush
		}
		return nil, ContinuationDone
	})
}

// String returns the binding name of a special key
func (s Special) String() string {
	if int(s) < len(specialNames) {
		return specialNames[s]
	}
	return ""
}

// Has reports whether every modifier in mod is set
func (k KeyModifiers) Has(mod KeyModifiers) bool {
	return k&mod != 0
}

// String returns the binding name of a key code
func (k KeyCode) String() string {
	if k.Char == ' ' {
		return "spc"
	}
	if k.Char != 0 {
		return string(k.Char)
	}
	return k.Special.String()
}

// String returns the binding notation for a key press
func (k KeyEvent) String() string {
	var parts []string
	if k.Mods.Has(ModCtrl) {
		parts = append(parts, "C")
	}
	if k.Mods.Has(ModAlt) {
		parts = append(parts, "A")
	}
	if k.Mods.Has(ModShift) {
		if k.Mods != ModShift || !unicode.IsUpper(k.Code.Char) {
			parts = append(parts, "S")
		}
	}
	s := k.Code.String()
	if len(parts) == 0 {
		return s
	}
	if k.Mods.Has(ModShift) && k.Code.Char != 0 {
		s = strings.ToLower(s)
	}
	return fmt.Sprintf("%s-%s", strings.Join(parts, "-"), s)
}

// WithMods returns a copy of k with the given modifiers added
func (k KeyEvent) WithMods(m KeyModifiers) KeyEvent {
	k.Mods |= m
	return k
}

// IsTypable reports whether k is a printable character that should be accepted
// as literal text input — Char is set and neither Ctrl nor Alt is held;
// ModShift alone is fine; it is already reflected in the Char value
func (k KeyEvent) IsTypable() bool {
	return k.Code.Char != 0 && !k.Mods.Has(ModCtrl) && !k.Mods.Has(ModAlt)
}

// ParseKeySequence parses a space-separated sequence of keycap names
func ParseKeySequence(input string) ([]KeyEvent, error) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil, ErrInvalidKey
	}
	res := make([]KeyEvent, len(parts))
	for i, part := range parts {
		key, err := parseKey(part)
		if err != nil {
			return nil, err
		}
		res[i] = key
	}
	return res, nil
}

func parseKey(input string) (KeyEvent, error) {
	s := input
	var mods KeyModifiers
	for len(s) > 2 {
		var mod KeyModifiers
		switch s[:2] {
		case "C-":
			mod = ModCtrl
		case "A-":
			mod = ModAlt
		case "S-":
			mod = ModShift
		default:
			goto parseCode
		}
		if mods.Has(mod) {
			return KeyEvent{}, fmt.Errorf("%w: %s", ErrInvalidKey, input)
		}
		mods |= mod
		s = s[2:]
	}

parseCode:
	if s == "spc" {
		return KeyEvent{Code: KeyCode{Char: ' '}, Mods: mods}, nil
	}
	for special, name := range specialNames {
		if special != int(SpecialUnknown) && name == s {
			return KeyEvent{
				Code: KeyCode{Special: Special(special)}, Mods: mods,
			}, nil
		}
	}
	runes := []rune(s)
	if len(runes) != 1 {
		return KeyEvent{}, fmt.Errorf("%w: %s", ErrInvalidKey, input)
	}
	ch := runes[0]
	if unicode.IsUpper(ch) {
		mods |= ModShift
	} else if mods.Has(ModShift) {
		ch = unicode.ToUpper(ch)
	}
	return KeyEvent{Code: KeyCode{Char: ch}, Mods: mods}, nil
}
