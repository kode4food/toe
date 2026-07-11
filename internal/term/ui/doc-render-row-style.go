package ui

import (
	"sort"

	"github.com/kode4food/toe/internal/term/highlight"
	"github.com/kode4food/toe/internal/tui"
)

type selectionAtRes struct {
	cursor   bool
	primary  bool
	selected bool
}

func (r *rowRender) selectionAt(pos int) selectionAtRes {
	for _, sp := range r.selSpans {
		if pos == sp.cur {
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

func (r *rowRender) diagnosticAt(pos int) (tui.Style, bool) {
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
	return best.style, ok
}

// baseStyleAt returns the syntax/glyph style that would apply to pos absent any
// selection or cursor overlay
func (r *rowRender) baseStyleAt(pos int, glyph documentGlyph) tui.Style {
	switch {
	case glyph == documentGlyphGuide:
		return r.tuiStyles.indentGuide
	case glyph == documentGlyphWhitespace:
		return r.tuiStyles.whitespace
	case r.hlSpans != nil:
		if scope, ok := r.hlScopeAt(pos); ok {
			return r.hlStyle(scope)
		}
	}
	return r.tuiStyles.text
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
func overlaySelStyle(base, sel tui.Style) tui.Style {
	if !sel.BgColor().IsReset() {
		base = base.Bg(sel.BgColor())
	}
	if !sel.FgColor().IsReset() {
		base = base.Fg(sel.FgColor())
	}
	return base
}

func overlayBgStyle(base, overlay tui.Style) tui.Style {
	if !overlay.BgColor().IsReset() {
		base = base.Bg(overlay.BgColor())
	}
	return base
}

func overlayDiagnosticStyle(base, diag tui.Style) tui.Style {
	if !diag.FgColor().IsReset() {
		base = base.Fg(diag.FgColor())
	}
	if !diag.BgColor().IsReset() {
		base = base.Bg(diag.BgColor())
	}
	if !diag.UnderlineColor().IsReset() {
		base = base.UlColor(diag.UnderlineColor())
	}
	if diag.UnderlineStyle() != tui.UnderlineReset {
		base = base.UlStyle(diag.UnderlineStyle())
	}
	if mod := diag.Modifier(); mod != 0 {
		base = base.Mod(mod)
	}
	return base
}
