package loader

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var ErrPathUnavailable = errors.New("path unavailable")

const (
	DirName           = "toe"
	LogFileName       = "toe.log"
	WorkspaceDirName  = "." + DirName
	RuntimeEnv        = "TOE_RUNTIME"
	DefaultRuntimeEnv = "TOE_DEFAULT_RUNTIME"
)

func RuntimeDirs() []string {
	var dirs []string
	if dir := os.Getenv("CARGO_MANIFEST_DIR"); dir != "" {
		dirs = append(dirs, filepath.Join(filepath.Dir(dir), "runtime"))
	}
	if dir, ok := ConfigDir(); ok {
		dirs = append(dirs, filepath.Join(dir, "runtime"))
	}
	if dir := os.Getenv(RuntimeEnv); dir != "" {
		dirs = append(dirs, normalizeRuntimeDir(dir))
	}
	if dir := os.Getenv(DefaultRuntimeEnv); dir != "" {
		dirs = append(dirs, dir)
	}
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		dirs = append(dirs, filepath.Join(filepath.Dir(exe), "runtime"))
	}
	return dirs
}

func RuntimeFile(rel string) string {
	dirs := RuntimeDirs()
	for _, dir := range dirs {
		path := filepath.Join(dir, rel)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	if len(dirs) == 0 {
		return rel
	}
	return filepath.Join(dirs[len(dirs)-1], rel)
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
		if workspaceMarkerExists(abs) {
			return abs, false
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

func normalizeRuntimeDir(dir string) string {
	switch {
	case dir == "~":
		if home, err := os.UserHomeDir(); err == nil {
			dir = home
		}
	case strings.HasPrefix(dir, "~/"):
		if home, err := os.UserHomeDir(); err == nil {
			dir = filepath.Join(home, strings.TrimPrefix(dir, "~/"))
		}
	}
	if abs, err := filepath.Abs(dir); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(dir)
}

func workspaceMarkerExists(dir string) bool {
	names := []string{".git", ".svn", ".jj", WorkspaceDirName}
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
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
	lines := make([]string, 0, len(set))
	for line := range set {
		lines = append(lines, line)
	}
	sort.Strings(lines)
	text := ""
	if len(lines) > 0 {
		text = strings.Join(lines, "\n") + "\n"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(text), 0o644)
}
