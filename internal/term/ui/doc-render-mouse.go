package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
	act "github.com/kode4food/toe/internal/view/action"
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

// contentViewAt returns the view whose editor content area contains screen
// point (x, y), excluding the pane's status row. Cursor positioning uses this
// so a click on the status or command line is ignored
func (r *renderPass) contentViewAt(x, y int) (*view.View, bool) {
	yOff := 0
	if bufferlineVisible(r.cx) {
		yOff = 1
	}
	contentY := y - yOff
	if contentY < 0 {
		return nil, false
	}
	for _, vs := range r.cx.Editor.Tree().Views() {
		a := vs.View.Area()
		contentH := max(a.Height-1, 0)
		if x >= a.X && x < a.X+a.Width &&
			contentY >= a.Y && contentY < a.Y+contentH {
			return vs.View, true
		}
	}
	return nil, false
}

func (r *renderPass) handleMouseClick(x, y int, mod tea.KeyMod) {
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
}

func (r *renderPass) handleMouseDrag(x, y int) {
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
		return
	}

	if r.ec.mouseDownRange == nil {
		return
	}

	contentY := y - yOff
	if contentY < 0 {
		return
	}

	doc, ok := r.cx.Editor.FocusedDocument()
	if !ok {
		return
	}
	v, ok := r.cx.Editor.FocusedView()
	if !ok {
		return
	}

	pos, ok := r.screenCharPos(doc, v, x, contentY)
	if !ok {
		return
	}

	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	primary := sel.Primary().PutCursor(text, pos, true)
	newSel, err := sel.Replace(sel.PrimaryIndex(), primary)
	if err != nil {
		return
	}
	tx := core.NewTransaction(text).WithSelection(newSel)
	_ = r.cx.Editor.Apply(tx)
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

type resolveClickPosRes struct {
	doc *view.Document
	v   *view.View
	pos int
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

type cursorScreenPosArgs struct {
	text    core.Rope
	cursor  int
	gutterW int
	rowMap  []viewRowEntry
	tabW    int
	// hOff is the view's horizontal scroll offset in content columns; the
	// cursor's content column is shifted left by it, the gutter is not
	hOff int
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
		if e.logLine != cursorLine {
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

type charPosInLineSegArgs struct {
	text    core.Rope
	docLine int
	charOff int
	targetX int
	tabW    int
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
