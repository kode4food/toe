package language

import (
	"errors"
	"fmt"
	"os"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/loader"
)

type (
	Languages struct {
		Languages        []Language
		LanguageServers  map[string]Server
		GrammarSelection GrammarSelection
		Grammars         []Grammar
	}

	Language struct {
		TextWidth          *int   `toml:"text-width"`
		Name               string `toml:"name"`
		LanguageID         string `toml:"language-id"`
		Scope              string `toml:"scope"`
		InjectionRegex     string `toml:"injection-regex"`
		FileTypes          []FileType
		Shebangs           []string `toml:"shebangs"`
		Roots              []string `toml:"roots"`
		LanguageServers    []ServerFeatures
		CommentTokens      []string
		BlockCommentTokens []core.BlockCommentToken
		Indent             Indent
		AutoPairs          AutoPairConfig
		AutoFormat         bool `toml:"auto-format"`
		Formatter          *Formatter
		SoftWrap           SoftWrap `toml:"soft-wrap"`
		Rulers             []int    `toml:"rulers"`
	}

	Indent struct {
		TabWidth *int
		Unit     string
	}

	FileType struct {
		Extension string
		Glob      string
	}

	ServerFeatures struct {
		Name string
	}

	Server struct {
		Command              string
		Args                 []string
		Environment          map[string]string
		Config               map[string]any
		Timeout              int
		RequiredRootPatterns []string
	}

	Formatter struct {
		Command string
		Args    []string
	}

	GrammarSelection struct {
		Only   []string
		Except []string
	}

	Grammar struct {
		Name   string
		Source GrammarSource
	}

	GrammarSource struct {
		Path    string
		Git     string
		Rev     string
		Subpath string
	}

	AutoPairConfig struct {
		Present bool
		Enable  *bool
		Pairs   [][2]rune
	}

	SoftWrap struct {
		Enable          *bool   `toml:"enable"`
		MaxWrap         *int    `toml:"max-wrap"`
		MaxIndentRetain *int    `toml:"max-indent-retain"`
		WrapIndicator   *string `toml:"wrap-indicator"`
		WrapAtTextWidth *bool   `toml:"wrap-at-text-width"`
	}
)

var ErrInvalidAutoPairConfig = errors.New("invalid auto-pair config")

const MinSoftWrapWidth = 10

func (a *AutoPairConfig) OrDefault() (core.AutoPairs, bool) {
	if !a.Present {
		return core.DefaultAutoPairs(), true
	}
	return a.AutoPairs()
}

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

func (a *AutoPairConfig) UnmarshalTOML(value any) error {
	cfg, ok := decodeAutoPairConfig(value)
	if !ok {
		return fmt.Errorf("%w: %v", ErrInvalidAutoPairConfig, value)
	}
	*a = cfg
	return nil
}

func LoadLanguage(lang string) *Language {
	if langs, ok := loadUserWorkspaceLanguages(); ok {
		for _, l := range langs.Languages {
			if l.Name == lang {
				return &l
			}
		}
	}
	return &Language{}
}

func DetectLanguage(path, content string) (string, bool) {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	global, ok := loader.LanguagesFile()
	if !ok {
		global = ""
	}
	langs, ok := LoadLanguagesForWorkspace(
		global, loader.WorkspaceLanguagesFile(cwd), cwd,
	)
	if !ok {
		return "", false
	}
	if lang, ok := languageForFilename(langs, path); ok {
		return lang.Name, true
	}
	if lang, ok := languageForShebang(langs, content); ok {
		return lang, true
	}
	return languageForMatch(langs, content)
}

func LoadBundledLanguages() (Languages, bool) {
	base, ok := loader.LoadDefaultLanguagesTOML()
	if !ok {
		return Languages{}, false
	}
	return decodeLanguagesMap(base)
}

func LoadLanguagesForWorkspace(
	global, workspace, dir string,
) (Languages, bool) {
	base, ok := loader.LoadDefaultLanguagesTOML()
	if !ok {
		return Languages{}, false
	}
	paths := []string{global}
	if loader.QueryWorkspaceTrust(dir, false) {
		paths = append(paths, workspace)
	}
	merged, ok := loader.LoadMergedTOMLWithBase(base, paths, 3)
	if !ok {
		return Languages{}, false
	}
	return decodeLanguagesMap(merged)
}

func loadUserWorkspaceLanguages() (Languages, bool) {
	path, ok := loader.LanguagesFile()
	if !ok {
		return Languages{}, false
	}
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	langs, ok := LoadLanguagesForWorkspace(
		path, loader.WorkspaceLanguagesFile(cwd), cwd,
	)
	return langs, ok
}
