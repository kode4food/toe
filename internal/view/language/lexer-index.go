package language

import (
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
)

type (
	// lexerIndex answers what chroma's Match would, but buckets each glob by
	// the extension any name it matches must end with
	lexerIndex struct {
		primary lexerGlobs
		alias   lexerGlobs
	}

	lexerGlobs struct {
		byExt map[string][]lexerGlob
		other []lexerGlob
	}

	lexerGlob struct {
		order int
		lexer chroma.Lexer
		glob  string
	}
)

const globMeta = `*?[\`

// mirrors chroma's unexported list of suffixes it accepts after any glob
var ignoredSuffixes = [...]string{
	"~", ".bak", ".old", ".orig",
	".dpkg-dist", ".dpkg-old", ".ucf-dist", ".ucf-new", ".ucf-old",
	".rpmnew", ".rpmorig", ".rpmsave",
	".in",
}

func (l lexerIndex) match(filename string) chroma.Lexer {
	base := filepath.Base(filename)
	if lex := l.primary.match(base); lex != nil {
		return lex
	}
	return l.alias.match(base)
}

// candidates are checked in registry order so chroma's priority sort sees the
// same input and breaks ties the same way
func (g lexerGlobs) match(base string) chroma.Lexer {
	cands := slices.Concat(g.byExt[filepath.Ext(base)], g.other)
	slices.SortFunc(cands, func(a, b lexerGlob) int {
		return a.order - b.order
	})
	var matched chroma.PrioritisedLexers
	for _, c := range cands {
		if globMatches(c.glob, base) {
			matched = append(matched, c.lexer)
		}
	}
	if len(matched) == 0 {
		return nil
	}
	sort.Sort(matched)
	return matched[0]
}

// a glob of "*" then a literal holding a dot only matches names sharing that
// literal's extension, with or without an ignored suffix
func (g *lexerGlobs) add(lg lexerGlob) {
	lit, ok := strings.CutPrefix(lg.glob, "*")
	simple := ok && !strings.ContainsAny(lit, globMeta) &&
		strings.Contains(lit, ".")
	if !simple {
		g.other = append(g.other, lg)
		return
	}
	g.addExt(filepath.Ext(lit), lg)
	for _, suf := range ignoredSuffixes {
		g.addExt(filepath.Ext(lit+suf), lg)
	}
}

func (g *lexerGlobs) addExt(ext string, lg lexerGlob) {
	list := g.byExt[ext]
	if n := len(list); n > 0 && list[n-1].order == lg.order {
		return
	}
	g.byExt[ext] = append(list, lg)
}

func newLexerIndex() *lexerIndex {
	idx := &lexerIndex{
		primary: lexerGlobs{byExt: map[string][]lexerGlob{}},
		alias:   lexerGlobs{byExt: map[string][]lexerGlob{}},
	}
	order := 0
	for _, lex := range lexers.GlobalLexerRegistry.Lexers {
		cfg := lex.Config()
		for _, glob := range cfg.Filenames {
			idx.primary.add(lexerGlob{order: order, lexer: lex, glob: glob})
			order++
		}
		for _, glob := range cfg.AliasFilenames {
			idx.alias.add(lexerGlob{order: order, lexer: lex, glob: glob})
			order++
		}
	}
	return idx
}

func globMatches(glob, base string) bool {
	if ok, _ := filepath.Match(glob, base); ok {
		return true
	}
	for _, suf := range ignoredSuffixes {
		if ok, _ := filepath.Match(glob+suf, base); ok {
			return true
		}
	}
	return false
}
