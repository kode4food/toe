package lsp

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"go.lsp.dev/uri"

	"github.com/kode4food/toe/internal/view/language"
)

type (
	workspaceServer struct {
		name       string
		root       string
		path       string
		languageID string
	}

	workspaceServerState struct {
		sync.Mutex
		scanned bool
	}
)

func (s *Session) ensureWorkspaceServers() {
	s.workspace.Lock()
	defer s.workspace.Unlock()
	if s.workspace.scanned {
		return
	}
	s.workspace.scanned = true
	for _, target := range s.discoverWorkspaceServers() {
		client, started, err := s.ensureServer(target)
		if err != nil || !started {
			continue
		}
		data, err := os.ReadFile(target.path)
		if err != nil {
			continue
		}
		_, _ = client.DidOpen(s.ctx, DocumentSnapshot{
			URI:        uri.File(target.path),
			LanguageID: target.languageID,
			Text:       string(data),
		})
	}
}

func (s *Session) discoverWorkspaceServers() []*workspaceServer {
	langs := s.servers.languageConfig()
	remaining := map[string]bool{}
	for _, lang := range langs.Languages {
		for _, name := range serverNames(lang.LanguageServers) {
			remaining[name] = true
		}
	}
	if len(remaining) == 0 {
		return nil
	}
	var out []*workspaceServer
	_ = filepath.WalkDir(s.cwd, func(
		path string, entry fs.DirEntry, err error,
	) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if path != s.cwd && skipWorkspaceDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if lang := language.ForFilename(langs, path); lang != nil {
			root := s.cwd
			langID := lang.Name
			if lang.LanguageID != "" {
				langID = lang.LanguageID
			}
			if resolved, ok := s.resolveWorkspaceRoot(path, lang); ok {
				root = resolved
			} else if len(lang.Roots) > 0 {
				return nil
			}
			for _, name := range serverNames(lang.LanguageServers) {
				if !remaining[name] {
					continue
				}
				delete(remaining, name)
				out = append(out, &workspaceServer{
					name:       name,
					root:       root,
					path:       path,
					languageID: langID,
				})
			}
		}
		if len(remaining) == 0 {
			return fs.SkipAll
		}
		return nil
	})
	return out
}

func (w *workspaceServerState) reset() {
	w.Lock()
	defer w.Unlock()
	w.scanned = false
}

func skipWorkspaceDir(name string) bool {
	switch name {
	case ".git", ".hg", ".svn", ".venv", "build", "coverage", "dist",
		"node_modules", "target", "vendor", "venv":
		return true
	default:
		return false
	}
}
