package language

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

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
		AutoFormat         *bool
		Formatter          *Formatter
		Debugger           *DebugAdapter
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

	ServerFeature string

	ServerFeatures struct {
		Name     string
		Only     []ServerFeature
		Excluded []ServerFeature
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

	DebugAdapter struct {
		Name      string
		Transport string
		Command   string
		Args      []string
		PortArg   string
		Templates []DebugTemplate
		Quirks    DebuggerQuirks
	}

	DebugTemplate struct {
		Name       string
		Request    string
		Completion []DebugCompletion
		Args       map[string]any
	}

	DebugCompletion struct {
		Name       string
		Completion string
		Default    string
	}

	DebuggerQuirks struct {
		AbsolutePaths bool
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

// LSPWorkspaceArgs holds the parameters for FindLSPWorkspace
type LSPWorkspaceArgs struct {
	File              string
	Language          *Language
	RootDirs          []string
	Workspace         string
	WorkspaceFallback bool
}

// SelectedGrammars applies the configured grammar include/exclude selection
func (l *Languages) SelectedGrammars() []Grammar {
	out := make([]Grammar, 0, len(l.Grammars))
	if len(l.GrammarSelection.Only) > 0 {
		for _, grammar := range l.Grammars {
			if slices.Contains(l.GrammarSelection.Only, grammar.Name) {
				out = append(out, grammar)
			}
		}
		return out
	}
	for _, grammar := range l.Grammars {
		if !slices.Contains(l.GrammarSelection.Except, grammar.Name) {
			out = append(out, grammar)
		}
	}
	return out
}

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

// LoadLanguageForScope resolves a language by its syntax scope
func LoadLanguageForScope(scope string) *Language {
	if langs, ok := loadUserWorkspaceLanguages(); ok {
		for _, l := range langs.Languages {
			if l.Scope == scope {
				return &l
			}
		}
	}
	return &Language{}
}

// FindLanguageRoot returns the topmost ancestor with a language root marker
func FindLanguageRoot(path string, lang *Language) (string, bool) {
	if len(lang.Roots) == 0 {
		return "", false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	if info, err := os.Stat(abs); err == nil && !info.IsDir() {
		abs = filepath.Dir(abs)
	}
	var found string
	for {
		if languageRootMarkerExists(abs, lang.Roots) {
			found = abs
		}
		next := filepath.Dir(abs)
		if next == abs {
			break
		}
		abs = next
	}
	return found, found != ""
}

// FindLSPWorkspace resolves a language server workspace root for a file
func FindLSPWorkspace(args LSPWorkspaceArgs) (string, bool) {
	abs, err := filepath.Abs(args.File)
	if err != nil {
		abs = args.File
	}
	abs = filepath.Clean(abs)
	workspace := filepath.Clean(args.Workspace)
	if rel, err := filepath.Rel(workspace, abs); err != nil || rel == ".." ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	if info, err := os.Stat(abs); err == nil && !info.IsDir() {
		abs = filepath.Dir(abs)
	}
	var top string
	for dir := abs; ; dir = filepath.Dir(dir) {
		if args.Language != nil &&
			languageRootMarkerExists(dir, args.Language.Roots) {
			top = dir
		}
		if lspRootDirMatches(dir, args.RootDirs, workspace) {
			if top != "" {
				return top, true
			}
			return workspace, true
		}
		if dir == workspace {
			if top != "" {
				return top, true
			}
			if args.WorkspaceFallback {
				return workspace, true
			}
			return "", false
		}
		next := filepath.Dir(dir)
		if next == dir {
			return "", false
		}
	}
}

func lspRootDirMatches(dir string, rootDirs []string, workspace string) bool {
	for _, root := range rootDirs {
		if filepath.Clean(filepath.Join(workspace, root)) == dir {
			return true
		}
	}
	return false
}

func DetectLanguage(path, content string) (string, bool) {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	global, ok := UserLanguagesPath()
	if !ok {
		global = ""
	}
	langs, ok := LoadLanguagesForWorkspace(
		global, WorkspaceLanguagesPath(cwd), cwd,
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

func loadUserWorkspaceLanguages() (Languages, bool) {
	path, ok := UserLanguagesPath()
	if !ok {
		return Languages{}, false
	}
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	langs, ok := LoadLanguagesForWorkspace(
		path, WorkspaceLanguagesPath(cwd), cwd,
	)
	return langs, ok
}

func LoadLanguages(path string) (Languages, bool) {
	merged, ok := loader.LoadMergedTOML([]string{path}, 3)
	if !ok {
		return Languages{}, false
	}
	return decodeLanguagesMap(merged)
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
	if loader.QueryWorkspaceTrust(dir, false) == loader.TrustTrusted {
		paths = append(paths, workspace)
	}
	merged, ok := loader.LoadMergedTOMLWithBase(base, paths, 3)
	if !ok {
		return Languages{}, false
	}
	return decodeLanguagesMap(merged)
}

func UserLanguagesPath() (string, bool) {
	return loader.LanguagesFile()
}

func WorkspaceLanguagesPath(dir string) string {
	return loader.WorkspaceLanguagesFile(dir)
}
