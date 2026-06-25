package ui

import (
	"os"
	"path/filepath"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"

	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/view/config"
)

type pickerIgnore struct {
	base string
	ig   *gitignore.GitIgnore
}

func skipPickerPath(rel string, d os.DirEntry, ignores []pickerIgnore) bool {
	name := d.Name()
	if strings.HasPrefix(name, ".") {
		return true
	}
	if excludedPickerType(name) {
		return true
	}
	for _, ig := range ignores {
		sub, ok := ignorePathForBase(rel, ig.base)
		if ok && ig.ig.MatchesPath(sub) {
			return true
		}
	}
	return false
}

func ignorePathForBase(rel, base string) (string, bool) {
	if base == "" {
		return rel, true
	}
	if rel == base {
		return "", true
	}
	sub, ok := strings.CutPrefix(rel, base+"/")
	return sub, ok
}

func loadIgnoreFiles(root, rel string) []pickerIgnore {
	var ignores []pickerIgnore
	for _, base := range ignoreBases(rel) {
		dir := filepath.Join(root, filepath.FromSlash(base))
		for _, name := range []string{
			".ignore",
			".gitignore",
			filepath.Join(loader.WorkspaceDirName, "ignore"),
		} {
			if ig, ok := compileIgnore(filepath.Join(dir, name)); ok {
				ignores = append(ignores, pickerIgnore{base, ig})
			}
		}
		if base == "" {
			if ig, ok := compileIgnore(
				filepath.Join(root, ".git", "info", "exclude"),
			); ok {
				ignores = append(ignores, pickerIgnore{base, ig})
			}
			if ig, ok := compileIgnore(config.IgnorePath()); ok {
				ignores = append(ignores, pickerIgnore{base, ig})
			}
		}
	}
	return ignores
}

func ignoreBases(rel string) []string {
	dir := filepath.ToSlash(filepath.Dir(rel))
	if dir == "." {
		return []string{""}
	}
	parts := strings.Split(dir, "/")
	bases := make([]string, 1, len(parts)+1)
	for i := range parts {
		bases = append(bases, strings.Join(parts[:i+1], "/"))
	}
	return bases
}

func compileIgnore(path string) (*gitignore.GitIgnore, bool) {
	ig, err := gitignore.CompileIgnoreFile(path)
	return ig, err == nil
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
