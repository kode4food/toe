package ui

import (
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

const (
	vertSplitChar  = "│" // '│' box drawings light vertical
	horizSplitChar = "─" // '─' box drawings light horizontal
)

// renderPass bundles the state needed for a single render pass so every render
// helper receives it without passing cx and ec separately
type renderPass struct {
	ec *EditorComponent
	cx *Context
	w  int
	h  int
}

func (r *renderPass) renderBufferline(buf *tui.Buffer, y int) {
	th := r.activeTheme()
	bgTUI := lipglossToTUIStyle(th.Get("ui.bufferline.background"))
	activeTUI := lipglossToTUIStyle(th.Get("ui.bufferline.active"))
	inactiveTUI := lipglossToTUIStyle(th.Get("ui.bufferline"))

	buf.SetString(0, y, strings.Repeat(" ", r.w), bgTUI)

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
		buf.SetString(x, y, label, style)
		x += runewidth.StringWidth(label)
	}
}

func (r *renderPass) editorCursor() (tea.Cursor, bool) {
	doc, ok := r.cx.Editor.FocusedDocument()
	if !ok {
		return tea.Cursor{}, false
	}
	v, ok := r.cx.Editor.FocusedView()
	if !ok {
		return tea.Cursor{}, false
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
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	cursor := sel.Primary().Cursor(text)
	g0 := opts.Gutters
	gutterW := max(lineNumberDigits(text), g0.LineNumberMinWidth()) + 1
	area := v.Area()
	if !g0.HasGutterType(view.GutterTypeLineNumbers) {
		gutterW = 0
	}
	xOff := area.X
	yOff := area.Y
	if bufferlineVisible(r.cx) {
		yOff++
	}

	rowMap := r.ec.cache.viewRowMaps[v.ID()]
	visualY, visualX := cursorScreenPos(cursorScreenPosArgs{
		text: text, cursor: cursor, gutterW: gutterW,
		rowMap: rowMap, tabW: doc.TabWidth(),
		hOff: v.Offset().HorizontalOffset,
	})
	return tea.Cursor{
		Position: tea.Position{
			X: xOff + visualX,
			Y: yOff + visualY,
		},
		Shape: cursorKindToShape(kind),
		Blink: false,
	}, true
}

type renderPaneArgs struct {
	doc     *view.Document
	view    *view.View
	buf     *tui.Buffer
	y0      int
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
	v.EnsureCursorVisible(
		text, doc.SelectionFor(v.ID()), contentH, scrolloff, vf,
	)
	r.renderContent(renderContentArgs{
		doc:     doc,
		view:    v,
		buf:     args.buf,
		x:       editorX,
		y:       args.y0 + a.Y,
		width:   editorW,
		height:  contentH,
		focused: args.focused,
	})
	r.renderStatus(renderStatusArgs{
		doc:     doc,
		view:    v,
		buf:     args.buf,
		x:       a.X,
		y:       args.y0 + a.Y + contentH,
		width:   a.Width,
		focused: args.focused,
	})
}

func (r *renderPass) renderEditorContent(buf *tui.Buffer) {
	th := r.activeTheme()

	bgTUI := lipglossToTUIStyle(th.Get("ui.background"))
	buf.Fill(bgTUI)

	y0 := 0
	if bufferlineVisible(r.cx) {
		r.renderBufferline(buf, 0)
		y0 = 1
	}

	for _, vs := range r.cx.Editor.Tree().Views() {
		v := vs.View
		doc, ok := r.cx.Editor.Document(v.DocID())
		if !ok {
			continue
		}
		r.renderPane(renderPaneArgs{
			doc: doc, view: v, buf: buf, y0: y0, focused: vs.Focused,
		})
	}

	sepTUI := lipglossToTUIStyle(th.Get("ui.border"))
	vertCells := make(map[[2]int]bool)
	horizCells := make(map[[2]int]bool)
	r.cx.Editor.Tree().WalkSeparators(func(s view.Separator) {
		if s.Layout == view.LayoutVertical {
			for row := s.Y; row < s.Y+s.H; row++ {
				vertCells[[2]int{s.X, row}] = true
			}
		} else {
			for col := s.X; col < s.X+s.W; col++ {
				horizCells[[2]int{col, s.Y}] = true
			}
		}
	})
	for cell := range vertCells {
		x, y := cell[0], cell[1]
		left := horizCells[[2]int{x - 1, y}]
		right := horizCells[[2]int{x + 1, y}]
		ch := vertSplitChar
		if left || right {
			ch = splitSepIntersectionChar(
				vertCells[[2]int{x, y - 1}], vertCells[[2]int{x, y + 1}],
				left, right,
			)
		}
		buf.SetString(x, y0+y, ch, sepTUI)
	}
	for cell := range horizCells {
		x, y := cell[0], cell[1]
		if vertCells[[2]int{x, y}] {
			continue
		}
		above := vertCells[[2]int{x, y - 1}]
		below := vertCells[[2]int{x, y + 1}]
		ch := horizSplitChar
		if above || below {
			ch = splitSepIntersectionChar(
				above, below,
				horizCells[[2]int{x - 1, y}],
				horizCells[[2]int{x + 1, y}],
			)
		}
		buf.SetString(x, y0+y, ch, sepTUI)
	}

	r.renderCmdline(buf, r.h-1)

	if r.ec.infoTitle != "" || len(r.ec.infoItems) > 0 {
		r.renderInfoOverlay(buf)
	}
}

func (r *renderPass) renderInfoOverlay(buf *tui.Buffer) {
	items := r.ec.infoItems
	title := r.ec.infoTitle
	th := r.activeTheme()

	popupSt := th.Get("ui.popup")
	popupTUI := lipglossToTUIStyle(popupSt)

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
	x := max(r.w-boxW, 0)
	y := max(r.h-boxH-1, 0)

	area := pop.drawInto(buf, x, y, boxW, boxH)

	if title != "" {
		buf.SetString(x+1, y, " "+title+" ", popupTUI)
	}
	for i, raw := range rawLines {
		buf.SetString(area.x, area.y+i, raw, popupTUI)
	}
}

func splitSepIntersectionChar(above, below, left, right bool) string {
	switch {
	case above && below && left && right:
		return "┼"
	case above && below && right:
		return "├"
	case above && below && left:
		return "┤"
	case above && below:
		return "│"
	case above && left && right:
		return "┴"
	case below && left && right:
		return "┬"
	default:
		return "─"
	}
}
