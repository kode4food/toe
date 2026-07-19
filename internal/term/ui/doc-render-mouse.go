package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
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
	doc *view.Document, v *view.View, at geom.Point,
) (int, bool) {
	a := v.Area()
	localY := at.Y - a.Y
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
	contentX := max(at.X-a.X-gutterW-entry.prefixW, 0) +
		v.Offset().HorizontalOffset
	return charPosInLineSeg(charPosInLineSegArgs{
		text: text, docLine: entry.logLine, charOff: entry.offset,
		targetX: contentX, tabW: doc.TabWidth(),
	})
}

// contentViewAt returns the view whose content area contains screen point
// (x, y); a click on the pane's own status row or the command line misses
func (r *renderPass) contentViewAt(at geom.Point) (*view.View, bool) {
	yOff := 0
	if bufferlineVisible(r.cx) {
		yOff = 1
	}
	contentY := at.Y - yOff
	if contentY < 0 {
		return nil, false
	}
	var found *view.View
	r.cx.Editor.Tree().Range(func(p view.Pane) bool {
		v, ok := p.(*view.View)
		if !ok {
			return true
		}
		content := v.Area()
		content.Height = max(content.Height-1, 0)
		if content.Contains(geom.Point{X: at.X, Y: contentY}) {
			found = v
			return false
		}
		return true
	})
	return found, found != nil
}

func (r *renderPass) handleMouseClick(at geom.Point, mod tea.KeyMod) {
	if p, ok := paneAt(r.cx, at); ok {
		wasFocused := r.cx.Editor.Tree().Focus() == p.ID()
		r.cx.Editor.FocusPane(p.ID())
		if sp, ok := p.(Draggable); ok {
			if wasFocused && sp.BeginDrag(r.cx, at, mod) {
				r.ec.mouseDownDrag = sp
			}
			return
		}
	}

	yOff := 0
	if bufferlineVisible(r.cx) {
		yOff = 1
	}
	sep, onSep :=
		r.cx.Editor.Tree().SeparatorAt(
			geom.Point{X: at.X, Y: at.Y - yOff},
		)
	if onSep {
		r.ec.mouseDownSep = &sepDrag{
			containerID: sep.ContainerID,
			childIdx:    sep.ChildIdx,
			layout:      sep.Layout,
		}
		return
	}

	// A click outside any editor content area (status line, command line, or a
	// gap) must not move the cursor
	res, ok := r.resolveClickPos(at)
	if !ok {
		return
	}

	text := res.doc.Text()
	prevSel := res.doc.SelectionFor(res.v.ID())
	r.ec.mouseDownRange = new(prevSel.Primary())
	r.ec.autoScrollV.last = at.Y - yOff
	r.ec.autoScrollH.last = at.X

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

func (r *renderPass) handleMouseDrag(at geom.Point) tea.Cmd {
	yOff := 0
	if bufferlineVisible(r.cx) {
		yOff = 1
	}

	if r.ec.mouseDownSep != nil {
		sep := r.ec.mouseDownSep
		newPos := at.X
		if sep.layout == view.LayoutHorizontal {
			newPos = at.Y - yOff
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

	contentY := at.Y - yOff
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
		at.X, contentX, contentX+contentW-1,
		autoScrollMargin(contentW, scrollOff),
	)

	pos, ok := r.screenCharPos(doc, v, geom.Point{X: clampedX, Y: clampedY})
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

func (r *renderPass) handleMouseMiddleRelease(at geom.Point, mod tea.KeyMod) {
	if mod&tea.ModAlt != 0 {
		act.PrimaryClipboardReplace(r.cx.Editor)
		return
	}

	res, ok := r.resolveClickPos(at)
	if !ok {
		return
	}
	text := res.doc.Text()
	tx := core.NewTransaction(text).WithSelection(core.PointSelection(res.pos))
	_ = r.cx.Editor.Apply(tx)
	act.PastePrimaryClipboardBefore(r.cx.Editor)
}

func (r *renderPass) resolveClickPos(at geom.Point) (resolveClickPosRes, bool) {
	v, ok := r.contentViewAt(at)
	if !ok {
		return resolveClickPosRes{}, false
	}
	r.cx.Editor.FocusView(v.ID())
	doc, ok := r.cx.Editor.Document(v.DocID())
	if !ok {
		return resolveClickPosRes{}, false
	}
	contentY := at.Y
	if bufferlineVisible(r.cx) {
		contentY--
	}
	pos, ok := r.screenCharPos(doc, v, geom.Point{X: at.X, Y: contentY})
	if !ok {
		return resolveClickPosRes{}, false
	}
	return resolveClickPosRes{doc: doc, v: v, pos: pos}, true
}

func cursorScreenPos(args cursorScreenPosArgs) geom.Point {
	text := args.text
	cursor := args.cursor
	gutterW := args.gutterW
	cursorLine, err := text.CharToLine(cursor)
	if err != nil {
		return geom.Point{X: gutterW}
	}
	lineStart, err := text.LineToChar(cursorLine)
	if err != nil {
		return geom.Point{X: gutterW}
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
		return geom.Point{X: gutterW}
	}

	lineEnd, err := text.LineEndCharIndex(cursorLine)
	if err != nil {
		return geom.Point{X: gutterW + segPrefixW, Y: segY}
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
	return geom.Point{
		X: gutterW + segPrefixW + col - args.hOff, Y: segY,
	}
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
