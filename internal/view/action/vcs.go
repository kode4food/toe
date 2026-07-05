package action

import (
	"errors"
	"strings"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

var (
	ErrDiffUnavailable      = errors.New("diff unavailable in this buffer")
	ErrNoChangesInSelection = errors.New("no changes under any selection")
)

// StatusDiffUnavailable is shown by change motions when no diff exists
const StatusDiffUnavailable = "Diff is not available in current buffer"

// GotoNextChange moves each cursor to the next diff change
func GotoNextChange(e *view.Editor) {
	gotoChange(e, core.DirectionForward)
}

// GotoPrevChange moves each cursor to the previous diff change
func GotoPrevChange(e *view.Editor) {
	gotoChange(e, core.DirectionBackward)
}

// GotoFirstChange selects the first diff change in the document
func GotoFirstChange(e *view.Editor) {
	gotoEdgeChange(e, false)
}

// GotoLastChange selects the last diff change in the document
func GotoLastChange(e *view.Editor) {
	gotoEdgeChange(e, true)
}

// ResetDiffChange reverts every diff hunk that intersects the selection back to
// the version-control base text. It returns how many hunks were reset
func ResetDiffChange(e *view.Editor) (int, error) {
	v, ok := e.FocusedView()
	if !ok {
		return 0, view.ErrNoView
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return 0, view.ErrNoDocument
	}
	vc := e.VersionControl()
	if vc == nil {
		return 0, ErrDiffUnavailable
	}
	base, ok := vc.DiffBase(doc)
	if !ok {
		return 0, ErrDiffUnavailable
	}
	hunks := vc.DiffHunks(doc)
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	lineRanges, err := sel.LineRanges(text)
	if err != nil {
		return 0, err
	}

	// ponytail: hunks are recomputed on a debounced background worker, so they
	// can trail the newest keystrokes by a beat; indices are clamped below. Add
	// a synchronous diff refresh here if that beat ever bites
	baseLines := strings.SplitAfter(base, "\n")
	var changes []core.Change
	for _, h := range hunks {
		if !hunkIntersects(h, lineRanges) {
			continue
		}
		from, to, ok := hunkCharRange(h, text)
		if !ok {
			continue
		}
		bf := min(h.BaseFrom, len(baseLines))
		bt := min(h.BaseTo, len(baseLines))
		replacement := strings.Join(baseLines[bf:bt], "")
		changes = append(changes, core.TextChange(from, to, replacement))
	}
	if len(changes) == 0 {
		return 0, ErrNoChangesInSelection
	}
	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return 0, err
	}
	if err := e.Apply(core.NewTransaction(text).WithChanges(cs)); err != nil {
		return 0, err
	}
	return len(changes), nil
}

func gotoChange(e *view.Editor, dir core.Direction) {
	count := countOrOne(e) - 1
	doc, v, hunks, ok := focusedDiffHunks(e)
	if !ok || len(hunks) == 0 {
		return
	}
	text := doc.Text()
	extend := e.Mode() == view.ModeSelect
	sel := doc.SelectionFor(v.ID())
	newSel := sel.Transform(func(r core.Range) core.Range {
		line, err := r.CursorLine(text)
		if err != nil {
			return r
		}
		var idx int
		if dir == core.DirectionForward {
			i, ok := nextHunkIdx(hunks, line)
			if !ok {
				return r
			}
			idx = min(i+count, len(hunks)-1)
		} else {
			i, ok := prevHunkIdx(hunks, line)
			if !ok {
				return r
			}
			idx = max(i-count, 0)
		}
		nr, ok := hunkRange(hunks[idx], text)
		if !ok {
			return r
		}
		if extend {
			head := nr.Head
			if nr.Head < r.Anchor {
				head = nr.Anchor
			}
			return core.NewRange(r.Anchor, head)
		}
		return nr.WithDirection(dir)
	})
	SaveSelection(e)
	doc.SetSelectionFor(v.ID(), newSel)
}

func gotoEdgeChange(e *view.Editor, last bool) {
	doc, v, hunks, ok := focusedDiffHunks(e)
	if !ok || len(hunks) == 0 {
		return
	}
	h := hunks[0]
	if last {
		h = hunks[len(hunks)-1]
	}
	r, ok := hunkRange(h, doc.Text())
	if !ok {
		return
	}
	newSel, err := core.NewSelection([]core.Range{r}, 0)
	if err != nil {
		return
	}
	SaveSelection(e)
	doc.SetSelectionFor(v.ID(), newSel)
}

func focusedDiffHunks(
	e *view.Editor,
) (*view.Document, *view.View, []view.DiffHunk, bool) {
	v, ok := e.FocusedView()
	if !ok {
		return nil, nil, nil, false
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return nil, nil, nil, false
	}
	vc := e.VersionControl()
	if vc == nil {
		e.SetStatusMsg(StatusDiffUnavailable)
		return nil, nil, nil, false
	}
	return doc, v, vc.DiffHunks(doc), true
}

// hunkRange returns the selection range covering a hunk. Additions and
// modifications cover the changed lines; a pure removal is the point at the
// start of the removal
func hunkRange(h view.DiffHunk, text core.Rope) (core.Range, bool) {
	from, to, ok := hunkCharRange(h, text)
	if !ok {
		return core.Range{}, false
	}
	if h.PureRemoval() {
		return core.NewRange(from, min(from+1, text.LenChars())), true
	}
	return core.NewRange(from, to), true
}

func hunkCharRange(h view.DiffHunk, text core.Rope) (int, int, bool) {
	from, err := text.LineToChar(min(h.From, text.LenLines()-1))
	if err != nil {
		return 0, 0, false
	}
	to := text.LenChars()
	if h.To < text.LenLines() {
		if to, err = text.LineToChar(h.To); err != nil {
			return 0, 0, false
		}
	}
	return from, to, true
}

func nextHunkIdx(hunks []view.DiffHunk, line int) (int, bool) {
	for i, h := range hunks {
		if h.From > line {
			return i, true
		}
	}
	return 0, false
}

// prevHunkIdx returns the last hunk that ends at or before line. A pure removal
// sitting exactly on line does not count; the cursor is inside it
func prevHunkIdx(hunks []view.DiffHunk, line int) (int, bool) {
	for i := len(hunks) - 1; i >= 0; i-- {
		h := hunks[i]
		if h.PureRemoval() {
			if h.From < line {
				return i, true
			}
			continue
		}
		if h.To <= line {
			return i, true
		}
	}
	return 0, false
}

func hunkIntersects(h view.DiffHunk, lineRanges []core.LineRange) bool {
	start := h.From
	end := max(h.To, h.From+1)
	for _, lr := range lineRanges {
		if start <= lr.To && end > lr.From {
			return true
		}
	}
	return false
}
