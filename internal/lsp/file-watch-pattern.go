package lsp

import (
	"path/filepath"
	"strings"

	"github.com/rjeczalik/notify"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/kode4food/toe/internal/glob"
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
	return matchWatchPattern(matchWatchPatternArgs{
		pattern: w.pattern,
		path:    candidate,
	})
}

func fileWatches(
	opts protocol.DidChangeWatchedFilesRegistrationOptions,
) []fileWatch {
	out := make([]fileWatch, 0, len(opts.Watchers))
	for _, watcher := range opts.Watchers {
		if watch, ok := fileWatchFor(watcher.GlobPattern); ok {
			out = append(out, watch)
		}
	}
	return out
}

func fileWatchFor(pattern protocol.GlobPattern) (fileWatch, bool) {
	switch p := pattern.(type) {
	case protocol.Pattern:
		return fileWatch{pattern: string(p)}, true
	case *protocol.RelativePattern:
		if p == nil {
			return fileWatch{}, false
		}
		if base, ok := relativePatternBase(p.BaseURI); ok {
			return fileWatch{
				pattern: string(p.Pattern),
				base:    base,
			}, true
		}
		return fileWatch{}, false
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

type matchWatchPatternArgs struct {
	pattern string
	path    string
}

func matchWatchPattern(args matchWatchPatternArgs) bool {
	if args.pattern == "" {
		return false
	}
	if glob.Match(glob.Candidate{
		Pattern: args.pattern,
		Path:    args.path,
	}) {
		return true
	}
	return glob.Match(glob.Candidate{
		Pattern: args.pattern,
		Path:    filepath.Base(args.path),
	})
}

func fileWatchChangeType(ev notify.Event) (protocol.FileChangeType, bool) {
	switch {
	case ev&(notify.Remove|notify.Rename) != 0:
		return protocol.FileChangeTypeDeleted, true
	case ev&notify.Create != 0:
		return protocol.FileChangeTypeCreated, true
	case ev&notify.Write != 0:
		return protocol.FileChangeTypeChanged, true
	default:
		return 0, false
	}
}
