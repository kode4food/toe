package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/loader"
)

type (
	TextFormat struct {
		ViewportWidth       int
		TabWidth            int
		SoftWrap            bool
		MaxWrap             int
		MaxIndentRetain     int
		WrapIndicator       string
		SoftWrapAtTextWidth bool
	}

	Config struct {
		Theme  Theme  `toml:"theme"`
		Editor Editor `toml:"editor"`
	}

	Theme struct {
		Name     string
		Light    string
		Dark     string
		Fallback string
		Adaptive bool
	}

	Editor struct {
		Scrolloff          *int               `toml:"scrolloff"`
		ScrollLines        *int               `toml:"scroll-lines"`
		LineNumber         LineNumber         `toml:"line-number"`
		Cursorline         *bool              `toml:"cursorline"`
		Cursorcolumn       *bool              `toml:"cursorcolumn"`
		Mouse              *bool              `toml:"mouse"`
		MiddleClickPaste   *bool              `toml:"middle-click-paste"`
		TextWidth          *int               `toml:"text-width"`
		InsertFinalNewline *bool              `toml:"insert-final-newline"`
		TrimFinalNewlines  *bool              `toml:"trim-final-newlines"`
		TrimTrailingWS     *bool              `toml:"trim-trailing-whitespace"`
		AutoPairs          AutoPairConfig     `toml:"auto-pairs"`
		AutoSave           AutoSave           `toml:"auto-save"`
		AtomicSave         *bool              `toml:"atomic-save"`
		Insecure           *bool              `toml:"insecure"`
		EditorConfig       *bool              `toml:"editor-config"`
		ContinueComments   *bool              `toml:"continue-comments"`
		CursorShape        CursorShapeConfig  `toml:"cursor-shape"`
		StatusLine         StatusLineConfig   `toml:"statusline"`
		DefaultLineEnding  core.LineEnding    `toml:"default-line-ending"`
		Search             SearchConfig       `toml:"search"`
		SoftWrap           SoftWrap           `toml:"soft-wrap"`
		Rulers             []int              `toml:"rulers"`
		BufferLine         BufferLine         `toml:"bufferline"`
		Whitespace         WhitespaceConfig   `toml:"whitespace"`
		IndentGuides       IndentGuidesConfig `toml:"indent-guides"`
		Gutters            GutterConfig       `toml:"gutters"`
		Shell              []string           `toml:"shell"`
	}

	AutoSave struct {
		FocusLost  *bool              `toml:"focus-lost"`
		AfterDelay AutoSaveAfterDelay `toml:"after-delay"`
	}

	AutoSaveAfterDelay struct {
		Enable  *bool `toml:"enable"`
		Timeout *int  `toml:"timeout"`
	}

	CursorShapeConfig struct {
		Normal CursorKind `toml:"normal"`
		Select CursorKind `toml:"select"`
		Insert CursorKind `toml:"insert"`
	}

	CursorKind string

	StatusLineConfig struct {
		Left      []StatusLineElement `toml:"left"`
		Center    []StatusLineElement `toml:"center"`
		Right     []StatusLineElement `toml:"right"`
		Separator string              `toml:"separator"`
		Mode      ModeConfig          `toml:"mode"`
	}

	StatusLineElement string

	ModeConfig struct {
		Normal string `toml:"normal"`
		Insert string `toml:"insert"`
		Select string `toml:"select"`
	}

	LineNumber string

	BufferLine string

	WhitespaceConfig struct {
		Render     WhitespaceRender     `toml:"render"`
		Characters WhitespaceCharacters `toml:"characters"`
	}

	// WhitespaceRender holds per-character render settings. When set from a
	// plain string in TOML (e.g. `render = "all"`), Default carries the value
	// and all per-character fields remain nil. When set as a table, each field
	// is independently nullable and falls back to Default then "none"
	WhitespaceRender struct {
		Default *WhitespaceRenderValue
		Space   *WhitespaceRenderValue
		Nbsp    *WhitespaceRenderValue
		Nnbsp   *WhitespaceRenderValue
		Tab     *WhitespaceRenderValue
		Newline *WhitespaceRenderValue
	}

	WhitespaceRenderValue string

	WhitespaceCharacters struct {
		Space   string `toml:"space"`
		Nbsp    string `toml:"nbsp"`
		Nnbsp   string `toml:"nnbsp"`
		Tab     string `toml:"tab"`
		Tabpad  string `toml:"tabpad"`
		Newline string `toml:"newline"`
	}

	IndentGuidesConfig struct {
		Render     bool   `toml:"render"`
		Character  string `toml:"character"`
		SkipLevels *int   `toml:"skip-levels"`
	}

	// GutterConfig controls which gutters are shown and in what order. In TOML
	// it can be an array of type strings or a table with layout/line-numbers.
	// Present tracks whether the config was explicitly set (to distinguish an
	// empty layout from an absent one
	GutterConfig struct {
		Present     bool
		Layout      []GutterType            `toml:"layout"`
		LineNumbers GutterLineNumbersConfig `toml:"line-numbers"`
	}

	GutterType string

	GutterLineNumbersConfig struct {
		MinWidth *int `toml:"min-width"`
	}

	SearchConfig struct {
		SmartCase  *bool `toml:"smart-case"`
		WrapAround *bool `toml:"wrap-around"`
	}

	SoftWrap struct {
		Enable          *bool   `toml:"enable"`
		MaxWrap         *int    `toml:"max-wrap"`
		MaxIndentRetain *int    `toml:"max-indent-retain"`
		WrapIndicator   *string `toml:"wrap-indicator"`
		WrapAtTextWidth *bool   `toml:"wrap-at-text-width"`
	}

	AutoPairConfig struct {
		Present bool
		Enable  *bool
		Pairs   [][2]rune
	}

	Languages struct {
		Languages        []Language
		LanguageServers  map[string]LanguageServer
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
		LanguageServers    []LanguageServerFeatures
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

	LanguageServerFeature string

	LanguageServerFeatures struct {
		Name     string
		Only     []LanguageServerFeature
		Excluded []LanguageServerFeature
	}

	LanguageServer struct {
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
)

const (
	BufferLineNever    BufferLine = "never"
	BufferLineAlways   BufferLine = "always"
	BufferLineMultiple BufferLine = "multiple"

	StatusLineMode             StatusLineElement = "mode"
	StatusLineSpinner          StatusLineElement = "spinner"
	StatusLineFileBaseName     StatusLineElement = "file-base-name"
	StatusLineFileName         StatusLineElement = "file-name"
	StatusLineFileAbsolutePath StatusLineElement = "file-absolute-path"
	StatusLineReadOnly         StatusLineElement = "read-only-indicator"
	StatusLineFileEncoding     StatusLineElement = "file-encoding"
	StatusLineFileLineEnding   StatusLineElement = "file-line-ending"
	StatusLineFileIndentStyle  StatusLineElement = "file-indent-style"
	StatusLineFileType         StatusLineElement = "file-type"
	StatusLineDiagnostics      StatusLineElement = "diagnostics"
	StatusLineWorkspaceDiag    StatusLineElement = "workspace-diagnostics"
	StatusLineSelections       StatusLineElement = "selections"
	StatusLinePrimaryLen       StatusLineElement = "primary-selection-length"
	StatusLinePosition         StatusLineElement = "position"
	StatusLineSeparator        StatusLineElement = "separator"
	StatusLinePercent          StatusLineElement = "position-percentage"
	StatusLineTotalLines       StatusLineElement = "total-line-numbers"
	StatusLineSpacer           StatusLineElement = "spacer"
	StatusLineVersionControl   StatusLineElement = "version-control"
	StatusLineRegister         StatusLineElement = "register"
	StatusLineModified         StatusLineElement = "file-modified-indicator"

	CursorKindBlock     CursorKind = "block"
	CursorKindBar       CursorKind = "bar"
	CursorKindUnderline CursorKind = "underline"
	CursorKindHidden    CursorKind = "hidden"

	LineNumberAbsolute LineNumber = "absolute"
	LineNumberRelative LineNumber = "relative"

	DefaultScrolloff   = 5
	DefaultScrollLines = 3

	DefaultTextWidth       = 80
	DefaultTabWidth        = 4
	DefaultMaxWrap         = 20
	DefaultMaxIndentRetain = 40
	DefaultWrapIndicator   = "↪ "
	DefaultAutoSaveDelay   = 3000
	MinSoftWrapWidth       = 10

	WhitespaceRenderNone WhitespaceRenderValue = "none"
	WhitespaceRenderAll  WhitespaceRenderValue = "all"

	DefaultWSSpace   = '·' // U+00B7 middle dot
	DefaultWSNbsp    = '⍽' // U+237D shouldered open box
	DefaultWSNnbsp   = '␣' // U+2423 open box
	DefaultWSTab     = '→' // U+2192 rightwards arrow
	DefaultWSTabpad  = ' ' // space
	DefaultWSNewline = '⏎' // U+23CE return symbol

	DefaultIndentGuideChar = '│' // U+2502 box drawings light vertical

	GutterTypeDiagnostics GutterType = "diagnostics"
	GutterTypeLineNumbers GutterType = "line-numbers"
	GutterTypeSpacer      GutterType = "spacer"
	GutterTypeDiff        GutterType = "diff"

	DefaultGutterLineNumberMinWidth = 3
)

var (
	ErrInvalidAutoPairConfig   = errors.New("invalid auto-pair config")
	ErrInvalidCursorKind       = errors.New("invalid cursor kind")
	ErrInvalidLineNumber       = errors.New("invalid line-number value")
	ErrInvalidStatusLine       = errors.New("invalid statusline element")
	ErrInvalidWhitespaceRender = errors.New("invalid whitespace render value")
	ErrInvalidGutterType       = errors.New("invalid gutter type")
	ErrInvalidBufferLine       = errors.New("invalid bufferline value")
)

func TextFormatForLanguage(lang string, w int) *TextFormat {
	cfg, _ := LoadUserConfig()
	return TextFormatForLanguageWithConfig(lang, cfg, w)
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

func TextFormatForLanguageWithConfig(
	lang string, cfg *Config, w int,
) *TextFormat {
	return TextFormatForConfig(LoadLanguage(lang), cfg, w)
}

func TextFormatForConfig(lang *Language, cfg *Config, w int) *TextFormat {
	textWidth := intValue(nil, cfg.Editor.TextWidth, DefaultTextWidth)
	if lang.TextWidth != nil {
		textWidth = *lang.TextWidth
	}

	wrapAt := boolValue(
		lang.SoftWrap.WrapAtTextWidth,
		cfg.Editor.SoftWrap.WrapAtTextWidth,
		false,
	)
	if wrapAt {
		if textWidth >= w {
			wrapAt = false
		} else {
			w = textWidth
		}
	}

	enabled := boolValue(
		lang.SoftWrap.Enable, cfg.Editor.SoftWrap.Enable, false,
	)
	format := DefaultTextFormat(w)
	format.SoftWrap = enabled && w > MinSoftWrapWidth
	format.MaxWrap = min(
		intValue(lang.SoftWrap.MaxWrap, cfg.Editor.SoftWrap.MaxWrap,
			DefaultMaxWrap,
		),
		w/4,
	)
	format.MaxIndentRetain = min(
		intValue(
			lang.SoftWrap.MaxIndentRetain,
			cfg.Editor.SoftWrap.MaxIndentRetain,
			DefaultMaxIndentRetain,
		),
		w*2/5,
	)
	format.WrapIndicator = stringValue(
		lang.SoftWrap.WrapIndicator,
		cfg.Editor.SoftWrap.WrapIndicator,
		DefaultWrapIndicator,
	)
	format.SoftWrapAtTextWidth = wrapAt
	return format
}

func DefaultConfig() *Config {
	return &Config{
		Theme: Theme{Name: "mocha"},
		Editor: Editor{
			TextWidth: new(DefaultTextWidth),
			SoftWrap: SoftWrap{
				Enable: new(false),
			},
		},
	}
}

func DefaultTextFormat(w int) *TextFormat {
	return &TextFormat{
		ViewportWidth:   w,
		TabWidth:        DefaultTabWidth,
		SoftWrap:        false,
		MaxWrap:         min(DefaultMaxWrap, w/4),
		MaxIndentRetain: min(DefaultMaxIndentRetain, w*2/5),
		WrapIndicator:   DefaultWrapIndicator,
	}
}

func (c *Config) DefaultLineEnding() core.LineEnding {
	if c.Editor.DefaultLineEnding == "" {
		return core.NativeLineEnding()
	}
	return c.Editor.DefaultLineEnding
}

func LoadUserConfig() (*Config, bool) {
	path, ok := UserConfigPath()
	if !ok {
		return DefaultConfig(), false
	}
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	cfg, ok := LoadConfigForWorkspace(path, WorkspaceConfigPath(cwd), cwd)
	if !ok {
		return DefaultConfig(), false
	}
	return cfg, true
}

func LoadConfig(path string) (*Config, bool) {
	merged, ok := loader.LoadMergedTOML([]string{path}, 3)
	if !ok {
		return nil, false
	}
	return decodeConfigMap(merged)
}

func LoadConfigForWorkspace(global, workspace, dir string) (*Config, bool) {
	paths := []string{global}
	globalCfg, _ := LoadConfig(global)
	insecure := false
	if globalCfg != nil {
		insecure = globalCfg.Insecure()
	}
	if loader.QueryWorkspaceTrust(dir, insecure) ==
		loader.TrustTrusted {
		paths = append(paths, workspace)
	}
	merged, ok := loader.LoadMergedTOML(paths, 3)
	if !ok {
		return nil, false
	}
	return decodeConfigMap(merged)
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

// LSPWorkspaceArgs holds the parameters for FindLSPWorkspace
type LSPWorkspaceArgs struct {
	File              string
	Language          *Language
	RootDirs          []string
	Workspace         string
	WorkspaceFallback bool
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

func UserConfigPath() (string, bool) {
	return loader.ConfigFile()
}

func WorkspaceConfigPath(dir string) string {
	return loader.WorkspaceConfigFile(dir)
}

func UserLanguagesPath() (string, bool) {
	return loader.LanguagesFile()
}

func WorkspaceLanguagesPath(dir string) string {
	return loader.WorkspaceLanguagesFile(dir)
}

func IgnorePath() string {
	return loader.ConfigIgnoreFile()
}

func LogFilePath() (string, bool) {
	return loader.LogFile()
}

func WorkspaceTrustPath() (string, bool) {
	return loader.WorkspaceTrustFile()
}

func TrustWorkspace(dir string) error {
	return loader.TrustWorkspace(dir)
}

func UntrustWorkspace(dir string) error {
	return loader.UntrustWorkspace(dir)
}

func (c *Config) InsertFinalNewline() bool {
	return boolValue(nil, c.Editor.InsertFinalNewline, true)
}

func (c *Config) TrimFinalNewlines() bool {
	return boolValue(nil, c.Editor.TrimFinalNewlines, false)
}

func (c *Config) TrimTrailingWhitespace() bool {
	return boolValue(nil, c.Editor.TrimTrailingWS, false)
}

func (c *Config) AutoSaveFocusLost() bool {
	return boolValue(nil, c.Editor.AutoSave.FocusLost, false)
}

func (c *Config) AutoSaveAfterDelay() bool {
	return boolValue(nil, c.Editor.AutoSave.AfterDelay.Enable, false)
}

func (c *Config) AutoSaveDelayTimeout() int {
	return intValue(nil, c.Editor.AutoSave.AfterDelay.Timeout,
		DefaultAutoSaveDelay,
	)
}

func (c *Config) AtomicSave() bool {
	return boolValue(nil, c.Editor.AtomicSave, true)
}

func (c *Config) Insecure() bool {
	return boolValue(nil, c.Editor.Insecure, false)
}

func (c *Config) EditorConfig() bool {
	return boolValue(nil, c.Editor.EditorConfig, true)
}

func (c *Config) ContinueComments() bool {
	return boolValue(nil, c.Editor.ContinueComments, true)
}

func (c *Config) Scrolloff() int {
	return intValue(nil, c.Editor.Scrolloff, DefaultScrolloff)
}

func (c *Config) ScrollLines() int {
	return intValue(nil, c.Editor.ScrollLines, DefaultScrollLines)
}

func (c *Config) Cursorline() bool {
	return boolValue(nil, c.Editor.Cursorline, false)
}

func (c *Config) Cursorcolumn() bool {
	return boolValue(nil, c.Editor.Cursorcolumn, false)
}

func (c *Config) Mouse() bool {
	return boolValue(nil, c.Editor.Mouse, true)
}

func (c *Config) MiddleClickPaste() bool {
	return boolValue(nil, c.Editor.MiddleClickPaste, true)
}

func (c *Config) GetBufferLine() BufferLine {
	if c.Editor.BufferLine == "" {
		return BufferLineNever
	}
	return c.Editor.BufferLine
}

func (c *Config) Whitespace() WhitespaceConfig {
	return c.Editor.Whitespace
}

func (c *Config) IndentGuides() IndentGuidesConfig {
	return c.Editor.IndentGuides
}

func (i IndentGuidesConfig) CharRune() rune {
	return runeOrDefault(i.Character, DefaultIndentGuideChar)
}

func (i IndentGuidesConfig) GetSkipLevels() int {
	if i.SkipLevels != nil {
		return *i.SkipLevels
	}
	return 0
}

func (c *Config) Gutters() GutterConfig {
	return c.Editor.Gutters
}

func (g *GutterConfig) GutterLayout() []GutterType {
	if g.Present {
		return g.Layout
	}
	return []GutterType{
		GutterTypeDiagnostics,
		GutterTypeSpacer,
		GutterTypeLineNumbers,
		GutterTypeSpacer,
		GutterTypeDiff,
	}
}

func (g *GutterConfig) HasGutterType(gt GutterType) bool {
	return slices.Contains(g.GutterLayout(), gt)
}

func (g *GutterConfig) LineNumberMinWidth() int {
	return intValue(nil, g.LineNumbers.MinWidth, DefaultGutterLineNumberMinWidth)
}

func (c *Config) Shell() []string {
	if len(c.Editor.Shell) > 0 {
		return c.Editor.Shell
	}
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/C"}
	}
	return []string{"sh", "-c"}
}

func (c *Config) Rulers() []int {
	return c.Editor.Rulers
}

func (c *Config) LineNumber() LineNumber {
	if c.Editor.LineNumber == "" {
		return LineNumberAbsolute
	}
	return c.Editor.LineNumber
}

func (c *Config) CursorShapeForMode(mode string) CursorKind {
	switch strings.ToLower(mode) {
	case "ins", "insert":
		return cursorKindOrDefault(c.Editor.CursorShape.Insert)
	case "sel", "select":
		return cursorKindOrDefault(c.Editor.CursorShape.Select)
	default:
		return cursorKindOrDefault(c.Editor.CursorShape.Normal)
	}
}

func (c *Config) ModeNameForMode(mode string) string {
	switch strings.ToLower(mode) {
	case "ins", "insert":
		return modeNameOrDefault(c.Editor.StatusLine.Mode.Insert, "INS")
	case "sel", "select":
		return modeNameOrDefault(c.Editor.StatusLine.Mode.Select, "SEL")
	default:
		return modeNameOrDefault(c.Editor.StatusLine.Mode.Normal, "NOR")
	}
}

func (c *Config) StatusLineSeparator() string {
	return modeNameOrDefault(c.Editor.StatusLine.Separator, "│")
}

func (c *Config) StatusLineLeft() []StatusLineElement {
	if len(c.Editor.StatusLine.Left) > 0 {
		return slices.Clone(c.Editor.StatusLine.Left)
	}
	return []StatusLineElement{
		StatusLineMode,
		StatusLineSpinner,
		StatusLineFileName,
		StatusLineReadOnly,
		StatusLineModified,
	}
}

func (c *Config) StatusLineCenter() []StatusLineElement {
	return slices.Clone(c.Editor.StatusLine.Center)
}

func (c *Config) StatusLineRight() []StatusLineElement {
	if len(c.Editor.StatusLine.Right) > 0 {
		return slices.Clone(c.Editor.StatusLine.Right)
	}
	return []StatusLineElement{
		StatusLineDiagnostics,
		StatusLineSelections,
		StatusLineRegister,
		StatusLinePosition,
		StatusLineFileEncoding,
	}
}

func (c *Config) SearchSmartCase() bool {
	return boolValue(nil, c.Editor.Search.SmartCase, true)
}

func (c *Config) SearchWrapAround() bool {
	return boolValue(nil, c.Editor.Search.WrapAround, true)
}

func (c *Config) AutoPairs() (core.AutoPairs, bool) {
	return c.Editor.AutoPairs.OrDefault()
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

func (a *AutoSave) UnmarshalTOML(value any) error {
	cfg, ok := decodeAutoSave(value)
	if !ok {
		return fmt.Errorf("%w: %v", ErrInvalidOption, value)
	}
	*a = cfg
	return nil
}

func (t Theme) Choose(light bool) string {
	if !t.Adaptive {
		return t.Name
	}
	if light {
		return t.Light
	}
	if t.Dark != "" {
		return t.Dark
	}
	return t.Fallback
}

func (c *CursorKind) UnmarshalText(text []byte) error {
	switch CursorKind(text) {
	case CursorKindBlock, CursorKindBar, CursorKindUnderline, CursorKindHidden:
		*c = CursorKind(text)
	default:
		return fmt.Errorf("%w: %s", ErrInvalidCursorKind, text)
	}
	return nil
}

func (l *LineNumber) UnmarshalText(text []byte) error {
	switch LineNumber(text) {
	case LineNumberAbsolute, LineNumberRelative:
		*l = LineNumber(text)
	default:
		return fmt.Errorf("%w: %s", ErrInvalidLineNumber, text)
	}
	return nil
}

func (s *StatusLineElement) UnmarshalText(text []byte) error {
	switch StatusLineElement(text) {
	case StatusLineMode, StatusLineSpinner, StatusLineFileBaseName,
		StatusLineFileName, StatusLineFileAbsolutePath,
		StatusLineModified, StatusLineReadOnly,
		StatusLineFileEncoding, StatusLineFileLineEnding,
		StatusLineFileIndentStyle, StatusLineFileType, StatusLineDiagnostics,
		StatusLineWorkspaceDiag, StatusLineSelections,
		StatusLinePrimaryLen, StatusLinePosition,
		StatusLineSeparator, StatusLinePercent,
		StatusLineTotalLines, StatusLineSpacer, StatusLineVersionControl,
		StatusLineRegister:
		*s = StatusLineElement(text)
	default:
		return fmt.Errorf("%w: %s", ErrInvalidStatusLine, text)
	}
	return nil
}

func (b *BufferLine) UnmarshalText(text []byte) error {
	switch BufferLine(text) {
	case BufferLineNever, BufferLineAlways, BufferLineMultiple:
		*b = BufferLine(text)
	default:
		return fmt.Errorf("%w: %s", ErrInvalidBufferLine, text)
	}
	return nil
}

func (g *GutterType) UnmarshalText(text []byte) error {
	switch GutterType(text) {
	case GutterTypeDiagnostics, GutterTypeLineNumbers,
		GutterTypeSpacer, GutterTypeDiff:
		*g = GutterType(text)
	default:
		return fmt.Errorf("%w: %s", ErrInvalidGutterType, text)
	}
	return nil
}

func (g *GutterConfig) UnmarshalTOML(value any) error {
	switch v := value.(type) {
	case []any:
		layout, err := decodeGutterLayout(v)
		if err != nil {
			return err
		}
		g.Present = true
		g.Layout = layout
	case map[string]any:
		g.Present = true
		if raw, ok := v["layout"]; ok {
			items, ok2 := raw.([]any)
			if !ok2 {
				return fmt.Errorf("%w: layout must be an array", ErrInvalidGutterType)
			}
			layout, err := decodeGutterLayout(items)
			if err != nil {
				return err
			}
			g.Layout = layout
		}
		if raw, ok := v["line-numbers"].(map[string]any); ok {
			g.LineNumbers.MinWidth = intPtrOrNil(raw["min-width"])
		}
	case nil:
		// leave zero value
	default:
		return fmt.Errorf("%w: %v", ErrInvalidGutterType, value)
	}
	return nil
}

func (w *WhitespaceRender) SpaceRender() WhitespaceRenderValue {
	return whitespaceRenderFor(w.Space, w.Default)
}

func (w *WhitespaceRender) NbspRender() WhitespaceRenderValue {
	return whitespaceRenderFor(w.Nbsp, w.Default)
}

func (w *WhitespaceRender) NnbspRender() WhitespaceRenderValue {
	return whitespaceRenderFor(w.Nnbsp, w.Default)
}

func (w *WhitespaceRender) TabRender() WhitespaceRenderValue {
	return whitespaceRenderFor(w.Tab, w.Default)
}

func (w *WhitespaceRender) NewlineRender() WhitespaceRenderValue {
	return whitespaceRenderFor(w.Newline, w.Default)
}

func (w *WhitespaceRender) UnmarshalTOML(value any) error {
	switch v := value.(type) {
	case string:
		rv, err := parseWhitespaceRenderValue(v)
		if err != nil {
			return err
		}
		w.Default = &rv
	case map[string]any:
		for key, raw := range v {
			s, ok := raw.(string)
			if !ok {
				return fmt.Errorf(
					"%w: %v", ErrInvalidWhitespaceRender, raw,
				)
			}
			rv, err := parseWhitespaceRenderValue(s)
			if err != nil {
				return err
			}
			switch key {
			case "default":
				w.Default = &rv
			case "space":
				w.Space = &rv
			case "nbsp":
				w.Nbsp = &rv
			case "nnbsp":
				w.Nnbsp = &rv
			case "tab":
				w.Tab = &rv
			case "newline":
				w.Newline = &rv
			}
		}
	case nil:
		// leave zero value
	default:
		return fmt.Errorf("%w: %v", ErrInvalidWhitespaceRender, value)
	}
	return nil
}

func (w *WhitespaceCharacters) SpaceRune() rune {
	return runeOrDefault(w.Space, DefaultWSSpace)
}

func (w *WhitespaceCharacters) NbspRune() rune {
	return runeOrDefault(w.Nbsp, DefaultWSNbsp)
}

func (w *WhitespaceCharacters) NnbspRune() rune {
	return runeOrDefault(w.Nnbsp, DefaultWSNnbsp)
}

func (w *WhitespaceCharacters) TabRune() rune {
	return runeOrDefault(w.Tab, DefaultWSTab)
}

func (w *WhitespaceCharacters) TabpadRune() rune {
	return runeOrDefault(w.Tabpad, DefaultWSTabpad)
}

func (w *WhitespaceCharacters) NewlineRune() rune {
	return runeOrDefault(w.Newline, DefaultWSNewline)
}

func cursorKindOrDefault(c CursorKind) CursorKind {
	if c == "" {
		return CursorKindBlock
	}
	return c
}

func boolValue(lang, editor *bool, fallback bool) bool {
	if lang != nil {
		return *lang
	}
	if editor != nil {
		return *editor
	}
	return fallback
}

func intValue(lang, editor *int, fallback int) int {
	if lang != nil {
		return *lang
	}
	if editor != nil {
		return *editor
	}
	return fallback
}

func stringValue(lang, editor *string, fallback string) string {
	if lang != nil {
		return *lang
	}
	if editor != nil {
		return *editor
	}
	return fallback
}

func modeNameOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func decodeConfigMap(m map[string]any) (*Config, bool) {
	var b bytes.Buffer
	if err := toml.NewEncoder(&b).Encode(m); err != nil {
		return nil, false
	}
	var raw struct {
		Editor Editor `toml:"editor"`
	}
	if _, err := toml.Decode(b.String(), &raw); err != nil {
		return nil, false
	}
	return &Config{
		Theme:  decodeTheme(m["theme"]),
		Editor: raw.Editor,
	}, true
}

func decodeTheme(value any) Theme {
	switch v := value.(type) {
	case string:
		return Theme{Name: v}
	case map[string]any:
		return Theme{
			Light:    stringValueFromMap(v, "light"),
			Dark:     stringValueFromMap(v, "dark"),
			Fallback: stringValueFromMap(v, "fallback"),
			Adaptive: true,
		}
	default:
		return Theme{Name: "mocha"}
	}
}

func decodeLanguagesMap(m map[string]any) (Languages, bool) {
	values, ok := languageValues(m["language"])
	if !ok {
		return Languages{}, false
	}
	langs := Languages{
		Languages:        make([]Language, 0, len(values)),
		LanguageServers:  decodeLanguageServers(m["language-server"]),
		GrammarSelection: decodeGrammarSelection(m["use-grammars"]),
		Grammars:         decodeGrammars(m["grammar"]),
	}
	for _, value := range values {
		l, ok := decodeLanguage(value)
		if !ok {
			return Languages{}, false
		}
		langs.Languages = append(langs.Languages, l)
	}
	return langs, true
}

func decodeGrammarSelection(value any) GrammarSelection {
	m, ok := value.(map[string]any)
	if !ok {
		return GrammarSelection{}
	}
	return GrammarSelection{
		Only:   decodeStringSlice(m["only"]),
		Except: decodeStringSlice(m["except"]),
	}
}

func decodeGrammars(value any) []Grammar {
	values, ok := anySlice(value)
	if !ok {
		return nil
	}
	out := make([]Grammar, 0, len(values))
	for _, value := range values {
		m, ok := value.(map[string]any)
		if !ok {
			continue
		}
		source, ok := decodeGrammarSource(m["source"])
		if !ok {
			continue
		}
		out = append(out, Grammar{
			Name:   stringValueFromMap(m, "name"),
			Source: source,
		})
	}
	return out
}

func decodeGrammarSource(value any) (GrammarSource, bool) {
	m, ok := value.(map[string]any)
	if !ok {
		return GrammarSource{}, false
	}
	if path, ok := m["path"].(string); ok {
		return GrammarSource{Path: path}, true
	}
	git, ok := m["git"].(string)
	if !ok {
		return GrammarSource{}, false
	}
	return GrammarSource{
		Git:     git,
		Rev:     stringValueFromMap(m, "rev"),
		Subpath: stringValueFromMap(m, "subpath"),
	}, true
}

func languageValues(value any) ([]map[string]any, bool) {
	switch values := value.(type) {
	case []map[string]any:
		return values, true
	case []any:
		out := make([]map[string]any, 0, len(values))
		for _, value := range values {
			m, ok := value.(map[string]any)
			if !ok {
				return nil, false
			}
			out = append(out, m)
		}
		return out, true
	default:
		return nil, false
	}
}

func decodeLanguage(m map[string]any) (Language, bool) {
	var l Language
	if name, ok := m["name"].(string); ok {
		l.Name = name
	}
	if id, ok := m["language-id"].(string); ok {
		l.LanguageID = id
	}
	if scope, ok := m["scope"].(string); ok {
		l.Scope = scope
	}
	if injection, ok := m["injection-regex"].(string); ok {
		l.InjectionRegex = injection
	}
	if n, ok := intPtr(m["text-width"]); ok {
		l.TextWidth = n
	}
	l.FileTypes = decodeFileTypes(m["file-types"])
	l.Shebangs = decodeStringSlice(m["shebangs"])
	l.Roots = decodeStringSlice(m["roots"])
	l.LanguageServers = decodeLanguageServerFeatures(m["language-servers"])
	l.CommentTokens = decodeCommentTokens(m)
	l.BlockCommentTokens = decodeBlockCommentTokens(m["block-comment-tokens"])
	if indent, ok := m["indent"].(map[string]any); ok {
		l.Indent = decodeIndent(indent)
	}
	if pairs, ok := decodeAutoPairConfig(m["auto-pairs"]); ok {
		l.AutoPairs = pairs
	}
	l.AutoFormat = boolPtr(m["auto-format"])
	if formatter, ok := decodeFormatter(m["formatter"]); ok {
		l.Formatter = &formatter
	}
	if debug, ok := decodeDebugAdapter(m["debugger"]); ok {
		l.Debugger = &debug
	}
	if soft, ok := m["soft-wrap"].(map[string]any); ok {
		l.SoftWrap = decodeSoftWrap(soft)
	}
	return l, l.Name != ""
}

func decodeLanguageServers(value any) map[string]LanguageServer {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]LanguageServer, len(m))
	for name, value := range m {
		cfg, ok := decodeLanguageServer(value)
		if ok {
			out[name] = cfg
		}
	}
	return out
}

func decodeLanguageServer(value any) (LanguageServer, bool) {
	m, ok := value.(map[string]any)
	if !ok {
		return LanguageServer{}, false
	}
	cmd, ok := m["command"].(string)
	if !ok {
		return LanguageServer{}, false
	}
	return LanguageServer{
		Command:              cmd,
		Args:                 decodeStringSlice(m["args"]),
		Environment:          decodeStringMap(m["environment"]),
		Config:               decodeAnyMap(m["config"]),
		Timeout:              intValueFromMap(m, "timeout", 20),
		RequiredRootPatterns: decodeStringSlice(m["required-root-patterns"]),
	}, true
}

func decodeLanguageServerFeatures(value any) []LanguageServerFeatures {
	values, ok := anySlice(value)
	if !ok {
		return nil
	}
	out := make([]LanguageServerFeatures, 0, len(values))
	for _, value := range values {
		if features, ok := decodeLanguageServerFeature(value); ok {
			out = append(out, features)
		}
	}
	return out
}

func decodeLanguageServerFeature(
	value any,
) (LanguageServerFeatures, bool) {
	switch v := value.(type) {
	case string:
		return LanguageServerFeatures{Name: v}, true
	case map[string]any:
		name, ok := v["name"].(string)
		if !ok {
			return LanguageServerFeatures{}, false
		}
		return LanguageServerFeatures{
			Name:     name,
			Only:     decodeLanguageServerFeatureNames(v["only-features"]),
			Excluded: decodeLanguageServerFeatureNames(v["except-features"]),
		}, true
	default:
		return LanguageServerFeatures{}, false
	}
}

func decodeLanguageServerFeatureNames(value any) []LanguageServerFeature {
	names := decodeStringSlice(value)
	out := make([]LanguageServerFeature, len(names))
	for i, name := range names {
		out[i] = LanguageServerFeature(name)
	}
	return out
}

func decodeFormatter(value any) (Formatter, bool) {
	m, ok := value.(map[string]any)
	if !ok {
		return Formatter{}, false
	}
	cmd, ok := m["command"].(string)
	if !ok {
		return Formatter{}, false
	}
	return Formatter{
		Command: cmd,
		Args:    decodeStringSlice(m["args"]),
	}, true
}

func decodeDebugAdapter(value any) (DebugAdapter, bool) {
	m, ok := value.(map[string]any)
	if !ok {
		return DebugAdapter{}, false
	}
	name, ok := m["name"].(string)
	if !ok {
		return DebugAdapter{}, false
	}
	transport, ok := m["transport"].(string)
	if !ok {
		return DebugAdapter{}, false
	}
	return DebugAdapter{
		Name:      name,
		Transport: transport,
		Command:   stringValueFromMap(m, "command"),
		Args:      decodeStringSlice(m["args"]),
		PortArg:   stringValueFromMap(m, "port-arg"),
		Templates: decodeDebugTemplates(m["templates"]),
		Quirks:    decodeDebuggerQuirks(m["quirks"]),
	}, true
}

func decodeDebugTemplates(value any) []DebugTemplate {
	values, ok := anySlice(value)
	if !ok {
		return nil
	}
	out := make([]DebugTemplate, 0, len(values))
	for _, value := range values {
		m, ok := value.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, DebugTemplate{
			Name:       stringValueFromMap(m, "name"),
			Request:    stringValueFromMap(m, "request"),
			Completion: decodeDebugCompletions(m["completion"]),
			Args:       decodeAnyMap(m["args"]),
		})
	}
	return out
}

func decodeDebugCompletions(value any) []DebugCompletion {
	values, ok := anySlice(value)
	if !ok {
		return nil
	}
	out := make([]DebugCompletion, 0, len(values))
	for _, value := range values {
		switch v := value.(type) {
		case string:
			out = append(out, DebugCompletion{Name: v})
		case map[string]any:
			out = append(out, DebugCompletion{
				Name:       stringValueFromMap(v, "name"),
				Completion: stringValueFromMap(v, "completion"),
				Default:    stringValueFromMap(v, "default"),
			})
		}
	}
	return out
}

func decodeDebuggerQuirks(value any) DebuggerQuirks {
	m, ok := value.(map[string]any)
	if !ok {
		return DebuggerQuirks{}
	}
	return DebuggerQuirks{AbsolutePaths: boolValueFromMap(m, "absolute-paths")}
}

func decodeAutoPairConfig(value any) (AutoPairConfig, bool) {
	switch v := value.(type) {
	case nil:
		return AutoPairConfig{}, false
	case bool:
		return AutoPairConfig{Present: true, Enable: &v}, true
	case map[string]any:
		return decodeAutoPairMap(v)
	case map[string]string:
		m := make(map[string]any, len(v))
		for k, value := range v {
			m[k] = value
		}
		return decodeAutoPairMap(m)
	default:
		return AutoPairConfig{}, false
	}
}

func decodeAutoPairMap(m map[string]any) (AutoPairConfig, bool) {
	pairs := make([][2]rune, 0, len(m))
	for k, value := range m {
		v, ok := value.(string)
		if !ok {
			return AutoPairConfig{}, false
		}
		openRunes := []rune(k)
		closeRunes := []rune(v)
		if len(openRunes) != 1 || len(closeRunes) != 1 {
			return AutoPairConfig{}, false
		}
		pairs = append(pairs, [2]rune{openRunes[0], closeRunes[0]})
	}
	return AutoPairConfig{Present: true, Pairs: pairs}, true
}

func decodeAutoSave(value any) (AutoSave, bool) {
	switch v := value.(type) {
	case nil:
		return AutoSave{}, false
	case bool:
		return AutoSave{FocusLost: &v}, true
	case map[string]any:
		return AutoSave{
			FocusLost:  boolPtr(v["focus-lost"]),
			AfterDelay: decodeAutoSaveAfterDelay(v["after-delay"]),
		}, true
	default:
		return AutoSave{}, false
	}
}

func decodeAutoSaveAfterDelay(value any) AutoSaveAfterDelay {
	m, ok := value.(map[string]any)
	if !ok {
		return AutoSaveAfterDelay{}
	}
	return AutoSaveAfterDelay{
		Enable:  boolPtr(m["enable"]),
		Timeout: intPtrOrNil(m["timeout"]),
	}
}

func decodeIndent(m map[string]any) Indent {
	return Indent{
		TabWidth: intPtrOrNil(m["tab-width"]),
		Unit:     stringValueFromMap(m, "unit"),
	}
}

func decodeCommentTokens(m map[string]any) []string {
	if tokens := decodeStringOrSlice(m["comment-tokens"]); len(tokens) > 0 {
		return tokens
	}
	return decodeStringOrSlice(m["comment-token"])
}

func decodeStringOrSlice(value any) []string {
	if s, ok := value.(string); ok {
		return []string{s}
	}
	return decodeStringSlice(value)
}

func decodeBlockCommentTokens(value any) []core.BlockCommentToken {
	if token, ok := decodeBlockCommentToken(value); ok {
		return []core.BlockCommentToken{token}
	}
	values, ok := anySlice(value)
	if !ok {
		return nil
	}
	out := make([]core.BlockCommentToken, 0, len(values))
	for _, value := range values {
		if token, ok := decodeBlockCommentToken(value); ok {
			out = append(out, token)
		}
	}
	return out
}

func decodeBlockCommentToken(value any) (core.BlockCommentToken, bool) {
	m, ok := value.(map[string]any)
	if !ok {
		return core.BlockCommentToken{}, false
	}
	start, ok := m["start"].(string)
	if !ok {
		return core.BlockCommentToken{}, false
	}
	end, ok := m["end"].(string)
	if !ok {
		return core.BlockCommentToken{}, false
	}
	return core.BlockCommentToken{Start: start, End: end}, true
}

func decodeFileTypes(value any) []FileType {
	values, ok := anySlice(value)
	if !ok {
		return nil
	}
	out := make([]FileType, 0, len(values))
	for _, value := range values {
		switch v := value.(type) {
		case string:
			out = append(out, FileType{Extension: v})
		case map[string]any:
			if glob, ok := v["glob"].(string); ok {
				out = append(out, FileType{Glob: normalizeGlob(glob)})
			}
		}
	}
	return out
}

func decodeStringSlice(value any) []string {
	values, ok := anySlice(value)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if s, ok := value.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func anySlice(value any) ([]any, bool) {
	switch values := value.(type) {
	case []any:
		return values, true
	case []map[string]any:
		out := make([]any, len(values))
		for i, value := range values {
			out[i] = value
		}
		return out, true
	case []string:
		out := make([]any, len(values))
		for i, value := range values {
			out[i] = value
		}
		return out, true
	default:
		return nil, false
	}
}

func decodeStringMap(value any) map[string]string {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, value := range m {
		if s, ok := value.(string); ok {
			out[k] = s
		}
	}
	return out
}

func decodeAnyMap(value any) map[string]any {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return m
}

func normalizeGlob(glob string) string {
	if filepath.IsAbs(glob) || strings.HasPrefix(glob, "*/") {
		return glob
	}
	return "*/" + glob
}

func languageForFilename(langs Languages, path string) (Language, bool) {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	var found Language
	foundLen := -1
	for _, lang := range langs.Languages {
		for _, ft := range lang.FileTypes {
			if ft.Glob != "" && globMatch(ft.Glob, abs) {
				if n := len(ft.Glob); n > foundLen {
					found = lang
					foundLen = n
				}
			}
		}
	}
	if foundLen >= 0 {
		return found, true
	}
	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	name := filepath.Base(path)
	for _, lang := range langs.Languages {
		for _, ft := range lang.FileTypes {
			if ext != "" && ft.Extension == ext {
				return lang, true
			}
			if ft.Extension == name {
				return lang, true
			}
		}
	}
	return Language{}, false
}

func languageForMatch(langs Languages, text string) (string, bool) {
	for _, lang := range langs.Languages {
		if lang.Name == text {
			return lang.Name, true
		}
	}
	bestLen := 0
	var found string
	for _, lang := range langs.Languages {
		if lang.InjectionRegex == "" {
			continue
		}
		re, err := regexp.Compile(lang.InjectionRegex)
		if err != nil {
			continue
		}
		loc := re.FindStringIndex(text)
		if loc == nil {
			continue
		}
		if n := loc[1] - loc[0]; n > bestLen {
			bestLen = n
			found = lang.Name
		}
	}
	return found, found != ""
}

func languageForShebang(langs Languages, content string) (string, bool) {
	line := content
	if before, _, ok := strings.Cut(content, "\n"); ok {
		line = before
	}
	if !strings.HasPrefix(line, "#!") {
		return "", false
	}
	fields := strings.Fields(strings.TrimPrefix(line, "#!"))
	if len(fields) == 0 {
		return "", false
	}
	marker := filepath.Base(fields[0])
	if marker == "env" && len(fields) > 1 {
		marker = fields[len(fields)-1]
	}
	for _, lang := range langs.Languages {
		if slices.Contains(lang.Shebangs, marker) {
			return lang.Name, true
		}
	}
	return "", false
}

func languageRootMarkerExists(dir string, roots []string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		for _, root := range roots {
			if rootMarkerMatches(root, entry.Name()) {
				return true
			}
		}
	}
	return false
}

func rootMarkerMatches(pattern, name string) bool {
	return globMatch(pattern, name) || pattern == name
}

func globMatch(pattern, path string) bool {
	for _, pattern := range expandGlobBraces(pattern) {
		if ok := pathGlobMatch(pattern, filepath.ToSlash(path)); ok {
			return true
		}
		if pathGlobMatch(pattern, path) {
			return true
		}
	}
	return false
}

func pathGlobMatch(pattern, path string) bool {
	parts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")
	if strings.HasPrefix(pattern, "*/") {
		for i := range len(pathParts) {
			if pathGlobParts(parts, pathParts[i:]) {
				return true
			}
		}
	}
	return pathGlobParts(parts, pathParts)
}

func pathGlobParts(pattern, path []string) bool {
	for len(pattern) > 0 {
		part := pattern[0]
		pattern = pattern[1:]
		if part == "**" {
			if len(pattern) == 0 {
				return true
			}
			for i := range len(path) + 1 {
				if pathGlobParts(pattern, path[i:]) {
					return true
				}
			}
			return false
		}
		if len(path) == 0 {
			return false
		}
		if ok, err := filepath.Match(part, path[0]); err != nil || !ok {
			return false
		}
		path = path[1:]
	}
	return len(path) == 0
}

func expandGlobBraces(pattern string) []string {
	start := strings.Index(pattern, "{")
	if start < 0 {
		return []string{pattern}
	}
	end := strings.Index(pattern[start:], "}")
	if end < 0 {
		return []string{pattern}
	}
	end += start
	pfx := pattern[:start]
	sfx := pattern[end+1:]
	alts := strings.Split(pattern[start+1:end], ",")
	out := make([]string, 0, len(alts))
	for _, alt := range alts {
		out = append(out, expandGlobBraces(pfx+alt+sfx)...)
	}
	return out
}

func decodeSoftWrap(m map[string]any) SoftWrap {
	return SoftWrap{
		Enable:          boolPtr(m["enable"]),
		MaxWrap:         intPtrOrNil(m["max-wrap"]),
		MaxIndentRetain: intPtrOrNil(m["max-indent-retain"]),
		WrapIndicator:   stringPtr(m["wrap-indicator"]),
		WrapAtTextWidth: boolPtr(m["wrap-at-text-width"]),
	}
}

func boolPtr(value any) *bool {
	v, ok := value.(bool)
	if !ok {
		return nil
	}
	return &v
}

func intPtr(value any) (*int, bool) {
	switch v := value.(type) {
	case int:
		return &v, true
	case int64:
		return new(int(v)), true
	default:
		return nil, false
	}
}

func intPtrOrNil(value any) *int {
	v, _ := intPtr(value)
	return v
}

func stringPtr(value any) *string {
	v, ok := value.(string)
	if !ok {
		return nil
	}
	return &v
}

func stringValueFromMap(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func intValueFromMap(m map[string]any, key string, fallback int) int {
	if n, ok := intPtr(m[key]); ok {
		return *n
	}
	return fallback
}

func boolValueFromMap(m map[string]any, key string) bool {
	v, _ := m[key].(bool)
	return v
}

func decodeGutterLayout(items []any) ([]GutterType, error) {
	layout := make([]GutterType, 0, len(items))
	for _, item := range items {
		s, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("%w: %v", ErrInvalidGutterType, item)
		}
		var gt GutterType
		if err := gt.UnmarshalText([]byte(s)); err != nil {
			return nil, err
		}
		layout = append(layout, gt)
	}
	return layout, nil
}

func whitespaceRenderFor(
	specific, def *WhitespaceRenderValue,
) WhitespaceRenderValue {
	if specific != nil {
		return *specific
	}
	if def != nil {
		return *def
	}
	return WhitespaceRenderNone
}

func parseWhitespaceRenderValue(s string) (WhitespaceRenderValue, error) {
	switch WhitespaceRenderValue(s) {
	case WhitespaceRenderNone, WhitespaceRenderAll:
		return WhitespaceRenderValue(s), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidWhitespaceRender, s)
	}
}

func runeOrDefault(s string, fallback rune) rune {
	if s == "" {
		return fallback
	}
	for _, r := range s {
		return r
	}
	return fallback
}
