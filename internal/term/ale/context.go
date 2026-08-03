package ale

import (
	"github.com/kode4food/ale"
	"github.com/kode4food/ale/data"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// buildContext snapshots editor state for a binding action as a nested Ale
// object; absent optional keys are omitted so lookups can supply a default
func buildContext(e *view.Editor) *data.Object {
	pairs := []data.Pair{
		data.NewCons(kw("cwd"), data.String(e.Cwd())),
	}
	if p := e.FocusedPane(); p != nil {
		pairs = append(pairs, data.NewCons(kw("pane"), buildPane(e, p)))
	}
	return data.NewObject(pairs...)
}

func buildPane(e *view.Editor, p view.Pane) *data.Object {
	pairs := []data.Pair{
		data.NewCons(kw("kind"), paneKind(p)),
		data.NewCons(kw("mode"), modeKeyword(p.Mode())),
	}
	if path := p.Path(); path != "" {
		pairs = append(pairs, data.NewCons(kw("path"), data.String(path)))
	}
	if v, ok := p.(*view.View); ok {
		if doc := e.Document(v.DocID()); doc != nil {
			pairs = append(pairs,
				data.NewCons(kw("document"), buildDocument(doc)),
				data.NewCons(kw("selection"), buildSelection(doc, v.ID())),
			)
		}
	}
	return data.NewObject(pairs...)
}

func buildDocument(doc *view.Document) *data.Object {
	pairs := []data.Pair{
		data.NewCons(kw("name"), data.String(doc.DisplayName())),
		data.NewCons(kw("modified"), data.Bool(doc.Modified())),
		data.NewCons(kw("read-only"), data.Bool(doc.ReadOnly())),
	}
	if path := doc.Path(); path != "" {
		pairs = append(pairs, data.NewCons(kw("path"), data.String(path)))
	}
	if lang := doc.Lang(); lang != "" {
		pairs = append(pairs, data.NewCons(kw("language"), data.String(lang)))
	}
	return data.NewObject(pairs...)
}

func buildSelection(doc *view.Document, viewID view.Id) *data.Object {
	sel := doc.SelectionFor(viewID)
	text := doc.Text()
	ranges := sel.Ranges()
	values := make([]ale.Value, len(ranges))
	for i, r := range ranges {
		values[i] = rangeObject(r, text)
	}
	return data.NewObject(
		data.NewCons(kw("primary"), data.Integer(sel.PrimaryIndex())),
		data.NewCons(kw("ranges"), data.NewVector(values...)),
	)
}

// rangeObject builds a concrete Ale object for a range; :anchor and :head
// keep direction, :from/:to/:cursor are derived zero-based offsets
func rangeObject(r core.Range, text core.Rope) *data.Object {
	return data.NewObject(
		data.NewCons(kw("anchor"), data.Integer(r.Anchor)),
		data.NewCons(kw("head"), data.Integer(r.Head)),
		data.NewCons(kw("from"), data.Integer(r.From())),
		data.NewCons(kw("to"), data.Integer(r.To())),
		data.NewCons(kw("cursor"), data.Integer(r.Cursor(text))),
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

func kw(name string) data.Keyword {
	return data.Keyword(name)
}
