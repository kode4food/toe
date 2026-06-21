package command

import (
	"fmt"
	"strings"

	"github.com/kode4food/toe/internal/view"
)

type (
	// KeyModifiers is a bitmask of modifier keys
	KeyModifiers uint8

	// KeyCode represents a single keyboard key
	KeyCode struct {
		// Char holds the rune for printable characters; 0 for special keys
		Char rune
		// Special holds the special key name when Char is 0
		Special string
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

func (k KeyModifiers) Has(mod KeyModifiers) bool {
	return k&mod != 0
}

func (k KeyCode) String() string {
	if k.Char != 0 {
		return string(k.Char)
	}
	return k.Special
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
		parts = append(parts, "S")
	}
	s := k.Code.String()
	if len(parts) == 0 {
		return s
	}
	return fmt.Sprintf("<%s-%s>", strings.Join(parts, "-"), s)
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
