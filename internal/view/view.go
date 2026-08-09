package view

import (
	"slices"

	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
)

type (
	// View is a viewport into a document
	View struct {
		id         Id
		editor     *Editor
		docID      DocumentId
		docHistory []DocumentId

		offset     Position
		mode       Mode
		jumps      JumpList
		freeScroll freeScrollState

		area      geom.Area
		visualCol visualColumnCache
		dirty     bool
	}

	visualColumnCache struct {
		doc      core.Rope
		cursor   int
		tabWidth int
		col      int
	}

	freeScrollState struct {
		active bool
		rev    int
		sel    core.Selection
	}

	// Id is the unique identifier for an open view
	Id int

	// Mode describes the current editing mode
	Mode int

	// Align describes vertical scroll alignment
	Align int

	// CursorScroll carries the viewport inputs that keep the cursor visible.
	// Height and Visual drive vertical scrolling, Width and TabWidth drive
	// horizontal scrolling
	CursorScroll struct {
		Doc       core.Rope
		Selection core.Selection
		Height    int
		Width     int
		TabWidth  int
		ScrollOff int
		Visual    *core.VisualMoveFormat
	}

	// Position holds the scroll offset for a view
	Position struct {
		Anchor           int
		HorizontalOffset int
		VerticalOffset   int
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

// ModeAny is the zero value; it is not a pane mode. It is the wildcard key
// in a Command's per-mode Keys map, applying to every mode the command
// supports unless a specific mode overrides it
const ModeAny Mode = 0

//go:generate go tool stringer -type=Mode -linecomment
const (
	ModeNormal   Mode = 1 << iota // NOR
	ModeInsert                    // INS
	ModeSelect                    // SEL
	ModeTerminal                  // TRM
	ModeImage                     // IMG
	ModeBinary                    // BIN

	// ModeCompletion is not a pane mode; it is the keymap dispatch bucket
	// used only while the completion popup owns key handling
	ModeCompletion // COM
)

const (
	// InvalidViewId is the zero value, indicating no view
	InvalidViewId Id = 0

	jumpListCap = 64
)

var (
	// allModes lists every single-bit Mode value, for decomposing an ORed set
	allModes = []Mode{
		ModeNormal, ModeInsert, ModeSelect, ModeTerminal, ModeImage, ModeBinary,
		ModeCompletion,
	}

	modeScopes = map[Mode]string{
		ModeNormal:   "normal",
		ModeInsert:   "insert",
		ModeSelect:   "select",
		ModeTerminal: "terminal",
		ModeImage:    "image",
		ModeBinary:   "binary",
	}
)

// ID returns the view identifier
func (v *View) ID() Id {
	return v.id
}

// SetID sets the view identifier (called by the tree on insertion)
func (v *View) SetID(id Id) {
	v.id = id
}

// Split returns another view of the same document
func (v *View) Split() (Pane, error) {
	return &View{
		editor: v.editor,
		docID:  v.docID,
		mode:   ModeNormal,
		jumps:  v.jumps.Clone(),
	}, nil
}

// Close closes this view
func (v *View) Close() {
	if v.editor != nil {
		v.editor.CloseView(v.id)
	}
}

// Discard closes this displaced view if no other view uses its document
func (v *View) Discard() {
	if v.editor != nil {
		v.editor.discardView(v)
	}
}

// Shutdown releases external resources owned by this view
func (v *View) Shutdown() {
}

// OnDisplace marks this view as stashed behind another pane. A view holds no
// heavy resources of its own, so there is nothing to release
func (v *View) OnDisplace() {}

// OnRevert marks this view as returned to the foreground. Nothing to reacquire
func (v *View) OnRevert() {}

// Area returns the screen rectangle assigned by the layout engine
func (v *View) Area() geom.Area {
	return v.area
}

// SetArea sets the screen rectangle (called by the layout engine)
func (v *View) SetArea(a geom.Area) {
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

// Path returns the path of the document this view displays
func (v *View) Path() string {
	if doc := v.editor.Document(v.docID); doc != nil {
		return doc.Path()
	}
	return ""
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
	return v.freeScroll.active
}

// BeginFreeScroll decouples the viewport from the cursor. The revision and
// selection snapshot the document state at this moment; free scroll ends
// automatically when either changes
func (v *View) BeginFreeScroll(rev int, sel core.Selection) {
	v.freeScroll = freeScrollState{active: true, rev: rev, sel: sel}
}

// EndFreeScroll re-couples the viewport to the cursor
func (v *View) EndFreeScroll() {
	if !v.freeScroll.active {
		return
	}
	v.dirty = true
	v.freeScroll = freeScrollState{}
}

// SyncFreeScroll ends free scroll when the document revision or selection
// changed since BeginFreeScroll, and reports whether it remains active
func (v *View) SyncFreeScroll(rev int, sel core.Selection) bool {
	if !v.freeScroll.active {
		return false
	}
	if rev != v.freeScroll.rev || !sel.Equal(v.freeScroll.sel) {
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

// EnsureCursorVisible scrolls so the cursor is visible within Height rows,
// respecting ScrollOff; measured in visual rows when Visual has active
// soft-wrap
func (v *View) EnsureCursorVisible(cs *CursorScroll) {
	if cs.Height <= 0 {
		return
	}
	if cs.Visual != nil && cs.Visual.ViewportWidth > 0 {
		v.ensureCursorVisibleVisual(cs)
		return
	}
	v.ensureCursorVisibleByLine(cs)
}

// EnsureCursorVisibleHorizontal scrolls so the cursor's visual column stays
// within Width content columns (gutter excluded). Width <= 0 disables
// horizontal scrolling and resets the offset to 0
func (v *View) EnsureCursorVisibleHorizontal(cs *CursorScroll) {
	defer v.trackOffsetChange()()
	if cs.Width <= 0 {
		v.offset.HorizontalOffset = 0
		return
	}
	cursor := cs.Selection.Primary().Cursor(cs.Doc)
	line, err := cs.Doc.CharToLine(cursor)
	if err != nil {
		return
	}
	lineStart, err := cs.Doc.LineToChar(line)
	if err != nil {
		return
	}
	col := v.cachedVisualColumn(cs.Doc, core.Span{
		From: lineStart,
		To:   cursor,
	}, cs.TabWidth)

	h := v.offset.HorizontalOffset
	// Clamp scrolloff so there is always at least one column in the middle
	so := min(cs.ScrollOff, max(cs.Width-1, 0)/2)

	leftEdge := h + so
	rightEdge := h + cs.Width - 1 - so

	if col < leftEdge {
		h = max(col-so, 0)
	} else if col > rightEdge {
		h = max(col-cs.Width+1+so, 0)
	}
	v.offset.HorizontalOffset = h
}

func (v *View) addDocHistory(did DocumentId) {
	if did == InvalidDocumentId {
		return
	}
	v.removeDocHistory(did)
	v.docHistory = append(v.docHistory, did)
}

func (v *View) removeDocHistory(did DocumentId) {
	v.docHistory = slices.DeleteFunc(v.docHistory, func(d DocumentId) bool {
		return d == did
	})
}

func (v *View) trackOffsetChange() func() {
	before := v.offset
	return func() {
		if v.offset != before {
			v.dirty = true
		}
	}
}

// cachedVisualColumn returns VisualColumn(doc, s, tabW), reusing the last
// result when doc, s.To, and tabW are unchanged since the previous call
func (v *View) cachedVisualColumn(doc core.Rope, s core.Span, tabW int) int {
	c := v.visualCol
	if c.doc == doc && c.cursor == s.To && c.tabWidth == tabW {
		return c.col
	}
	col := VisualColumn(doc, s, tabW)
	v.visualCol = visualColumnCache{
		doc:      doc,
		cursor:   s.To,
		tabWidth: tabW,
		col:      col,
	}
	return col
}

// Scope returns the theme scope suffix for the pane's status line and, for
// document panes, the cursor style: e.g. normal, insert, terminal, binary
func (m Mode) Scope() string {
	if s, ok := modeScopes[m]; ok {
		return s
	}
	return modeScopes[ModeNormal]
}

// Split decomposes an ORed set of modes into its constituent single-bit
// values, in declaration order
func (m Mode) Split() []Mode {
	out := make([]Mode, 0, len(allModes))
	for _, v := range allModes {
		if m&v != 0 {
			out = append(out, v)
		}
	}
	return out
}

// ParseMode returns the Mode for a short name (NOR, INS, …), defaulting to
// ModeNormal for anything unrecognized
func ParseMode(name string) Mode {
	for _, m := range allModes {
		if m.String() == name {
			return m
		}
	}
	return ModeNormal
}

// Entries returns all jump history entries from oldest to newest
func (j *JumpList) Entries() []JumpEntry {
	return slices.Clone(j.items)
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

// Clone returns an independent copy of the jump list
func (j *JumpList) Clone() JumpList {
	return JumpList{
		items: slices.Clone(j.items),
		head:  j.head,
	}
}

// Push adds a new jump selection, discarding forward history
func (j *JumpList) Push(docID DocumentId, anchor int, sel core.Selection) {
	j.push(JumpEntry{
		DocID:     docID,
		Anchor:    anchor,
		Selection: sel,
	})
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

func (j *JumpList) push(item JumpEntry) {
	if len(j.items) > 0 && j.head < len(j.items) {
		j.items = j.items[:j.head]
	}
	if len(j.items) > 0 && j.items[len(j.items)-1].equal(item) {
		j.head = len(j.items)
		return
	}
	j.items = append(j.items, item)
	if len(j.items) > jumpListCap {
		j.items = j.items[len(j.items)-jumpListCap:]
	}
	j.head = len(j.items)
}

// RuneWidth returns the display width of ch at the given tab stop, expanding
// tabs to the next boundary. The ASCII fast path avoids a per-rune string
// allocation in the render and cursor-positioning hot paths
func RuneWidth(ch rune, at core.TabStop) int {
	if uint32(ch)-0x20 < 0x5f {
		return 1
	}
	return runeWidthSlow(ch, at)
}

func (v *View) ensureCursorVisibleByLine(cs *CursorScroll) {
	defer v.trackOffsetChange()()
	// Text-line scrolling never scrolls within a line
	v.offset.VerticalOffset = 0
	cursor := cs.Selection.Primary().Cursor(cs.Doc)
	line, err := cs.Doc.CharToLine(cursor)
	if err != nil {
		return
	}
	anchorLine, err := cs.Doc.CharToLine(v.offset.Anchor)
	if err != nil {
		anchorLine = 0
	}

	// asymmetric: bottom margin holds even at EOF, so the view scrolls past
	// the last line instead of pinning it to the bottom edge
	soTop := min(cs.ScrollOff, max(cs.Height-1, 0)/2)
	soBottom := min(cs.ScrollOff, cs.Height/2)

	var newFirstLine int
	switch {
	case line < anchorLine+soTop:
		newFirstLine = max(line-soTop, 0)
	case line > anchorLine+cs.Height-1-soBottom:
		newFirstLine = max(line-(cs.Height-1-soBottom), 0)
	default:
		return
	}
	if newAnchor, err := cs.Doc.LineToChar(newFirstLine); err == nil {
		v.offset.Anchor = newAnchor
	}
}

func (v *View) ensureCursorVisibleVisual(cs *CursorScroll) {
	defer v.trackOffsetChange()()
	cursor := cs.Selection.Primary().Cursor(cs.Doc)
	cursorLine, err := cs.Doc.CharToLine(cursor)
	if err != nil {
		return
	}
	cursorLineStart, err := cs.Doc.LineToChar(cursorLine)
	if err != nil {
		return
	}
	cursorRow := cs.Visual.VisualRowOfOffset(core.VisualRowOfOffsetArgs{
		Doc:     cs.Doc,
		Line:    cursorLine,
		CharOff: cursor - cursorLineStart,
	})

	anchorLine, err := cs.Doc.CharToLine(v.offset.Anchor)
	if err != nil {
		anchorLine = 0
	}
	vOff := max(v.offset.VerticalOffset, 0)

	soTop := min(cs.ScrollOff, max(cs.Height-1, 0)/2)
	soBottom := min(cs.ScrollOff, cs.Height/2)

	// ok is false when the cursor sits above the anchor line entirely
	rows, ok := visualRowsToCursor(visualRowsArgs{
		doc:        cs.Doc,
		visual:     cs.Visual,
		anchorLine: anchorLine,
		cursorLine: cursorLine,
		cursorRow:  cursorRow,
		limit:      cs.Height + vOff,
	})
	fromTop := rows - vOff

	switch {
	case !ok || fromTop < soTop:
		res := cs.Visual.VisualScrollUp(core.VisualScrollUpArgs{
			Doc:  cs.Doc,
			Line: cursorLine,
			Row:  cursorRow,
			Up:   soTop,
		})
		anchorLine, vOff = res.Line, res.Row
	case fromTop > cs.Height-1-soBottom:
		res := cs.Visual.VisualScrollUp(core.VisualScrollUpArgs{
			Doc:  cs.Doc,
			Line: cursorLine,
			Row:  cursorRow,
			Up:   cs.Height - 1 - soBottom,
		})
		anchorLine, vOff = res.Line, res.Row
	default:
		return
	}
	if newAnchor, err := cs.Doc.LineToChar(anchorLine); err == nil {
		v.offset.Anchor = newAnchor
		v.offset.VerticalOffset = vOff
	}
}

// VisualColumn returns the display column of the span's end, measured from
// its start, expanding tabs to the next tabW boundary. It folds rune widths
// over the span directly, allocating no intermediate substring
func VisualColumn(doc core.Rope, s core.Span, tabW int) int {
	col := 0
	doc.ForEachSegment(s, func(seg string) {
		for _, ch := range seg {
			col += RuneWidth(ch, core.TabStop{Column: col, TabWidth: tabW})
		}
	})
	return col
}

func (e JumpEntry) equal(other JumpEntry) bool {
	return e.DocID == other.DocID &&
		e.Anchor == other.Anchor &&
		e.Selection.Equal(other.Selection)
}

// Outlined slow path to allow inlining of RuneWidth's fast path
//
//go:noinline
func runeWidthSlow(ch rune, at core.TabStop) int {
	if ch == '\t' {
		return core.TabWidthAt(at)
	}
	return runewidth.RuneWidth(ch)
}

type visualRowsArgs struct {
	doc        core.Rope
	visual     *core.VisualMoveFormat
	anchorLine int
	cursorLine int
	cursorRow  int
	limit      int
}

func visualRowsToCursor(args visualRowsArgs) (int, bool) {
	if args.cursorLine < args.anchorLine {
		return 0, false
	}
	for l := args.anchorLine; l < args.cursorLine; l++ {
		args.cursorRow += args.visual.VisualRows(args.doc, l)
		if args.cursorRow > args.limit {
			return args.cursorRow, true
		}
	}
	return args.cursorRow, true
}
