package ui

import (
	"fmt"
	"slices"
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

var splitSepIntersectionChars = [...]string{
	borderH, borderH, borderH, borderV,
	borderH, borderH, borderH, borderMR,
	borderH, borderH, borderH, borderML,
	borderH, borderMB, borderMT, borderX,
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
	p := r.context.Editor.Tree().Get(r.context.Editor.Tree().Focus())
	if pc, ok := p.(PaneCursor); ok {
		return pc.Cursor(r.context)
	}
	opts := r.context.Editor.Options()
	kind := opts.CursorShapeForMode(r.context.Editor.Mode())
	switch kind {
	case view.CursorKindHidden:
		return tea.Cursor{}, false
	case view.CursorKindBlock:
		if r.editor.focused {
			// block cursor drawn manually in content; terminal cursor hidden
			return tea.Cursor{}, false
		}
		// terminal lost focus: use underline so position is still visible
		kind = view.CursorKindUnderline
	}
	if at, ok := r.editor.caretScreenPos(r.context); ok {
		return tea.Cursor{
			Position: tea.Position{X: at.X, Y: at.Y},
			Shape:    cursorKindToShape(kind),
		}, true
	}
	return tea.Cursor{}, false
}

type renderPaneArgs struct {
	doc     *view.Document
	view    *view.View
	buf     *tui.Buffer
	yOff    int
	focused bool
}

func (r *renderPass) renderPane(args renderPaneArgs) {
	doc := args.doc
	v := args.view
	a := v.Area()
	opts := r.context.Editor.Options()
	scrolloff := opts.ScrollOff
	contentH := max(a.Height-1, 0)
	editorX := a.X
	editorW := a.Width

	// Build the soft-wrap layout so vertical visibility is measured in visual
	// rows; nil keeps the text-line fallback when soft-wrap is off
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
			Point: geom.Point{X: editorX, Y: args.yOff + a.Y},
			Size:  geom.Size{Width: editorW, Height: contentH},
		},
		focused: args.focused,
	})
	r.renderStatus(renderStatusArgs{
		doc:     doc,
		view:    v,
		buf:     args.buf,
		at:      geom.Point{X: a.X, Y: args.yOff + a.Y + contentH},
		width:   a.Width,
		focused: args.focused,
	})
}

func (r *renderPass) forceFullRedraw(cache *renderCache, th *theme.Theme) bool {
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

	if cache.lastInfoTitle != r.editor.keys.infoTitle ||
		!slices.Equal(cache.lastInfoItems, r.editor.keys.infoItems) {
		cache.lastInfoTitle = r.editor.keys.infoTitle
		cache.lastInfoItems = r.editor.keys.infoItems
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

	if cache.lastSpinner != r.editor.spinner {
		cache.lastSpinner = r.editor.spinner
		force = true
	}

	return force
}

type beginPaneRedrawArgs struct {
	buf        *tui.Buffer
	pane       view.Pane
	yOff       int
	dirty      bool
	redrawAll  bool
	focused    bool
	background tui.Style
}

// a focused pane's background already matches the frame's base fill, so a
// full redraw re-clears only an unfocused (dimmed) one
func (r *renderPass) beginPaneRedraw(args beginPaneRedrawArgs) bool {
	redraw := args.redrawAll
	if !redraw {
		forced := !r.context.composition.singleLayer &&
			paneUnderOverlay(r.context, args.pane.Area(), args.yOff)
		redraw = forced || args.dirty
	}
	if !redraw {
		return false
	}
	if !args.redrawAll || !args.focused {
		clearPaneRect(args.buf, args.pane.Area(), args.yOff, args.background)
	}
	return true
}

func (r *renderPass) renderEditorContent(buf *tui.Buffer) {
	th := r.context.Theme()
	cache := r.editor.cache

	redrawAll := r.forceFullRedraw(cache, th)
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
				yOff:       y0,
				dirty:      dirty,
				redrawAll:  redrawAll,
				focused:    focused,
				background: paneBg,
			}) {
				r.renderPane(renderPaneArgs{
					doc:     doc,
					view:    pane,
					buf:     buf,
					yOff:    y0,
					focused: focused,
				})
			}
		case *ImagePane:
			if r.beginPaneRedraw(beginPaneRedrawArgs{
				buf:        buf,
				pane:       pane,
				yOff:       y0,
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
				yOff:       y0,
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
				yOff:       y0,
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
		x, y := cell[0], cell[1]
		left := horizCells[[2]int{x - 1, y}]
		right := horizCells[[2]int{x + 1, y}]
		ch := borderV
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
		x, y := cell[0], cell[1]
		if vertCells[[2]int{x, y}] {
			continue
		}
		above := vertCells[[2]int{x, y - 1}]
		below := vertCells[[2]int{x, y + 1}]
		ch := borderH
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

	r.renderCmdline(buf, r.size.Height-1)

	r.renderDiagnosticPopup(buf)

	if r.editor.keys.infoTitle != "" || len(r.editor.keys.infoItems) > 0 {
		r.renderInfoOverlay(buf)
	}
}

func (r *renderPass) renderInfoOverlay(buf *tui.Buffer) {
	items := r.editor.keys.infoItems
	title := r.editor.keys.infoTitle
	th := r.context.Theme()

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
