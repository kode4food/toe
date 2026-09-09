package ui

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
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
		editor *view.Editor
		doc    *view.Document
		mode   view.Mode

		baseTUI tui.Style
		modeSt  tui.Style
		spinSt  tui.Style

		cursor     core.Position
		selCount   int
		primIdx    int
		primLen    int
		totalLines int

		macroReg   rune
		macroSt    tui.Style
		maximizeSt tui.Style
		blinkFrame int

		recording bool
		spinFrame int
	}
)

type renderStatusArgs struct {
	doc     *view.Document
	view    *view.View
	buf     *tui.Buffer
	at      geom.Point
	width   int
	focused bool
}

func (r *renderPass) renderStatus(args renderStatusArgs) {
	cx := r.context
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

	opts := cx.Editor.Options()
	th := cx.Theme()

	st := th.Get("ui.statusline.inactive")
	modeSt := st
	if args.focused {
		st = th.Get("ui.statusline")
		modeSt = th.Get("ui.statusline." + v.Mode().Scope())
	}

	spinSt := applyAccentStyle(styleOverlay{
		base:    st,
		overlay: th.Get("ui.prompt"),
	})

	nSel := len(sel.Ranges())
	primIdx := sel.PrimaryIndex()
	primLen := prim.Len()
	totalLines := text.LenLines()

	baseTUI := st
	ms := r.editor.macroSlot
	src := &statusElemCtx{
		editor:     cx.Editor,
		doc:        doc,
		mode:       v.Mode(),
		baseTUI:    baseTUI,
		modeSt:     modeSt,
		spinSt:     spinSt,
		selCount:   nSel,
		primIdx:    primIdx,
		primLen:    primLen,
		totalLines: totalLines,
		cursor:     at,
		macroReg:   ms.reg,
		macroSt:    th.Get("ui.statusline.macro"),
		maximizeSt: th.Get("ui.statusline.maximized"),
		blinkFrame: r.editor.macroBlink.phase,
		recording:  ms.recording,
		spinFrame:  r.editor.spinner.phase,
	}

	r.paintStatus(buf, statusRow{
		at:        args.at,
		width:     width,
		baseStyle: baseTUI,
		focused:   args.focused,
		left:      src.collect(opts.StatusLineLeft()),
		right:     src.collect(opts.StatusLineRight()),
	})
}

// paintStatus is the only way a status row reaches the buffer, so the badges
// for editor-wide state land on the corner row whatever pane owns it
func (r *renderPass) paintStatus(buf *tui.Buffer, row statusRow) {
	if row.at.Y == r.size.Height-1 && row.at.X+row.width == r.size.Width {
		row.right = append(row.right, r.cornerBadges(row.baseStyle)...)
	}
	cx := r.context
	opts := cx.Editor.Options()
	row.edge = statusEdges[edgesFor(opts)]
	row.sectionStyle = cx.Theme().Get("ui.statusline.section")
	row.paint(buf)
}

func (r *renderPass) cornerBadges(base tui.Style) []statusElem {
	cx := r.context
	ms := r.editor.macroSlot
	reg := cx.Editor.ActiveRegister()
	maximized := cx.Editor.Tree().Maximized()
	if !ms.recording && !maximized && reg == 0 {
		return nil
	}
	th := cx.Theme()
	src := &statusElemCtx{
		editor:     cx.Editor,
		baseTUI:    base,
		macroReg:   ms.reg,
		macroSt:    th.Get("ui.statusline.macro"),
		maximizeSt: th.Get("ui.statusline.maximized"),
		blinkFrame: r.editor.macroBlink.phase,
		recording:  ms.recording,
	}
	var out []statusElem
	for _, fn := range []statusElemFn{
		statusElemRegister, statusElemPaneMaximized, statusElemMacroRecording,
	} {
		if se := fn(src); se.text != "" {
			se.pinned = true
			out = append(out, se)
		}
	}
	return out
}

func (s *statusElemCtx) collect(items []view.StatusLineItem) []statusElem {
	var out []statusElem
	for _, item := range items {
		se := s.elem(item)
		if se.text == "" {
			continue
		}
		if out == nil {
			out = make([]statusElem, 0, len(items))
		}
		out = append(out, se)
	}
	return out
}

func (s *statusElemCtx) elem(e view.StatusLineItem) statusElem {
	if fn, ok := statusElemFns[e.Element]; ok {
		se := fn(s)
		se.pinned = se.pinned || e.Pinned
		return se
	}
	return statusElem{}
}

// edgesFor is the configured divider shape, or the plain one when the font has
// no powerline glyphs to draw the others with
func edgesFor(opts *view.Options) view.StatusLineEdges {
	if !opts.NerdFonts {
		return view.StatusLineEdgesLine
	}
	return opts.StatusLineEdges()
}
