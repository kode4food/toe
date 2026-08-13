package ui

import (
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type (
	styles struct {
		text         tui.Style
		line         tui.Style
		lineSelected tui.Style

		selection      tui.Style
		cursor         tui.Style
		cursorPrim     tui.Style
		cursorLinePrim tui.Style
		cursorLineSec  tui.Style
		cursorColumn   tui.Style

		whitespace  tui.Style
		indentGuide tui.Style
		rulerBg     tui.Color

		inlayHint      tui.Style
		inlayHintType  tui.Style
		inlayHintParam tui.Style

		severityHint    tui.Style
		severityInfo    tui.Style
		severityWarning tui.Style
		severityError   tui.Style

		diagnostic        tui.Style
		diagnosticHint    tui.Style
		diagnosticInfo    tui.Style
		diagnosticWarning tui.Style
		diagnosticError   tui.Style

		documentHighlight tui.Style
		documentLink      tui.Style
		searchMatch       tui.Style

		diffAdded    tui.Style
		diffModified tui.Style
		diffRemoved  tui.Style
	}

	statusElemCtx struct {
		doc  *view.Document
		mode view.Mode

		baseTUI tui.Style
		modeSt  tui.Style
		sepSt   tui.Style
		spinSt  tui.Style
		sep     string

		cursor     core.Position
		selCount   int
		primIdx    int
		primLen    int
		totalLines int

		reg     rune
		cwd     string
		vcsHead string

		busy      bool
		spinFrame int
	}
)

// one column keeps the caret visible without wrapping the terminal
const commandLineRightPad = 1

func (r *renderPass) renderCmdline(buf *tui.Buffer, y int) {
	msg := r.editor.keys.message
	st := r.cmdlineStyle(msg != nil && msg.error)
	badge := r.editor.macroElems(r.context, st)
	row := statusRow{
		at:        geom.Point{X: 0, Y: y},
		width:     r.size.Width,
		baseStyle: st,
		right:     badge,
	}
	width := max(row.contentWidth()-commandLineRightPad, 0)
	text := runewidth.Truncate(r.cmdlineText(), width, "")
	if text != "" {
		row.left = []statusElem{{text: text, style: st, compact: true}}
	}
	row.paint(buf)
}

func (r *renderPass) cmdlineText() string {
	if hint := r.editor.keys.hint; hint != "" {
		return hint
	}
	if status := r.editor.keys.status; status != "" {
		return status
	}
	if msg := r.editor.keys.message; msg != nil {
		return msg.value
	}
	return ""
}

func (r *renderPass) cmdlineStyle(errorMsg bool) tui.Style {
	th := r.context.Theme()
	if errorMsg {
		return th.Get("error")
	}
	if r.editor.keys.hint != "" || r.editor.keys.status != "" ||
		r.editor.macroSlot.recording {
		return th.Get("ui.statusline").Bg(promptBackground(th))
	}
	return th.Get("ui.statusline")
}

type renderStatusArgs struct {
	doc     *view.Document
	view    *view.View
	buf     *tui.Buffer
	at      geom.Point
	width   int
	focused bool
}

func (r *renderPass) renderStatus(args renderStatusArgs) {
	doc := args.doc
	v := args.view
	buf := args.buf
	width := args.width
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	prim := sel.Primary()
	cursor := prim.Cursor(text)

	at := core.Position{Line: 1, Column: 1}
	if p, err := text.Position(cursor); err == nil {
		at = p
	}

	opts := r.context.Editor.Options()

	th := r.context.Theme()

	st := th.Get("ui.statusline.inactive")
	modeSt := st
	if args.focused {
		st = th.Get("ui.statusline")
		modeSt = th.Get("ui.statusline." + v.Mode().Scope())
	}

	sepSt := st
	if s, ok := th.TryGet("ui.statusline.separator"); ok {
		sepSt = s
	}
	spinSt := applyAccentStyle(styleOverlay{
		base:    st,
		overlay: th.Get("ui.prompt"),
	})

	nSel := len(sel.Ranges())
	primIdx := sel.PrimaryIndex()
	primLen := prim.Len()
	totalLines := text.LenLines()
	reg := r.context.Editor.ActiveRegister()
	cwd := r.context.Editor.Cwd()
	sep := opts.StatusLineSeparator()
	var vcsHead string
	if vc := r.context.Editor.VersionControl(); vc != nil {
		vcsHead, _ = vc.HeadName(doc)
	}
	var busy bool
	if ls := r.context.Editor.LanguageServerController(); ls != nil {
		busy = ls.Busy()
	}

	baseTUI := st
	src := &statusElemCtx{
		doc:     doc,
		mode:    v.Mode(),
		baseTUI: baseTUI,
		modeSt:  modeSt,
		sepSt:   sepSt,
		spinSt:  spinSt,
		sep:     sep, selCount: nSel, primIdx: primIdx, primLen: primLen,
		totalLines: totalLines, reg: reg, cwd: cwd,
		cursor:    at,
		vcsHead:   vcsHead,
		busy:      busy,
		spinFrame: r.editor.spinner.phase,
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
	right = r.withMaximizedStatus(right)
	statusRow{
		at:        args.at,
		width:     width,
		baseStyle: baseTUI,
		left:      left,
		right:     right,
	}.paint(buf)
}

func (r *renderPass) withMaximizedStatus(elems []statusElem) []statusElem {
	if !r.context.Editor.Tree().Maximized() {
		return elems
	}
	return append(
		elems,
		statusBadge(
			i18n.Text(i18n.StatusPaneMaximized),
			r.context.Theme().Get("ui.statusline.maximized"),
		),
	)
}

func (s *statusElemCtx) elem(e view.StatusLineItem) statusElem {
	if fn, ok := statusElemFns[e.Element]; ok {
		se := fn(s)
		se.pinned = se.pinned || e.Pinned
		return se
	}
	return statusElem{}
}
