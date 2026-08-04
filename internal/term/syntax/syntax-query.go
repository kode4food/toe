package syntax

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/kode4food/toe/internal/term/highlight"
)

func (sc *Cache) langCacheFor(
	lang string, language *sitter.Language,
) *langEntry {
	sc.mu.RLock()
	if e, ok := sc.langCache[lang]; ok {
		sc.mu.RUnlock()
		return e
	}
	sc.mu.RUnlock()

	qb, ok := sc.queryFor(lang)
	if !ok {
		return nil
	}
	p := sitter.NewParser()
	if err := p.SetLanguage(language); err != nil {
		p.Close()
		return nil
	}
	q, qErr := sitter.NewQuery(language, string(qb))
	if qErr != nil {
		p.Close()
		return nil
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
	return e
}

func (sc *Cache) queryFor(lang string) ([]byte, bool) {
	sc.mu.RLock()
	if b, ok := sc.rawQuery[lang]; ok {
		sc.mu.RUnlock()
		return b, true
	}
	sc.mu.RUnlock()

	b, ok := resolveQueryDir("queries", lang, map[string]bool{})
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
