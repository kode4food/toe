package ui

import (
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	ec   *EditorComponent
	cx   *Context
	size geom.Size
}

const (
	splitIntersect = "\u253c" // '┼' - box drawings light cross
	splitLeftT     = "\u251c" // '├' - box drawings light left T
	splitRightT    = "\u2524" // '┤' - box drawings light vertical and left
	vertSplit      = "\u2502" // '│' - box drawings light vertical
	splitUpT       = "\u2534" // '┴' - box drawings light up and horizontal
	splitDownT     = "\u252c" // '┬' - box drawings light down T
	horizSplit     = "\u2500" // '─' - box drawings light horizontal
)

var splitSepIntersectionChars = [...]string{
	horizSplit, horizSplit, horizSplit, vertSplit,
	horizSplit, horizSplit, horizSplit, splitRightT,
	horizSplit, horizSplit, horizSplit, splitLeftT,
	horizSplit, splitUpT, splitDownT, splitIntersect,
}

func (r *renderPass) renderBufferline(buf *tui.Buffer, y int) {
	th := r.activeTheme()
	bgTUI := th.Get("ui.bufferline.background")
	activeTUI := th.Get("ui.bufferline.active")
	inactiveTUI := th.Get("ui.bufferline")

	buf.SetString(geom.Point{Y: y}, strings.Repeat(" ", r.size.Width), bgTUI)

	focusedDoc, _ := r.cx.Editor.FocusedDocument()
	docs := r.cx.Editor.AllDocuments()
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
	p := r.cx.Editor.Tree().Get(r.cx.Editor.Tree().Focus())
	if pc, ok := p.(PaneCursor); ok {
		return pc.Cursor(r.cx)
	}
	opts := r.cx.Editor.Options()
	kind := opts.CursorShapeForMode(r.cx.Editor.Mode().String())
	switch kind {
	case view.CursorKindHidden:
		return tea.Cursor{}, false
	case view.CursorKindBlock:
		if r.ec.focused {
			// block cursor drawn manually in content; terminal cursor hidden
			return tea.Cursor{}, false
		}
		// terminal lost focus: use underline so position is still visible
		kind = view.CursorKindUnderline
	}
	at, ok := r.ec.caretScreenPos(r.cx)
	if !ok {
		return tea.Cursor{}, false
	}
	return tea.Cursor{
		Position: tea.Position{X: at.X, Y: at.Y},
		Shape:    cursorKindToShape(kind),
		Blink:    false,
	}, true
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
	opts := r.cx.Editor.Options()
	scrolloff := opts.ScrollOff
	contentH := max(a.Height-1, 0)
	editorX := a.X
	editorW := a.Width

	// Build the soft-wrap layout so vertical visibility is measured in visual
	// rows; nil keeps the text-line fallback when soft-wrap is off
	text := doc.Text()
	gutterW := gutterWidthFor(text, opts.Gutters)
	format := doc.TextFormatForConfig(
		max(editorW-gutterW, 0), r.cx.Editor.Options(),
	)
	var vf *core.VisualMoveFormat
	if format.SoftWrap && gutterW < editorW {
		vf = &core.VisualMoveFormat{
			ViewportWidth:    format.ViewportWidth,
			TabWidth:         format.TabWidth,
			MaxWrap:          format.MaxWrap,
			MaxIndentRetain:  format.MaxIndentRetain,
			WrapIndicatorLen: runewidth.StringWidth(format.WrapIndicator),
		}
	}
	sel := doc.SelectionFor(v.ID())
	if !v.SyncFreeScroll(doc.Revision(), sel) {
		v.EnsureCursorVisible(text, sel, contentH, scrolloff, vf)
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

func (r *renderPass) forceFullRedraw(
	cache *renderCache, th *theme.Theme,
) (force bool) {
	key := styleKey{theme: th.Name(), mode: r.cx.Editor.Mode()}
	if cache.stylesKey != key {
		force = true
	}

	if gen := r.cx.Editor.Options().Gen; cache.lastOptionsGen != gen {
		cache.lastOptionsGen = gen
		force = true
	}

	if r.cx.composition.changed {
		force = true
	}

	if cache.lastInfoTitle != r.ec.keys.infoTitle ||
		!slices.Equal(cache.lastInfoItems, r.ec.keys.infoItems) {
		cache.lastInfoTitle = r.ec.keys.infoTitle
		cache.lastInfoItems = r.ec.keys.infoItems
		force = true
	}

	if cache.lastW != r.size.Width || cache.lastH != r.size.Height {
		cache.lastW, cache.lastH = r.size.Width, r.size.Height
		force = true
	}

	if key := currentDiagnosticPopupKey(r.cx); cache.lastDiagKey != key {
		cache.lastDiagKey = key
		force = true
	}

	if cache.lastSpinFrame != r.ec.spinFrame {
		cache.lastSpinFrame = r.ec.spinFrame
		force = true
	}

	return
}

type beginPaneRedrawArgs struct {
	buf        *tui.Buffer
	pane       view.Pane
	yOffset    int
	dirty      bool
	redrawAll  bool
	background tui.Style
}

// beginPaneRedraw reports whether pane needs repainting this frame, clearing
// its cell rectangle first on an incremental (non-full) redraw
func (r *renderPass) beginPaneRedraw(args beginPaneRedrawArgs) bool {
	forced := !r.cx.composition.singleLayer &&
		paneUnderOverlay(r.cx, args.pane.Area(), args.yOffset)
	switch {
	case args.redrawAll:
		return true
	case forced || args.dirty:
		clearPaneRect(args.buf, args.pane.Area(), args.yOffset, args.background)
		return true
	default:
		return false
	}
}

func (r *renderPass) renderEditorContent(buf *tui.Buffer) {
	th := r.activeTheme()
	cache := r.ec.cache

	redrawAll := r.forceFullRedraw(cache, th)
	bgTUI := th.Get("ui.background")
	if redrawAll {
		buf.Fill(bgTUI)
	}

	y0 := 0
	if bufferlineVisible(r.cx) {
		r.renderBufferline(buf, 0)
		y0 = 1
	}

	focus := r.cx.Editor.Tree().Focus()
	r.cx.Editor.Tree().Range(func(p view.Pane) bool {
		focused := p.ID() == focus
		switch pane := p.(type) {
		case *view.View:
			doc, ok := r.cx.Editor.Document(pane.DocID())
			if !ok {
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
				background: bgTUI,
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
				background: bgTUI,
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
				background: bgTUI,
			}) {
				r.renderTerminalPane(buf, pane, y0, focused)
			}
		}
		return true
	})

	sepTUI := th.Get("ui.border")
	vertCells := make(map[[2]int]bool)
	horizCells := make(map[[2]int]bool)
	r.cx.Editor.Tree().WalkSeparators(func(s view.Separator) {
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
		x, y := cell[0], cell[1]
		left := horizCells[[2]int{x - 1, y}]
		right := horizCells[[2]int{x + 1, y}]
		ch := vertSplit
		if left || right {
			ch = splitSepIntersectionChar(
				vertCells[[2]int{x, y - 1}], vertCells[[2]int{x, y + 1}],
				left, right,
			)
		}
		buf.SetString(geom.Point{X: x, Y: y0 + y}, ch, sepTUI)
	}
	for cell := range horizCells {
		x, y := cell[0], cell[1]
		if vertCells[[2]int{x, y}] {
			continue
		}
		above := vertCells[[2]int{x, y - 1}]
		below := vertCells[[2]int{x, y + 1}]
		ch := horizSplit
		if above || below {
			ch = splitSepIntersectionChar(
				above, below,
				horizCells[[2]int{x - 1, y}],
				horizCells[[2]int{x + 1, y}],
			)
		}
		buf.SetString(geom.Point{X: x, Y: y0 + y}, ch, sepTUI)
	}

	r.renderCmdline(buf, r.size.Height-1)

	r.renderDiagnosticPopup(buf)

	if r.ec.keys.infoTitle != "" || len(r.ec.keys.infoItems) > 0 {
		r.renderInfoOverlay(buf)
	}
}

func paneUnderOverlay(cx *Context, a geom.Area, y0 int) bool {
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

func (r *renderPass) renderInfoOverlay(buf *tui.Buffer) {
	items := r.ec.keys.infoItems
	title := r.ec.keys.infoTitle
	th := r.activeTheme()

	popupSt := th.Get("ui.popup")
	popupTUI := popupSt

	keyW := 0
	for _, item := range items {
		if w := runewidth.StringWidth(item.Key); w > keyW {
			keyW = w
		}
	}
	rawLines := make([]string, len(items))
	bodyW := 0
	for i, item := range items {
		rawLines[i] = fmt.Sprintf("%-*s  %s", keyW, item.Key, item.Label)
		if w := runewidth.StringWidth(rawLines[i]); w > bodyW {
			bodyW = w
		}
	}
	if tw := runewidth.StringWidth(title); tw > bodyW {
		bodyW = tw
	}

	pop := popup{
		border:       lipgloss.RoundedBorder(),
		borderStyle:  popupTUI,
		contentStyle: popupTUI,
		padX:         1,
	}
	boxW := bodyW + 2 + 2*pop.padX
	boxH := len(rawLines) + 2
	x := max(r.size.Width-boxW, 0)
	y := max(r.size.Height-boxH-1, 0)

	area := pop.drawInto(buf, geom.Area{
		Point: geom.Point{X: x, Y: y},
		Size:  geom.Size{Width: boxW, Height: boxH},
	})

	if title != "" {
		buf.SetString(geom.Point{X: x + 1, Y: y}, " "+title+" ", popupTUI)
	}
	for i, raw := range rawLines {
		buf.SetString(area.Point.Add(geom.Point{Y: i}), raw, popupTUI)
	}
}

func splitSepIntersectionChar(above, below, left, right bool) string {
	idx := 0
	if above {
		idx |= 1
	}
	if below {
		idx |= 2
	}
	if left {
		idx |= 4
	}
	if right {
		idx |= 8
	}
	return splitSepIntersectionChars[idx]
}
