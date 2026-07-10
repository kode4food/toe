package view

import (
	"strconv"
	"strings"
)

const (
	RegisterSearch           = '/'
	RegisterDefaultYank      = '"'
	RegisterSelectionIndices = '#'
	RegisterSelectionText    = '.'
	RegisterDocumentPath     = '%'
	RegisterClipboard        = '+'
	RegisterPrimaryClipboard = '*'
	RegisterBlackHole        = '_'
)

// ReadRegister returns regular and computed register contents for the current
// editor state
func (e *Editor) ReadRegister(name rune) []string {
	switch name {
	case RegisterBlackHole:
		return nil
	case RegisterSelectionIndices:
		return e.selectionIndexRegister()
	case RegisterSelectionText:
		return e.selectionTextRegister()
	case RegisterDocumentPath:
		return e.documentPathRegister()
	case RegisterClipboard, RegisterPrimaryClipboard:
		return e.clipboardRegister(name)
	default:
		return e.registers.Read(name)
	}
}

// FirstRegister returns the first value from ReadRegister
func (e *Editor) FirstRegister(name rune) (string, bool) {
	vals := e.ReadRegister(name)
	if len(vals) == 0 {
		return "", false
	}
	return vals[0], true
}

// WriteRegister stores regular register contents, syncing special clipboard
// registers to the system clipboard provider
func (e *Editor) WriteRegister(name rune, values []string) {
	switch name {
	case RegisterBlackHole:
		return
	case RegisterSelectionIndices, RegisterSelectionText, RegisterDocumentPath:
		return
	case RegisterClipboard:
		e.registers.Write(name, values)
		_ = e.clipboard.Write(strings.Join(values, "\n"))
	case RegisterPrimaryClipboard:
		e.registers.Write(name, values)
		_ = e.clipboard.WritePrimary(strings.Join(values, "\n"))
	default:
		e.registers.Write(name, values)
	}
}

func (e *Editor) selectionIndexRegister() []string {
	v, ok := e.FocusedView()
	if !ok {
		return nil
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return nil
	}
	n := len(doc.SelectionFor(v.ID()).Ranges())
	out := make([]string, n)
	for i := range out {
		out[i] = strconv.Itoa(i + 1)
	}
	return out
}

func (e *Editor) selectionTextRegister() []string {
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
	out := make([]string, 0, len(sel.Ranges()))
	for _, r := range sel.Ranges() {
		frag, err := r.Fragment(text)
		if err != nil {
			continue
		}
		out = append(out, frag)
	}
	return out
}

func (e *Editor) documentPathRegister() []string {
	doc, ok := e.FocusedDocument()
	if !ok {
		return nil
	}
	return []string{doc.Path()}
}

func (e *Editor) clipboardRegister(name rune) []string {
	var val string
	var err error
	if name == RegisterClipboard {
		val, err = e.clipboard.Read()
	} else {
		val, err = e.clipboard.ReadPrimary()
	}
	saved := e.registers.Read(name)
	if err != nil {
		return saved
	}
	if registerContentsMatch(saved, val) {
		return saved
	}
	if val == "" {
		return saved
	}
	return []string{val}
}

func registerContentsMatch(values []string, content string) bool {
	if len(values) == 0 {
		return content == ""
	}
	return strings.Join(values, "\n") == content
}
