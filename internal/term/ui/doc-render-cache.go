package ui

import (
	"regexp"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/command"
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
		stylesKey  styleKey
		lgStyles   *lipglossStyles
		tuiStyles  *tuiStyles
		hlFn       func(string) lipgloss.Style
		hlTUICache map[string]tui.Style

		viewRowMaps map[view.Id][]viewRowEntry

		lastInfoTitle string
		lastInfoItems []command.KeyHint

		lastOptionsGen int

		lastW, lastH  int
		lastDiagKey   diagPopupKey
		lastSpinFrame int
	}

	// styleKey identifies the theme+mode combination the cached lipgloss/tui
	// styles were built for
	styleKey struct {
		theme string
		mode  view.Mode
	}

	// diagPopupKey identifies the diagnostic popup's rendered content, so a
	// change (including disappearing) can be detected across frames
	diagPopupKey struct {
		severity view.DiagnosticSeverity
		text     string
	}

	viewRowEntry struct {
		logLine int
		offset  int
		prefixW int
		filler  bool
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

		prefixRev  int
		prefixHOff int
		prefixTabW int

		// linePrefix caches scanLinePrefix results per line; a change to
		// the revision, horizontal offset, or tab width invalidates all
		// lines at once
		linePrefix map[int]linePrefixScan

		// lineIndex holds one entry per line plus a sentinel, built in a
		// single pass over rawTextCached
		lineIndex []lineIndexEntry
		liRev     int
	}

	linePrefixScan struct {
		indentCol, windowPos, windowCol int
	}

	lineIndexEntry struct {
		charStart int
		byteStart int
		endingLen int
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

func newRenderCache() *renderCache {
	return &renderCache{
		docCaches:   map[view.DocumentId]*docRenderCache{},
		viewRowMaps: map[view.Id][]viewRowEntry{},
	}
}

// evictClosed drops entries for closed documents and views because docCaches
// retains each document's full text
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

func (dc *docRenderCache) ensureHL(
	sc *syntax.Cache, rev int, lang, rawText string,
) []highlight.Span {
	if lang != "text" && (dc.hlRev != rev || dc.hlLang != lang) {
		dc.hlRev = rev
		dc.hlLang = lang
		dc.hlSpans = sc.Tokenize(
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
		idx[len(idx)-1].endingLen = endingLen
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
	if dc.prefixRev != args.rev || dc.prefixHOff != args.horizontalOffset ||
		dc.prefixTabW != args.tabWidth {
		dc.prefixRev = args.rev
		dc.prefixHOff = args.horizontalOffset
		dc.prefixTabW = args.tabWidth
		dc.linePrefix = make(map[int]linePrefixScan, len(dc.linePrefix))
	}
	if r, ok := dc.linePrefix[args.lineNum]; ok {
		return r
	}
	res := scanLinePrefix(args)
	dc.linePrefix[args.lineNum] = res
	return res
}
