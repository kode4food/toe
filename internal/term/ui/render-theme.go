package ui

import (
	"image/color"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/term/highlight"
	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

const (
	rulerBackgroundPct = 0.06
	cursorHighlightPct = 0.03
	grayRampStep       = 10
)

func buildStyles(th *theme.Theme, mode view.Mode) *styles {
	return buildStylesWithBackground(
		th, mode, th.Get("ui.background").BgColor(),
	)
}

func buildStylesWithBackground(
	th *theme.Theme, mode view.Mode, bg tui.Color,
) *styles {
	sel := th.Get("ui.selection")
	cur, _ := modeCursorStyleFor(th, mode, false)
	curPrim, _ := modeCursorStyleFor(th, mode, true)
	light := isLightTheme(th)
	ruler := deriveBackground(bg, rulerBackgroundPct, light)
	cursorHighlight := deriveBackground(bg, cursorHighlightPct, light)
	cl := tui.Style{}.Bg(cursorHighlight)
	st := &styles{
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
		rulerBg:           ruler,
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
		searchMatch:       searchMatchStyle(th),
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
	if next, ok := th.TryGet("ui.search.match"); ok {
		st.searchMatch = next
	}
	return st
}

func isLightTheme(th *theme.Theme) bool {
	return th.Name() == "latte"
}

func deriveBackground(bg tui.Color, pct float64, light bool) tui.Color {
	if !TrueColorSupported() {
		return rampBackground(bg, pct, light)
	}
	r, g, b, _ := bg.RGBA()
	red, green, blue := uint8(r>>8), uint8(g>>8), uint8(b>>8)
	toward := 255.0
	if light {
		toward = 0
	}
	shift := func(v uint8) uint8 {
		return uint8(float64(v) + (toward-float64(v))*pct)
	}
	return tui.ColorRGB(shift(red), shift(green), shift(blue))
}

func rampBackground(bg tui.Color, pct float64, light bool) tui.Color {
	// a 3% tint is finer than the color cube can express, and the background
	// quantizes too, so step from the entry it actually lands on
	lum := colorLuma(bg.Quantized()) / lumaScale
	steps := int(pct/cursorHighlightPct + 0.5)
	if light {
		steps = -steps
	}
	v := uint8(min(max(lum+steps*grayRampStep, 0), 255))
	return tui.ColorRGB(v, v, v).Quantized()
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

func highlighterFor(th *theme.Theme) func(string) tui.Style {
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

func insertCursorAt(cx *Context, at geom.Point) (tea.Cursor, bool) {
	kind := cx.Editor.Options().CursorShapeForMode(view.ModeInsert)
	if kind == view.CursorKindHidden {
		return tea.Cursor{}, false
	}
	return tea.Cursor{
		X:     at.X,
		Y:     at.Y,
		Shape: cursorKindToShape(kind),
		Color: cursorColor(cx, view.ModeInsert),
	}, true
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

func promptBackground(th *theme.Theme) tui.Color {
	return deriveBackground(
		th.Get("ui.popup").BgColor(), cursorHighlightPct, isLightTheme(th),
	)
}

// the terminal's own cursor color suits its configured background, so a bar or
// underline can vanish under a theme that inverts it
func cursorColor(cx *Context, mode view.Mode) color.Color {
	st, ok := modeCursorStyleFor(cx.Theme(), mode, true)
	if !ok {
		return nil
	}
	if bg := st.BgColor(); !bg.IsReset() {
		return bg
	}
	return nil
}
