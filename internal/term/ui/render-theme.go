package ui

import (
	"image/color"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kode4food/toe/internal/term/highlight"
	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/view"
)

func buildTUIStyles(th *theme.Theme, mode view.Mode) *tuiStyles {
	sel := th.Get("ui.selection")
	cur, _ := modeCursorStyleFor(th, mode, false)
	curPrim, _ := modeCursorStyleFor(th, mode, true)
	cl := th.Get("ui.cursorline.primary")
	st := &tuiStyles{
		text:              styleToTUI(th.Get("ui.text")),
		line:              styleToTUI(th.Get("ui.linenr")),
		lineSelected:      styleToTUI(th.Get("ui.linenr.selected")),
		selection:         styleToTUI(sel),
		cursor:            styleToTUI(cur),
		cursorPrim:        styleToTUI(curPrim),
		cursorLinePrim:    styleToTUI(cl),
		cursorLineSec:     styleToTUI(cl),
		cursorColumn:      styleToTUI(cl),
		whitespace:        styleToTUI(th.Get("ui.virtual.whitespace")),
		indentGuide:       styleToTUI(th.Get("ui.virtual.indent-guide")),
		ruler:             styleToTUI(th.Get("ui.virtual.ruler")),
		inlayHint:         styleToTUI(th.Get("ui.virtual")),
		inlayHintType:     styleToTUI(th.Get("ui.virtual")),
		inlayHintParam:    styleToTUI(th.Get("ui.virtual")),
		severityHint:      styleToTUI(th.Get("hint")),
		severityInfo:      styleToTUI(th.Get("info")),
		severityWarning:   styleToTUI(th.Get("warning")),
		severityError:     styleToTUI(th.Get("error")),
		diagnostic:        styleToTUI(th.Get("diagnostic")),
		diagnosticHint:    styleToTUI(th.Get("diagnostic.hint")),
		diagnosticInfo:    styleToTUI(th.Get("diagnostic.info")),
		diagnosticWarning: styleToTUI(th.Get("diagnostic.warning")),
		diagnosticError:   styleToTUI(th.Get("diagnostic.error")),
		documentHighlight: styleToTUI(th.Get("ui.highlight")),
		documentLink:      styleToTUI(th.Get("markup.link.url")),
		searchMatch:       styleToTUI(searchMatchStyle()),
		diffAdded:         styleToTUI(th.Get("diff.plus.gutter")),
		diffModified:      styleToTUI(th.Get("diff.delta.gutter")),
		diffRemoved:       styleToTUI(th.Get("diff.minus.gutter")),
	}
	if next, ok := th.TryGet("ui.virtual.inlay-hint"); ok {
		converted := styleToTUI(next)
		st.inlayHint = converted
		st.inlayHintType = converted
		st.inlayHintParam = converted
	}
	if next, ok := th.TryGet("ui.virtual.inlay-hint.type"); ok {
		st.inlayHintType = styleToTUI(next)
	}
	if next, ok := th.TryGet("ui.virtual.inlay-hint.parameter"); ok {
		st.inlayHintParam = styleToTUI(next)
	}
	if next, ok := th.TryGetExact("markup.link"); ok {
		st.documentLink = styleToTUI(next)
	}
	if next, ok := th.TryGetExact("markup.link.url"); ok {
		st.documentLink = styleToTUI(next)
	}
	if next, ok := th.TryGetExact("ui.selection.primary"); ok {
		st.selection = styleToTUI(next)
	}
	if next, ok := th.TryGet("ui.cursorline.secondary"); ok {
		st.cursorLineSec = styleToTUI(next)
	}
	if next, ok := th.TryGetExact("ui.cursorcolumn"); ok {
		st.cursorColumn = styleToTUI(next)
	}
	if next, ok := th.TryGetExact("ui.cursorcolumn.primary"); ok {
		st.cursorColumn = styleToTUI(next)
	}
	if next, ok := th.TryGet("ui.search.match"); ok {
		st.searchMatch = styleToTUI(next)
	}
	return st
}

func clearStyleBackground(st lipgloss.Style) lipgloss.Style {
	if colorToTUI(st.GetBackground()).IsReset() {
		return st
	}
	return st.Background(lipgloss.NoColor{})
}

func inheritStyleBackground(st lipgloss.Style, bg color.Color) lipgloss.Style {
	if colorToTUI(bg).IsReset() {
		return st
	}
	if !colorToTUI(st.GetBackground()).IsReset() {
		return st
	}
	return st.Background(bg)
}

// Resolved styles render transparently over lower layers unless the theme
// gives the scope an explicit background
func hlStyleFnFor(th *theme.Theme) func(string) lipgloss.Style {
	return func(scope string) lipgloss.Style {
		if s, ok := th.TryGet(scope); ok {
			return s
		}
		return highlight.DefaultStyle(scope)
	}
}

func modeCursorStyleFor(
	th *theme.Theme, mode view.Mode, primary bool,
) (lipgloss.Style, bool) {
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
