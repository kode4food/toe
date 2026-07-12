package loader

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var ErrPathUnavailable = errors.New("path unavailable")

const (
	DirName          = "toe"
	LogFileName      = "toe.log"
	WorkspaceDirName = "." + DirName
)

// ExpandUserPath expands leading home-directory shorthand and environment vars
func ExpandUserPath(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return os.ExpandEnv(path)
}

func ConfigFile() (string, bool) {
	dir, ok := ConfigDir()
	if !ok {
		return "", false
	}
	return filepath.Join(dir, "config.toml"), true
}

func LanguagesFile() (string, bool) {
	dir, ok := ConfigDir()
	if !ok {
		return "", false
	}
	return filepath.Join(dir, "languages.toml"), true
}

func ConfigIgnoreFile() string {
	dir, ok := ConfigDir()
	if !ok {
		return ""
	}
	return filepath.Join(dir, "ignore")
}

func ConfigDir() (string, bool) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, DirName), true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	return filepath.Join(home, ".config", DirName), true
}

func LogFile() (string, bool) {
	dir, ok := CacheDir()
	if !ok {
		return "", false
	}
	return filepath.Join(dir, LogFileName), true
}

func CacheDir() (string, bool) {
	if dir := os.Getenv("XDG_CACHE_HOME"); dir != "" {
		return filepath.Join(dir, DirName), true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	return filepath.Join(home, ".cache", DirName), true
}

func DataDir() (string, bool) {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, DirName), true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	return filepath.Join(home, ".local", "share", DirName), true
}

func WorkspaceConfigFile(dir string) string {
	root, _ := FindWorkspace(dir)
	return filepath.Join(root, WorkspaceDirName, "config.toml")
}

func WorkspaceLanguagesFile(dir string) string {
	root, _ := FindWorkspace(dir)
	return filepath.Join(root, WorkspaceDirName, "languages.toml")
}

func WorkspaceTrustFile() (string, bool) {
	dir, ok := DataDir()
	if !ok {
		return "", false
	}
	return filepath.Join(dir, "trusted_workspaces"), true
}

func FindWorkspace(dir string) (string, bool) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}
	for {
		for _, name := range []string{".git", WorkspaceDirName} {
			if _, err := os.Stat(filepath.Join(abs, name)); err == nil {
				return abs, false
			}
		}
		next := filepath.Dir(abs)
		if next == abs {
			return dir, true
		}
		abs = next
	}
}

func TrustWorkspace(dir string) error {
	path, ok := WorkspaceTrustFile()
	if !ok {
		return ErrPathUnavailable
	}
	root, _ := FindWorkspace(dir)
	return updateWorkspaceSet(path, root, true)
}

func UntrustWorkspace(dir string) error {
	path, ok := WorkspaceTrustFile()
	if !ok {
		return ErrPathUnavailable
	}
	root, _ := FindWorkspace(dir)
	return updateWorkspaceSet(path, root, false)
}

func updateWorkspaceSet(path, workspace string, add bool) error {
	set := map[string]bool{}
	if data, err := os.ReadFile(path); err == nil {
		for line := range strings.SplitSeq(string(data), "\n") {
			if line != "" {
				set[line] = true
			}
		}
	}
	if add {
		set[workspace] = true
	} else {
		delete(set, workspace)
	}
	lines := slices.Sorted(maps.Keys(set))
	text := ""
	if len(lines) > 0 {
		text = strings.Join(lines, "\n") + "\n"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(text), 0o644)
}
