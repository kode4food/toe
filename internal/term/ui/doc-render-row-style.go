package ui

import (
	"sort"

	"github.com/kode4food/toe/internal/term/highlight"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

// styleOverlay is a base style and the style being layered onto it
type styleOverlay struct {
	base    tui.Style
	overlay tui.Style
}

type selectionAtRes struct {
	cursor   bool
	primary  bool
	selected bool
}

func (r *rowRender) selectionAt(pos int) selectionAtRes {
	for _, sp := range r.selSpans {
		if pos == sp.cursor {
			return selectionAtRes{cursor: true, primary: sp.primary}
		}
		if pos >= sp.from && pos < sp.to {
			return selectionAtRes{selected: true}
		}
	}
	return selectionAtRes{}
}

func (r *rowRender) colorAt(pos int) (tui.Style, bool) {
	lo, hi := 0, len(r.docColors)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		sp := r.docColors[mid]
		if pos < sp.from {
			hi = mid - 1
		} else if pos >= sp.to {
			lo = mid + 1
		} else {
			return sp.style, true
		}
	}
	return tui.Style{}, false
}

func (r *rowRender) diagnosticAt(pos int) (diagnosticSpan, bool) {
	var best diagnosticSpan
	ok := false
	for _, sp := range r.diagnostics {
		if pos < sp.from {
			break
		}
		if pos >= sp.to {
			continue
		}
		if !ok || sp.severity > best.severity {
			best = sp
			ok = true
		}
	}
	return best, ok
}

// baseStyleAt returns the syntax/glyph style that would apply to pos absent any
// selection or cursor overlay
func (r *rowRender) baseStyleAt(pos int, glyph documentGlyph) tui.Style {
	switch {
	case glyph == documentGlyphGuide:
		return r.styles.indentGuide
	case glyph == documentGlyphWhitespace:
		return r.styles.whitespace
	case r.hlSpans != nil:
		scope, ok := r.hlScopeAt(pos)
		if !ok {
			break
		}
		if !isIdentifierScope(scope) {
			return r.hlStyle(scope)
		}
		if diag, dOk := r.diagnosticAt(pos); dOk &&
			diag.severity >= view.DiagnosticSeverityError {
			return r.styles.text
		}
		return r.hlStyle(scope)
	}
	return r.styles.text
}

// hlScopeAt resolves the highlight scope at pos by advancing hlIdx; callers
// must present non-decreasing positions, which rows() guarantees
func (r *rowRender) hlScopeAt(pos int) (string, bool) {
	spans := r.hlSpans
	for r.hlIdx < len(spans) && pos >= spans[r.hlIdx].End {
		r.hlIdx++
	}
	if r.hlIdx < len(spans) && pos >= spans[r.hlIdx].Start {
		return spans[r.hlIdx].Scope, true
	}
	return "", false
}

func isIdentifierScope(scope string) bool {
	switch scope {
	case "variable", "variable.parameter", "variable.other.member",
		"function", "function.macro", "type", "type.enum.variant",
		"namespace", "constructor":
		return true
	default:
		return false
	}
}

func rangeMatch(ranges []matchSpan, pos int) bool {
	lo, hi := 0, len(ranges)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		sp := ranges[mid]
		if pos < sp.from {
			hi = mid - 1
		} else if pos >= sp.to {
			lo = mid + 1
		} else {
			return true
		}
	}
	return false
}

func spanLowerBound(spans []highlight.Span, pos int) int {
	return sort.Search(len(spans), func(i int) bool {
		return spans[i].End > pos
	})
}

// overlaySelStyle overlays the bg (and explicit fg) of sel onto base,
// preserving the syntax foreground and attributes when sel has none
func overlaySelStyle(args styleOverlay) tui.Style {
	base := args.base
	if !args.overlay.BgColor().IsReset() {
		base = base.Bg(args.overlay.BgColor())
	}
	if !args.overlay.FgColor().IsReset() {
		base = base.Fg(args.overlay.FgColor())
	}
	return base
}

func overlayBgStyle(args styleOverlay) tui.Style {
	base := args.base
	if !args.overlay.BgColor().IsReset() {
		base = base.Bg(args.overlay.BgColor())
	}
	return base
}

func overlayDiagnosticStyle(args styleOverlay) tui.Style {
	base := args.base
	if !args.overlay.FgColor().IsReset() {
		base = base.Fg(args.overlay.FgColor())
	}
	if !args.overlay.BgColor().IsReset() {
		base = base.Bg(args.overlay.BgColor())
	}
	if !args.overlay.UnderlineColor().IsReset() {
		base = base.UlColor(args.overlay.UnderlineColor())
	}
	if args.overlay.UnderlineStyle() != tui.UnderlineReset {
		base = base.UlStyle(args.overlay.UnderlineStyle())
	}
	if mod := args.overlay.Modifier(); mod != 0 {
		base = base.Mod(mod)
	}
	return base
}
