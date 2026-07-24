package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/term/highlight"
	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

func buildTUIStyles(th *theme.Theme, mode view.Mode) *tuiStyles {
	sel := th.Get("ui.selection")
	cur, _ := modeCursorStyleFor(th, mode, false)
	curPrim, _ := modeCursorStyleFor(th, mode, true)
	cl := th.Get("ui.cursorline.primary")
	st := &tuiStyles{
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

func clearStyleBackground(st tui.Style) tui.Style {
	return st.Bg(tui.ColorReset)
}

func inheritStyleBackground(st tui.Style, bg tui.Color) tui.Style {
	if bg.IsReset() {
		return st
	}
	if !st.BgColor().IsReset() {
		return st
	}
	return st.Bg(bg)
}

// Resolved styles render transparently over lower layers unless the theme
// gives the scope an explicit background
func hlStyleFnFor(th *theme.Theme) func(string) tui.Style {
	return func(scope string) tui.Style {
		if s, ok := th.TryGet(scope); ok {
			return s
		}
		return highlight.DefaultStyle(scope)
	}
}

func modeCursorStyleFor(
	th *theme.Theme, mode view.Mode, primary bool,
) (tui.Style, bool) {
	scope := "ui.cursor." + mode.Scope()
	if primary {
		scope = "ui.cursor.primary." + mode.Scope()
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
