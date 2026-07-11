package ui

import (
	"image/color"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kode4food/toe/internal/term/highlight"
	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/view"
)

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
