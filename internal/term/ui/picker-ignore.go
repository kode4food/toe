package ui

import (
	"bufio"
	"os"
	"path/filepath"
	"slices"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"

	"github.com/kode4food/toe/internal/loader"
)

type (
	PickerIgnore struct {
		base    string
		dir     string
		matcher *gitignore.GitIgnore
	}

	PickerIgnoreOptions struct {
		Hidden      bool
		Parents     bool
		IgnoreFiles bool
	}

	// IgnoreTarget is the walk root and the path being tested against the
	// ignore rules that apply to it
	IgnoreTarget struct {
		Root string
		Path string
	}

	// ignoreCandidate is a path under test, both workspace-relative and full
	ignoreCandidate struct {
		rel  string
		full string
	}

	// ignoreFile is a rules file and the directory its patterns are anchored
	// to
	ignoreFile struct {
		dir  string
		path string
	}
)

var excludedPickerTypes = map[string]struct{}{
	".zip":  {},
	".gz":   {},
	".bz2":  {},
	".zst":  {},
	".lzo":  {},
	".sz":   {},
	".tgz":  {},
	".tbz2": {},
	".lz":   {},
	".lz4":  {},
	".lzma": {},
	".z":    {},
	".xz":   {},
	".7z":   {},
	".rar":  {},
	".cab":  {},
}

// SkipPickerPathArgs holds the entry a picker file-walk is considering and
// the ignore state to test it against
type SkipPickerPathArgs struct {
	Rel     string
	Path    string
	Entry   os.DirEntry
	Ignores []PickerIgnore
	Opts    PickerIgnoreOptions
}

// SkipPickerPath reports whether a walked entry should be excluded from a
// picker's file listing under the given ignore rules
func SkipPickerPath(args SkipPickerPathArgs) bool {
	name := args.Entry.Name()
	if args.Opts.Hidden && strings.HasPrefix(name, ".") {
		return true
	}
	if isExcludedPickerType(name) {
		return true
	}
	for _, ig := range args.Ignores {
		at := ignoreCandidate{rel: args.Rel, full: args.Path}
		if sub, ok := ignorePathForBase(at, ig); ok &&
			ig.matcher.MatchesPath(sub) {
			return true
		}
	}
	return false
}

// LoadIgnoreFiles collects .ignore, .toe/ignore, and .gitignore rules that
// apply to path, nearest directory last
func LoadIgnoreFiles(
	target IgnoreTarget, opts PickerIgnoreOptions,
) []PickerIgnore {
	if !opts.IgnoreFiles {
		return nil
	}
	var ignores []PickerIgnore
	for _, dir := range ignoreDirs(target, opts.Parents) {
		for _, name := range []string{
			".ignore",
			filepath.Join(loader.WorkspaceDirName, "ignore"),
			".gitignore",
		} {
			ignores = appendIgnorePath(ignores, ignoreFile{
				dir:  dir,
				path: filepath.Join(dir, name),
			})
		}
	}
	root := target.Root
	ignores = appendIgnorePath(ignores, ignoreFile{
		dir:  root,
		path: filepath.Join(root, ".git", "info", "exclude"),
	})
	ignores = appendIgnorePath(ignores, ignoreFile{
		dir:  root,
		path: gitGlobalIgnorePath(),
	})
	ignores = appendIgnorePath(ignores, ignoreFile{
		path: loader.ConfigIgnoreFile(),
	})
	return ignores
}

// DefaultPickerIgnoreOptions is the ignore behavior file-walking pickers use
// when a caller does not need to customize it
func DefaultPickerIgnoreOptions() PickerIgnoreOptions {
	return PickerIgnoreOptions{
		Hidden: true, Parents: true, IgnoreFiles: true,
	}
}

func ignorePathForBase(at ignoreCandidate, ig PickerIgnore) (string, bool) {
	if ig.dir != "" {
		sub, err := filepath.Rel(ig.dir, at.full)
		parent := ".." + string(filepath.Separator)
		if err != nil || strings.HasPrefix(sub, parent) {
			return "", false
		}
		if sub == ".." {
			return "", false
		}
		return filepath.ToSlash(sub), true
	}
	if ig.base == "" {
		return at.rel, true
	}
	if at.rel == ig.base {
		return "", true
	}
	sub, ok := strings.CutPrefix(at.rel, ig.base+"/")
	return sub, ok
}

func appendIgnorePath(
	ignores []PickerIgnore, f ignoreFile,
) []PickerIgnore {
	if ig := compileIgnore(f.path); ig != nil {
		return append(ignores, PickerIgnore{dir: f.dir, matcher: ig})
	}
	return ignores
}

func ignoreDirs(target IgnoreTarget, parents bool) []string {
	root := filepath.Clean(target.Root)
	dir := filepath.Dir(target.Path)
	if !parents {
		return []string{root}
	}
	var dirs []string
	for p := dir; ; p = filepath.Dir(p) {
		dirs = append(dirs, p)
		if p == filepath.Dir(p) {
			break
		}
	}
	slices.Reverse(dirs)
	return dirs
}

func compileIgnore(path string) *gitignore.GitIgnore {
	ig, err := gitignore.CompileIgnoreFile(path)
	if err != nil {
		return nil
	}
	return ig
}

func gitGlobalIgnorePath() string {
	for _, path := range gitConfigPaths() {
		if found := readGitExcludesFile(path); found != "" {
			return loader.ExpandUserPath(found)
		}
	}
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "git", "ignore")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "git", "ignore")
	}
	return ""
}

func gitConfigPaths() []string {
	if path := os.Getenv("GIT_CONFIG_GLOBAL"); path != "" {
		return []string{path}
	}
	var paths []string
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".gitconfig"))
	}
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		paths = append(paths, filepath.Join(dir, "git", "config"))
	}
	return paths
}

func readGitExcludesFile(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()
	inCore := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			inCore = strings.EqualFold(line, "[core]")
			continue
		}
		if !inCore ||
			strings.HasPrefix(line, "#") ||
			strings.HasPrefix(line, ";") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if ok && strings.EqualFold(strings.TrimSpace(key), "excludesfile") {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func isExcludedPickerType(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	_, ok := excludedPickerTypes[ext]
	return ok
}
