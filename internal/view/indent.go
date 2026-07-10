package view

// Indenter computes indentation for a new line at pos in doc
type Indenter func(doc *Document, line, pos int) (string, bool)

// SetIndenter installs syntax-aware indentation support
func (e *Editor) SetIndenter(p Indenter) {
	e.indenter = p
}

// IndentForNewline returns syntax-aware indentation when a provider exists
func (e *Editor) IndentForNewline(doc *Document, line, pos int) (string, bool) {
	if e.indenter == nil {
		return "", false
	}
	return e.indenter(doc, line, pos)
}
