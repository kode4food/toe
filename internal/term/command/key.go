package command

import (
	"fmt"
	"strings"
	"unicode"

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
		Char rune
		// Special names the key when Char is 0
		Special Special
	}

	// KeyEvent is a key code combined with modifier state
	KeyEvent struct {
		Code KeyCode
		Mods KeyModifiers
	}

	// Continuation is called with subsequent keys while an action is in
	// progress. Returns nil to signal completion, or another Continuation to
	// consume more keys
	Continuation func(*view.Editor, KeyEvent) Continuation

	// KeyAction handles a key sequence and may return a continuation
	KeyAction func(*view.Editor) Continuation

	// KeyBinding describes default key sequences for a command
	KeyBinding [][]KeyEvent
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

func (s Special) String() string {
	if int(s) < len(specialNames) {
		return specialNames[s]
	}
	return ""
}

func (k KeyModifiers) Has(mod KeyModifiers) bool {
	return k&mod != 0
}

func (k KeyModifiers) HasOnly(mod KeyModifiers) bool {
	return k == mod
}

func (k KeyCode) String() string {
	if k.Char == ' ' {
		return "spc"
	}
	if k.Char != 0 {
		return string(k.Char)
	}
	return k.Special.String()
}

func (k KeyEvent) String() string {
	var parts []string
	if k.Mods.Has(ModCtrl) {
		parts = append(parts, "C")
	}
	if k.Mods.Has(ModAlt) {
		parts = append(parts, "A")
	}
	if k.Mods.Has(ModShift) {
		if !k.Mods.HasOnly(ModShift) || !unicode.IsUpper(k.Code.Char) {
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
