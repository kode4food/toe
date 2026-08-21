package action

import (
	"errors"
	"slices"
	"strings"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/view"
)

const StatusDiffUnavailable i18n.Key = "status.diffUnavailable"

var (
	ErrDiffUnavailable      = errors.New("diff unavailable in this buffer")
	ErrNoChangesInSelection = errors.New("no changes under any selection")
)

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
	v := e.FocusedView()
	if v == nil {
		return 0, view.ErrNoView
	}
	doc := e.FocusedDocument()
	if doc == nil {
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

	// hunks trail the newest keystrokes (debounced background worker), so
	// indices are clamped below
	baseLines := strings.SplitAfter(base, "\n")
	var changes []core.Change
	for _, h := range hunks {
		if !hunkIntersects(h, lineRanges) {
			continue
		}
		hr, ok := hunkCharRange(h, text)
		if !ok {
			continue
		}
		bf := min(h.BaseFrom, len(baseLines))
		bt := min(h.BaseTo, len(baseLines))
		replacement := strings.Join(baseLines[bf:bt], "")
		changes = append(
			changes, core.TextChange(core.Span{
				From: hr.From(),
				To:   hr.To(),
			}, replacement),
		)
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
	count := e.CountOr(1) - 1
	res, ok := focusedDiffHunks(e)
	if !ok || len(res.hunks) == 0 {
		return
	}
	doc := res.doc
	hunks := res.hunks
	text := doc.Text()
	extend := e.Mode() == view.ModeSelect
	sel := doc.SelectionFor(res.view.ID())
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
			return core.Range{Anchor: r.Anchor, Head: head}
		}
		return nr.WithDirection(dir)
	})
	SaveSelection(e)
	doc.SetSelectionFor(res.view.ID(), newSel)
}

func gotoEdgeChange(e *view.Editor, last bool) {
	res, ok := focusedDiffHunks(e)
	if !ok || len(res.hunks) == 0 {
		return
	}
	doc := res.doc
	hunks := res.hunks
	h := hunks[0]
	if last {
		h = hunks[len(hunks)-1]
	}
	r, ok := hunkRange(h, doc.Text())
	if !ok {
		return
	}
	if newSel, err := core.NewSelection([]core.Range{r}, 0); err == nil {
		SaveSelection(e)
		doc.SetSelectionFor(res.view.ID(), newSel)
	}
}

type focusedDiffHunksRes struct {
	doc   *view.Document
	view  *view.View
	hunks []view.DiffHunk
}

func focusedDiffHunks(e *view.Editor) (focusedDiffHunksRes, bool) {
	v := e.FocusedView()
	if v == nil {
		return focusedDiffHunksRes{}, false
	}
	doc := e.FocusedDocument()
	if doc == nil {
		return focusedDiffHunksRes{}, false
	}
	vc := e.VersionControl()
	if vc == nil {
		e.SetStatusMsg(i18n.Text(StatusDiffUnavailable))
		return focusedDiffHunksRes{}, false
	}
	return focusedDiffHunksRes{
		doc:   doc,
		view:  v,
		hunks: vc.DiffHunks(doc),
	}, true
}

func hunkRange(h view.DiffHunk, text core.Rope) (core.Range, bool) {
	r, ok := hunkCharRange(h, text)
	if !ok {
		return core.Range{}, false
	}
	if h.PureRemoval() {
		return core.Range{
			Anchor: r.From(),
			Head:   min(r.From()+1, text.LenChars()),
		}, true
	}
	return r, true
}

func hunkCharRange(h view.DiffHunk, text core.Rope) (core.Range, bool) {
	from, err := text.LineToChar(min(h.From, text.LenLines()-1))
	if err != nil {
		return core.Range{}, false
	}
	to := text.LenChars()
	if h.To < text.LenLines() {
		if to, err = text.LineToChar(h.To); err != nil {
			return core.Range{}, false
		}
	}
	return core.Range{Anchor: from, Head: to}, true
}

func nextHunkIdx(hunks []view.DiffHunk, line int) (int, bool) {
	for i, h := range hunks {
		if h.From > line {
			return i, true
		}
	}
	return 0, false
}

func prevHunkIdx(hunks []view.DiffHunk, line int) (int, bool) {
	for i, h := range slices.Backward(hunks) {

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

func hunkIntersects(h view.DiffHunk, lineRanges []core.Span) bool {
	start := h.From
	end := max(h.To, h.From+1)
	for _, lr := range lineRanges {
		if start <= lr.To && end > lr.From {
			return true
		}
	}
	return false
}
