package ale

import (
	"github.com/kode4food/ale"
	"github.com/kode4food/ale/data"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// contextValue is a live, read-only view of editor state for Ale. Each lookup
// reads current state and builds only the requested branch
type (
	contextValue struct {
		editor *view.Editor
	}

	paneValue struct {
		editor *view.Editor
		pane   view.Pane
	}

	documentValue struct {
		doc *view.Document
	}

	selectionValue struct {
		doc    *view.Document
		viewID view.Id
	}
)

const (
	keyCWD       data.Keyword = "cwd"
	keyPane      data.Keyword = "pane"
	keyKind      data.Keyword = "kind"
	keyMode      data.Keyword = "mode"
	keyPath      data.Keyword = "path"
	keyDocument  data.Keyword = "document"
	keySelection data.Keyword = "selection"
	keyName      data.Keyword = "name"
	keyLanguage  data.Keyword = "language"
	keyModified  data.Keyword = "modified"
	keyReadOnly  data.Keyword = "read-only"
	keyPrimary   data.Keyword = "primary"
	keyRanges    data.Keyword = "ranges"
	keyAnchor    data.Keyword = "anchor"
	keyHead      data.Keyword = "head"
	keyFrom      data.Keyword = "from"
	keyTo        data.Keyword = "to"
	keyCursor    data.Keyword = "cursor"
)

// Get resolves an editor context field for a script
func (c contextValue) Get(key ale.Value) (ale.Value, bool) {
	switch key {
	case keyCWD:
		return data.String(c.editor.Cwd()), true
	case keyPane:
		if p := c.editor.FocusedPane(); p != nil {
			return paneValue{editor: c.editor, pane: p}, true
		}
	}
	return data.Null, false
}

// Equal reports whether other is the same context
func (c contextValue) Equal(other ale.Value) bool {
	o, ok := other.(contextValue)
	return ok && o == c
}

// Get resolves a pane field for a script
func (p paneValue) Get(key ale.Value) (ale.Value, bool) {
	switch key {
	case keyKind:
		return paneKind(p.pane), true
	case keyMode:
		return modeKeyword(p.pane.Mode()), true
	case keyPath:
		if path := p.pane.Path(); path != "" {
			return data.String(path), true
		}
	case keyDocument:
		if doc, _, ok := p.document(); ok {
			return documentValue{doc: doc}, true
		}
	case keySelection:
		if doc, id, ok := p.document(); ok {
			return selectionValue{doc: doc, viewID: id}, true
		}
	}
	return data.Null, false
}

// Equal reports whether other wraps the same pane
func (p paneValue) Equal(other ale.Value) bool {
	o, ok := other.(paneValue)
	return ok && o == p
}

func (p paneValue) document() (*view.Document, view.Id, bool) {
	v, ok := p.pane.(*view.View)
	if !ok {
		return nil, 0, false
	}
	doc := p.editor.Document(v.DocID())
	if doc == nil {
		return nil, 0, false
	}
	return doc, v.ID(), true
}

// Get resolves a document field for a script
func (d documentValue) Get(key ale.Value) (ale.Value, bool) {
	switch key {
	case keyName:
		return data.String(d.doc.DisplayName()), true
	case keyPath:
		if path := d.doc.Path(); path != "" {
			return data.String(path), true
		}
	case keyLanguage:
		if lang := d.doc.Lang(); lang != "" {
			return data.String(lang), true
		}
	case keyModified:
		return data.Bool(d.doc.Modified()), true
	case keyReadOnly:
		return data.Bool(d.doc.ReadOnly()), true
	}
	return data.Null, false
}

// Equal reports whether other wraps the same document
func (d documentValue) Equal(other ale.Value) bool {
	o, ok := other.(documentValue)
	return ok && o == d
}

// Get resolves a selection field for a script
func (s selectionValue) Get(key ale.Value) (ale.Value, bool) {
	sel := s.doc.SelectionFor(s.viewID)
	switch key {
	case keyPrimary:
		return data.Integer(sel.PrimaryIndex()), true
	case keyRanges:
		text := s.doc.Text()
		ranges := sel.Ranges()
		values := make([]ale.Value, len(ranges))
		for i, r := range ranges {
			values[i] = rangeObject(r, text)
		}
		return data.NewVector(values...), true
	}
	return data.Null, false
}

// Equal reports whether other wraps the same selection
func (s selectionValue) Equal(other ale.Value) bool {
	o, ok := other.(selectionValue)
	return ok && o == s
}

func buildContext(e *view.Editor) contextValue {
	return contextValue{editor: e}
}

func rangeObject(r core.Range, text core.Rope) *data.Object {
	return data.NewObject(
		data.NewCons(keyAnchor, data.Integer(r.Anchor)),
		data.NewCons(keyHead, data.Integer(r.Head)),
		data.NewCons(keyFrom, data.Integer(r.From())),
		data.NewCons(keyTo, data.Integer(r.To())),
		data.NewCons(keyCursor, data.Integer(r.Cursor(text))),
	)
}

func paneKind(p view.Pane) data.Keyword {
	if _, ok := p.(*view.View); ok {
		return "view"
	}
	if p.Mode() == view.ModeImage {
		return "image"
	}
	return "terminal"
}

func modeKeyword(mode view.Mode) data.Keyword {
	switch mode {
	case view.ModeInsert:
		return modeInsert
	case view.ModeSelect:
		return modeSelect
	case view.ModeTerminal:
		return modeTerminal
	case view.ModeImage:
		return modeImage
	default:
		return modeNormal
	}
}
