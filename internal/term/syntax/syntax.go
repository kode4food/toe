// Package syntax provides Tree-sitter based syntax highlighting. It tokenizes
// source text using Tree-sitter grammars and highlight queries, returning
// scope-named spans that map directly to theme scopes
package syntax

import (
	"slices"
	"strings"
	"sync"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/kode4food/toe/internal/term/highlight"
)

type (
	// Cache holds Tree-sitter parser and query caches for a UI context
	Cache struct {
		mu        sync.RWMutex
		langCache map[string]*langEntry
		rawQuery  map[string][]byte
		rawInject map[string][]byte
	}

	// langEntry holds the compiled parser and query for a single language
	langEntry struct {
		parser         *sitter.Parser
		query          *sitter.Query
		capNames       []string
		injectionQuery *sitter.Query
		injectionCaps  []string
	}

	tsCapture struct {
		start, end int
		scope      string
		idx        uint32
	}
)

// NewSyntaxCache returns an initialized SyntaxCache
func NewSyntaxCache() *Cache {
	return &Cache{
		langCache: map[string]*langEntry{},
		rawQuery:  map[string][]byte{},
		rawInject: map[string][]byte{},
	}
}

// Tokenize parses text for lang and returns highlight spans with theme
// scope names. Tree-sitter is tried first; Chroma is the fallback
func (sc *Cache) Tokenize(text, lang string) []highlight.Span {
	if spans := sc.treeTokenize(text, lang); spans != nil {
		return spans
	}
	return highlight.Tokenize(text, lang)
}

func (sc *Cache) treeTokenize(text, lang string) []highlight.Span {
	language, ok := languageFor(lang)
	if !ok {
		return nil
	}
	lc, ok := sc.langCacheFor(lang, language)
	if !ok {
		return nil
	}

	src := []byte(text)
	tree := lc.parser.Parse(src, nil)
	if tree == nil {
		return nil
	}
	defer tree.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	root := tree.RootNode()
	matches := qc.Matches(lc.query, root, src)

	b2c := buildByteToChar(text)

	var captures []tsCapture
	for {
		m := matches.Next()
		if m == nil {
			break
		}
		if !m.SatisfiesTextPredicate(lc.query, nil, nil, src) {
			continue
		}
		for _, c := range m.Captures {
			name := lc.capNames[c.Index]
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
	captures = append(captures, sc.injectionCaptures(lc, root, src, b2c)...)

	slices.SortFunc(captures, func(a, b tsCapture) int {
		if a.start != b.start {
			return a.start - b.start
		}
		// lower capture index = higher priority
		return int(a.idx) - int(b.idx)
	})
	return buildSpans(captures)
}

func (sc *Cache) injectionCaptures(
	lc *langEntry, root *sitter.Node, src []byte, b2c []int,
) []tsCapture {
	if lc.injectionQuery == nil {
		return nil
	}
	qc := sitter.NewQueryCursor()
	defer qc.Close()

	matches := qc.Matches(lc.injectionQuery, root, src)
	var out []tsCapture
	for {
		m := matches.Next()
		if m == nil {
			break
		}
		lang := injectionLanguage(lc.injectionQuery, lc.injectionCaps, m, src)
		if lang == "" {
			continue
		}
		for _, c := range m.Captures {
			if lc.injectionCaps[c.Index] != "injection.content" {
				continue
			}
			sb := int(c.Node.StartByte())
			eb := int(c.Node.EndByte())
			if eb <= sb {
				continue
			}
			start := b2c[sb]
			injected := string(src[sb:eb])
			for _, sp := range sc.Tokenize(injected, lang) {
				out = append(out, tsCapture{
					start: start + sp.Start,
					end:   start + sp.End,
					scope: sp.Scope,
				})
			}
		}
	}
	return out
}

func (sc *Cache) langCacheFor(
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
	if err := p.SetLanguage(language); err != nil {
		p.Close()
		return nil, false
	}
	q, qErr := sitter.NewQuery(language, string(qb))
	if qErr != nil {
		p.Close()
		return nil, false
	}
	e := &langEntry{
		parser:   p,
		query:    q,
		capNames: q.CaptureNames(),
	}
	if iqb, ok := sc.injectionQueryFor(lang); ok {
		if iq, err := sitter.NewQuery(language, string(iqb)); err == nil {
			e.injectionQuery = iq
			e.injectionCaps = iq.CaptureNames()
		}
	}
	sc.mu.Lock()
	sc.langCache[lang] = e
	sc.mu.Unlock()
	return e, true
}

func (sc *Cache) queryFor(lang string) ([]byte, bool) {
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

func (sc *Cache) injectionQueryFor(lang string) ([]byte, bool) {
	sc.mu.RLock()
	if b, ok := sc.rawInject[lang]; ok {
		sc.mu.RUnlock()
		return b, true
	}
	sc.mu.RUnlock()

	b, ok := embeddedInjectionQuery(lang)
	if !ok {
		return nil, false
	}
	sc.mu.Lock()
	sc.rawInject[lang] = b
	sc.mu.Unlock()
	return b, true
}

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
		best := tsCapture{
			start: start, end: c.end,
			scope: c.scope, idx: c.idx,
		}
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

func injectionLanguage(
	q *sitter.Query, names []string, m *sitter.QueryMatch, src []byte,
) string {
	for _, p := range q.PropertySettings(m.PatternIndex) {
		if p.Key == "injection.language" && p.Value != nil {
			return *p.Value
		}
	}
	for _, c := range m.Captures {
		if names[c.Index] == "injection.language" {
			sb := int(c.Node.StartByte())
			eb := int(c.Node.EndByte())
			return string(src[sb:eb])
		}
	}
	return ""
}

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
