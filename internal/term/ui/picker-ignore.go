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
		base string
		dir  string
		ig   *gitignore.GitIgnore
	}

	PickerIgnoreOptions struct {
		Hidden     bool
		Parents    bool
		Ignore     bool
		GitIgnore  bool
		GitGlobal  bool
		GitExclude bool
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
	if excludedPickerType(name) {
		return true
	}
	for _, ig := range args.Ignores {
		sub, ok := ignorePathForBase(args.Rel, args.Path, ig)
		if ok && ig.ig.MatchesPath(sub) {
			return true
		}
	}
	return false
}

func ignorePathForBase(rel, path string, ig PickerIgnore) (string, bool) {
	if ig.dir != "" {
		sub, err := filepath.Rel(ig.dir, path)
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
		return rel, true
	}
	if rel == ig.base {
		return "", true
	}
	sub, ok := strings.CutPrefix(rel, ig.base+"/")
	return sub, ok
}

func LoadIgnoreFiles(
	root, path string, opts PickerIgnoreOptions,
) []PickerIgnore {
	var ignores []PickerIgnore
	for _, dir := range ignoreDirs(root, path, opts.Parents) {
		if opts.Ignore {
			ignores = appendIgnore(ignores, dir, ".ignore")
			ignores = appendIgnore(
				ignores, dir, filepath.Join(loader.WorkspaceDirName, "ignore"),
			)
		}
		if opts.GitIgnore {
			ignores = appendIgnore(ignores, dir, ".gitignore")
		}
	}
	if opts.GitExclude {
		ignores = appendIgnore(
			ignores, root, filepath.Join(".git", "info", "exclude"),
		)
	}
	if opts.GitGlobal {
		ignores = appendIgnorePath(ignores, root, gitGlobalIgnorePath())
	}
	if opts.Ignore {
		ignores = appendIgnorePath(ignores, "", loader.ConfigIgnoreFile())
	}
	return ignores
}

func appendIgnore(ignores []PickerIgnore, dir, name string) []PickerIgnore {
	return appendIgnorePath(ignores, dir, filepath.Join(dir, name))
}

func appendIgnorePath(ignores []PickerIgnore, dir, path string) []PickerIgnore {
	if ig, ok := compileIgnore(path); ok {
		return append(ignores, PickerIgnore{dir: dir, ig: ig})
	}
	return ignores
}

func ignoreDirs(root, path string, parents bool) []string {
	root = filepath.Clean(root)
	dir := filepath.Dir(path)
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

// DefaultPickerIgnoreOptions is the ignore behavior file-walking pickers use
// when a caller does not need to customize it
func DefaultPickerIgnoreOptions() PickerIgnoreOptions {
	return PickerIgnoreOptions{
		Hidden: true, Parents: true, Ignore: true,
		GitIgnore: true, GitGlobal: true, GitExclude: true,
	}
}

func compileIgnore(path string) (*gitignore.GitIgnore, bool) {
	ig, err := gitignore.CompileIgnoreFile(path)
	return ig, err == nil
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
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "git", "ignore")
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

func excludedPickerType(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	_, ok := excludedPickerTypes[ext]
	return ok
}
