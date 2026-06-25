package syntax

import (
	"slices"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/bash"
	"github.com/smacker/go-tree-sitter/css"
	"github.com/smacker/go-tree-sitter/dockerfile"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/hcl"
	"github.com/smacker/go-tree-sitter/html"
	"github.com/smacker/go-tree-sitter/javascript"
	md "github.com/smacker/go-tree-sitter/markdown/tree-sitter-markdown"
	"github.com/smacker/go-tree-sitter/protobuf"
	"github.com/smacker/go-tree-sitter/sql"
	"github.com/smacker/go-tree-sitter/toml"
	"github.com/smacker/go-tree-sitter/typescript/tsx"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
	"github.com/smacker/go-tree-sitter/yaml"
)

var langRegistry = map[string]*sitter.Language{
	"bash":       bash.GetLanguage(),
	"css":        css.GetLanguage(),
	"dockerfile": dockerfile.GetLanguage(),
	"go":         golang.GetLanguage(),
	"hcl":        hcl.GetLanguage(),
	"html":       html.GetLanguage(),
	"javascript": javascript.GetLanguage(),
	"markdown":   md.GetLanguage(),
	"protobuf":   protobuf.GetLanguage(),
	"sql":        sql.GetLanguage(),
	"toml":       toml.GetLanguage(),
	"tsx":        tsx.GetLanguage(),
	"typescript": typescript.GetLanguage(),
	"yaml":       yaml.GetLanguage(),
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
