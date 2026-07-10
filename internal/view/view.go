package view

import (
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/core"
)

type (
	// View is a viewport into a document
	View struct {
		id         Id
		docID      DocumentId
		docHistory []DocumentId
		offset     Position
		mode       Mode
		jumps      JumpList
		freeScroll bool
		// fsRev and fsSel snapshot the document state when free scroll
		// began; free scroll ends when either changes
		fsRev int
		fsSel core.Selection
		// area is the screen rectangle assigned by the layout engine
		area Area
		// vcol memoizes the last VisualColumn result; Rope is immutable
		// and comparable, so equal fields mean an identical result
		vcol vcolCache
		// dirty is set whenever area or offset changes value, or MarkDirty
		// is called; render code consumes it to decide whether to repaint
		dirty bool
	}

	vcolCache struct {
		doc    core.Rope
		cursor int
		tabW   int
		col    int
	}

	// Id is the unique identifier for an open view
	Id int

	// Mode describes the current editing mode
	Mode int

	// Align describes vertical scroll alignment
	Align int

	// Area is the screen rectangle assigned to a view by the layout engine
	Area struct {
		X, Y, Width, Height int
	}

	// Position holds the scroll offset for a view
	Position struct {
		// Anchor is the first visible char position in the document
		Anchor int
		// HorizontalOffset is the number of columns scrolled right
		HorizontalOffset int
		// VerticalOffset is lines of context above the visible area
		VerticalOffset int
	}

	// JumpList manages a bounded history of cursor positions
	JumpList struct {
		items []JumpEntry
		head  int
	}

	// JumpEntry is a single entry in the jump history
	JumpEntry struct {
		DocID     DocumentId
		Anchor    int
		Selection core.Selection
	}
)

const (
	ModeNormal Mode = iota
	ModeInsert
	ModeSelect
)

const (
	// InvalidViewId is the zero value, indicating no view
	InvalidViewId Id = 0

	jumpListCap = 64
)

// ID returns the view identifier
func (v *View) ID() Id {
	return v.id
}

// Area returns the screen rectangle assigned by the layout engine
func (v *View) Area() Area {
	return v.area
}

// SetArea sets the screen rectangle (called by the layout engine)
func (v *View) SetArea(a Area) {
	if a != v.area {
		v.area = a
		v.dirty = true
	}
}

// MarkDirty flags the view as needing a repaint on the next frame
func (v *View) MarkDirty() {
	v.dirty = true
}

// ConsumeDirty reports whether the view has changed since the last call,
// clearing the flag
func (v *View) ConsumeDirty() bool {
	d := v.dirty
	v.dirty = false
	return d
}

// DocID returns the document this view displays
func (v *View) DocID() DocumentId {
	return v.docID
}

func (v *View) addDocHistory(did DocumentId) {
	if did == InvalidDocumentId {
		return
	}
	for i, existing := range v.docHistory {
		if existing == did {
			v.docHistory = append(
				v.docHistory[:i], v.docHistory[i+1:]...,
			)
			break
		}
	}
	v.docHistory = append(v.docHistory, did)
}

func (v *View) removeDocHistory(did DocumentId) {
	for i := 0; i < len(v.docHistory); i++ {
		if v.docHistory[i] == did {
			v.docHistory = append(
				v.docHistory[:i], v.docHistory[i+1:]...,
			)
			i--
		}
	}
}

// Mode returns the current editing mode
func (v *View) Mode() Mode {
	return v.mode
}

// SetMode sets the current editing mode
func (v *View) SetMode(m Mode) {
	v.mode = m
}

// Offset returns the current scroll position
func (v *View) Offset() Position {
	return v.offset
}

// SetOffset updates the scroll position
func (v *View) SetOffset(p Position) {
	if p != v.offset {
		v.offset = p
		v.dirty = true
	}
}

// FreeScroll reports whether the viewport is decoupled from the cursor
func (v *View) FreeScroll() bool {
	return v.freeScroll
}

// BeginFreeScroll decouples the viewport from the cursor. The revision and
// selection snapshot the document state at this moment; free scroll ends
// automatically when either changes
func (v *View) BeginFreeScroll(rev int, sel core.Selection) {
	v.freeScroll = true
	v.fsRev = rev
	v.fsSel = sel
}

// EndFreeScroll re-couples the viewport to the cursor
func (v *View) EndFreeScroll() {
	v.freeScroll = false
	v.fsSel = core.Selection{}
}

// SyncFreeScroll ends free scroll when the document revision or selection
// changed since BeginFreeScroll, and reports whether it remains active
func (v *View) SyncFreeScroll(rev int, sel core.Selection) bool {
	if !v.freeScroll {
		return false
	}
	if rev != v.fsRev || !sel.Equal(v.fsSel) {
		v.EndFreeScroll()
		return false
	}
	return true
}

// PushJump records a selection in the jump list
func (v *View) PushJump(docID DocumentId, anchor int, sel core.Selection) {
	v.jumps.Push(docID, anchor, sel)
}

// JumpBackward moves to the previous position in the jump list
func (v *View) JumpBackward() (DocumentId, int, bool) {
	return v.jumps.Backward()
}

// JumpForward moves to the next position in the jump list
func (v *View) JumpForward() (DocumentId, int, bool) {
	return v.jumps.Forward()
}

// Jumps returns all entries in the jump history, oldest first
func (v *View) Jumps() []JumpEntry {
	return v.jumps.Entries()
}

// EnsureCursorVisible adjusts the view offset so the cursor is visible within
// height terminal rows, respecting the scrolloff margin. When vf describes an
// active soft-wrap layout, visibility is measured in visual (wrapped) rows;
// otherwise it falls back to text-line counting
func (v *View) EnsureCursorVisible(
	doc core.Rope, sel core.Selection, height, scrolloff int,
	vf *core.VisualMoveFormat,
) {
	if height <= 0 {
		return
	}
	if vf != nil && vf.ViewportWidth > 0 {
		v.ensureCursorVisibleVisual(doc, sel, height, scrolloff, vf)
		return
	}
	v.ensureCursorVisibleByLine(doc, sel, height, scrolloff)
}

func (v *View) trackOffsetChange() func() {
	before := v.offset
	return func() {
		if v.offset != before {
			v.dirty = true
		}
	}
}

// EnsureCursorVisibleHorizontal adjusts the horizontal scroll offset so the
// cursor's visual column stays within width content columns, respecting the
// scrolloff margin. The gutter is never shifted — width is the content area
// (viewport minus gutter). A width <= 0 disables horizontal scrolling (used for
// soft-wrapped views) and resets the offset to 0
func (v *View) EnsureCursorVisibleHorizontal(
	doc core.Rope, sel core.Selection, width, tabW, scrolloff int,
) {
	defer v.trackOffsetChange()()
	if width <= 0 {
		v.offset.HorizontalOffset = 0
		return
	}
	cursor := sel.Primary().Cursor(doc)
	line, err := doc.CharToLine(cursor)
	if err != nil {
		return
	}
	lineStart, err := doc.LineToChar(line)
	if err != nil {
		return
	}
	col := v.cachedVisualColumn(doc, lineStart, cursor, tabW)

	h := v.offset.HorizontalOffset
	// Clamp scrolloff so there is always at least one column in the middle
	so := min(scrolloff, max(width-1, 0)/2)

	leftEdge := h + so
	rightEdge := h + width - 1 - so

	if col < leftEdge {
		h = max(col-so, 0)
	} else if col > rightEdge {
		h = max(col-width+1+so, 0)
	}
	v.offset.HorizontalOffset = h
}

// cachedVisualColumn returns VisualColumn(doc, from, to, tabW), reusing the
// last result when doc, to, and tabW are unchanged since the previous call
func (v *View) cachedVisualColumn(doc core.Rope, from, to, tabW int) int {
	if v.vcol.doc == doc && v.vcol.cursor == to && v.vcol.tabW == tabW {
		return v.vcol.col
	}
	col := VisualColumn(doc, from, to, tabW)
	v.vcol = vcolCache{doc: doc, cursor: to, tabW: tabW, col: col}
	return col
}

func (m Mode) String() string {
	switch m {
	case ModeNormal:
		return "NOR"
	case ModeInsert:
		return "INS"
	case ModeSelect:
		return "SEL"
	}
	return "NOR"
}

// Entries returns all jump history entries from oldest to newest
func (j *JumpList) Entries() []JumpEntry {
	return append([]JumpEntry(nil), j.items...)
}

// Head returns the current head index in the jump list
func (j *JumpList) Head() int {
	return j.head
}

// Restore replaces the jump list contents and head position
func (j *JumpList) Restore(items []JumpEntry, head int) {
	j.items = items
	j.head = head
}

// Push adds a new jump selection, discarding forward history
func (j *JumpList) Push(docID DocumentId, anchor int, sel core.Selection) {
	j.push(JumpEntry{
		DocID:     docID,
		Anchor:    anchor,
		Selection: sel,
	})
}

func (j *JumpList) push(item JumpEntry) {
	if len(j.items) > 0 && j.head < len(j.items) {
		j.items = j.items[:j.head]
	}
	if len(j.items) > 0 && jumpEntryEqual(j.items[len(j.items)-1], item) {
		j.head = len(j.items)
		return
	}
	j.items = append(j.items, item)
	if len(j.items) > jumpListCap {
		j.items = j.items[len(j.items)-jumpListCap:]
	}
	j.head = len(j.items)
}

func jumpEntryEqual(a, b JumpEntry) bool {
	return a.DocID == b.DocID &&
		a.Anchor == b.Anchor &&
		a.Selection.Equal(b.Selection)
}

// Backward moves to the previous jump and returns it
func (j *JumpList) Backward() (DocumentId, int, bool) {
	if j.head <= 1 {
		return 0, 0, false
	}
	j.head--
	it := j.items[j.head-1]
	return it.DocID, it.Anchor, true
}

// Forward moves to the next jump and returns it
func (j *JumpList) Forward() (DocumentId, int, bool) {
	if j.head >= len(j.items) {
		return 0, 0, false
	}
	it := j.items[j.head]
	j.head++
	return it.DocID, it.Anchor, true
}

// RuneWidth returns the display width of ch at visual column col, expanding
// tabs to the next tabW boundary. The ASCII fast path avoids a per-rune string
// allocation in the render and cursor-positioning hot paths
func RuneWidth(ch rune, col, tabW int) int {
	if uint32(ch)-0x20 < 0x5f {
		return 1
	}
	if ch == '\t' {
		return tabW - col%tabW
	}
	return runeWidthWide(ch)
}

//go:noinline
func runeWidthWide(ch rune) int {
	return runewidth.RuneWidth(ch)
}

func (v *View) ensureCursorVisibleByLine(
	doc core.Rope, sel core.Selection, height, scrolloff int,
) {
	defer v.trackOffsetChange()()
	// Text-line scrolling never scrolls within a line
	v.offset.VerticalOffset = 0
	cursor := sel.Primary().Cursor(doc)
	line, err := doc.CharToLine(cursor)
	if err != nil {
		return
	}
	anchorLine, err := doc.CharToLine(v.offset.Anchor)
	if err != nil {
		anchorLine = 0
	}

	// Asymmetric scrolloff: the top loses one row so a gap always remains in
	// the middle, while the bottom keeps the full margin. The bottom margin
	// holds even at end-of-file, so the view scrolls past the last line rather
	// than pinning it to the bottom edge
	soTop := min(scrolloff, max(height-1, 0)/2)
	soBottom := min(scrolloff, height/2)

	var newFirstLine int
	switch {
	case line < anchorLine+soTop:
		newFirstLine = max(line-soTop, 0)
	case line > anchorLine+height-1-soBottom:
		newFirstLine = max(line-(height-1-soBottom), 0)
	default:
		return
	}
	if newAnchor, err := doc.LineToChar(newFirstLine); err == nil {
		v.offset.Anchor = newAnchor
	}
}

// ensureCursorVisibleVisual keeps the cursor within the scrolloff margin
// measured in visual rows. The view top is the anchor line plus a vertical
// offset of visual rows scrolled into it, so a single soft-wrapped line taller
// than the viewport can scroll within itself. Each walk is bounded by the
// viewport height
func (v *View) ensureCursorVisibleVisual(
	doc core.Rope, sel core.Selection, height, scrolloff int,
	vf *core.VisualMoveFormat,
) {
	defer v.trackOffsetChange()()
	cursor := sel.Primary().Cursor(doc)
	cursorLine, err := doc.CharToLine(cursor)
	if err != nil {
		return
	}
	cursorLineStart, err := doc.LineToChar(cursorLine)
	if err != nil {
		return
	}
	cursorRow := vf.VisualRowOfOffset(doc, cursorLine, cursor-cursorLineStart)

	anchorLine, err := doc.CharToLine(v.offset.Anchor)
	if err != nil {
		anchorLine = 0
	}
	vOff := max(v.offset.VerticalOffset, 0)

	soTop := min(scrolloff, max(height-1, 0)/2)
	soBottom := min(scrolloff, height/2)

	// fromTop is the cursor's visual row measured from the current viewport top
	// (anchor line top, minus the rows already scrolled past). ok is false when
	// the cursor sits above the anchor line entirely
	rows, ok := visualRowsToCursor(
		doc, vf, anchorLine, cursorLine, cursorRow, height+vOff,
	)
	fromTop := rows - vOff

	switch {
	case !ok || fromTop < soTop:
		anchorLine, vOff = vf.VisualScrollUp(doc, cursorLine, cursorRow, soTop)
	case fromTop > height-1-soBottom:
		anchorLine, vOff = vf.VisualScrollUp(
			doc, cursorLine, cursorRow, height-1-soBottom,
		)
	default:
		return
	}
	if newAnchor, err := doc.LineToChar(anchorLine); err == nil {
		v.offset.Anchor = newAnchor
		v.offset.VerticalOffset = vOff
	}
}

// visualRowsToCursor returns the cursor's visual-row distance from the top of
// anchorLine (sum of the wrapped row counts of the lines in between plus the
// cursor's row within its own line). ok is false when the cursor is above
// anchorLine. The walk stops once it exceeds cap, since callers only compare
// against the viewport height
func visualRowsToCursor(
	doc core.Rope, vf *core.VisualMoveFormat,
	anchorLine, cursorLine, cursorRow, limit int,
) (int, bool) {
	if cursorLine < anchorLine {
		return 0, false
	}
	rows := cursorRow
	for l := anchorLine; l < cursorLine; l++ {
		rows += vf.VisualRows(doc, l)
		if rows > limit {
			return rows, true
		}
	}
	return rows, true
}

// VisualColumn returns the display column of position to, measured from from,
// expanding tabs to the next tabW boundary. It folds rune widths over the
// range directly, allocating no intermediate substring
func VisualColumn(doc core.Rope, from, to, tabW int) int {
	col := 0
	doc.ForEachSegment(from, to, func(seg string) {
		for _, ch := range seg {
			col += RuneWidth(ch, col, tabW)
		}
	})
	return col
}
