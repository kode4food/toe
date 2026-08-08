package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

func (r *renderPass) screenCharPos(
	doc *view.Document, v *view.View, at geom.Point,
) (int, bool) {
	a := v.Area()
	localY := at.Y - a.Y
	if localY < 0 {
		return 0, false
	}
	rowMap := r.editor.cache.viewRowMaps[v.ID()]
	if len(rowMap) == 0 {
		return 0, false
	}
	if localY >= len(rowMap) {
		localY = len(rowMap) - 1
	}
	entry := rowMap[localY]

	text := doc.Text()
	gutterW := gutterWidthFor(text, r.context.Editor.Options().Gutters)
	// Add the horizontal scroll offset: screen column 0 of the content maps to
	// content column hOff. The gutter is fixed and excluded from the offset
	contentX := max(at.X-a.X-gutterW-entry.prefixWidth, 0) +
		v.Offset().HorizontalOffset
	return charPosInLineSeg(charPosInLineSegArgs{
		text:     text,
		docLine:  entry.logLine,
		charOff:  entry.offset,
		targetX:  contentX,
		tabWidth: doc.TabWidth(),
	})
}

// contentViewAt returns the view whose content area contains screen point
// (x, y); a click on the pane's own status row or the command line misses
func (r *renderPass) contentViewAt(at geom.Point) *view.View {
	yOff := 0
	if bufferlineVisible(r.context) {
		yOff = 1
	}
	contentY := at.Y - yOff
	if contentY < 0 {
		return nil
	}
	var found *view.View
	r.context.Editor.Tree().RangeVisible(func(p view.Pane) bool {
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
	return found
}

func (r *renderPass) handleMouseClick(msg tea.MouseClickMsg) {
	at := geom.Point{X: msg.X, Y: msg.Y}
	if p, ok := paneAt(r.context, at); ok {
		wasFocused := r.context.Editor.Tree().Focus() == p.ID()
		r.context.Editor.FocusPane(p.ID())
		if pi, ok := p.(PaneInput); ok {
			if _, handled := pi.HandleEvent(r.context, msg); handled {
				return
			}
		}
		if sp, ok := p.(Draggable); ok && msg.Button == tea.MouseLeft {
			if wasFocused && sp.BeginDrag(r.context, at, msg.Mod) {
				r.editor.mouse.downDrag = sp
			}
			return
		}
	}
	if msg.Button != tea.MouseLeft {
		return
	}

	yOff := 0
	if bufferlineVisible(r.context) {
		yOff = 1
	}
	sep, onSep :=
		r.context.Editor.Tree().SeparatorAt(at.Sub(geom.Point{Y: yOff}))
	if onSep {
		r.editor.mouse.downSep = &sepDrag{
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
	prevSel := res.doc.SelectionFor(res.view.ID())
	r.editor.mouse.downRange = new(prevSel.Primary())
	r.editor.mouse.vertical.last = at.Y - yOff
	r.editor.mouse.horizontal.last = at.X

	var newSel core.Selection
	switch {
	case msg.Mod&tea.ModAlt != 0:
		newSel = prevSel.Push(core.PointRange(res.pos))
	case r.context.Editor.Mode() == view.ModeSelect:
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
	action.ApplySelection(r.context.Editor, newSel)
	res.view.EndFreeScroll()
}

func (r *renderPass) handleMouseDrag(at geom.Point) tea.Cmd {
	yOff := 0
	if bufferlineVisible(r.context) {
		yOff = 1
	}

	if r.editor.mouse.downSep != nil {
		sep := r.editor.mouse.downSep
		newPos := at.X
		if sep.layout == view.LayoutHorizontal {
			newPos = at.Y - yOff
		}
		r.context.Editor.Tree().MoveSeparator(
			sep.containerID, sep.childIdx, sep.layout, newPos,
		)
		return nil
	}

	if r.editor.mouse.downRange == nil {
		return nil
	}

	doc := r.context.Editor.FocusedDocument()
	if doc == nil {
		return nil
	}
	v := r.context.Editor.FocusedView()
	if v == nil {
		return nil
	}

	contentY := at.Y - yOff
	area := v.Area()
	contentH := max(area.Height-1, 0)
	scrollOff := r.context.Editor.Options().ScrollOff

	vEdge := r.editor.mouse.vertical.update(dragBounds{
		pos:      contentY,
		lowEdge:  area.Y,
		highEdge: area.Y + contentH - 1,
		margin: autoScrollMargin(autoScrollMarginArgs{
			span:      contentH,
			scrollOff: scrollOff,
		}),
	})

	gutterW := gutterWidthFor(doc.Text(), r.context.Editor.Options().Gutters)
	contentX := area.X + gutterW
	contentW := max(area.Width-gutterW, 0)
	hEdge := r.editor.mouse.horizontal.update(dragBounds{
		pos:      at.X,
		lowEdge:  contentX,
		highEdge: contentX + contentW - 1,
		margin: autoScrollMargin(autoScrollMarginArgs{
			span:      contentW,
			scrollOff: scrollOff,
		}),
	})

	clampedX, clampedY := hEdge.clamped, vEdge.clamped
	pos, ok := r.screenCharPos(doc, v, geom.Point{X: clampedX, Y: clampedY})
	if !ok {
		return nil
	}
	if !extendSelectionTo(r.context, doc, v, pos) {
		return nil
	}

	vAxis := &r.editor.mouse.vertical
	hAxis := &r.editor.mouse.horizontal
	vCmd := vAxis.trigger(vEdge, clampedX, vAxis.schedule)
	hCmd := hAxis.trigger(hEdge, clampedY, hAxis.schedule)
	return tea.Batch(vCmd, hCmd)
}

func (r *renderPass) handleMouseMiddleRelease(at geom.Point, mod tea.KeyMod) {
	if mod&tea.ModAlt != 0 {
		action.PrimaryClipboardReplace(r.context.Editor)
		return
	}

	res, ok := r.resolveClickPos(at)
	if !ok {
		return
	}
	action.ApplySelection(r.context.Editor, core.PointSelection(res.pos))
	action.PastePrimaryClipboardBefore(r.context.Editor)
}

type resolveClickPosRes struct {
	doc  *view.Document
	view *view.View
	pos  int
}

func (r *renderPass) resolveClickPos(at geom.Point) (resolveClickPosRes, bool) {
	v := r.contentViewAt(at)
	if v == nil {
		return resolveClickPosRes{}, false
	}
	r.context.Editor.FocusView(v.ID())
	doc := r.context.Editor.Document(v.DocID())
	if doc == nil {
		return resolveClickPosRes{}, false
	}
	contentY := at.Y
	if bufferlineVisible(r.context) {
		contentY--
	}
	pos, ok := r.screenCharPos(doc, v, geom.Point{X: at.X, Y: contentY})
	if !ok {
		return resolveClickPosRes{}, false
	}
	return resolveClickPosRes{doc: doc, view: v, pos: pos}, true
}

type cursorScreenPosArgs struct {
	text        core.Rope
	cursor      int
	gutterWidth int
	rowMap      []viewRowEntry
	tabWidth    int
	horzOff     int
}

func cursorScreenPos(args cursorScreenPosArgs) geom.Point {
	text := args.text
	cursor := args.cursor
	gutterW := args.gutterWidth
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
		segPrefixW = e.prefixWidth
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
	for _, ch := range lineString(text, core.Span{
		From: lineStart,
		To:   lineEnd,
	}) {
		if runeIdx >= cursorOff {
			break
		}
		if runeIdx >= segStart {
			col += view.RuneWidth(ch, core.TabStop{
				Column:   col,
				TabWidth: args.tabWidth,
			})
		}
		runeIdx++
	}
	return geom.Point{
		X: gutterW + segPrefixW + col - args.horzOff, Y: segY,
	}
}

type charPosInLineSegArgs struct {
	text     core.Rope
	docLine  int
	charOff  int
	targetX  int
	tabWidth int
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
	for _, ch := range lineString(text, core.Span{
		From: lineStart,
		To:   lineEnd,
	}) {
		if runeIdx < charOff {
			runeIdx++
			continue
		}
		var w int
		if ch == '\t' {
			w = args.tabWidth - col%args.tabWidth
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
	action.ApplySelection(cx.Editor, newSel)
	v.BeginFreeScroll(doc.Revision(), newSel)
	return true
}

type autoScrollMarginArgs struct {
	span      int
	scrollOff int
}

func autoScrollMargin(args autoScrollMarginArgs) int {
	return min(args.scrollOff, max(args.span/2-1, 0))
}
