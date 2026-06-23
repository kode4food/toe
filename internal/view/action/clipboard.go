package action

import (
	"errors"
	"os/exec"
	"strings"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

const (
	clipboardRegister        = '+'
	primaryClipboardRegister = '*'
)

var (
	ErrNoClipboardProvider = errors.New("no clipboard provider found")
)

// ShowClipboardProvider returns the name of the detected clipboard tool
func ShowClipboardProvider() string {
	tools := []string{"pbcopy", "xclip", "xsel", "wl-copy"}
	for _, t := range tools {
		if _, err := exec.LookPath(t); err == nil {
			return t
		}
	}
	return "none"
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
	if doc.Readonly() {
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
	_ = e.Apply(core.NewTransaction(text).WithChanges(cs).WithSelection(newSel))
	e.SetMode(view.ModeNormal)
}

func YankToPrimaryClipboard(e *view.Editor) {
	values := selectionFragments(e)
	if len(values) == 0 {
		return
	}
	e.Registers().Write(primaryClipboardRegister, values)
	_ = writePrimaryClipboard(strings.Join(values, "\n"))
	e.SetMode(view.ModeNormal)
}

func PastePrimaryClipboardAfter(e *view.Editor) {
	withPrimaryClipboard(e, PasteAfter)
}

func PastePrimaryClipboardBefore(e *view.Editor) {
	withPrimaryClipboard(e, PasteBefore)
}

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

func tryReadCmds(cmds [][]string) (string, bool) {
	for _, args := range cmds {
		if out, err := exec.Command(args[0], args[1:]...).Output(); err == nil {
			return string(out), true
		}
	}
	return "", false
}

func tryWriteCmds(cmds [][]string, text string) bool {
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdin = strings.NewReader(text)
		if cmd.Run() == nil {
			return true
		}
	}
	return false
}

func readClipboard() (string, error) {
	if v, ok := tryReadCmds([][]string{
		{"pbpaste"},
		{"xclip", "-selection", "clipboard", "-o"},
		{"xsel", "--clipboard", "--output"},
		{"wl-paste", "--no-newline"},
	}); ok {
		return v, nil
	}
	return "", ErrNoClipboardProvider
}

func writeClipboard(text string) error {
	if tryWriteCmds([][]string{
		{"pbcopy"},
		{"xclip", "-selection", "clipboard"},
		{"xsel", "--clipboard", "--input"},
		{"wl-copy"},
	}, text) {
		return nil
	}
	return ErrNoClipboardProvider
}

func readPrimaryClipboard() (string, error) {
	if v, ok := tryReadCmds([][]string{
		{"xclip", "-selection", "primary", "-o"},
		{"xsel", "--primary", "--output"},
		{"wl-paste", "--primary", "--no-newline"},
	}); ok {
		return v, nil
	}
	return readClipboard()
}

func writePrimaryClipboard(text string) error {
	if tryWriteCmds([][]string{
		{"xclip", "-selection", "primary"},
		{"xsel", "--primary", "--input"},
		{"wl-copy", "--primary"},
	}, text) {
		return nil
	}
	return writeClipboard(text)
}
