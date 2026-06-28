package ui

import (
	"strconv"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type signatureCall struct {
	docID  view.DocumentId
	viewID view.Id
	open   int
}

type signatureHelpComponent struct {
	ec     *EditorComponent
	call   signatureCall
	help   view.SignatureHelp
	cursor int
}

const signaturePopupMaxH = 12

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
	msg tea.Msg, cx *Context,
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

func (s *signatureHelpComponent) Render(int, int, *Context) string {
	return ""
}

func (s *signatureHelpComponent) Cursor(
	int, int, *Context,
) (tea.Cursor, bool) {
	return tea.Cursor{}, false
}

func (s *signatureHelpComponent) RenderOverBuffer(
	buf *tui.Buffer, cx *Context,
) {
	if len(s.help.Signatures) == 0 {
		return
	}
	if !s.valid(cx) {
		return
	}
	sig := s.help.Signatures[s.cursor]
	lines := popupTextLines(signatureDocs(sig), buf.Width-2)
	w := max(runewidth.StringWidth(sig.Label), popupTextWidth(lines)) + 2
	if len(s.help.Signatures) > 1 {
		w += runewidth.StringWidth(s.indexText()) + 1
	}
	w = min(max(w, 2), buf.Width)
	h := min(max(len(lines)+4, 3), signaturePopupMaxH)
	x, y := 0, 0
	if cur, ok := s.ec.Cursor(buf.Width, buf.Height, cx); ok {
		x = s.openScreenX(cx)
		y = max(cur.Y-h-1, 0)
		if cur.Y < h+1 {
			y = min(cur.Y+1, max(buf.Height-h, 0))
		}
	}
	if x+w > buf.Width {
		x = max(buf.Width-w, 0)
	}
	st := lipglossToTUIStyle(cx.Theme().Get("ui.popup"))
	border := lipgloss.RoundedBorder()
	pop := popup{
		border:       border,
		borderStyle:  st,
		contentStyle: st,
		padX:         0,
	}
	area := pop.drawInto(buf, x, y, w, h)
	s.renderSignature(buf, area, sig, cx)
	if len(s.help.Signatures) > 1 {
		index := s.indexText()
		buf.SetString(area.x+area.w-runewidth.StringWidth(index),
			area.y, index, st)
	}
	if len(lines) == 0 || area.h < 3 {
		return
	}
	renderSignatureSeparator(buf, x, area.y+1, w, border, st)
	docArea := area
	docArea.y += 2
	docArea.h -= 2
	r := popupTextRenderer{buf: buf, cx: cx, area: docArea, base: st}
	r.render(lines)
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
	_, visualX := cursorScreenPos(cursorScreenPosArgs{
		text:    doc.Text(),
		cursor:  s.call.open,
		gutterW: gutterWidthFor(doc.Text(), opts.Gutters),
		rowMap:  rowMap,
		tabW:    doc.TabWidth(),
		hOff:    v.Offset().HorizontalOffset,
	})
	return v.Area().X + visualX
}

func (s *signatureHelpComponent) move(n int) {
	if len(s.help.Signatures) <= 1 {
		return
	}
	s.cursor = (s.cursor + n + len(s.help.Signatures)) % len(s.help.Signatures)
}

func (s *signatureHelpComponent) refresh(
	comp *Compositor, cx *Context,
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

func (s *signatureHelpComponent) dismiss(
	comp *Compositor, _ *Context,
) tea.Cmd {
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
	buf *tui.Buffer, area popupArea, sig view.SignatureInformation,
	cx *Context,
) {
	label := ansi.Truncate(sig.Label, area.w, "")
	base := lipglossToTUIStyle(cx.Theme().Get("ui.popup"))
	buf.SetString(area.x, area.y, label, base)
	if sig.ActiveEnd <= sig.ActiveStart {
		return
	}
	rs := []rune(label)
	start := min(sig.ActiveStart, len(rs))
	end := min(sig.ActiveEnd, len(rs))
	if end <= start {
		return
	}
	x := area.x + runewidth.StringWidth(string(rs[:start]))
	text := string(rs[start:end])
	bg := cx.Theme().Get("ui.popup").GetBackground()
	st := inheritStyleBackground(cx.Theme().Get("ui.selection"), bg)
	buf.SetString(x, area.y, text, lipglossToTUIStyle(st))
}

func renderSignatureSeparator(
	buf *tui.Buffer, x, y, w int, border lipgloss.Border, st tui.Style,
) {
	line := border.MiddleLeft +
		strings.Repeat(border.Top, max(w-2, 0)) +
		border.MiddleRight
	buf.SetString(x, y, line, st)
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
	for i := len(comp.layers) - 1; i >= 1; i-- {
		if _, ok := comp.layers[i].(*signatureHelpComponent); ok {
			comp.layers = append(comp.layers[:i], comp.layers[i+1:]...)
			return
		}
	}
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
	stack := []int{}
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
