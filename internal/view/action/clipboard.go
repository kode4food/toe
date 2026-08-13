package action

import (
	"errors"
	"os"

	"github.com/charmbracelet/x/ansi"

	"github.com/kode4food/toe/internal/view"
)

// TTYWriter writes text via OSC 52 to a bound device. Returns true if the write
// succeeded
type TTYWriter func(text string, primary bool) bool

const (
	clipboardRegister        = '+'
	primaryClipboardRegister = '*'
)

var (
	ErrNoClipboardProvider = errors.New("no clipboard provider found")
)

// YankToClipboard copies all selection text to the system clipboard
func YankToClipboard(e *view.Editor) {
	values := selectionFragments(e)
	if len(values) == 0 {
		return
	}
	e.WriteRegister(clipboardRegister, values)
	e.SetMode(view.ModeNormal)
}

// YankMain copies only the primary selection to the active register
func YankMain(e *view.Editor) {
	v := e.FocusedView()
	if v == nil {
		return
	}
	doc := e.FocusedDocument()
	if doc == nil {
		return
	}
	sel := doc.SelectionFor(v.ID())
	frag, err := sel.Primary().Fragment(doc.Text())
	if err != nil {
		return
	}
	writeYank(e, []string{frag})
}

// YankToPrimaryClipboard copies all selections to the primary clipboard
func YankToPrimaryClipboard(e *view.Editor) {
	values := selectionFragments(e)
	if len(values) == 0 {
		return
	}
	e.WriteRegister(primaryClipboardRegister, values)
	e.SetMode(view.ModeNormal)
}

// PastePrimaryClipboardAfter reads the primary clipboard and pastes after each
// selection
func PastePrimaryClipboardAfter(e *view.Editor) {
	withPrimaryClipboard(e, PasteAfter)
}

// PastePrimaryClipboardBefore reads the primary clipboard and pastes before
// each selection
func PastePrimaryClipboardBefore(e *view.Editor) {
	withPrimaryClipboard(e, PasteBefore)
}

// PrimaryClipboardReplace replaces each selection with the primary clipboard
func PrimaryClipboardReplace(e *view.Editor) {
	withPrimaryClipboard(e, ReplaceWithYanked)
}

func selectionFragments(e *view.Editor) []string {
	v := e.FocusedView()
	if v == nil {
		return nil
	}
	doc := e.FocusedDocument()
	if doc == nil {
		return nil
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	values := make([]string, 0, len(sel.Ranges()))
	for _, r := range sel.Ranges() {
		if frag, err := r.Fragment(text); err == nil {
			values = append(values, frag)
		}
	}
	return values
}

func withPrimaryClipboard(e *view.Editor, fn func(*view.Editor)) {
	if len(e.ReadRegister(primaryClipboardRegister)) == 0 {
		return
	}
	prev := e.ActiveRegister()
	e.SetRegister(primaryClipboardRegister)
	fn(e)
	e.SetRegister(prev)
}

func writeTTY(text string, primary bool) bool {
	f, err := os.OpenFile(ttyDevice, os.O_WRONLY, 0)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	var selection byte = ansi.SystemClipboard
	if primary {
		selection = ansi.PrimaryClipboard
	}
	_, err = f.WriteString(ansi.SetClipboard(selection, text))
	return err == nil
}
