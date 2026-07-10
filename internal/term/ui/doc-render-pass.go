package ui

import (
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

// renderPass bundles the state needed for a single render pass so every render
// helper receives it without passing cx and ec separately
type renderPass struct {
	ec *EditorComponent
	cx *Context
	w  int
	h  int
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
	x, y, ok := r.ec.caretScreenPos(r.cx)
	if !ok {
		return tea.Cursor{}, false
	}
	return tea.Cursor{
		Position: tea.Position{X: x, Y: y},
		Shape:    cursorKindToShape(kind),
		Blink:    false,
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
	sel := doc.SelectionFor(v.ID())
	if !v.SyncFreeScroll(doc.Revision(), sel) {
		v.EnsureCursorVisible(text, sel, contentH, scrolloff, vf)
	}
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

func (r *renderPass) forceFullRedraw(cache *renderCache, th *theme.Theme) bool {
	stylesKey := th.Name() + "\x00" + r.cx.Editor.Mode().String()
	stylesChanged := cache.stylesKey != stylesKey

	gen := r.cx.Editor.Options().Gen
	optionsChanged := cache.lastOptionsGen != gen
	cache.lastOptionsGen = gen

	infoChanged := cache.lastInfoTitle != r.ec.infoTitle ||
		!slices.Equal(cache.lastInfoItems, r.ec.infoItems)
	cache.lastInfoTitle = r.ec.infoTitle
	cache.lastInfoItems = r.ec.infoItems

	return stylesChanged || optionsChanged || r.cx.OverlaysChanged ||
		infoChanged
}

func (r *renderPass) renderEditorContent(buf *tui.Buffer) {
	th := r.activeTheme()
	cache := r.ec.cache

	redrawAll := r.forceFullRedraw(cache, th)
	bgTUI := lipglossToTUIStyle(th.Get("ui.background"))
	if redrawAll {
		buf.Fill(bgTUI)
	}

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
		vDirty := v.ConsumeDirty()
		dDirty := doc.ConsumeDirty(v.ID())
		forced := !r.cx.SingleLayer && paneUnderOverlay(r.cx, v.Area(), y0)
		if !redrawAll && !forced && !vDirty && !dDirty {
			continue
		}
		if !redrawAll {
			clearPaneRect(buf, v.Area(), y0, bgTUI)
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
		ch := vertSplit
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
		ch := horizSplit
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

	r.renderDiagnosticPopup(buf)

	if r.ec.infoTitle != "" || len(r.ec.infoItems) > 0 {
		r.renderInfoOverlay(buf)
	}
}

func (r *renderPass) renderDiagnosticPopup(buf *tui.Buffer) {
	doc, ok := r.cx.Editor.FocusedDocument()
	if !ok {
		return
	}
	v, ok := r.cx.Editor.FocusedView()
	if !ok {
		return
	}
	diag, ok := diagnosticAtCursor(doc, v)
	if !ok {
		return
	}
	text := diagnosticPopupText(diag)
	if text == "" {
		return
	}
	r.drawDiagnosticPopup(buf, text, diag.Severity)
}

func (r *renderPass) drawDiagnosticPopup(
	buf *tui.Buffer, text string, severity view.DiagnosticSeverity,
) {
	maxW := min(buf.Width, 60)
	lines := diagnosticPopupLines(text, max(maxW-4, 1), 4)
	if len(lines) == 0 {
		return
	}
	bodyW := 0
	for _, line := range lines {
		bodyW = max(bodyW, runewidth.StringWidth(line))
	}
	st := diagnosticPopupStyle(r.cx, severity)
	pop := popup{
		border:       lipgloss.RoundedBorder(),
		borderStyle:  st,
		contentStyle: st,
		padX:         1,
	}
	w := min(bodyW+2+2*pop.padX, maxW)
	h := len(lines) + 2
	x := max(buf.Width-w, 0)
	y := 0
	if bufferlineVisible(r.cx) {
		y = 1
	}
	if y+h > buf.Height {
		y = max(buf.Height-h, 0)
	}
	area := pop.drawInto(buf, x, y, w, h)
	for i, line := range lines {
		buf.SetString(area.x, area.y+i, line, st)
	}
}

func paneUnderOverlay(cx *Context, a view.Area, y0 int) bool {
	if !cx.OverlayRegionsPrecise {
		return true
	}
	ax, ay := a.X, y0+a.Y
	for _, b := range cx.OverlayRegions {
		if ax < b.x+b.w && b.x < ax+a.Width &&
			ay < b.y+b.h && b.y < ay+a.Height {
			return true
		}
	}
	return false
}

func clearPaneRect(buf *tui.Buffer, a view.Area, y0 int, style tui.Style) {
	// redo the full-buffer Fill writeFillToBuffer trusts, just this pane
	top := y0 + a.Y
	for y := top; y < top+a.Height; y++ {
		buf.FillRange(a.X, y, a.Width, style)
	}
}

func diagnosticPopupStyle(
	cx *Context, severity view.DiagnosticSeverity,
) tui.Style {
	bg := lipglossToTUIStyle(cx.Theme().Get("ui.popup")).BgColor()
	var scope string
	switch severity {
	case view.DiagnosticSeverityError:
		scope = "diagnostic.error"
	case view.DiagnosticSeverityWarning:
		scope = "diagnostic.warning"
	case view.DiagnosticSeverityInfo:
		scope = "diagnostic.info"
	case view.DiagnosticSeverityHint:
		scope = "diagnostic.hint"
	default:
		return lipglossToTUIStyle(cx.Theme().Get("ui.popup"))
	}
	st := lipglossToTUIStyle(cx.Theme().Get(scope))
	fg := st.FgColor()
	if fg.IsReset() {
		fg = st.UnderlineColor()
	}
	return tui.Style{}.Fg(fg).Bg(bg)
}

func diagnosticAtCursor(
	doc *view.Document, v *view.View,
) (view.Diagnostic, bool) {
	cursor := doc.SelectionFor(v.ID()).Primary().Cursor(doc.Text())
	var best view.Diagnostic
	ok := false
	for _, diag := range doc.Diagnostics() {
		from, to := diagnosticRangeBounds(diag)
		if cursor < from || cursor >= to || diag.Message == "" {
			continue
		}
		if !ok || diag.Severity > best.Severity {
			best = diag
			ok = true
		}
	}
	return best, ok
}

func diagnosticRangeBounds(diag view.Diagnostic) (int, int) {
	from := diag.Range.From
	to := diag.Range.To
	if from > to {
		from, to = to, from
	}
	if from == to {
		to++
	}
	return from, to
}

func diagnosticPopupText(diag view.Diagnostic) string {
	msg := diagnosticMessageText(diag.Message)
	if diag.Source == "" {
		return msg
	}
	return diag.Source + ": " + msg
}

func diagnosticMessageText(message string) string {
	lines := strings.FieldsFunc(message, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
	return strings.Join(lines, "  ")
}

func diagnosticPopupLines(text string, width, maxLines int) []string {
	text = strings.TrimSpace(text)
	if text == "" || width <= 0 || maxLines <= 0 {
		return nil
	}
	var lines []string
	for line := range strings.SplitSeq(lipgloss.Wrap(text, width, ""), "\n") {
		line = strings.TrimRight(line, " \t")
		if line == "" {
			continue
		}
		lines = append(lines, line)
		if len(lines) == maxLines {
			break
		}
	}
	return lines
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
		return splitIntersect
	case above && below && right:
		return splitLeftT
	case above && below && left:
		return splitRightT
	case above && below:
		return vertSplit
	case above && left && right:
		return splitUpT
	case below && left && right:
		return splitDownT
	default:
		return horizSplit
	}
}
