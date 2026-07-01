// Package syntax provides Tree-sitter based syntax highlighting. It tokenizes
// source text using Tree-sitter grammars and highlight queries, returning
// scope-named spans that map directly to theme scopes
package syntax

import (
	"context"
	"slices"
	"strings"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/kode4food/toe/internal/term/highlight"
)

type (
	// SyntaxCache holds Tree-sitter parser and query caches for a UI context
	SyntaxCache struct {
		mu        sync.RWMutex
		langCache map[string]*langEntry
		rawQuery  map[string][]byte
	}

	// langEntry holds the compiled parser and query for a single language
	langEntry struct {
		parser *sitter.Parser
		query  *sitter.Query
	}

	tsCapture struct {
		start, end int
		scope      string
		idx        uint32
	}
)

// NewSyntaxCache returns an initialized SyntaxCache
func NewSyntaxCache() *SyntaxCache {
	return &SyntaxCache{
		langCache: map[string]*langEntry{},
		rawQuery:  map[string][]byte{},
	}
}

// Tokenize parses text for lang and returns highlight spans with theme
// scope names. Tree-sitter is tried first; Chroma is the fallback
func (sc *SyntaxCache) Tokenize(text, lang string) []highlight.Span {
	if spans := sc.treeTokenize(text, lang); spans != nil {
		return spans
	}
	return highlight.Tokenize(text, lang)
}

func (sc *SyntaxCache) treeTokenize(text, lang string) []highlight.Span {
	language, ok := languageFor(lang)
	if !ok {
		return nil
	}
	lc, ok := sc.langCacheFor(lang, language)
	if !ok {
		return nil
	}

	src := []byte(text)
	tree, err := lc.parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return nil
	}
	if tree == nil {
		return nil
	}
	defer tree.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()
	qc.Exec(lc.query, tree.RootNode())

	b2c := buildByteToChar(text)

	var captures []tsCapture
	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}
		m = qc.FilterPredicates(m, src)
		for _, c := range m.Captures {
			name := lc.query.CaptureNameForId(c.Index)
			scope := strings.TrimPrefix(name, "@")
			if scope == "" {
				continue
			}
			sb := int(c.Node.StartByte())
			eb := int(c.Node.EndByte())
			if eb <= sb {
				continue
			}
			captures = append(captures, tsCapture{
				start: b2c[sb],
				end:   b2c[eb],
				scope: scope,
				idx:   c.Index,
			})
		}
	}

	if len(captures) == 0 {
		return nil
	}

	slices.SortFunc(captures, func(a, b tsCapture) int {
		if a.start != b.start {
			return a.start - b.start
		}
		// lower capture index = higher priority
		return int(a.idx) - int(b.idx)
	})
	return buildSpans(captures)
}

func (sc *SyntaxCache) langCacheFor(
	lang string, language *sitter.Language,
) (*langEntry, bool) {
	sc.mu.RLock()
	if e, ok := sc.langCache[lang]; ok {
		sc.mu.RUnlock()
		return e, true
	}
	sc.mu.RUnlock()

	qb, ok := sc.queryFor(lang)
	if !ok {
		return nil, false
	}
	p := sitter.NewParser()
	p.SetLanguage(language)
	q, err := sitter.NewQuery(qb, language)
	if err != nil {
		return nil, false
	}
	e := &langEntry{parser: p, query: q}
	sc.mu.Lock()
	sc.langCache[lang] = e
	sc.mu.Unlock()
	return e, true
}

func (sc *SyntaxCache) queryFor(lang string) ([]byte, bool) {
	sc.mu.RLock()
	if b, ok := sc.rawQuery[lang]; ok {
		sc.mu.RUnlock()
		return b, true
	}
	sc.mu.RUnlock()

	b, ok := resolveQuery(lang, map[string]bool{})
	if !ok {
		return nil, false
	}
	sc.mu.Lock()
	sc.rawQuery[lang] = b
	sc.mu.Unlock()
	return b, true
}

// buildByteToChar builds a table mapping UTF-8 byte offset → rune offset
func buildByteToChar(text string) []int {
	table := make([]int, len(text)+1)
	ri := 0
	for bi := range text {
		table[bi] = ri
		ri++
	}
	table[len(text)] = ri
	return table
}

// buildSpans converts a sorted capture list into non-overlapping Spans,
// keeping the highest-priority (lowest index) capture at each position
func buildSpans(cs []tsCapture) []highlight.Span {
	spans := make([]highlight.Span, 0, len(cs))
	pos := 0
	for i := 0; i < len(cs); {
		c := cs[i]
		if c.end <= pos {
			i++
			continue
		}
		start := max(c.start, pos)
		best := tsCapture{start: start, end: c.end, scope: c.scope, idx: c.idx}
		j := i + 1
		for j < len(cs) && cs[j].start == c.start {
			if cs[j].idx < best.idx {
				best.end = cs[j].end
				best.scope = cs[j].scope
				best.idx = cs[j].idx
			}
			j++
		}
		if best.end > best.start {
			spans = append(spans, highlight.Span{
				Start: best.start,
				End:   best.end,
				Scope: best.scope,
			})
			pos = best.end
		}
		i = j
	}
	return spans
}

// resolveQuery loads and resolves ;; inherits: directives for a language
func resolveQuery(lang string, seen map[string]bool) ([]byte, bool) {
	if seen[lang] {
		return nil, false
	}
	seen[lang] = true

	raw, ok := embeddedQuery(lang)
	if !ok {
		return nil, false
	}

	var out []byte
	for line := range strings.SplitSeq(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(trimmed, "; inherits:"); ok {
			for parent := range strings.SplitSeq(after, ",") {
				parent = strings.TrimSpace(parent)
				if parent == "" {
					continue
				}
				if pb, ok := resolveQuery(parent, seen); ok {
					out = append(out, pb...)
					out = append(out, '\n')
				}
			}
			continue
		}
		out = append(out, line...)
		out = append(out, '\n')
	}
	return out, true
}
