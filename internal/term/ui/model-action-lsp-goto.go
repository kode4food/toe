package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

type locationGetter func(
	view.LanguageServerController, *view.Document, view.Id,
) ([]view.Location, error)

// GotoDeclarationAction jumps to the declaration of the symbol at the cursor
func (m Model) GotoDeclarationAction(e *view.Editor) {
	m.gotoLocation(e,
		i18n.Text(i18n.StatusNoDeclaration),
		view.LanguageServerController.GotoDeclaration,
	)
}

// GotoDefinitionAction jumps to the definition of the symbol at the cursor
func (m Model) GotoDefinitionAction(e *view.Editor) {
	m.gotoLocation(e,
		i18n.Text(i18n.StatusNoDefinition),
		view.LanguageServerController.GotoDefinition,
	)
}

// GotoTypeDefinitionAction jumps to the type of the symbol at the cursor
func (m Model) GotoTypeDefinitionAction(e *view.Editor) {
	m.gotoLocation(e,
		i18n.Text(i18n.StatusNoTypeDefinition),
		view.LanguageServerController.GotoTypeDefinition,
	)
}

// GotoImplementationAction jumps to implementations of the symbol at the cursor
func (m Model) GotoImplementationAction(e *view.Editor) {
	m.gotoLocation(e,
		i18n.Text(i18n.StatusNoImplementation),
		view.LanguageServerController.GotoImplementation,
	)
}

// GotoReferenceAction lists references to the symbol at the cursor
func (m Model) GotoReferenceAction(e *view.Editor) {
	m.gotoLocationPicker(e, view.LanguageServerController.GotoReference)
}

// SelectReferencesAction selects every reference to the symbol at the cursor
func (m Model) SelectReferencesAction(e *view.Editor) {
	doc := e.FocusedDocument()
	if doc == nil {
		return
	}
	v := e.FocusedView()
	if v == nil {
		return
	}
	ls := e.LanguageServerController()
	if ls == nil {
		e.SetStatusMsg(i18n.Text(i18n.StatusLSPNoHighlights))
		return
	}
	highlights, err := ls.DocumentHighlights(doc, v.ID())
	if err != nil {
		e.SetStatusMsg(i18n.ErrorText(err))
		return
	}
	if len(highlights) == 0 {
		e.SetStatusMsg(i18n.Text(i18n.StatusNoSymbolReferences))
		return
	}
	setSelectionFromHighlights(doc, v.ID(), highlights)
}

func (m Model) gotoLocation(
	e *view.Editor, notFound string, get locationGetter,
) {
	ec := m.component
	cx := m.context
	doc := e.FocusedDocument()
	if doc == nil {
		return
	}
	v := e.FocusedView()
	if v == nil {
		return
	}
	ls := e.LanguageServerController()
	if ls == nil {
		e.SetStatusMsg(i18n.Text(i18n.StatusLSPNoNavigation))
		return
	}
	locations, err := get(ls, doc, v.ID())
	if err != nil {
		e.SetStatusMsg(i18n.ErrorText(err))
		return
	}
	switch len(locations) {
	case 0:
		e.SetStatusMsg(notFound)
	case 1:
		jumpToLocation(e, locations[0])
	default:
		opener := locationPickerLayer(func() ([]view.Location, error) {
			return locations, nil
		})
		cx.lastLayer = opener
		ec.queueNextLayer(opener(e))
	}
}

func (m Model) gotoLocationPicker(e *view.Editor, get locationGetter) {
	ec := m.component
	cx := m.context
	doc := e.FocusedDocument()
	if doc == nil {
		return
	}
	v := e.FocusedView()
	if v == nil {
		return
	}
	ls := e.LanguageServerController()
	if ls == nil {
		e.SetStatusMsg(i18n.Text(i18n.StatusLSPNoNavigation))
		return
	}
	viewID := v.ID()
	opener := locationPickerLayer(func() ([]view.Location, error) {
		return get(ls, doc, viewID)
	})
	cx.lastLayer = opener
	ec.queueNextLayer(opener(e))
}

func locationPickerLayer(
	request locationRequest,
) func(*view.Editor) layerFunc {
	return func(e *view.Editor) layerFunc {
		p := newLSPLocationPicker(e, request)
		cmd := p.load.feedCmd
		p.load.feedCmd = nil
		return func(cx *Context) (Component, tea.Cmd) {
			return newPickerComponent(cx, p), cmd
		}
	}
}

func jumpToLocation(e *view.Editor, loc view.Location) {
	action.SaveSelection(e)
	v, err := e.OpenFile(loc.Path)
	if err != nil {
		e.SetStatusMsg(i18n.ErrorText(err))
		return
	}
	doc := e.FocusedDocument()
	if doc == nil {
		return
	}
	if sel, ok := locationSelection(doc.Text(), loc); ok {
		doc.SetSelectionFor(v.ID(), sel)
	}
}

func setSelectionFromHighlights(
	doc *view.Document, viewID view.Id, highlights []view.DocumentHighlight,
) {
	text := doc.Text()
	cursor := doc.SelectionFor(viewID).Primary().Cursor(text)
	ranges := make([]core.Range, 0, len(highlights))
	primary := 0
	for i, h := range highlights {
		r := core.Range{Anchor: h.From, Head: h.To}
		if r.Contains(cursor) {
			primary = i
		}
		ranges = append(ranges, r)
	}
	if sel, err := core.NewSelection(ranges, primary); err == nil {
		doc.SetSelectionFor(viewID, sel)
	}
}
