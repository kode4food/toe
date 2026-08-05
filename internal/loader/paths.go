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

// CanonicalPath resolves symlinks in path's parent directory and joins the
// unresolved final element
func CanonicalPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	dir, base := filepath.Split(abs)
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return abs
	}
	return filepath.Join(resolved, base)
}

// ExpandUserPath expands leading home-directory shorthand and environment vars
func ExpandUserPath(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return os.ExpandEnv(path)
}

// ConfigFile returns the user's config.toml path
func ConfigFile() (string, bool) {
	if dir, ok := ConfigDir(); ok {
		return filepath.Join(dir, "config.toml"), true
	}
	return "", false
}

// LanguagesFile returns the user's languages.toml path
func LanguagesFile() (string, bool) {
	if dir, ok := ConfigDir(); ok {
		return filepath.Join(dir, "languages.toml"), true
	}
	return "", false
}

// ConfigIgnoreFile returns the user's picker ignore-file path
func ConfigIgnoreFile() string {
	if dir, ok := ConfigDir(); ok {
		return filepath.Join(dir, "ignore")
	}
	return ""
}

// ConfigDir returns the user's toe configuration directory
func ConfigDir() (string, bool) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, DirName), true
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", DirName), true
	}
	return "", false
}

// LogFile returns the editor's log path
func LogFile() (string, bool) {
	if dir, ok := CacheDir(); ok {
		return filepath.Join(dir, LogFileName), true
	}
	return "", false
}

// CacheDir returns the user's toe cache directory
func CacheDir() (string, bool) {
	if dir := os.Getenv("XDG_CACHE_HOME"); dir != "" {
		return filepath.Join(dir, DirName), true
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".cache", DirName), true
	}
	return "", false
}

// DataDir returns the user's toe data directory
func DataDir() (string, bool) {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, DirName), true
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share", DirName), true
	}
	return "", false
}

// WorkspaceConfigFile returns the workspace config.toml path under dir
func WorkspaceConfigFile(dir string) string {
	root, _ := FindWorkspace(dir)
	return filepath.Join(root, WorkspaceDirName, "config.toml")
}

// WorkspaceLanguagesFile returns the workspace languages.toml path under dir
func WorkspaceLanguagesFile(dir string) string {
	root, _ := FindWorkspace(dir)
	return filepath.Join(root, WorkspaceDirName, "languages.toml")
}

// WorkspaceInitFile returns the workspace Ale initialization path
func WorkspaceInitFile(dir string) string {
	root, _ := FindWorkspace(dir)
	return filepath.Join(root, WorkspaceDirName, "init.ale")
}

// WorkspaceTrustFile returns the path of the trusted-workspace list
func WorkspaceTrustFile() (string, bool) {
	if dir, ok := DataDir(); ok {
		return filepath.Join(dir, "trusted_workspaces"), true
	}
	return "", false
}

// FindWorkspace walks up for a .git or .toe directory. The bool reports a
// fallback to dir itself, not success
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

// TrustWorkspace adds the workspace containing dir to the trusted list
func TrustWorkspace(dir string) error {
	if path, ok := WorkspaceTrustFile(); ok {
		root, _ := FindWorkspace(dir)
		return updateWorkspaceSet(path, root, true)
	}
	return ErrPathUnavailable
}

// UntrustWorkspace drops the workspace containing dir from the trusted list
func UntrustWorkspace(dir string) error {
	if path, ok := WorkspaceTrustFile(); ok {
		root, _ := FindWorkspace(dir)
		return updateWorkspaceSet(path, root, false)
	}
	return ErrPathUnavailable
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
