package syntax

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/kode4food/toe/internal/term/highlight"
)

func (sc *Cache) langCacheFor(lang string, language *sitter.Language) *langEntry {
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

	b, ok := resolveQueryDir(resolveQueryDirArgs{
		dir:  "queries",
		lang: lang,
	}, map[string]bool{})
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

// buildSpans: innermost capture wins; cs must list outer captures before inner ones
func buildSpans(cs []tsCapture) []highlight.Span {
	spans := make([]highlight.Span, 0, len(cs))
	var stack []tsCapture
	pos := 0
	for _, c := range cs {
		for len(stack) > 0 && stack[len(stack)-1].end <= c.start {
			top := stack[len(stack)-1]
			spans = appendSpan(spans, pos, top)
			pos = max(pos, top.end)
			stack = stack[:len(stack)-1]
		}
		if len(stack) > 0 {
			spans = appendSpan(spans, pos, tsCapture{
				end:   c.start,
				scope: stack[len(stack)-1].scope,
			})
		}
		pos = max(pos, c.start)
		stack = append(stack, c)
	}
	for i := len(stack) - 1; i >= 0; i-- {
		spans = appendSpan(spans, pos, stack[i])
		pos = max(pos, stack[i].end)
	}
	return spans
}

func injectionLanguage(q *sitter.Query, names []string, m *sitter.QueryMatch, src []byte) string {
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

func appendSpan(spans []highlight.Span, from int, c tsCapture) []highlight.Span {
	if c.end <= from {
		return spans
	}
	return append(spans, highlight.Span{
		Start: from,
		End:   c.end,
		Scope: c.scope,
	})
}
