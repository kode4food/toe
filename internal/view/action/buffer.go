package action

import (
	"strings"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// GotoLastAccessedFile switches to the most recently accessed alternate file
func GotoLastAccessedFile(e *view.Editor) {
	did, ok := e.PopPrevDocID()
	if !ok {
		return
	}
	for _, v := range e.AllViews() {
		if v.DocID() == did {
			e.FocusView(v.ID())
			return
		}
	}
}

// GotoLastModifiedFile switches focus to the most recently modified document
// that is not currently focused
func GotoLastModifiedFile(e *view.Editor) {
	curDID := view.InvalidDocumentId
	if v, ok := e.FocusedView(); ok {
		curDID = v.DocID()
	}
	ids := e.LastModifiedDocIDs()
	for _, did := range ids {
		if did == view.InvalidDocumentId || did == curDID {
			continue
		}
		for _, v := range e.AllViews() {
			if v.DocID() == did {
				e.FocusView(v.ID())
				return
			}
		}
	}
}

// RepeatLastMotion replays the most recently recorded repeatable motion
func RepeatLastMotion(e *view.Editor) {
	fn := e.LastMotion()
	if fn == nil {
		return
	}
	n := max(e.Count(), 1)
	for range n {
		fn(e)
	}
}

// ExtendToColumn extends each selection to the Nth character column
func ExtendToColumn(e *view.Editor) {
	gotoColumn(e, true)
}

// PasteRegisterAtCursor inserts the contents of the given register at each
// cursor position (for use in insert mode)
func PasteRegisterAtCursor(e *view.Editor, reg rune) {
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
	val, ok := e.FirstRegister(reg)
	if !ok {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	changes := make([]core.Change, 0, len(ranges))
	seen := map[int]bool{}
	for _, r := range ranges {
		pos := r.Cursor(text)
		if seen[pos] {
			continue
		}
		seen[pos] = true
		changes = append(changes, core.TextChange(pos, pos, val))
	}
	if len(changes) == 0 {
		return
	}
	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return
	}
	newSel, err := sel.Map(cs)
	if err != nil {
		return
	}
	_ = e.Apply(core.NewTransaction(text).WithChanges(cs).WithSelection(newSel))
}

// YankJoin yanks all selection text joined by a separator to the active
// register (default '"'). Mirrors :yank-join
func YankJoin(e *view.Editor, sep string) {
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
	reg := e.ActiveRegister()
	if reg == 0 {
		reg = defaultYankRegister
	}
	parts := yankFragments(text, sel)
	if len(parts) == 0 {
		return
	}
	e.WriteRegister(reg, []string{strings.Join(parts, sep)})
	setYankStatus(e, reg, 1)
	e.SetMode(view.ModeNormal)
}

func gotoColumn(e *view.Editor, extend bool) {
	col := max(e.Count(), 1)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		cursor := r.Cursor(doc)
		line, err := doc.CharToLine(cursor)
		if err != nil {
			return r
		}
		lineStart, err := doc.LineToChar(line)
		if err != nil {
			return r
		}
		lineEnd, err := doc.LineEndCharIndex(line)
		if err != nil {
			return r
		}
		return r.PutCursor(doc, min(lineStart+col-1, lineEnd), extend)
	})
}
