package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

type locationGetter func(
	view.LanguageServerController, *view.Document, view.Id,
) ([]view.Location, error)

func (m Model) GotoDeclarationAction() command.KeyAction {
	return m.gotoLocationAction(
		i18n.Text(i18n.StatusNoDeclaration),
		view.LanguageServerController.GotoDeclaration,
	)
}

func (m Model) GotoDefinitionAction() command.KeyAction {
	return m.gotoLocationAction(
		i18n.Text(i18n.StatusNoDefinition),
		view.LanguageServerController.GotoDefinition,
	)
}

func (m Model) GotoTypeDefinitionAction() command.KeyAction {
	return m.gotoLocationAction(
		i18n.Text(i18n.StatusNoTypeDefinition),
		view.LanguageServerController.GotoTypeDefinition,
	)
}

func (m Model) GotoImplementationAction() command.KeyAction {
	return m.gotoLocationAction(
		i18n.Text(i18n.StatusNoImplementation),
		view.LanguageServerController.GotoImplementation,
	)
}

func (m Model) GotoReferenceAction() command.KeyAction {
	return m.gotoLocationAction(
		i18n.Text(i18n.StatusNoReferences),
		view.LanguageServerController.GotoReference,
	)
}

func (m Model) SelectReferencesAction() command.KeyAction {
	return func(e *view.Editor) command.Continuation {
		doc, ok := e.FocusedDocument()
		if !ok {
			return nil
		}
		v, ok := e.FocusedView()
		if !ok {
			return nil
		}
		ls := e.LanguageServerController()
		if ls == nil {
			e.SetStatusMsg(i18n.Text(i18n.StatusLSPNoHighlights))
			return nil
		}
		highlights, err := ls.DocumentHighlights(doc, v.ID())
		if err != nil {
			e.SetStatusMsg(i18n.ErrorText(err))
			return nil
		}
		if len(highlights) == 0 {
			e.SetStatusMsg(i18n.Text(i18n.StatusNoSymbolReferences))
			return nil
		}
		setSelectionFromHighlights(doc, v.ID(), highlights)
		return nil
	}
}

func (m Model) gotoLocationAction(
	notFound string, get locationGetter,
) command.KeyAction {
	ec := m.component
	cx := m.context
	return func(e *view.Editor) command.Continuation {
		doc, ok := e.FocusedDocument()
		if !ok {
			return nil
		}
		v, ok := e.FocusedView()
		if !ok {
			return nil
		}
		ls := e.LanguageServerController()
		if ls == nil {
			e.SetStatusMsg(i18n.Text(i18n.StatusLSPNoNavigation))
			return nil
		}
		locations, err := get(ls, doc, v.ID())
		if err != nil {
			e.SetStatusMsg(i18n.ErrorText(err))
			return nil
		}
		switch len(locations) {
		case 0:
			e.SetStatusMsg(notFound)
		case 1:
			jumpToLocation(e, locations[0])
		default:
			opener := locationPickerLayer(locations)
			cx.lastLayer = opener
			ec.keys.nextLayer = opener(e)
		}
		return nil
	}
}

func locationPickerLayer(
	locations []view.Location,
) func(*view.Editor) layerFunc {
	return func(e *view.Editor) layerFunc {
		p := newLSPLocationPicker(e, locations)
		cmd := p.load.feedCmd
		p.load.feedCmd = nil
		return func(_ *Context) (Component, tea.Cmd) {
			return newPickerComponent(p), cmd
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
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	sel, err := locationSelection(loc)
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), sel)
}

func setSelectionFromHighlights(
	doc *view.Document, viewID view.Id, highlights []view.DocumentHighlight,
) {
	text := doc.Text()
	cursor := doc.SelectionFor(viewID).Primary().Cursor(text)
	ranges := make([]core.Range, 0, len(highlights))
	primary := 0
	for i, h := range highlights {
		r := core.NewRange(h.From, h.To)
		if r.Contains(cursor) {
			primary = i
		}
		ranges = append(ranges, r)
	}
	sel, err := core.NewSelection(ranges, primary)
	if err != nil {
		return
	}
	doc.SetSelectionFor(viewID, sel)
}
