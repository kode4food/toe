package lsp

import (
	"os"
	"path/filepath"
	"strings"

	"go.lsp.dev/uri"
)

type (
	// WorkspaceRequest carries the parameters for resolving a project workspace root
	WorkspaceRequest struct {
		FilePath       string
		Workspace      string
		WorkspaceIsCWD bool
		RootMarkers    []string
		RootDirs       []string
	}

	// Workspace represents a resolved project workspace directory
	Workspace struct {
		Path string
	}
)

// ResolveWorkspace locates the nearest workspace root for the given file path
func ResolveWorkspace(req WorkspaceRequest) (Workspace, bool) {
	file, ok := workspaceFile(req.FilePath)
	if !ok {
		return Workspace{}, false
	}
	workspace, ok := cleanAbs(req.Workspace)
	if !ok || !inside(file, workspace) {
		return Workspace{}, false
	}
	var marked string
	for dir := file; ; dir = filepath.Dir(dir) {
		if dirHasAny(dir, req.RootMarkers) {
			marked = dir
		}
		if matchesRootDir(workspace, dir, req.RootDirs) {
			if marked != "" {
				return Workspace{Path: marked}, true
			}
			return Workspace{Path: workspace}, true
		}
		if dir == workspace {
			if marked != "" {
				return Workspace{Path: marked}, true
			}
			if !req.WorkspaceIsCWD {
				return Workspace{Path: workspace}, true
			}
			return Workspace{}, false
		}
		next := filepath.Dir(dir)
		if next == dir {
			return Workspace{}, false
		}
	}
}

func (w Workspace) URI() uri.URI {
	return uri.File(w.Path)
}

// RequiredRootFound reports whether any required root pattern matches a directory entry
func RequiredRootFound(root string, patterns []string) (bool, error) {
	if len(patterns) == 0 {
		return true, nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		for _, pattern := range patterns {
			ok, err := filepath.Match(pattern, entry.Name())
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
	}
	return false, nil
}

func workspaceFile(path string) (string, bool) {
	if path == "" {
		path = "."
	}
	if !filepath.IsAbs(path) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", false
		}
		path = filepath.Join(cwd, path)
	}
	return filepath.Clean(path), true
}

func cleanAbs(path string) (string, bool) {
	if path == "" {
		return "", false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	return filepath.Clean(abs), true
}

func inside(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if rel == ".." || filepath.IsAbs(rel) {
		return false
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func dirHasAny(dir string, names []string) bool {
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

func matchesRootDir(workspace, dir string, roots []string) bool {
	for _, root := range roots {
		if filepath.Clean(filepath.Join(workspace, root)) == dir {
			return true
		}
	}
	return false
}
