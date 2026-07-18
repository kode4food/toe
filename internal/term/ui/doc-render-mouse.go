package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
	act "github.com/kode4food/toe/internal/view/action"
)

type (
	resolveClickPosRes struct {
		doc *view.Document
		v   *view.View
		pos int
	}

	cursorScreenPosArgs struct {
		text    core.Rope
		cursor  int
		gutterW int
		rowMap  []viewRowEntry
		tabW    int
		hOff    int
	}

	charPosInLineSegArgs struct {
		text    core.Rope
		docLine int
		charOff int
		targetX int
		tabW    int
	}
)

func (r *renderPass) screenCharPos(
	doc *view.Document, v *view.View, x, contentY int,
) (int, bool) {
	a := v.Area()
	localY := contentY - a.Y
	if localY < 0 {
		return 0, false
	}
	rowMap := r.ec.cache.viewRowMaps[v.ID()]
	if len(rowMap) == 0 {
		return 0, false
	}
	if localY >= len(rowMap) {
		localY = len(rowMap) - 1
	}
	entry := rowMap[localY]

	text := doc.Text()
	gutterW := gutterWidthFor(text, r.cx.Editor.Options().Gutters)
	// Add the horizontal scroll offset: screen column 0 of the content maps to
	// content column hOff. The gutter is fixed and excluded from the offset
	contentX := max(x-a.X-gutterW-entry.prefixW, 0) +
		v.Offset().HorizontalOffset
	return charPosInLineSeg(charPosInLineSegArgs{
		text: text, docLine: entry.logLine, charOff: entry.offset,
		targetX: contentX, tabW: doc.TabWidth(),
	})
}

// contentViewAt returns the view whose content area contains screen point
// (x, y); a click on the pane's own status row or the command line misses
func (r *renderPass) contentViewAt(x, y int) (*view.View, bool) {
	yOff := 0
	if bufferlineVisible(r.cx) {
		yOff = 1
	}
	contentY := y - yOff
	if contentY < 0 {
		return nil, false
	}
	var found *view.View
	r.cx.Editor.Tree().Range(func(p view.Pane) bool {
		v, ok := p.(*view.View)
		if !ok {
			return true
		}
		a := v.Area()
		contentH := max(a.Height-1, 0)
		if x >= a.X && x < a.X+a.Width &&
			contentY >= a.Y && contentY < a.Y+contentH {
			found = v
			return false
		}
		return true
	})
	return found, found != nil
}

func (r *renderPass) handleMouseClick(x, y int, mod tea.KeyMod) {
	if p, ok := paneAt(r.cx, x, y); ok {
		wasFocused := r.cx.Editor.Tree().Focus() == p.ID()
		r.cx.Editor.FocusPane(p.ID())
		if sp, ok := p.(Draggable); ok {
			if wasFocused && sp.BeginDrag(r.cx, x, y, mod) {
				r.ec.mouseDownDrag = sp
			}
			return
		}
	}

	yOff := 0
	if bufferlineVisible(r.cx) {
		yOff = 1
	}
	containerID, childIdx, layout, onSep :=
		r.cx.Editor.Tree().SeparatorAt(x, y-yOff)
	if onSep {
		r.ec.mouseDownSep = &sepDrag{
			containerID: containerID,
			childIdx:    childIdx,
			layout:      layout,
		}
		return
	}

	// A click outside any editor content area (status line, command line, or a
	// gap) must not move the cursor
	res, ok := r.resolveClickPos(x, y)
	if !ok {
		return
	}

	text := res.doc.Text()
	prevSel := res.doc.SelectionFor(res.v.ID())
	r.ec.mouseDownRange = new(prevSel.Primary())
	r.ec.autoScrollV.last = y - yOff
	r.ec.autoScrollH.last = x

	var newSel core.Selection
	switch {
	case mod&tea.ModAlt != 0:
		newSel = prevSel.Push(core.PointRange(res.pos))
	case r.cx.Editor.Mode() == view.ModeSelect:
		// In select mode a click extends the primary selection rather than
		// collapsing it, discarding any secondary selections
		primary := prevSel.Primary().PutCursor(text, res.pos, true)
		if s, err := core.NewSelection([]core.Range{primary}, 0); err == nil {
			newSel = s
		} else {
			newSel = core.PointSelection(res.pos)
		}
	default:
		newSel = core.PointSelection(res.pos)
	}
	tx := core.NewTransaction(text).WithSelection(newSel)
	_ = r.cx.Editor.Apply(tx)
	res.v.BeginFreeScroll(res.doc.Revision(), newSel)
}

func (r *renderPass) handleMouseDrag(x, y int) tea.Cmd {
	yOff := 0
	if bufferlineVisible(r.cx) {
		yOff = 1
	}

	if r.ec.mouseDownSep != nil {
		sep := r.ec.mouseDownSep
		newPos := x
		if sep.layout == view.LayoutHorizontal {
			newPos = y - yOff
		}
		r.cx.Editor.Tree().MoveSeparator(
			sep.containerID, sep.childIdx, sep.layout, newPos,
		)
		return nil
	}

	if r.ec.mouseDownRange == nil {
		return nil
	}

	doc, ok := r.cx.Editor.FocusedDocument()
	if !ok {
		return nil
	}
	v, ok := r.cx.Editor.FocusedView()
	if !ok {
		return nil
	}

	contentY := y - yOff
	area := v.Area()
	contentH := max(area.Height-1, 0)
	scrollOff := r.cx.Editor.Options().ScrollOff

	atTop, atBottom, clampedY := r.ec.autoScrollV.update(
		contentY, area.Y, area.Y+contentH-1,
		autoScrollMargin(contentH, scrollOff),
	)

	gutterW := gutterWidthFor(doc.Text(), r.cx.Editor.Options().Gutters)
	contentX := area.X + gutterW
	contentW := max(area.Width-gutterW, 0)
	atLeft, atRight, clampedX := r.ec.autoScrollH.update(
		x, contentX, contentX+contentW-1,
		autoScrollMargin(contentW, scrollOff),
	)

	pos, ok := r.screenCharPos(doc, v, clampedX, clampedY)
	if !ok {
		return nil
	}
	if !extendSelectionTo(r.cx, doc, v, pos) {
		return nil
	}

	vAxis, hAxis := &r.ec.autoScrollV, &r.ec.autoScrollH
	vCmd := vAxis.trigger(atTop, atBottom, clampedX, vAxis.schedule)
	hCmd := hAxis.trigger(atLeft, atRight, clampedY, hAxis.schedule)
	return tea.Batch(vCmd, hCmd)
}

func (r *renderPass) handleMouseMiddleRelease(x, y int, mod tea.KeyMod) {
	if mod&tea.ModAlt != 0 {
		act.PrimaryClipboardReplace(r.cx.Editor)
		return
	}

	res, ok := r.resolveClickPos(x, y)
	if !ok {
		return
	}
	text := res.doc.Text()
	tx := core.NewTransaction(text).WithSelection(core.PointSelection(res.pos))
	_ = r.cx.Editor.Apply(tx)
	act.PastePrimaryClipboardBefore(r.cx.Editor)
}

func (r *renderPass) resolveClickPos(x, y int) (resolveClickPosRes, bool) {
	v, ok := r.contentViewAt(x, y)
	if !ok {
		return resolveClickPosRes{}, false
	}
	r.cx.Editor.FocusView(v.ID())
	doc, ok := r.cx.Editor.Document(v.DocID())
	if !ok {
		return resolveClickPosRes{}, false
	}
	contentY := y
	if bufferlineVisible(r.cx) {
		contentY--
	}
	pos, ok := r.screenCharPos(doc, v, x, contentY)
	if !ok {
		return resolveClickPosRes{}, false
	}
	return resolveClickPosRes{doc: doc, v: v, pos: pos}, true
}

func cursorScreenPos(args cursorScreenPosArgs) (visualY, visualX int) {
	text := args.text
	cursor := args.cursor
	gutterW := args.gutterW
	cursorLine, err := text.CharToLine(cursor)
	if err != nil {
		return 0, gutterW
	}
	lineStart, err := text.LineToChar(cursorLine)
	if err != nil {
		return 0, gutterW
	}
	cursorOff := cursor - lineStart

	segY := -1
	segStart := 0
	segPrefixW := 0
	for i, e := range args.rowMap {
		if e.filler || e.logLine != cursorLine {
			if segY >= 0 {
				break
			}
			continue
		}
		if cursorOff < e.offset {
			break
		}
		segY = i
		segStart = e.offset
		segPrefixW = e.prefixW
	}
	if segY < 0 {
		return 0, gutterW
	}

	lineEnd, err := text.LineEndCharIndex(cursorLine)
	if err != nil {
		return segY, gutterW + segPrefixW
	}
	col := 0
	runeIdx := 0
	for _, ch := range lineString(text, lineStart, lineEnd) {
		if runeIdx >= cursorOff {
			break
		}
		if runeIdx >= segStart {
			col += view.RuneWidth(ch, col, args.tabW)
		}
		runeIdx++
	}
	return segY, gutterW + segPrefixW + col - args.hOff
}

func charPosInLineSeg(args charPosInLineSegArgs) (int, bool) {
	text := args.text
	docLine := args.docLine
	charOff := args.charOff
	lineStart, err := text.LineToChar(docLine)
	if err != nil {
		return 0, false
	}
	lineEnd, err := text.LineEndCharIndex(docLine)
	if err != nil {
		return 0, false
	}
	col := 0
	charPos := lineStart + charOff
	runeIdx := 0
	for _, ch := range lineString(text, lineStart, lineEnd) {
		if runeIdx < charOff {
			runeIdx++
			continue
		}
		var w int
		if ch == '\t' {
			w = args.tabW - col%args.tabW
		} else {
			w = runewidth.RuneWidth(ch)
		}
		if col+w > args.targetX {
			break
		}
		col += w
		charPos++
		runeIdx++
	}
	return charPos, true
}

func extendSelectionTo(
	cx *Context, doc *view.Document, v *view.View, pos int,
) bool {
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	primary := sel.Primary().PutCursor(text, pos, true)
	newSel, err := sel.Replace(sel.PrimaryIndex(), primary)
	if err != nil {
		return false
	}
	tx := core.NewTransaction(text).WithSelection(newSel)
	_ = cx.Editor.Apply(tx)
	v.BeginFreeScroll(doc.Revision(), newSel)
	return true
}

func autoScrollMargin(span, scrollOff int) int {
	return min(scrollOff, max(span/2-1, 0))
}
