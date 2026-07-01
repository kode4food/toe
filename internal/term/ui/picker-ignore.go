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
	pickerIgnore struct {
		base string
		dir  string
		ig   *gitignore.GitIgnore
	}

	pickerIgnoreOptions struct {
		hidden     bool
		parents    bool
		ignore     bool
		gitIgnore  bool
		gitGlobal  bool
		gitExclude bool
	}
)

type skipPickerPathArgs struct {
	rel     string
	path    string
	entry   os.DirEntry
	ignores []pickerIgnore
	opts    pickerIgnoreOptions
}

func skipPickerPath(args skipPickerPathArgs) bool {
	name := args.entry.Name()
	if args.opts.hidden && strings.HasPrefix(name, ".") {
		return true
	}
	if excludedPickerType(name) {
		return true
	}
	for _, ig := range args.ignores {
		sub, ok := ignorePathForBase(args.rel, args.path, ig)
		if ok && ig.ig.MatchesPath(sub) {
			return true
		}
	}
	return false
}

func ignorePathForBase(rel, path string, ig pickerIgnore) (string, bool) {
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

func loadIgnoreFiles(
	root, path, rel string, opts pickerIgnoreOptions,
) []pickerIgnore {
	var ignores []pickerIgnore
	for _, dir := range ignoreDirs(root, path, opts.parents) {
		if opts.ignore {
			ignores = appendIgnore(ignores, dir, ".ignore")
			ignores = appendIgnore(
				ignores, dir, filepath.Join(loader.WorkspaceDirName, "ignore"),
			)
		}
		if opts.gitIgnore {
			ignores = appendIgnore(ignores, dir, ".gitignore")
		}
	}
	if opts.gitExclude {
		ignores = appendIgnore(
			ignores, root, filepath.Join(".git", "info", "exclude"),
		)
	}
	if opts.gitGlobal {
		ignores = appendIgnorePath(ignores, root, gitGlobalIgnorePath())
	}
	if opts.ignore {
		ignores = appendIgnorePath(ignores, "", loader.ConfigIgnoreFile())
	}
	return ignores
}

func appendIgnore(
	ignores []pickerIgnore, dir, name string,
) []pickerIgnore {
	return appendIgnorePath(ignores, dir, filepath.Join(dir, name))
}

func appendIgnorePath(
	ignores []pickerIgnore, dir, path string,
) []pickerIgnore {
	if ig, ok := compileIgnore(path); ok {
		return append(ignores, pickerIgnore{dir: dir, ig: ig})
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

func defaultPickerIgnoreOptions() pickerIgnoreOptions {
	return pickerIgnoreOptions{
		hidden: true, parents: true, ignore: true,
		gitIgnore: true, gitGlobal: true, gitExclude: true,
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
	switch ext {
	case ".zip", ".gz", ".bz2", ".zst", ".lzo", ".sz", ".tgz", ".tbz2",
		".lz", ".lz4", ".lzma", ".z", ".xz", ".7z", ".rar", ".cab":
		return true
	default:
		return false
	}
}
