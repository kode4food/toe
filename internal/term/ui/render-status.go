package ui

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"

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
		text    string
		style   tui.Style
		kind    view.StatusLineElement
		pinned  bool
		compact bool
	}

	statusElemCtx struct {
		doc        *view.Document
		opts       *view.Options
		mode       string
		baseTUI    tui.Style
		modeSt     tui.Style
		sepSt      tui.Style
		spinSt     tui.Style
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
		busy       bool
		spinFrame  int
	}
)

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
		scope := "ui.statusline." + v.Mode().Scope()
		modeSt = th.Get(scope)
	}

	sepSt := st
	if s, ok := th.TryGet("ui.statusline.separator"); ok {
		sepSt = s
	}
	spinSt := applyAccentStyle(st, th.Get("ui.prompt"))

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
	var busy bool
	if ls := r.cx.Editor.LanguageServerController(); ls != nil {
		busy = ls.Busy()
	}

	baseTUI := lipglossToTUIStyle(st)

	src := &statusElemCtx{
		doc: doc, opts: opts, mode: mode,
		baseTUI: baseTUI,
		modeSt:  lipglossToTUIStyle(modeSt),
		sepSt:   lipglossToTUIStyle(sepSt),
		spinSt:  lipglossToTUIStyle(spinSt),
		sep:     sep, nSel: nSel, primIdx: primIdx, primLen: primLen,
		totalLines: totalLines, reg: reg, cwd: cwd,
		row: row, col: col,
		vcsHead:   vcsHead,
		busy:      busy,
		spinFrame: r.ec.spinFrame,
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
			if !e.compact {
				w += 2
			}
		}
		return w
	}

	// sheds pinned-excluded elements from each section's inner edge; right
	// section first, then left
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
			if !e.compact {
				buf.SetString(x, y, " ", baseTUI)
				x++
			}
			buf.SetString(x, y, e.text, e.style)
			x += runewidth.StringWidth(e.text)
			if !e.compact {
				buf.SetString(x, y, " ", baseTUI)
				x++
			}
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
