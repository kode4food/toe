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
