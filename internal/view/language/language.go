package language

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kode4food/toe/internal/toml"

	"github.com/alecthomas/chroma/v2/lexers"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/loader"
)

type (
	// Languages is the loaded language configuration: every known language and
	// the language servers they refer to by name
	Languages struct {
		Languages       []Language
		LanguageServers map[string]Server
	}

	// Language describes one language: how files are recognized as it, and the
	// comment, indent, wrap, and server behavior that follows
	Language struct {
		Name           string `toml:"name"`
		LanguageID     string `toml:"language-id"`
		InjectionRegex string `toml:"injection-regex"`

		FileTypes []FileType
		Shebangs  []string `toml:"shebangs"`
		Roots     []string `toml:"roots"`

		CommentTokens      []string
		BlockCommentTokens []core.BlockCommentToken

		Indent    Indent
		AutoPairs AutoPairConfig

		TextWidth *int     `toml:"text-width"`
		SoftWrap  SoftWrap `toml:"soft-wrap"`
		Rulers    []int    `toml:"rulers"`

		LanguageServers []string
		AutoFormat      bool `toml:"auto-format"`
		Formatter       *Formatter
	}

	// Indent is a language's indentation style, with a nil TabWidth deferring
	// to the editor default
	Indent struct {
		TabWidth *int
		Unit     string
	}

	// FileType matches files to a language, by extension or by glob
	FileType struct {
		Extension string
		Glob      string
	}

	// Server is how to launch and configure one language server
	Server struct {
		Command      string
		Args         []string
		Environment  map[string]string
		Config       map[string]any
		Timeout      int
		RootPatterns []string
	}

	// Formatter is the external command that formats a language's documents
	Formatter struct {
		Command string
		Args    []string
	}

	// AutoPairConfig is a language's bracket-pairing behavior. Present marks
	// the section as configured, separating an explicit disable from silence
	AutoPairConfig struct {
		Present bool
		Enable  *bool
		Pairs   [][2]rune
	}

	// SoftWrap is a language's wrapping behavior, each nil field deferring to
	// the editor setting
	SoftWrap struct {
		Enable          *bool   `toml:"enable"`
		WrapIndicator   *string `toml:"wrap-indicator"`
		WrapAtTextWidth *bool   `toml:"wrap-at-text-width"`
	}
)

var ErrInvalidAutoPairConfig = errors.New("invalid auto-pair config")

// MinSoftWrapWidth is the width soft-wrap must exceed to engage, below which
// the wrapped rows are too narrow to read
const MinSoftWrapWidth = 10

// OrDefault returns the configured pairs, or the built-in set when unset
func (a *AutoPairConfig) OrDefault() (core.AutoPairs, bool) {
	if !a.Present {
		return core.DefaultAutoPairs(), true
	}
	return a.AutoPairs()
}

// AutoPairs returns the configured pairs only
func (a *AutoPairConfig) AutoPairs() (core.AutoPairs, bool) {
	if a.Enable != nil {
		if !*a.Enable {
			return core.AutoPairs{}, false
		}
		return core.DefaultAutoPairs(), true
	}
	if len(a.Pairs) == 0 {
		return core.AutoPairs{}, false
	}
	return core.NewAutoPairs(a.Pairs), true
}

// UnmarshalTOML accepts either a bool or a table of pair characters
func (a *AutoPairConfig) UnmarshalTOML(value any) error {
	if cfg, ok := decodeAutoPairConfig(value); ok {
		*a = cfg
		return nil
	}
	return fmt.Errorf("%w: %v", ErrInvalidAutoPairConfig, value)
}

// LoadBundledLanguages returns the definitions embedded in the binary
func LoadBundledLanguages() (Languages, bool) {
	if base, ok := loader.DefaultLanguages(); ok {
		return decodeLanguagesMap(base)
	}
	return Languages{}, false
}

// LoadLanguagesForWorkspace merges bundled, user, and workspace definitions
func LoadLanguagesForWorkspace(args loader.WorkspaceFiles) (Languages, bool) {
	base, ok := loader.DefaultLanguages()
	if !ok {
		return Languages{}, false
	}
	paths := []string{args.Global}
	if loader.QueryWorkspaceTrust(args.Dir, false) {
		paths = append(paths, args.Workspace)
	}
	if merged, ok := toml.LoadMergedWithBase(base, paths, 3); ok {
		return decodeLanguagesMap(merged)
	}
	return Languages{}, false
}

func definedLanguage(langs Languages, args DetectArgs) (string, bool) {
	if lang := ForFilename(langs, args.Path); lang != nil {
		return lang.Name, true
	}
	content := args.Content
	if lang, ok := languageForShebang(langs, content); ok {
		return lang, true
	}
	return languageForMatch(langs, content)
}

func chromaLanguage(index *lexerIndex, args DetectArgs) (string, bool) {
	if lex := index.match(args.Path); lex != nil {
		return strings.ToLower(lex.Config().Name), true
	}
	if lex := lexers.Analyse(args.Content); lex != nil {
		return strings.ToLower(lex.Config().Name), true
	}
	return "", false
}
