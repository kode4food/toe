package syntax

import (
	"slices"

	hcl "github.com/tree-sitter-grammars/tree-sitter-hcl/bindings/go"
	toml "github.com/tree-sitter-grammars/tree-sitter-toml/bindings/go"
	yaml "github.com/tree-sitter-grammars/tree-sitter-yaml/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
	bash "github.com/tree-sitter/tree-sitter-bash/bindings/go"
	css "github.com/tree-sitter/tree-sitter-css/bindings/go"
	golang "github.com/tree-sitter/tree-sitter-go/bindings/go"
	html "github.com/tree-sitter/tree-sitter-html/bindings/go"
	javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

var langRegistry = map[string]*sitter.Language{
	"bash":       sitter.NewLanguage(bash.Language()),
	"css":        sitter.NewLanguage(css.Language()),
	"go":         sitter.NewLanguage(golang.Language()),
	"hcl":        sitter.NewLanguage(hcl.Language()),
	"html":       sitter.NewLanguage(html.Language()),
	"javascript": sitter.NewLanguage(javascript.Language()),
	"toml":       sitter.NewLanguage(toml.Language()),
	"tsx":        sitter.NewLanguage(typescript.LanguageTSX()),
	"typescript": sitter.NewLanguage(typescript.LanguageTypescript()),
	"yaml":       sitter.NewLanguage(yaml.Language()),
}

func SupportedLanguages() []string {
	names := make([]string, 0, len(langRegistry))
	for name := range langRegistry {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func languageFor(name string) (*sitter.Language, bool) {
	l, ok := langRegistry[name]
	return l, ok
}
