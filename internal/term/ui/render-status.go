package ui

import (
	"fmt"
	"image/color"
	"path/filepath"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/highlight"
	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type (
	lipglossStyles struct {
		text              lipgloss.Style
		line              lipgloss.Style
		lineSelected      lipgloss.Style
		selection         lipgloss.Style
		cursor            lipgloss.Style
		cursorPrim        lipgloss.Style
		cursorLinePrim    lipgloss.Style
		cursorLineSec     lipgloss.Style
		cursorColumn      lipgloss.Style
		whitespace        lipgloss.Style
		indentGuide       lipgloss.Style
		ruler             lipgloss.Style
		inlayHint         lipgloss.Style
		inlayHintType     lipgloss.Style
		inlayHintParam    lipgloss.Style
		severityHint      lipgloss.Style
		severityInfo      lipgloss.Style
		severityWarning   lipgloss.Style
		severityError     lipgloss.Style
		diagnostic        lipgloss.Style
		diagnosticHint    lipgloss.Style
		diagnosticInfo    lipgloss.Style
		diagnosticWarning lipgloss.Style
		diagnosticError   lipgloss.Style
		documentHighlight lipgloss.Style
		documentLink      lipgloss.Style
		searchMatch       lipgloss.Style
		diffAdded         lipgloss.Style
		diffModified      lipgloss.Style
		diffRemoved       lipgloss.Style
	}

	tuiStyles struct {
		text              tui.Style
		selection         tui.Style
		cursor            tui.Style
		cursorPrim        tui.Style
		cursorLinePrim    tui.Style
		cursorLineSec     tui.Style
		cursorColumn      tui.Style
		whitespace        tui.Style
		indentGuide       tui.Style
		inlayHint         tui.Style
		inlayHintType     tui.Style
		inlayHintParam    tui.Style
		severityHint      tui.Style
		severityInfo      tui.Style
		severityWarning   tui.Style
		severityError     tui.Style
		diagnostic        tui.Style
		diagnosticHint    tui.Style
		diagnosticInfo    tui.Style
		diagnosticWarning tui.Style
		diagnosticError   tui.Style
		documentHighlight tui.Style
		documentLink      tui.Style
		searchMatch       tui.Style
		diffAdded         tui.Style
		diffModified      tui.Style
		diffRemoved       tui.Style
	}

	// statusElem is a single rendered piece of a status bar
	statusElem struct {
		text   string
		style  tui.Style
		kind   view.StatusLineElement
		pinned bool
	}

	statusElemCtx struct {
		doc        *view.Document
		opts       *view.Options
		mode       string
		baseTUI    tui.Style
		modeSt     tui.Style
		sepSt      tui.Style
		sep        string
		nSel       int
		primIdx    int
		primLen    int
		totalLines int
		reg        rune
		cwd        string
		row        int
		col        int
		vcsHead    string
	}
)

var statusElemFns = map[view.StatusLineElement]func(*statusElemCtx) statusElem{
	view.StatusLineMode:             statusElemMode,
	view.StatusLineSeparator:        statusElemSeparator,
	view.StatusLineFileName:         statusElemFileName,
	view.StatusLineFileBaseName:     statusElemFileBaseName,
	view.StatusLineFileAbsolutePath: statusElemFileAbsPath,
	view.StatusLineReadOnly:         statusElemReadOnly,
	view.StatusLineModified:         statusElemModified,
	view.StatusLineSelections:       statusElemSelections,
	view.StatusLinePrimaryLen:       statusElemPrimaryLen,
	view.StatusLinePosition:         statusElemPosition,
	view.StatusLinePercent:          statusElemPercent,
	view.StatusLineTotalLines:       statusElemTotalLines,
	view.StatusLineFileEncoding:     statusElemEncoding,
	view.StatusLineFileLineEnding:   statusElemLineEnding,
	view.StatusLineSpacer:           statusElemSpacer,
	view.StatusLineFileIndentStyle:  statusElemIndentStyle,
	view.StatusLineFileType:         statusElemFileType,
	view.StatusLineDiagnostics:      statusElemDiagnostics,
	view.StatusLineRegister:         statusElemRegister,
	view.StatusLineVersionControl:   statusElemVersionControl,
}

func buildLipglossStyles(th *theme.Theme, mode view.Mode) lipglossStyles {
	sel := th.Get("ui.selection")
	cur, _ := modeCursorStyleFor(th, mode, false)
	curPrim, _ := modeCursorStyleFor(th, mode, true)
	cl := th.Get("ui.cursorline.primary")
	st := lipglossStyles{
		text:              th.Get("ui.text"),
		line:              th.Get("ui.linenr"),
		lineSelected:      th.Get("ui.linenr.selected"),
		selection:         sel,
		cursor:            cur,
		cursorPrim:        curPrim,
		cursorLinePrim:    cl,
		cursorLineSec:     cl,
		cursorColumn:      cl,
		whitespace:        th.Get("ui.virtual.whitespace"),
		indentGuide:       th.Get("ui.virtual.indent-guide"),
		ruler:             th.Get("ui.virtual.ruler"),
		inlayHint:         th.Get("ui.virtual"),
		inlayHintType:     th.Get("ui.virtual"),
		inlayHintParam:    th.Get("ui.virtual"),
		severityHint:      th.Get("hint"),
		severityInfo:      th.Get("info"),
		severityWarning:   th.Get("warning"),
		severityError:     th.Get("error"),
		diagnostic:        th.Get("diagnostic"),
		diagnosticHint:    th.Get("diagnostic.hint"),
		diagnosticInfo:    th.Get("diagnostic.info"),
		diagnosticWarning: th.Get("diagnostic.warning"),
		diagnosticError:   th.Get("diagnostic.error"),
		documentHighlight: th.Get("ui.highlight"),
		documentLink:      th.Get("markup.link.url"),
		searchMatch:       searchMatchStyle(),
		diffAdded:         th.Get("diff.plus.gutter"),
		diffModified:      th.Get("diff.delta.gutter"),
		diffRemoved:       th.Get("diff.minus.gutter"),
	}
	if next, ok := th.TryGet("ui.virtual.inlay-hint"); ok {
		st.inlayHint = next
		st.inlayHintType = next
		st.inlayHintParam = next
	}
	if next, ok := th.TryGet("ui.virtual.inlay-hint.type"); ok {
		st.inlayHintType = next
	}
	if next, ok := th.TryGet("ui.virtual.inlay-hint.parameter"); ok {
		st.inlayHintParam = next
	}
	if next, ok := th.TryGetExact("markup.link"); ok {
		st.documentLink = next
	}
	if next, ok := th.TryGetExact("markup.link.url"); ok {
		st.documentLink = next
	}
	if next, ok := th.TryGetExact("ui.selection.primary"); ok {
		st.selection = next
	}
	if next, ok := th.TryGet("ui.cursorline.secondary"); ok {
		st.cursorLineSec = next
	}
	if next, ok := th.TryGetExact("ui.cursorcolumn"); ok {
		st.cursorColumn = next
	}
	if next, ok := th.TryGetExact("ui.cursorcolumn.primary"); ok {
		st.cursorColumn = next
	}
	if next, ok := th.TryGet("ui.search.match"); ok {
		st.searchMatch = next
	}
	return st
}

func buildTUIStyles(s *lipglossStyles) *tuiStyles {
	return &tuiStyles{
		text:              lipglossToTUIStyle(s.text),
		selection:         lipglossToTUIStyle(s.selection),
		cursor:            lipglossToTUIStyle(s.cursor),
		cursorPrim:        lipglossToTUIStyle(s.cursorPrim),
		cursorLinePrim:    lipglossToTUIStyle(s.cursorLinePrim),
		cursorLineSec:     lipglossToTUIStyle(s.cursorLineSec),
		cursorColumn:      lipglossToTUIStyle(s.cursorColumn),
		whitespace:        lipglossToTUIStyle(s.whitespace),
		indentGuide:       lipglossToTUIStyle(s.indentGuide),
		inlayHint:         lipglossToTUIStyle(s.inlayHint),
		inlayHintType:     lipglossToTUIStyle(s.inlayHintType),
		inlayHintParam:    lipglossToTUIStyle(s.inlayHintParam),
		severityHint:      lipglossToTUIStyle(s.severityHint),
		severityInfo:      lipglossToTUIStyle(s.severityInfo),
		severityWarning:   lipglossToTUIStyle(s.severityWarning),
		severityError:     lipglossToTUIStyle(s.severityError),
		diagnostic:        lipglossToTUIStyle(s.diagnostic),
		diagnosticHint:    lipglossToTUIStyle(s.diagnosticHint),
		diagnosticInfo:    lipglossToTUIStyle(s.diagnosticInfo),
		diagnosticWarning: lipglossToTUIStyle(s.diagnosticWarning),
		diagnosticError:   lipglossToTUIStyle(s.diagnosticError),
		documentHighlight: lipglossToTUIStyle(s.documentHighlight),
		documentLink:      lipglossToTUIStyle(s.documentLink),
		searchMatch:       lipglossToTUIStyle(s.searchMatch),
		diffAdded:         lipglossToTUIStyle(s.diffAdded),
		diffModified:      lipglossToTUIStyle(s.diffModified),
		diffRemoved:       lipglossToTUIStyle(s.diffRemoved),
	}
}

func (r *renderPass) renderCmdline(buf *tui.Buffer, y int) {
	w := r.w
	isErr := r.ec.cmdMsg != "" &&
		strings.HasPrefix(r.ec.cmdMsg, "error:")
	st := r.cmdlineStyle(isErr)
	tuiSt := lipglossToTUIStyle(st)

	left := r.ec.cmdMsg
	right := r.ec.hint
	if right == "" {
		right = r.ec.status
	}
	if r.ec.macroSlot.recording {
		right += fmt.Sprintf("[%c]", r.ec.macroSlot.reg)
	}

	buf.SetString(0, y, strings.Repeat(" ", w), tuiSt)
	if left == "" && right == "" {
		return
	}
	rightW := runewidth.StringWidth(right)
	leftW := max(w-rightW, 0)
	leftStr := ansi.Truncate(left, leftW, "")
	buf.SetString(0, y, leftStr, tuiSt)
	if rightW > 0 && rightW <= w {
		buf.SetString(w-rightW, y, right, tuiSt)
	}
}

type renderStatusArgs struct {
	doc     *view.Document
	view    *view.View
	buf     *tui.Buffer
	x, y    int
	width   int
	focused bool
}

func (r *renderPass) renderStatus(args renderStatusArgs) {
	doc := args.doc
	v := args.view
	buf := args.buf
	x := args.x
	y := args.y
	width := args.width
	isFocused := args.focused
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	prim := sel.Primary()
	cursor := prim.Cursor(text)

	row, col := 1, 1
	if l, err := text.CharToLine(cursor); err == nil {
		row = l + 1
		if lineStart, err2 := text.LineToChar(l); err2 == nil {
			col = cursor - lineStart + 1
		}
	}

	opts := r.cx.Editor.Options()
	mode := v.Mode().String()

	th := r.activeTheme()

	statusKey := "ui.statusline"
	if !isFocused {
		statusKey = "ui.statusline.inactive"
	}
	st := th.Get(statusKey)

	modeSt := st
	if isFocused {
		scope := "ui.statusline." + modeScopeName(mode)
		modeSt = th.Get(scope)
	}

	sepSt := st
	if s, ok := th.TryGet("ui.statusline.separator"); ok {
		sepSt = s
	}

	nSel := len(sel.Ranges())
	primIdx := sel.PrimaryIndex()
	primLen := prim.Len()
	totalLines := text.LenLines()
	reg := r.cx.Editor.ActiveRegister()
	cwd := r.cx.Editor.Cwd()
	sep := opts.StatusLineSeparator()
	var vcsHead string
	if vc := r.cx.Editor.VersionControl(); vc != nil {
		vcsHead, _ = vc.HeadName(doc)
	}

	baseTUI := lipglossToTUIStyle(st)

	src := &statusElemCtx{
		doc: doc, opts: opts, mode: mode,
		baseTUI: baseTUI,
		modeSt:  lipglossToTUIStyle(modeSt),
		sepSt:   lipglossToTUIStyle(sepSt),
		sep:     sep, nSel: nSel, primIdx: primIdx, primLen: primLen,
		totalLines: totalLines, reg: reg, cwd: cwd,
		row: row, col: col,
		vcsHead: vcsHead,
	}

	collectElems := func(items []view.StatusLineItem) []statusElem {
		out := make([]statusElem, 0, len(items))
		for _, e := range items {
			if se := src.elem(e); se.text != "" {
				out = append(out, se)
			}
		}
		return out
	}

	left := collectElems(opts.StatusLineLeft())
	right := collectElems(opts.StatusLineRight())

	elemsWidth := func(elems []statusElem) int {
		w := 0
		for _, e := range elems {
			w += runewidth.StringWidth(e.text)
		}
		return w
	}

	// Elements suffixed "!" in config are pinned and never dropped. When the
	// bar is too narrow, sections shed from their inner edge so edge-anchored
	// items survive longest: the right section drops from its left, the left
	// section from its right. Right section first, then left
	dropOne := func(elems []statusElem, fromEnd bool) ([]statusElem, bool) {
		for n, i := len(elems), 0; i < n; i++ {
			idx := i
			if fromEnd {
				idx = n - 1 - i
			}
			if !elems[idx].pinned {
				return slices.Delete(elems, idx, idx+1), true
			}
		}
		return elems, false
	}
	for elemsWidth(left)+elemsWidth(right) > width {
		var ok bool
		if right, ok = dropOne(right, false); ok {
			continue
		}
		if left, ok = dropOne(left, true); !ok {
			break
		}
	}

	writeElems := func(elems []statusElem, x int) {
		for _, e := range elems {
			buf.SetString(x, y, e.text, e.style)
			x += runewidth.StringWidth(e.text)
		}
	}

	buf.SetString(x, y, strings.Repeat(" ", width), baseTUI)

	writeElems(left, x)

	rightW := elemsWidth(right)
	writeElems(right, x+width-rightW)
}

func (r *renderPass) activeTheme() *theme.Theme {
	return r.cx.Theme()
}

func (r *renderPass) cmdlineStyle(errorMsg bool) lipgloss.Style {
	th := r.activeTheme()
	if errorMsg {
		return th.Get("error")
	}
	return th.Get("ui.statusline")
}

func (s *statusElemCtx) elem(e view.StatusLineItem) statusElem {
	if fn, ok := statusElemFns[e.Element]; ok {
		se := fn(s)
		se.kind = e.Element
		se.pinned = e.Pinned
		return se
	}
	return statusElem{}
}

func clearStyleBackground(st lipgloss.Style) lipgloss.Style {
	if lipglossColorToTUI(st.GetBackground()).IsReset() {
		return st
	}
	return st.Background(lipgloss.NoColor{})
}

func inheritStyleBackground(st lipgloss.Style, bg color.Color) lipgloss.Style {
	if lipglossColorToTUI(bg).IsReset() {
		return st
	}
	if !lipglossColorToTUI(st.GetBackground()).IsReset() {
		return st
	}
	return st.Background(bg)
}

// Resolved styles carry no background of their own: they render transparently
// over whatever layer sits beneath (base fill, ruler, cursorline). A scope with
// an explicit background in the theme is treated as an intentional override
func hlStyleFnFor(th *theme.Theme) func(string) lipgloss.Style {
	return func(scope string) lipgloss.Style {
		if s, ok := th.TryGet(scope); ok {
			return s
		}
		return highlight.DefaultStyle(scope)
	}
}

func modeScopeName(mode string) string {
	switch mode {
	case "INS":
		return "insert"
	case "SEL":
		return "select"
	default:
		return "normal"
	}
}

func modeCursorStyleFor(
	th *theme.Theme, mode view.Mode, primary bool,
) (lipgloss.Style, bool) {
	modeStr := modeScopeName(mode.String())
	scope := "ui.cursor." + modeStr
	if primary {
		scope = "ui.cursor.primary." + modeStr
	}
	return th.TryGetExact(scope)
}

func cursorKindToShape(kind view.CursorKind) tea.CursorShape {
	switch kind {
	case view.CursorKindBar:
		return tea.CursorBar
	case view.CursorKindUnderline:
		return tea.CursorUnderline
	default:
		return tea.CursorBlock
	}
}

func statusElemMode(s *statusElemCtx) statusElem {
	return statusElem{
		text:  " " + s.opts.ModeNameForMode(s.mode) + " ",
		style: s.modeSt,
	}
}

func statusElemSeparator(s *statusElemCtx) statusElem {
	return statusElem{text: s.sep, style: s.sepSt}
}

func statusElemFileName(s *statusElemCtx) statusElem {
	return statusElem{
		text:  " " + s.doc.RelativeName(s.cwd) + " ",
		style: s.baseTUI,
	}
}

func statusElemFileBaseName(s *statusElemCtx) statusElem {
	return statusElem{
		text:  " " + filepath.Base(s.doc.Path()) + " ",
		style: s.baseTUI,
	}
}

func statusElemFileAbsPath(s *statusElemCtx) statusElem {
	return statusElem{text: " " + s.doc.Path() + " ", style: s.baseTUI}
}

func statusElemReadOnly(s *statusElemCtx) statusElem {
	if !s.doc.ReadOnly() {
		return statusElem{}
	}
	return statusElem{text: " [readonly]", style: s.baseTUI}
}

func statusElemModified(s *statusElemCtx) statusElem {
	if !s.doc.Modified() {
		return statusElem{}
	}
	return statusElem{text: "[modified] ", style: s.baseTUI}
}

func statusElemSelections(s *statusElemCtx) statusElem {
	if s.nSel == 1 {
		return statusElem{text: " 1 sel ", style: s.baseTUI}
	}
	return statusElem{
		text:  fmt.Sprintf(" %d/%d sels ", s.primIdx+1, s.nSel),
		style: s.baseTUI,
	}
}

func statusElemPrimaryLen(s *statusElemCtx) statusElem {
	return statusElem{
		text:  fmt.Sprintf(" %d ", s.primLen),
		style: s.baseTUI,
	}
}

func statusElemPosition(s *statusElemCtx) statusElem {
	return statusElem{
		text:  fmt.Sprintf(" %d:%d ", s.row, s.col),
		style: s.baseTUI,
	}
}

func statusElemPercent(s *statusElemCtx) statusElem {
	pct := 0
	if s.totalLines > 0 {
		pct = (s.row * 100) / s.totalLines
	}
	return statusElem{
		text:  fmt.Sprintf(" %d%% ", pct),
		style: s.baseTUI,
	}
}

func statusElemTotalLines(s *statusElemCtx) statusElem {
	return statusElem{
		text:  fmt.Sprintf(" %d ", s.totalLines),
		style: s.baseTUI,
	}
}

func statusElemEncoding(s *statusElemCtx) statusElem {
	label := "utf-8"
	if s.doc.HasBOM() {
		label = "utf-8-bom"
	}
	return statusElem{text: " " + label + " ", style: s.baseTUI}
}

func statusElemSpacer(s *statusElemCtx) statusElem {
	return statusElem{text: " ", style: s.baseTUI}
}

func statusElemLineEnding(s *statusElemCtx) statusElem {
	label := "lf"
	if s.doc.LineEnding() == core.LineEndingCRLF {
		label = "crlf"
	}
	return statusElem{text: " " + label + " ", style: s.baseTUI}
}

func statusElemIndentStyle(s *statusElemCtx) statusElem {
	indent := s.doc.IndentStyle()
	var label string
	if indent.IsTabs() {
		label = "tabs"
	} else {
		label = fmt.Sprintf("spaces:%d", indent.Width())
	}
	return statusElem{text: " " + label + " ", style: s.baseTUI}
}

func statusElemFileType(s *statusElemCtx) statusElem {
	lang := s.doc.Lang()
	if lang == "" {
		lang = "text"
	}
	return statusElem{text: " " + lang + " ", style: s.baseTUI}
}

func statusElemDiagnostics(s *statusElemCtx) statusElem {
	counts := s.doc.DiagnosticCounts()
	var parts []string
	if counts.Errors > 0 {
		parts = append(parts, fmt.Sprintf("E:%d", counts.Errors))
	}
	if counts.Warnings > 0 {
		parts = append(parts, fmt.Sprintf("W:%d", counts.Warnings))
	}
	if counts.Info > 0 {
		parts = append(parts, fmt.Sprintf("I:%d", counts.Info))
	}
	if counts.Hints > 0 {
		parts = append(parts, fmt.Sprintf("H:%d", counts.Hints))
	}
	if len(parts) == 0 {
		return statusElem{}
	}
	return statusElem{
		text:  " " + strings.Join(parts, " ") + " ",
		style: s.baseTUI,
	}
}

func statusElemVersionControl(s *statusElemCtx) statusElem {
	if s.vcsHead == "" {
		return statusElem{}
	}
	return statusElem{text: " " + s.vcsHead + " ", style: s.baseTUI}
}

func statusElemRegister(s *statusElemCtx) statusElem {
	if s.reg == 0 {
		return statusElem{}
	}
	return statusElem{
		text:  fmt.Sprintf(" reg=%c ", s.reg),
		style: s.baseTUI,
	}
}
