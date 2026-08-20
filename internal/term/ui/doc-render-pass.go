package ui

import (
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

// renderPass bundles the state needed for a single render pass so every render
// helper receives it without passing cx and ec separately
type renderPass struct {
	editor  *EditorComponent
	context *Context
	size    geom.Size
}

const countArrow = "\u2192" // '→' - rightwards arrow

const (
	infoPopupChrome   = 3 // top border, breadcrumb, bottom border
	infoPopupRule     = 1 // divider between the breadcrumb and the hints
	infoPopupTitlePad = 2 // spaces flanking the title on the top border
	inputCaretGap     = 1 // space between the breadcrumb and its caret
)

// marks a hint that opens another menu, a command's row is blank here
const hintPrefixMark = "\u25b8" // '▸' - black right-pointing small triangle

// splitSepIntersectionChars is an immutable separator glyph lookup
var splitSepIntersectionChars = [...]string{
	tui.BorderH, tui.BorderH, tui.BorderH, tui.BorderV,
	tui.BorderH, tui.BorderH, tui.BorderH, tui.BorderMR,
	tui.BorderH, tui.BorderH, tui.BorderH, tui.BorderML,
	tui.BorderH, tui.BorderMB, tui.BorderMT, tui.BorderX,
}

func (r *renderPass) renderBufferline(buf *tui.Buffer, y int) {
	th := r.context.Theme()
	bgTUI := th.Get("ui.bufferline.background")
	activeTUI := th.Get("ui.bufferline.active")
	inactiveTUI := th.Get("ui.bufferline")

	buf.SetString(geom.Point{Y: y}, strings.Repeat(" ", r.size.Width), bgTUI)

	focusedDoc := r.context.Editor.FocusedDocument()
	docs := r.context.Editor.AllDocuments()
	slices.SortFunc(docs, func(a, b *view.Document) int {
		return int(a.ID() - b.ID())
	})

	x := 0
	for _, doc := range docs {
		name := doc.DisplayName()
		if name == "" {
			name = "[scratch]"
		}
		mod := ""
		if doc.Modified() {
			mod = "[+]"
		}
		label := " " + name + mod + " "
		style := inactiveTUI
		if focusedDoc != nil && doc.ID() == focusedDoc.ID() {
			style = activeTUI
		}
		buf.SetString(geom.Point{X: x, Y: y}, label, style)
		x += runewidth.StringWidth(label)
	}
}

func (r *renderPass) editorCursor() (tea.Cursor, bool) {
	opts := r.context.Editor.Options()
	// the popup is taking input, so the caret belongs at its breadcrumb
	if r.editor.overlayHead() != "" {
		return insertCursorAt(r.context, r.editor.cache.inputCaret)
	}
	p := r.context.Editor.Tree().Get(r.context.Editor.Tree().Focus())
	if pc, ok := p.(PaneCursor); ok {
		return pc.Cursor(r.context)
	}
	kind := opts.CursorShapeForMode(r.context.Editor.Mode())
	switch kind {
	case view.CursorKindHidden:
		return tea.Cursor{}, false
	case view.CursorKindBlock:
		if r.editor.focused {
			// block cursor drawn manually in content, terminal cursor hidden
			return tea.Cursor{}, false
		}
		// terminal lost focus: use underline so position is still visible
		kind = view.CursorKindUnderline
	}
	if at, ok := r.editor.caretScreenPos(r.context); ok {
		return tea.Cursor{
			Position: tea.Position{X: at.X, Y: at.Y},
			Shape:    cursorKindToShape(kind),
			Color:    cursorColor(r.context, r.context.Editor.Mode()),
		}, true
	}
	return tea.Cursor{}, false
}

type renderPaneArgs struct {
	doc     *view.Document
	view    *view.View
	buf     *tui.Buffer
	yOffset int
	focused bool
}

func (r *renderPass) renderPane(args renderPaneArgs) {
	doc := args.doc
	v := args.view
	a := v.Area()
	opts := r.context.Editor.Options()
	scrolloff := opts.ScrollOff
	contentH := v.ContentHeight()
	editorX := a.X
	editorW := a.Width

	// Build the soft-wrap layout so vertical visibility is measured in visual
	// rows. A nil layout keeps the text-line fallback when soft-wrap is off
	text := doc.Text()
	gutterW := gutterWidthFor(text, opts.Gutters)
	format := doc.TextFormatForConfig(
		max(editorW-gutterW, 0), r.context.Editor.Options(),
	)
	var vf *core.VisualMoveFormat
	if format.SoftWrap && gutterW < editorW {
		vf = &core.VisualMoveFormat{
			ViewportWidth:   format.ViewportWidth,
			TabWidth:        format.TabWidth,
			MaxWrap:         format.MaxWrap,
			MaxIndentRetain: format.MaxIndentRetain,
			WrapIndicatorWidth: runewidth.StringWidth(
				format.WrapIndicatorPrefix(),
			),
		}
	}
	sel := doc.SelectionFor(v.ID())
	if !v.SyncFreeScroll(doc.Revision(), sel) {
		v.EnsureCursorVisible(&view.CursorScroll{
			Doc:       text,
			Selection: sel,
			Height:    contentH,
			ScrollOff: scrolloff,
			Visual:    vf,
		})
	}
	r.renderContent(renderContentArgs{
		doc:  doc,
		view: v,
		buf:  args.buf,
		area: geom.Area{
			Point: geom.Point{X: editorX, Y: args.yOffset + a.Y},
			Size:  geom.Size{Width: editorW, Height: contentH},
		},
		focused: args.focused,
	})
	r.renderStatus(renderStatusArgs{
		doc:     doc,
		view:    v,
		buf:     args.buf,
		at:      geom.Point{X: a.X, Y: args.yOffset + a.Y + contentH},
		width:   a.Width,
		focused: args.focused,
	})
}

func (r *renderPass) needsFullRedraw(cache *renderCache, th *theme.Theme) bool {
	var force bool

	key := styleKey{theme: th.Name(), mode: r.context.Editor.Mode()}
	if cache.stylesKey != key {
		force = true
	}

	if gen := r.context.Editor.Options().Gen; cache.lastOptionsGen != gen {
		cache.lastOptionsGen = gen
		force = true
	}

	if r.context.composition.changed {
		force = true
	}

	if info := r.currentInfoPopupKey(); !cache.lastInfoKey.equals(info) {
		cache.lastInfoKey = info
		force = true
	}

	if cache.lastW != r.size.Width || cache.lastH != r.size.Height {
		cache.lastW, cache.lastH = r.size.Width, r.size.Height
		force = true
	}

	if key := currentDiagnosticPopupKey(r.context); cache.lastDiagKey != key {
		cache.lastDiagKey = key
		force = true
	}

	if reg := r.context.Editor.ActiveRegister(); cache.lastReg != reg {
		cache.lastReg = reg
		force = true
	}

	if cache.lastToastRev != r.editor.toasts.rev {
		cache.lastToastRev = r.editor.toasts.rev
		force = true
	}

	if cache.lastBlink != r.editor.macroBlink {
		cache.lastBlink = r.editor.macroBlink
		force = true
	}

	if cache.lastSpinner != r.editor.spinner {
		cache.lastSpinner = r.editor.spinner
		force = true
	}

	return force
}

type beginPaneRedrawArgs struct {
	buf        *tui.Buffer
	pane       view.Pane
	yOffset    int
	dirty      bool
	redrawAll  bool
	focused    bool
	background tui.Style
}

func (r *renderPass) beginPaneRedraw(args beginPaneRedrawArgs) bool {
	redraw := args.redrawAll
	if !redraw {
		forced := !r.context.composition.singleLayer &&
			isPaneUnderOverlay(r.context, args.pane.Area(), args.yOffset)
		redraw = forced || args.dirty
	}
	if !redraw {
		return false
	}
	if !args.redrawAll || !args.focused {
		clearPaneRect(args.buf, args.pane.Area(), args.yOffset, args.background)
	}
	return true
}

func (r *renderPass) renderEditorContent(buf *tui.Buffer) {
	th := r.context.Theme()
	cache := r.editor.cache

	redrawAll := r.needsFullRedraw(cache, th)
	bgTUI := th.Get("ui.background")
	if redrawAll {
		buf.Fill(bgTUI)
	}

	y0 := 0
	if bufferlineVisible(r.context) {
		r.renderBufferline(buf, 0)
		y0 = 1
	}

	focus := r.context.Editor.Tree().Focus()
	r.context.Editor.Tree().RangeVisible(func(p view.Pane) bool {
		focused := p.ID() == focus
		paneBg := r.context.ThemeFor(focused).Get("ui.background")
		switch pane := p.(type) {
		case *view.View:
			doc := r.context.Editor.Document(pane.DocID())
			if doc == nil {
				return true
			}
			dirty := pane.ConsumeDirty()
			dirty = doc.ConsumeDirty(pane.ID()) || dirty
			if r.beginPaneRedraw(beginPaneRedrawArgs{
				buf:        buf,
				pane:       pane,
				yOffset:    y0,
				dirty:      dirty,
				redrawAll:  redrawAll,
				focused:    focused,
				background: paneBg,
			}) {
				r.renderPane(renderPaneArgs{
					doc:     doc,
					view:    pane,
					buf:     buf,
					yOffset: y0,
					focused: focused,
				})
			}
		case *ImagePane:
			if r.beginPaneRedraw(beginPaneRedrawArgs{
				buf:        buf,
				pane:       pane,
				yOffset:    y0,
				dirty:      pane.ConsumeDirty(),
				redrawAll:  redrawAll,
				focused:    focused,
				background: paneBg,
			}) {
				r.renderImagePane(buf, pane, y0, focused)
			}
		case *TerminalPane:
			if r.beginPaneRedraw(beginPaneRedrawArgs{
				buf:        buf,
				pane:       pane,
				yOffset:    y0,
				dirty:      pane.ConsumeDirty(),
				redrawAll:  redrawAll,
				focused:    focused,
				background: paneBg,
			}) {
				r.renderTerminalPane(buf, pane, y0, focused)
			}
		case *BinaryPane:
			if r.beginPaneRedraw(beginPaneRedrawArgs{
				buf:        buf,
				pane:       pane,
				yOffset:    y0,
				dirty:      pane.ConsumeDirty(),
				redrawAll:  redrawAll,
				focused:    focused,
				background: paneBg,
			}) {
				r.renderBinaryPane(buf, pane, y0, focused)
			}
		}
		return true
	})

	sepTUI := th.Get("ui.border")
	vertCells := make(map[[2]int]bool)
	horizCells := make(map[[2]int]bool)
	r.context.Editor.Tree().WalkSeparators(func(s view.Separator) {
		if s.Layout == view.LayoutVertical {
			for row := s.Y; row < s.Y+s.Height; row++ {
				vertCells[[2]int{s.X, row}] = true
			}
		} else {
			for col := s.X; col < s.X+s.Width; col++ {
				horizCells[[2]int{col, s.Y}] = true
			}
		}
	})
	for cell := range vertCells {
		x := cell[0]
		y := cell[1]
		left := horizCells[[2]int{x - 1, y}]
		right := horizCells[[2]int{x + 1, y}]
		ch := tui.BorderV
		if left || right {
			ch = splitSepIntersectionChar(splitSepIntersectionArgs{
				above: vertCells[[2]int{x, y - 1}],
				below: vertCells[[2]int{x, y + 1}],
				left:  left,
				right: right,
			})
		}
		buf.SetString(geom.Point{X: x, Y: y0 + y}, ch, sepTUI)
	}
	for cell := range horizCells {
		x := cell[0]
		y := cell[1]
		if vertCells[[2]int{x, y}] {
			continue
		}
		above := vertCells[[2]int{x, y - 1}]
		below := vertCells[[2]int{x, y + 1}]
		ch := tui.BorderH
		if above || below {
			ch = splitSepIntersectionChar(splitSepIntersectionArgs{
				above: above,
				below: below,
				left:  horizCells[[2]int{x - 1, y}],
				right: horizCells[[2]int{x + 1, y}],
			})
		}
		buf.SetString(geom.Point{X: x, Y: y0 + y}, ch, sepTUI)
	}

	r.renderToasts(buf, r.size.Height-1)

	r.renderDiagnosticPopup(buf)

	if r.editor.overlayHead() == "" {
		r.editor.cache.infoBounds = geom.Area{}
		return
	}
	r.renderInfoOverlay(buf)
}

func (r *renderPass) renderInfoOverlay(buf *tui.Buffer) {
	items := r.editor.keys.infoItems
	th := r.context.Theme()

	popupSt := th.Get("ui.popup")
	popupTUI := popupSt

	title := r.editor.keys.infoTitle
	head := r.editor.overlayHead()

	keyW := 0
	for _, item := range items {
		if w := runewidth.StringWidth(item.Key); w > keyW {
			keyW = w
		}
	}
	rawLines := make([]string, len(items))
	caretX := runewidth.StringWidth(head) + inputCaretGap
	bodyW := caretX + 1
	for i, item := range items {
		lead := " "
		if item.Prefix {
			lead = hintPrefixMark
		}
		// pad by display width, a wide glyph is one rune but two columns
		pad := strings.Repeat(" ", keyW-runewidth.StringWidth(item.Key))
		rawLines[i] = item.Key + pad + " " + lead + " " + item.Label
		if w := runewidth.StringWidth(rawLines[i]); w > bodyW {
			bodyW = w
		}
	}
	if tw := runewidth.StringWidth(title) + infoPopupTitlePad; tw > bodyW {
		bodyW = tw
	}

	pop := popup{
		borderStyle:  popupTUI,
		contentStyle: popupTUI,
		padX:         1,
	}
	boxW := bodyW + 2 + 2*pop.padX
	within := geom.Size{
		Width:  r.size.Width,
		Height: max(r.size.Height-overlayKeepClear, 0),
	}
	rows := max(within.Height-infoPopupChrome-infoPopupRule, 0)
	if len(rawLines) > rows {
		rawLines = rawLines[:rows]
	}
	boxH := infoPopupChrome
	if len(rawLines) > 0 {
		boxH += len(rawLines) + infoPopupRule
	}

	size := geom.Size{Width: boxW, Height: boxH}
	origin := geom.Area{Size: within}.Center(size)
	box := geom.Area{Point: origin, Size: size}
	r.editor.cache.infoBounds = box
	area := pop.drawInto(buf, box)

	if title != "" {
		buf.SetString(
			geom.Point{X: origin.X + 1, Y: origin.Y}, " "+title+" ", popupTUI,
		)
	}
	buf.SetString(area.Point, head, popupTUI)
	r.editor.cache.inputCaret = area.Add(geom.Point{X: caretX})

	if len(rawLines) == 0 {
		return
	}
	drawPopupRule(drawPopupRuleArgs{
		buf:   buf,
		at:    geom.Point{X: area.X - 1 - pop.padX, Y: area.Y + 1},
		width: area.Width + 2*pop.padX + 2,
		style: popupSt,
	})
	for i, raw := range rawLines {
		buf.SetString(area.Add(geom.Point{Y: i + 2}), raw, popupTUI)
	}
}

func (r *renderPass) currentInfoPopupKey() infoPopupKey {
	return infoPopupKey{
		head:  r.editor.overlayHead(),
		title: r.editor.keys.infoTitle,
		items: r.editor.keys.infoItems,
	}
}

func (k infoPopupKey) equals(o infoPopupKey) bool {
	return k.head == o.head && k.title == o.title &&
		slices.Equal(k.items, o.items)
}

func (e *EditorComponent) overlayHead() string {
	keys := e.keys.path
	if len(keys) == 0 && e.keys.count == 0 {
		return ""
	}
	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(k.String())
	}
	return withCount(sb.String(), e.keys.count)
}

func withCount(keys string, count int) string {
	if count == 0 {
		return keys
	}
	if keys == "" {
		return strconv.Itoa(count)
	}
	return keys + " " + countArrow + " " + strconv.Itoa(count)
}

func isPaneUnderOverlay(cx *Context, a geom.Area, y0 int) bool {
	if !cx.composition.precise {
		return true
	}
	pane := a.Translate(geom.Point{Y: y0})
	return slices.ContainsFunc(cx.composition.regions, pane.Intersects)
}

func clearPaneRect(buf *tui.Buffer, a geom.Area, y0 int, style tui.Style) {
	// redo the full-buffer Fill writeFillToBuffer trusts, just this pane
	top := y0 + a.Y
	for y := top; y < top+a.Height; y++ {
		buf.FillRange(geom.Point{X: a.X, Y: y}, a.Width, style)
	}
}

type splitSepIntersectionArgs struct {
	above bool
	below bool
	left  bool
	right bool
}

func splitSepIntersectionChar(at splitSepIntersectionArgs) string {
	idx := 0
	if at.above {
		idx |= 1
	}
	if at.below {
		idx |= 2
	}
	if at.left {
		idx |= 4
	}
	if at.right {
		idx |= 8
	}
	return splitSepIntersectionChars[idx]
}
