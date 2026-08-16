package ui

import (
	"regexp"
	"strings"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/highlight"
	"github.com/kode4food/toe/internal/term/syntax"
	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type (
	renderCache struct {
		docCaches map[view.DocumentId]*docRenderCache

		stylesKey styleKey
		styles    docStyleSet
		stylesDim docStyleSet

		viewRowMaps map[view.Id][]viewRowEntry

		lastInfoKey infoPopupKey
		inputCaret  geom.Point
		infoBounds  geom.Area

		lastOptionsGen int

		lastW, lastH int
		lastDiagKey  diagPopupKey
		lastSpinner  animationState
	}

	// styleKey identifies the theme+mode combination for cached styles
	// styles were built for
	styleKey struct {
		theme string
		mode  view.Mode
	}

	// docStyleSet bundles a theme's derived style tables so a focused and a
	// dimmed variant are always selected as one unit, never field by field
	docStyleSet struct {
		styles    *styles
		highlight func(string) tui.Style
		hlCache   map[string]tui.Style
	}

	// diagPopupKey identifies the diagnostic popup's rendered content, so a
	// change (including disappearing) can be detected across frames
	diagPopupKey struct {
		severity view.DiagnosticSeverity
		text     string
	}

	// infoPopupKey identifies the pending-key popup's rendered content, so a
	// change (including disappearing) can be detected across frames
	infoPopupKey struct {
		head  string
		title string
		items []command.KeyHint
	}

	viewRowEntry struct {
		logLine     int
		offset      int
		prefixWidth int
		filler      bool
	}

	// docRenderCache memoizes a single document's derived render state, keyed
	// internally by revision so it is recomputed only when the document changes
	docRenderCache struct {
		rawTextRev    int
		rawTextCached string

		hlRev   int
		hlLang  string
		hlSpans []highlight.Span

		searchRev     int
		searchPattern string
		searchSpans   []matchSpan

		prefixRev      int
		prefixHOff     int
		prefixTabWidth int

		linePrefix map[int]linePrefixScan

		lineIndex []lineIndexEntry
		liRev     int
	}

	linePrefixScan struct {
		indentCol, windowPos, windowCol int
	}

	lineIndexEntry struct {
		charStart   int
		byteStart   int
		endingWidth int
	}

	matchSpan struct{ from, to int }

	colorSpan struct {
		from, to int
		style    tui.Style
	}

	diagnosticSpan struct {
		from, to int
		severity view.DiagnosticSeverity
		style    tui.Style
	}

	inlineAnnotation struct {
		pos   int
		text  string
		style tui.Style
	}
)

func newDocStyleSet(th *theme.Theme, mode view.Mode) docStyleSet {
	return docStyleSet{
		styles:    buildStyles(th, mode),
		highlight: highlighterFor(th),
		hlCache:   make(map[string]tui.Style, 64),
	}
}

func newRenderCache() *renderCache {
	return &renderCache{
		docCaches:   map[view.DocumentId]*docRenderCache{},
		viewRowMaps: map[view.Id][]viewRowEntry{},
	}
}

func (c *renderCache) evictClosed(e *view.Editor) {
	docs := e.AllDocuments()
	if len(c.docCaches) > len(docs) {
		live := make(map[view.DocumentId]struct{}, len(docs))
		for _, d := range docs {
			live[d.ID()] = struct{}{}
		}
		for id := range c.docCaches {
			if _, ok := live[id]; !ok {
				delete(c.docCaches, id)
			}
		}
	}
	views := e.AllViews()
	if len(c.viewRowMaps) > len(views) {
		live := make(map[view.Id]struct{}, len(views))
		for _, v := range views {
			live[v.ID()] = struct{}{}
		}
		for id := range c.viewRowMaps {
			if _, ok := live[id]; !ok {
				delete(c.viewRowMaps, id)
			}
		}
	}
}

func (dc *docRenderCache) ensureRawText(rev int, text core.Rope) string {
	if dc.rawTextRev != rev || dc.rawTextCached == "" {
		dc.rawTextRev = rev
		dc.rawTextCached = text.String()
	}
	return dc.rawTextCached
}

type ensureHighlightArgs struct {
	cache   *syntax.Cache
	rev     int
	lang    string
	rawText string
}

func (dc *docRenderCache) ensureHightlight(
	args ensureHighlightArgs,
) []highlight.Span {
	lang := args.lang
	rev := args.rev
	if lang != view.DefaultLanguage && (dc.hlRev != rev || dc.hlLang != lang) {
		dc.hlRev = rev
		dc.hlLang = lang
		dc.hlSpans = args.cache.Tokenize(core.Source{
			Text: highlight.NormalizeNewlines(args.rawText),
			Lang: lang,
		})

	}
	if lang == view.DefaultLanguage {
		return nil
	}
	return dc.hlSpans
}

type ensureSearchSpansArgs struct {
	rev     int
	pattern string
	rawText string
}

func (dc *docRenderCache) ensureSearchSpans(args ensureSearchSpansArgs) {
	if dc.searchRev == args.rev && dc.searchPattern == args.pattern {
		return
	}
	dc.searchRev = args.rev
	dc.searchPattern = args.pattern
	dc.searchSpans = nil
	if args.pattern == "" {
		return
	}
	re, err := regexp.Compile(args.pattern)
	if err != nil {
		return
	}
	locs := re.FindAllStringIndex(args.rawText, -1)
	if len(locs) == 0 {
		return
	}
	b2r := make([]int, len(args.rawText)+1)
	ri := 0
	for bi := range args.rawText {
		b2r[bi] = ri
		ri++
	}
	b2r[len(args.rawText)] = ri
	for _, loc := range locs {
		from, to := b2r[loc[0]], b2r[loc[1]]
		if to > from {
			dc.searchSpans = append(dc.searchSpans, matchSpan{from, to})
		}
	}
}

func (dc *docRenderCache) ensureLineIndex(
	rev int, rawText string,
) []lineIndexEntry {
	if dc.liRev == rev && dc.lineIndex != nil {
		return dc.lineIndex
	}
	idx := make([]lineIndexEntry, 1, strings.Count(rawText, "\n")+2)
	charPos := 0
	for bytePos, ch := range rawText {
		charPos++
		if ch != '\n' {
			continue
		}
		endingLen := 1
		if bytePos > 0 && rawText[bytePos-1] == '\r' {
			endingLen = 2
		}
		idx[len(idx)-1].endingWidth = endingLen
		idx = append(idx, lineIndexEntry{
			charStart: charPos, byteStart: bytePos + 1,
		})
	}
	idx = append(idx, lineIndexEntry{
		charStart: charPos, byteStart: len(rawText),
	})
	dc.liRev = rev
	dc.lineIndex = idx
	return idx
}

func (dc *docRenderCache) ensureLinePrefix(args linePrefixArgs) linePrefixScan {
	if dc.prefixRev != args.rev || dc.prefixHOff != args.horzOff ||
		dc.prefixTabWidth != args.tabWidth {
		dc.prefixRev = args.rev
		dc.prefixHOff = args.horzOff
		dc.prefixTabWidth = args.tabWidth
		dc.linePrefix = make(map[int]linePrefixScan, len(dc.linePrefix))
	}
	if r, ok := dc.linePrefix[args.lineNum]; ok {
		return r
	}
	res := scanLinePrefix(args)
	dc.linePrefix[args.lineNum] = res
	return res
}
