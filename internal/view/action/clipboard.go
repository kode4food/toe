package action

import (
	"errors"
	"os"
	"strings"

	"github.com/aymanbagabas/go-osc52/v2"

	"github.com/kode4food/toe/internal/core"
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

// MakeTTYAvailable returns a function that reports whether the TTY can be
// opened for writing. Call once at the term layer; pass the result where needed
func MakeTTYAvailable() func() bool {
	return func() bool {
		f, err := os.OpenFile(ttyDevice, os.O_WRONLY, 0)
		if err != nil {
			return false
		}
		_ = f.Close()
		return true
	}
}

// MakeTTYWriter returns a function that writes text via OSC 52 to the TTY.
// Returns false if the TTY is unavailable or the write fails. Call once at the
// term layer; pass the result where needed
func MakeTTYWriter() TTYWriter {
	return func(text string, primary bool) bool {
		f, err := os.OpenFile(ttyDevice, os.O_WRONLY, 0)
		if err != nil {
			return false
		}
		defer func() { _ = f.Close() }()
		seq := osc52.New(text)
		if primary {
			seq = seq.Primary()
		}
		_, err = seq.WriteTo(f)
		return err == nil
	}
}

// YankToClipboard copies all selection text to the system clipboard
func YankToClipboard(e *view.Editor) {
	values := selectionFragments(e)
	if len(values) == 0 {
		return
	}
	e.Registers().Write(clipboardRegister, values)
	_ = writeClipboard(strings.Join(values, "\n"))
	e.SetMode(view.ModeNormal)
}

// YankMainToClipboard copies only the primary selection to clipboard
func YankMainToClipboard(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	frag, err := sel.Primary().Fragment(text)
	if err != nil {
		return
	}
	e.Registers().Write(clipboardRegister, []string{frag})
	_ = writeClipboard(frag)
	e.SetMode(view.ModeNormal)
}

// PasteClipboardAfter reads the clipboard and pastes after each selection
func PasteClipboardAfter(e *view.Editor) {
	val, err := readClipboard()
	if err != nil || val == "" {
		return
	}
	e.Registers().Write(clipboardRegister, []string{val})
	old := e.ActiveRegister()
	e.SetRegister(clipboardRegister)
	pasteImpl(e, false)
	e.SetRegister(old)
	e.SetMode(view.ModeNormal)
}

// PasteClipboardBefore reads the clipboard and pastes before each selection
func PasteClipboardBefore(e *view.Editor) {
	val, err := readClipboard()
	if err != nil || val == "" {
		return
	}
	e.Registers().Write(clipboardRegister, []string{val})
	old := e.ActiveRegister()
	e.SetRegister(clipboardRegister)
	pasteImpl(e, true)
	e.SetRegister(old)
	e.SetMode(view.ModeNormal)
}

// ClipboardReplace replaces each selection with the clipboard
func ClipboardReplace(e *view.Editor) {
	val, err := readClipboard()
	if err != nil || val == "" {
		return
	}
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	if doc.ReadOnly() {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	changes := make([]core.Change, 0, len(ranges))
	for _, r := range ranges {
		changes = append(changes, core.TextChange(r.From(), r.To(), val))
	}
	cs, err2 := core.NewChangeSetFromChanges(text, changes)
	if err2 != nil {
		return
	}
	newSel, err2 := sel.Map(cs)
	if err2 != nil {
		return
	}
	_ = e.Apply(
		core.NewTransaction(text).WithChanges(cs).WithSelection(newSel),
	)
	e.SetMode(view.ModeNormal)
}

// YankToPrimaryClipboard copies all selections to the primary clipboard
func YankToPrimaryClipboard(e *view.Editor) {
	values := selectionFragments(e)
	if len(values) == 0 {
		return
	}
	e.Registers().Write(primaryClipboardRegister, values)
	_ = writePrimaryClipboard(strings.Join(values, "\n"))
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
	v, ok := e.FocusedView()
	if !ok {
		return nil
	}
	doc, ok := e.FocusedDocument()
	if !ok {
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
	val, err := readPrimaryClipboard()
	if err != nil {
		e.SetStatusMsg("error: " + err.Error())
		return
	}
	e.Registers().Write(primaryClipboardRegister, []string{val})
	prev := e.ActiveRegister()
	e.SetRegister(primaryClipboardRegister)
	fn(e)
	e.SetRegister(prev)
}
