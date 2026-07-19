package ui

import (
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type (
	signatureCall struct {
		docID  view.DocumentId
		viewID view.Id
		open   int
	}

	signatureHelpComponent struct {
		overlayBuf
		ec     *EditorComponent
		call   signatureCall
		help   view.SignatureHelp
		cursor int
		lines  []popupLine
	}
)

const signaturePopupMaxH = 12

var _ BufferOverlayComponent = (*signatureHelpComponent)(nil)

func newSignatureHelpComponent(
	ec *EditorComponent, call signatureCall, help view.SignatureHelp,
) *signatureHelpComponent {
	cursor := help.Active
	if cursor < 0 || cursor >= len(help.Signatures) {
		cursor = 0
	}
	return &signatureHelpComponent{
		ec: ec, call: call, help: help, cursor: cursor,
	}
}

func (s *signatureHelpComponent) HandleEvent(
	cx *Context, msg tea.Msg,
) (EventResult, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return ignored(), nil
	}
	switch {
	case key.Code == tea.KeyEscape:
		return consumedWith(s.dismiss), nil
	case key.Mod&tea.ModAlt != 0 && key.Text == "p":
		s.move(-1)
		return consumed(), nil
	case key.Mod&tea.ModAlt != 0 && key.Text == "n":
		s.move(1)
		return consumed(), nil
	default:
		k := FromTeaKey(key)
		if cx.Editor.Mode() == view.ModeInsert && k.IsTypable() {
			return ignoredWith(s.refresh), nil
		}
		return ignoredWith(popLayer), nil
	}
}

func (s *signatureHelpComponent) Cursor(
	*Context, geom.Size,
) (tea.Cursor, bool) {
	return tea.Cursor{}, false
}

func (s *signatureHelpComponent) Layout(
	cx *Context, screen geom.Size,
) (geom.Area, bool) {
	if len(s.help.Signatures) == 0 || !s.valid(cx) {
		return geom.Area{}, false
	}
	sig := s.help.Signatures[s.cursor]
	lines := popupTextLines(signatureDocs(sig), screen.Width-2)
	w := max(runewidth.StringWidth(sig.Label), popupTextWidth(lines)) + 2
	if len(s.help.Signatures) > 1 {
		w += runewidth.StringWidth(s.indexText()) + 1
	}
	w = min(max(w, 2), screen.Width)
	h := min(max(len(lines)+4, 3), signaturePopupMaxH)
	x, y := 0, 0
	if cur, ok := s.ec.Cursor(cx, screen); ok {
		x = s.openScreenX(cx)
		y = cur.Y + 1
		if y+h > screen.Height {
			y = max(cur.Y-h-1, 0)
		}
	}
	if x+w > screen.Width {
		x = max(screen.Width-w, 0)
	}
	s.lines = lines
	return geom.Area{
		Point: geom.Point{X: x, Y: y},
		Size:  geom.Size{Width: w, Height: h},
	}, true
}

func (s *signatureHelpComponent) PaintBuffer(
	cx *Context, pl geom.Area,
) *tui.Buffer {
	return s.maybePaint(cx, pl.Size, func(buf *tui.Buffer) {
		s.paint(cx, buf, pl)
	})
}

func (s *signatureHelpComponent) paint(
	cx *Context, buf *tui.Buffer, pl geom.Area,
) {
	sig := s.help.Signatures[s.cursor]
	w, h := pl.Width, pl.Height
	st := lipglossToTUIStyle(cx.Theme().Get("ui.popup"))
	border := lipgloss.RoundedBorder()
	pop := popup{
		border:       border,
		borderStyle:  st,
		contentStyle: st,
		padX:         0,
	}
	area := pop.drawInto(buf, geom.Area{
		Size: geom.Size{Width: w, Height: h},
	})
	s.renderSignature(cx, buf, area, sig)
	if len(s.help.Signatures) > 1 {
		index := s.indexText()
		buf.SetString(geom.Point{
			X: area.X + area.Width - runewidth.StringWidth(index),
			Y: area.Y,
		}, index, st)
	}
	if len(s.lines) == 0 || area.Height < 3 {
		return
	}
	renderSignatureSeparator(
		buf, geom.Point{Y: area.Y + 1}, w, border, st,
	)
	docArea := area
	docArea.Y += 2
	docArea.Height -= 2
	r := popupTextRenderer{buf: buf, cx: cx, area: docArea, base: st}
	r.render(s.lines)
}

func (s *signatureHelpComponent) openScreenX(cx *Context) int {
	doc, ok := cx.Editor.FocusedDocument()
	if !ok {
		return 0
	}
	v, ok := cx.Editor.FocusedView()
	if !ok {
		return 0
	}
	opts := cx.Editor.Options()
	rowMap := s.ec.cache.viewRowMaps[v.ID()]
	visual := cursorScreenPos(cursorScreenPosArgs{
		text:    doc.Text(),
		cursor:  s.call.open,
		gutterW: gutterWidthFor(doc.Text(), opts.Gutters),
		rowMap:  rowMap,
		tabW:    doc.TabWidth(),
		hOff:    v.Offset().HorizontalOffset,
	})
	return v.Area().X + visual.X
}

func (s *signatureHelpComponent) move(n int) {
	if len(s.help.Signatures) <= 1 {
		return
	}
	s.markDirty()
	s.cursor = (s.cursor + n + len(s.help.Signatures)) % len(s.help.Signatures)
}

func (s *signatureHelpComponent) refresh(
	cx *Context, comp *Compositor,
) tea.Cmd {
	if !s.valid(cx) {
		removeSignatureHelpLayer(comp)
		return nil
	}
	doc, ok := cx.Editor.FocusedDocument()
	if !ok {
		removeSignatureHelpLayer(comp)
		return nil
	}
	v, ok := cx.Editor.FocusedView()
	if !ok {
		removeSignatureHelpLayer(comp)
		return nil
	}
	ls := cx.Editor.LanguageServerController()
	if ls == nil {
		removeSignatureHelpLayer(comp)
		return nil
	}
	help, err := ls.SignatureHelp(doc, v.ID())
	if err != nil {
		cx.Editor.SetStatusMsg(err.Error())
		removeSignatureHelpLayer(comp)
		return nil
	}
	if len(help.Signatures) == 0 {
		removeSignatureHelpLayer(comp)
		return nil
	}
	pushSignatureHelpLayer(comp, newSignatureHelpComponent(s.ec, s.call, help))
	return nil
}

func (s *signatureHelpComponent) dismiss(_ *Context, comp *Compositor) tea.Cmd {
	s.ec.signatureHidden = &s.call
	comp.Pop()
	return nil
}

func (s *signatureHelpComponent) valid(cx *Context) bool {
	call, ok := currentSignatureCall(cx)
	return ok && call == s.call
}

func (s *signatureHelpComponent) indexText() string {
	return "(" + strconv.Itoa(s.cursor+1) + "/" +
		strconv.Itoa(len(s.help.Signatures)) + ")"
}

func (s *signatureHelpComponent) renderSignature(
	cx *Context, buf *tui.Buffer, area geom.Area,
	sig view.SignatureInformation,
) {
	label := ansi.Truncate(sig.Label, area.Width, "")
	base := lipglossToTUIStyle(cx.Theme().Get("ui.popup"))
	buf.SetString(area.Point, label, base)
	if sig.ActiveEnd <= sig.ActiveStart {
		return
	}
	rs := []rune(label)
	start := min(sig.ActiveStart, len(rs))
	end := min(sig.ActiveEnd, len(rs))
	if end <= start {
		return
	}
	x := area.X + runewidth.StringWidth(string(rs[:start]))
	text := string(rs[start:end])
	bg := cx.Theme().Get("ui.popup").GetBackground()
	st := inheritStyleBackground(cx.Theme().Get("ui.selection"), bg)
	buf.SetString(geom.Point{X: x, Y: area.Y}, text, lipglossToTUIStyle(st))
}

func renderSignatureSeparator(
	buf *tui.Buffer, at geom.Point, w int, border lipgloss.Border, st tui.Style,
) {
	line := border.MiddleLeft +
		strings.Repeat(border.Top, max(w-2, 0)) +
		border.MiddleRight
	buf.SetString(at, line, st)
}

func pushSignatureHelpLayer(comp *Compositor, layer *signatureHelpComponent) {
	for i := len(comp.layers) - 1; i >= 1; i-- {
		if _, ok := comp.layers[i].(*signatureHelpComponent); ok {
			comp.layers[i] = layer
			return
		}
	}
	comp.Push(layer)
}

func removeSignatureHelpLayer(comp *Compositor) {
	comp.layers = slices.DeleteFunc(comp.layers, func(l Component) bool {
		_, ok := l.(*signatureHelpComponent)
		return ok
	})
}

func currentSignatureCall(cx *Context) (signatureCall, bool) {
	doc, ok := cx.Editor.FocusedDocument()
	if !ok {
		return signatureCall{}, false
	}
	v, ok := cx.Editor.FocusedView()
	if !ok {
		return signatureCall{}, false
	}
	sel := doc.SelectionFor(v.ID())
	pos := sel.Primary().Cursor(doc.Text())
	open, ok := signatureCallOpen(doc, pos)
	if !ok {
		return signatureCall{}, false
	}
	return signatureCall{
		docID:  doc.ID(),
		viewID: v.ID(),
		open:   open,
	}, true
}

func signatureCallOpen(doc *view.Document, pos int) (int, bool) {
	before, err := doc.Text().SliceString(0, pos)
	if err != nil {
		return 0, false
	}
	var stack []int
	charIdx := 0
	for len(before) > 0 {
		ch, size := utf8.DecodeRuneInString(before)
		switch ch {
		case '(':
			stack = append(stack, charIdx)
		case ')':
			if len(stack) == 0 {
				return 0, false
			}
			stack = stack[:len(stack)-1]
		}
		before = before[size:]
		charIdx++
	}
	if len(stack) == 0 {
		return 0, false
	}
	return stack[len(stack)-1], true
}

func signatureDocs(sig view.SignatureInformation) string {
	switch {
	case sig.Docs != "" && sig.ParamDocs != "":
		return sig.Docs + "\n\n" + sig.ParamDocs
	case sig.Docs != "":
		return sig.Docs
	default:
		return sig.ParamDocs
	}
}
