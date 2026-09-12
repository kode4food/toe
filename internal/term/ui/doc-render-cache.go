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

		viewRowMaps     map[view.Id][]viewRowEntry
		viewAnnotations map[view.Id][]inlineAnnotation

		lastInfoKey infoPopupKey
		inputCaret  geom.Point
		infoBounds  geom.Area

		lastOptionsGen int

		lastW, lastH int
		lastDiagKey  diagPopupKey
		lastSpinner  animationState
		lastBlink    animationState
		lastToastRev int
		lastReg      rune
	}

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
		annotations []inlineAnnotation
	}

	// docRenderCache memoizes a single document's derived render state, keyed
	// internally by revision so it is recomputed only when the document changes
	docRenderCache struct {
		rawTextRev    int
		rawTextCached string
		hexScratch    []view.DocumentColor

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
		windowByte                      int
	}

	lineIndexEntry struct {
		charStart   int
		byteStart   int
		endingWidth int
	}

	matchSpan struct{ from, to int }

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
		docCaches:       map[view.DocumentId]*docRenderCache{},
		viewRowMaps:     map[view.Id][]viewRowEntry{},
		viewAnnotations: map[view.Id][]inlineAnnotation{},
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
				delete(c.viewAnnotations, id)
			}
		}
	}
}

func (d *docRenderCache) ensureRawText(rev int, text core.Rope) string {
	if d.rawTextRev != rev || d.rawTextCached == "" {
		d.rawTextRev = rev
		d.rawTextCached = text.String()
	}
	return d.rawTextCached
}

type ensureHighlightArgs struct {
	cache   *syntax.Cache
	rev     int
	lang    string
	rawText string
}

func (d *docRenderCache) ensureHightlight(
	args ensureHighlightArgs,
) []highlight.Span {
	lang := args.lang
	rev := args.rev
	if lang != view.DefaultLanguage && (d.hlRev != rev || d.hlLang != lang) {
		d.hlRev = rev
		d.hlLang = lang
		d.hlSpans = args.cache.Tokenize(core.Source{
			Text: highlight.NormalizeNewlines(args.rawText),
			Lang: lang,
		})

	}
	if lang == view.DefaultLanguage {
		return nil
	}
	return d.hlSpans
}

type ensureSearchSpansArgs struct {
	rev     int
	pattern string
	rawText string
}

func (d *docRenderCache) ensureSearchSpans(args ensureSearchSpansArgs) {
	if d.searchRev == args.rev && d.searchPattern == args.pattern {
		return
	}
	d.searchRev = args.rev
	d.searchPattern = args.pattern
	d.searchSpans = nil
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
		from := b2r[loc[0]]
		to := b2r[loc[1]]
		if to > from {
			d.searchSpans = append(d.searchSpans, matchSpan{from, to})
		}
	}
}

func (d *docRenderCache) ensureLineIndex(
	rev int, rawText string,
) []lineIndexEntry {
	if d.liRev == rev && d.lineIndex != nil {
		return d.lineIndex
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
			charStart: charPos,
			byteStart: bytePos + 1,
		})
	}
	idx = append(idx, lineIndexEntry{
		charStart: charPos,
		byteStart: len(rawText),
	})
	d.liRev = rev
	d.lineIndex = idx
	return idx
}

func (d *docRenderCache) ensureLinePrefix(
	req *linePrefixRequest,
) linePrefixScan {
	if d.prefixRev != req.rev || d.prefixHOff != req.horzOff ||
		d.prefixTabWidth != req.tabWidth {
		d.prefixRev = req.rev
		d.prefixHOff = req.horzOff
		d.prefixTabWidth = req.tabWidth
		d.linePrefix = make(map[int]linePrefixScan, len(d.linePrefix))
	}
	if r, ok := d.linePrefix[req.lineNum]; ok {
		return r
	}
	res := scanLinePrefix(req)
	d.linePrefix[req.lineNum] = res
	return res
}
