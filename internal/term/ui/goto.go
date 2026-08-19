package ui

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

// GotoSelector resolves the selection to place once the destination document is
// open; a nil selector leaves the document's selection alone
type GotoSelector func(core.Rope) (core.Selection, bool)

// GotoDocument navigates the pane to an open document, placing the selection
// the selector resolves and scrolling it into view
func GotoDocument(
	e *view.Editor, id view.DocumentId, sel GotoSelector,
	accept PickerAcceptAction,
) (*view.View, bool) {
	v := acceptDocumentID(e, id, accept)
	if v == nil {
		return nil, false
	}
	return gotoLanded(e, v, sel)
}

// GotoPath navigates the pane to a file, switching to it when it is already
// open, placing the selection the selector resolves and scrolling it into
// view. Image and binary files open as their own panes and take no selection
func GotoPath(
	e *view.Editor, path string, sel GotoSelector, accept PickerAcceptAction,
) (*view.View, bool) {
	if path == "" {
		return nil, false
	}
	v, ok, err := OpenPath(e, path, accept)
	if err != nil {
		e.SetStatusMsg(i18n.ErrorText(err))
		return nil, false
	}
	// an image or binary file opened as its own pane; no cursor to place
	if !ok || v == nil {
		return nil, ok
	}
	return gotoLanded(e, v, sel)
}

// GotoJump navigates to the jump list entry at index, moving the list head onto
// it instead of recording a new jump, so the history either side of it
// survives. A split accept opens a new pane and records a jump as usual
func GotoJump(
	e *view.Editor, index int, accept PickerAcceptAction,
) (*view.View, bool) {
	v := e.FocusedView()
	if v == nil {
		return nil, false
	}
	entry, ok := v.JumpTo(index)
	if !ok {
		return nil, false
	}
	sel := GotoSelection(entry.Selection)
	if accept != PickerAcceptReplace {
		return GotoDocument(e, entry.DocID, sel, accept)
	}
	if entry.DocID != v.DocID() && !e.SwitchBuffer(entry.DocID) {
		return nil, false
	}
	return gotoLanded(e, v, sel)
}

// GotoSelection returns a selector that always resolves to sel
func GotoSelection(sel core.Selection) GotoSelector {
	return func(core.Rope) (core.Selection, bool) {
		return sel, true
	}
}

// GotoLines returns a selector covering the first line of lines
func GotoLines(lines *core.Span) GotoSelector {
	return func(text core.Rope) (core.Selection, bool) {
		return lineRangeSelection(text, lines)
	}
}

// acceptDocumentID shows the document by id, splitting per accept, and
// records the departed position on the jump list
func acceptDocumentID(
	e *view.Editor, id view.DocumentId, accept PickerAcceptAction,
) *view.View {
	action.SaveSelection(e)
	switch accept {
	case PickerAcceptHorizontalSplit:
		return e.HSplit(id)
	case PickerAcceptVerticalSplit:
		return e.VSplit(id)
	default:
		return e.ShowDocument(id)
	}
}

func alignAcceptedView(e *view.Editor, v *view.View, doc *view.Document) {
	cs := &view.CursorScroll{
		Doc:       doc.Text(),
		Selection: doc.SelectionFor(v.ID()),
		Height:    max(v.Area().Height, e.ViewHeight()),
		Width:     e.ViewContentWidth(),
		TabWidth:  doc.TabWidth(),
		ScrollOff: e.Options().ScrollOff,
	}
	v.EnsureCursorVisible(cs)
	v.EnsureCursorVisibleHorizontal(cs)
}

func gotoLanded(
	e *view.Editor, v *view.View, sel GotoSelector,
) (*view.View, bool) {
	doc := e.Document(v.DocID())
	if doc == nil {
		return nil, false
	}
	if sel != nil {
		if landed, ok := sel(doc.Text()); ok {
			doc.SetSelectionFor(v.ID(), landed)
		}
	}
	alignAcceptedView(e, v, doc)
	return v, true
}
