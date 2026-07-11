package lsp

import (
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func (w fileWatch) match(path string) bool {
	candidate := path
	if w.base != "" {
		rel, err := filepath.Rel(w.base, path)
		if err != nil ||
			strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return false
		}
		candidate = rel
	}
	return matchWatchPattern(w.pattern, candidate)
}

func fileWatches(
	opts protocol.DidChangeWatchedFilesRegistrationOptions,
) []fileWatch {
	out := make([]fileWatch, 0, len(opts.Watchers))
	for _, watcher := range opts.Watchers {
		watch, ok := fileWatchFor(watcher.GlobPattern)
		if ok {
			out = append(out, watch)
		}
	}
	return out
}

func fileWatchFor(pattern protocol.GlobPattern) (fileWatch, bool) {
	switch p := pattern.(type) {
	case protocol.Pattern:
		return fileWatch{pattern: filepath.FromSlash(string(p))}, true
	case *protocol.RelativePattern:
		if p == nil {
			return fileWatch{}, false
		}
		base, ok := relativePatternBase(p.BaseURI)
		if !ok {
			return fileWatch{}, false
		}
		return fileWatch{
			pattern: filepath.FromSlash(string(p.Pattern)),
			base:    base,
		}, true
	default:
		return fileWatch{}, false
	}
}

func relativePatternBase(base protocol.RelativePatternBaseURI) (string, bool) {
	switch b := base.(type) {
	case protocol.URI:
		return uri.URI(b).FsPath(), true
	case *protocol.WorkspaceFolder:
		if b == nil {
			return "", false
		}
		return b.URI.FsPath(), true
	default:
		return "", false
	}
}

func watchRegistrationsMatch(regs map[string][]fileWatch, path string) bool {
	for _, watches := range regs {
		for _, watch := range watches {
			if watch.match(path) {
				return true
			}
		}
	}
	return false
}

func matchWatchPattern(pattern, path string) bool {
	if pattern == "" {
		return false
	}
	if ok, _ := filepath.Match(pattern, path); ok {
		return true
	}
	if ok, _ := filepath.Match(pattern, filepath.Base(path)); ok {
		return true
	}
	const recursive = "**" + string(filepath.Separator)
	if after, ok := strings.CutPrefix(pattern, recursive); ok {
		if ok, _ := filepath.Match(after, path); ok {
			return true
		}
		if ok, _ := filepath.Match(after, filepath.Base(path)); ok {
			return true
		}
		return strings.HasSuffix(path, after) ||
			strings.HasSuffix(filepath.Base(path), after)
	}
	return false
}

func fileWatchChangeType(op fsnotify.Op) (protocol.FileChangeType, bool) {
	switch {
	case op&(fsnotify.Remove|fsnotify.Rename) != 0:
		return protocol.FileChangeTypeDeleted, true
	case op&fsnotify.Create != 0:
		return protocol.FileChangeTypeCreated, true
	case op&fsnotify.Write != 0:
		return protocol.FileChangeTypeChanged, true
	default:
		return 0, false
	}
}
