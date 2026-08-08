package view

// Indenter computes indentation for a new line at pos in doc
type Indenter func(doc *Document, line, pos int) (string, bool)

// SetIndenter installs syntax-aware indentation support
func (e *Editor) SetIndenter(p Indenter) {
	e.indenter = p
}

// IndentForNewlineArgs is the document and the point a newline is inserted at
type IndentForNewlineArgs struct {
	Doc  *Document
	Line int
	Pos  int
}

// IndentForNewline returns syntax-aware indentation when a provider exists
func (e *Editor) IndentForNewline(args IndentForNewlineArgs) (string, bool) {
	if e.indenter == nil {
		return "", false
	}
	return e.indenter(args.Doc, args.Line, args.Pos)
}
