package ui

import (
	"fmt"
	"image/color"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/highlight"
	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type statusElemCtx struct {
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
	rightW := ansi.StringWidth(right)
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
	x0 := args.x
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

	baseTUI := lipglossToTUIStyle(st)

	src := &statusElemCtx{
		doc: doc, opts: opts, mode: mode,
		baseTUI: baseTUI,
		modeSt:  lipglossToTUIStyle(modeSt),
		sepSt:   lipglossToTUIStyle(sepSt),
		sep:     sep, nSel: nSel, primIdx: primIdx, primLen: primLen,
		totalLines: totalLines, reg: reg, cwd: cwd,
		row: row, col: col,
	}

	collectElems := func(elems []view.StatusLineElement) []statusElem {
		out := make([]statusElem, 0, len(elems))
		for _, e := range elems {
			if se := src.elem(e); se.text != "" {
				out = append(out, se)
			}
		}
		return out
	}

	left := collectElems(opts.StatusLineLeft())
	center := collectElems(opts.StatusLineCenter())
	right := collectElems(opts.StatusLineRight())

	elemsWidth := func(elems []statusElem) int {
		w := 0
		for _, e := range elems {
			w += ansi.StringWidth(e.text)
		}
		return w
	}

	writeElems := func(elems []statusElem, x int) {
		for _, e := range elems {
			buf.SetString(x, y, e.text, e.style)
			x += ansi.StringWidth(e.text)
		}
	}

	buf.SetString(x0, y, strings.Repeat(" ", width), baseTUI)

	writeElems(left, x0)

	rightW := elemsWidth(right)
	writeElems(right, x0+width-rightW)

	if len(center) > 0 {
		centerW := elemsWidth(center)
		writeElems(center, x0+width/2-centerW/2)
	}
}

func (r *renderPass) activeTheme() *theme.Theme {
	return r.cx.Theme()
}

// buildLipglossStyles constructs a lipglossStyles from a loaded theme and mode
func buildLipglossStyles(th *theme.Theme, mode view.Mode) lipglossStyles {
	sel := th.Get("ui.selection")
	cur, _ := modeCursorStyleFor(th, mode, false)
	curPrim, _ := modeCursorStyleFor(th, mode, true)
	cl := th.Get("ui.cursorline.primary")
	st := lipglossStyles{
		background:       th.Get("ui.background"),
		text:             th.Get("ui.text"),
		line:             th.Get("ui.linenr"),
		lineSelected:     th.Get("ui.linenr.selected"),
		selection:        sel,
		selectionPrim:    sel,
		cursor:           cur,
		cursorPrim:       curPrim,
		cursorlinePrim:   cl,
		cursorlineSec:    cl,
		cursorcolumnPrim: cl,
		cursorcolumnSec:  cl,
		whitespace:       th.Get("ui.virtual.whitespace"),
		indentGuide:      th.Get("ui.virtual.indent-guide"),
		ruler:            th.Get("ui.virtual.ruler"),
		searchMatch:      searchMatchStyle(),
	}
	if next, ok := th.TryGetExact("ui.selection.primary"); ok {
		st.selectionPrim = next
	}
	if next, ok := th.TryGet("ui.cursorline.secondary"); ok {
		st.cursorlineSec = next
		st.cursorcolumnSec = next
	}
	if next, ok := th.TryGetExact("ui.cursorcolumn"); ok {
		st.cursorcolumnPrim = next
		st.cursorcolumnSec = next
	}
	if next, ok := th.TryGetExact("ui.cursorcolumn.primary"); ok {
		st.cursorcolumnPrim = next
	}
	if next, ok := th.TryGetExact("ui.cursorcolumn.secondary"); ok {
		st.cursorcolumnSec = next
	}
	if next, ok := th.TryGet("ui.search.match"); ok {
		st.searchMatch = next
	}
	st.inheritBackground()
	return st
}

func buildTUIStyles(s *lipglossStyles) *tuiStyles {
	return &tuiStyles{
		text:             lipglossToTUIStyle(s.text),
		selection:        lipglossToTUIStyle(s.selection),
		selectionPrim:    lipglossToTUIStyle(s.selectionPrim),
		cursor:           lipglossToTUIStyle(s.cursor),
		cursorPrim:       lipglossToTUIStyle(s.cursorPrim),
		cursorlinePrim:   lipglossToTUIStyle(s.cursorlinePrim),
		cursorlineSec:    lipglossToTUIStyle(s.cursorlineSec),
		cursorcolumnPrim: lipglossToTUIStyle(s.cursorcolumnPrim),
		cursorcolumnSec:  lipglossToTUIStyle(s.cursorcolumnSec),
		whitespace:       lipglossToTUIStyle(s.whitespace),
		indentGuide:      lipglossToTUIStyle(s.indentGuide),
		searchMatch:      lipglossToTUIStyle(s.searchMatch),
	}
}

func (r *lipglossStyles) inheritBackground() {
	bg := r.background.GetBackground()
	if lipglossColorToTUI(bg).IsReset() {
		return
	}
	r.text = inheritStyleBackground(r.text, bg)
	r.line = inheritStyleBackground(r.line, bg)
	r.lineSelected = inheritStyleBackground(r.lineSelected, bg)
	r.whitespace = inheritStyleBackground(r.whitespace, bg)
	r.indentGuide = inheritStyleBackground(r.indentGuide, bg)
}

// clearBackground strips the document background from all inherited styles so
// that preview content is background-transparent; the containing pane provides
// the background uniformly via its outer Render call
func (r *lipglossStyles) clearBackground() {
	r.background = lipgloss.NewStyle()
	r.text = clearStyleBackground(r.text)
	r.line = clearStyleBackground(r.line)
	r.lineSelected = clearStyleBackground(r.lineSelected)
	r.whitespace = clearStyleBackground(r.whitespace)
	r.indentGuide = clearStyleBackground(r.indentGuide)
}

func clearStyleBackground(st lipgloss.Style) lipgloss.Style {
	if lipglossColorToTUI(st.GetBackground()).IsReset() {
		return st
	}
	return st.Background(lipgloss.NoColor{})
}

func inheritStyleBackground(
	st lipgloss.Style, bg color.Color,
) lipgloss.Style {
	if lipglossColorToTUI(bg).IsReset() {
		return st
	}
	if !lipglossColorToTUI(st.GetBackground()).IsReset() {
		return st
	}
	return st.Background(bg)
}

// hlStyleFnFor returns a function that maps highlight scope names to styles,
// falling back to the default Chroma palette when the theme has no entry
func hlStyleFnFor(th *theme.Theme) func(string) lipgloss.Style {
	bg := th.Get("ui.background").GetBackground()
	return func(scope string) lipgloss.Style {
		if s, ok := th.TryGet(scope); ok {
			return inheritStyleBackground(s, bg)
		}
		return inheritStyleBackground(highlight.DefaultStyle(scope), bg)
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

func (r *renderPass) cmdlineStyle(errorMsg bool) lipgloss.Style {
	th := r.activeTheme()
	if errorMsg {
		return th.Get("error")
	}
	return th.Get("ui.statusline")
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

func (s *statusElemCtx) elem(e view.StatusLineElement) statusElem {
	switch e {
	case view.StatusLineMode:
		return statusElem{
			text:  " " + s.opts.ModeNameForMode(s.mode) + " ",
			style: s.modeSt,
		}
	case view.StatusLineSeparator:
		return statusElem{
			text:  s.sep,
			style: s.sepSt,
		}
	case view.StatusLineSpacer, view.StatusLineSpinner:
		return statusElem{}
	case view.StatusLineFileName:
		return statusElem{
			text:  " " + s.doc.RelativeName(s.cwd) + " ",
			style: s.baseTUI,
		}
	case view.StatusLineFileBaseName:
		return statusElem{
			text:  " " + filepath.Base(s.doc.Path()) + " ",
			style: s.baseTUI,
		}
	case view.StatusLineFileAbsolutePath:
		return statusElem{
			text:  " " + s.doc.Path() + " ",
			style: s.baseTUI,
		}
	case view.StatusLineReadOnly:
		if s.doc.Readonly() {
			return statusElem{text: " [readonly]", style: s.baseTUI}
		}
		return statusElem{}
	case view.StatusLineModified:
		if s.doc.Modified() {
			return statusElem{text: "[modified] ", style: s.baseTUI}
		}
		return statusElem{}
	case view.StatusLineSelections:
		if s.nSel == 1 {
			return statusElem{text: " 1 sel ", style: s.baseTUI}
		}
		return statusElem{
			text: fmt.Sprintf(
				" %d/%d sels ", s.primIdx+1, s.nSel,
			),
			style: s.baseTUI,
		}
	case view.StatusLinePrimaryLen:
		return statusElem{
			text:  fmt.Sprintf(" %d ", s.primLen),
			style: s.baseTUI,
		}
	case view.StatusLinePosition:
		return statusElem{
			text:  fmt.Sprintf(" %d:%d ", s.row, s.col),
			style: s.baseTUI,
		}
	case view.StatusLinePercent:
		pct := 0
		if s.totalLines > 0 {
			pct = (s.row * 100) / s.totalLines
		}
		return statusElem{
			text:  fmt.Sprintf(" %d%% ", pct),
			style: s.baseTUI,
		}
	case view.StatusLineTotalLines:
		return statusElem{
			text:  fmt.Sprintf(" %d ", s.totalLines),
			style: s.baseTUI,
		}
	case view.StatusLineFileEncoding:
		return statusElem{}
	case view.StatusLineFileLineEnding:
		le := s.doc.LineEnding()
		label := "lf"
		if le == core.LineEndingCRLF {
			label = "crlf"
		}
		return statusElem{text: " " + label + " ", style: s.baseTUI}
	case view.StatusLineFileIndentStyle:
		indent := s.doc.IndentStyle()
		var label string
		if indent.IsTabs() {
			label = "tabs"
		} else {
			label = fmt.Sprintf("spaces:%d", indent.Width())
		}
		return statusElem{text: " " + label + " ", style: s.baseTUI}
	case view.StatusLineFileType:
		lang := s.doc.Lang()
		if lang == "" {
			lang = "text"
		}
		return statusElem{text: " " + lang + " ", style: s.baseTUI}
	case view.StatusLineRegister:
		if s.reg != 0 {
			return statusElem{
				text:  fmt.Sprintf(" reg=%c ", s.reg),
				style: s.baseTUI,
			}
		}
		return statusElem{}
	case view.StatusLineDiagnostics,
		view.StatusLineWorkspaceDiag,
		view.StatusLineVersionControl:
		return statusElem{}
	default:
		return statusElem{}
	}
}
