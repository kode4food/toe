package language_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/view/language"
)

const lexerTestDefault = "text"

var (
	lexerTestSuffixes = []string{"", "~", ".bak", ".in", ".rpmsave"}
	lexerTestClass    = regexp.MustCompile(`\[([^\]])[^\]]*\]`)
	lexerTestNames    = []string{
		"x.txt", "README", "Makefile", "CMakeLists.txt", ".bashrc",
		"x.unknownext", "x", "x.go.orig", "x.tar.gz", "Dockerfile.in",
	}
)

func TestLexerIndex(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	bundled, ok := language.LoadBundledLanguages()
	assert.True(t, ok)

	t.Run("matches chroma for every glob", func(t *testing.T) {
		if testing.Short() {
			t.Skip("slow: full chroma scan per name")
		}
		registry := new(language.Registry)
		for _, name := range lexerTestCorpus() {
			if language.ForFilename(bundled, name) != nil {
				continue
			}
			got := registry.Detect(language.DetectArgs{
				Path:    name,
				Default: lexerTestDefault,
			})
			assert.Equal(t, name+"="+chromaOracle(name), name+"="+got)
		}
	})
}

func lexerTestCorpus() []string {
	names := append([]string{}, lexerTestNames...)
	sample := strings.NewReplacer("*", "x", "?", "x")
	for _, lex := range lexers.GlobalLexerRegistry.Lexers {
		cfg := lex.Config()
		globs := append(cfg.Filenames[:len(cfg.Filenames):len(cfg.Filenames)],
			cfg.AliasFilenames...,
		)
		for _, glob := range globs {
			base := sample.Replace(lexerTestClass.ReplaceAllString(glob, "$1"))
			for _, suf := range lexerTestSuffixes {
				names = append(names, base+suf)
			}
		}
	}
	return names
}

func chromaOracle(name string) string {
	if lex := lexers.Match(name); lex != nil {
		return strings.ToLower(lex.Config().Name)
	}
	if lex := lexers.Analyse(""); lex != nil {
		return strings.ToLower(lex.Config().Name)
	}
	return lexerTestDefault
}
