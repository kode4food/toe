package ui

import (
	"regexp"

	"charm.land/lipgloss/v2"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/highlight"
	"github.com/kode4food/toe/internal/term/syntax"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type (
	renderCache struct {
		// docCaches holds per-document raw-text, highlight, and search-match
		// caches so multiple panes showing different documents do not evict
		// each other's tokenization every frame
		docCaches map[view.DocumentId]*docRenderCache

		// rebuilt only when theme or mode changes between frames
		stylesKey  string
		lgStyles   *lipglossStyles
		tuiStyles  *tuiStyles
		hlFn       func(string) lipgloss.Style
		hlTUICache map[string]tui.Style

		viewRowMaps map[view.Id][]viewRowEntry
	}

	viewRowEntry struct {
		logLine int
		offset  int
		prefixW int
	}

	// docRenderCache memoizes a single document's derived render state, keyed
	// internally by revision so it is recomputed only when the document changes
	docRenderCache struct {
		rawTextRev    int
		rawTextCached string

		hlRev   int
		hlLang  string
		hlSpans []highlight.Span

		smRev   int
		smPat   string
		smSpans []matchSpan
	}

	matchSpan struct{ from, to int }
)

func newRenderCache() *renderCache {
	return &renderCache{
		docCaches:   map[view.DocumentId]*docRenderCache{},
		viewRowMaps: map[view.Id][]viewRowEntry{},
	}
}

func (dc *docRenderCache) ensureRawText(rev int, text core.Rope) string {
	if dc.rawTextRev != rev || dc.rawTextCached == "" {
		dc.rawTextRev = rev
		dc.rawTextCached = text.String()
	}
	return dc.rawTextCached
}

func (dc *docRenderCache) ensureHL(
	rev int, lang, rawText string,
) []highlight.Span {
	if lang != "text" && (dc.hlRev != rev || dc.hlLang != lang) {
		dc.hlRev = rev
		dc.hlLang = lang
		dc.hlSpans = syntax.Tokenize(
			highlight.NormalizeNewlines(rawText), lang,
		)
	}
	if lang == "text" {
		return nil
	}
	return dc.hlSpans
}

func (dc *docRenderCache) ensureSearchSpans(rev int, pat, rawText string) {
	if dc.smRev == rev && dc.smPat == pat {
		return
	}
	dc.smRev = rev
	dc.smPat = pat
	dc.smSpans = nil
	if pat == "" {
		return
	}
	re, err := regexp.Compile(pat)
	if err != nil {
		return
	}
	locs := re.FindAllStringIndex(rawText, -1)
	if len(locs) == 0 {
		return
	}
	b2r := make([]int, len(rawText)+1)
	ri := 0
	for bi := range rawText {
		b2r[bi] = ri
		ri++
	}
	b2r[len(rawText)] = ri
	for _, loc := range locs {
		from, to := b2r[loc[0]], b2r[loc[1]]
		if to > from {
			dc.smSpans = append(dc.smSpans, matchSpan{from, to})
		}
	}
}
