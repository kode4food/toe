package lsp

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/go-json-experiment/json"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

type (
	fileWatchEvent struct {
		path string
		kind protocol.FileChangeType
	}

	fileWatch struct {
		pattern string
		base    string
	}

	fsWatcher struct {
		watcher *fsnotify.Watcher
		done    chan struct{}
		dirs    map[string]struct{}
	}
)

func (c *Client) DidChangeWatchedFile(
	ctx context.Context, path string,
) error {
	return c.DidChangeWatchedFiles(ctx, []fileWatchEvent{{
		path: path,
		kind: protocol.FileChangeTypeChanged,
	}})
}

func (c *Client) DidChangeWatchedFiles(
	ctx context.Context, events []fileWatchEvent,
) error {
	if len(events) == 0 {
		return nil
	}
	changes := make([]protocol.FileEvent, 0, len(events))
	for _, event := range events {
		if event.path == "" {
			continue
		}
		changes = append(changes, protocol.FileEvent{
			URI:  uri.File(event.path),
			Type: event.kind,
		})
	}
	if len(changes) == 0 {
		return nil
	}
	params := &protocol.DidChangeWatchedFilesParams{
		Changes: changes,
	}
	return c.server.DidChangeWatchedFiles(ctx, params)
}

func (s *Session) registerCapability(
	server string, params *protocol.RegistrationParams,
) error {
	if params == nil {
		return nil
	}
	for _, reg := range params.Registrations {
		if reg.Method != protocol.MethodWorkspaceDidChangeWatchedFiles {
			continue
		}
		var opts protocol.DidChangeWatchedFilesRegistrationOptions
		if err := json.Unmarshal(reg.RegisterOptions, &opts); err != nil {
			return err
		}
		s.registerWatches(server, reg.ID, fileWatches(opts))
	}
	return nil
}

func (s *Session) unregisterCapability(
	server string, params *protocol.UnregistrationParams,
) {
	if params == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, unreg := range params.Unregisterations {
		if unreg.Method != protocol.MethodWorkspaceDidChangeWatchedFiles {
			continue
		}
		delete(s.watches[server], unreg.ID)
		if len(s.watches[server]) == 0 {
			delete(s.watches, server)
		}
	}
}

func (s *Session) registerWatches(
	server, id string, watches []fileWatch,
) {
	s.mu.Lock()
	if s.watches[server] == nil {
		s.watches[server] = map[string][]fileWatch{}
	}
	s.watches[server][id] = watches
	s.mu.Unlock()
	if len(watches) > 0 {
		go s.ensureFileWatcher()
	}
}

func (s *Session) didChangeWatchedFile(path string) {
	s.didChangeWatchedFileEvent(fileWatchEvent{
		path: path,
		kind: protocol.FileChangeTypeChanged,
	})
}

func (s *Session) didChangeWatchedFileEvent(event fileWatchEvent) {
	if event.path == "" {
		return
	}
	clients := s.clientsWatching(event.path)
	for _, client := range clients {
		_ = client.DidChangeWatchedFiles(s.ctx, []fileWatchEvent{event})
	}
}

func (s *Session) ensureFileWatcher() {
	roots := s.fileWatchRoots()
	if len(roots) == 0 {
		return
	}
	s.mu.Lock()
	if s.watcher == nil {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			s.mu.Unlock()
			return
		}
		state := &fsWatcher{
			watcher: watcher,
			done:    make(chan struct{}),
			dirs:    map[string]struct{}{},
		}
		s.watcher = state
		go s.runFileWatcher(state)
	}
	s.mu.Unlock()
	for _, root := range roots {
		s.addFileWatchRoot(root)
	}
}

func (s *Session) closeFileWatcher() {
	s.mu.Lock()
	state := s.watcher
	s.watcher = nil
	s.mu.Unlock()
	if state == nil {
		return
	}
	close(state.done)
	_ = state.watcher.Close()
}

func (s *Session) fileWatchRoots() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := map[string]struct{}{}
	out := []string{}
	add := func(path string) {
		if path == "" {
			return
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return
		}
		abs = filepath.Clean(abs)
		if _, ok := seen[abs]; ok {
			return
		}
		seen[abs] = struct{}{}
		out = append(out, abs)
	}
	add(s.cwd)
	for _, root := range s.roots {
		add(root)
	}
	return out
}

func (s *Session) runFileWatcher(state *fsWatcher) {
	for {
		select {
		case event, ok := <-state.watcher.Events:
			if !ok {
				return
			}
			s.handleFileWatchEvent(event)
		case _, ok := <-state.watcher.Errors:
			if !ok {
				return
			}
		case <-state.done:
			return
		}
	}
}

func (s *Session) handleFileWatchEvent(event fsnotify.Event) {
	path := filepath.Clean(event.Name)
	if event.Op&fsnotify.Create != 0 {
		s.addCreatedWatchPath(path)
	}
	kind, ok := fileWatchChangeType(event.Op)
	if !ok {
		return
	}
	s.didChangeWatchedFileEvent(fileWatchEvent{path: path, kind: kind})
}

func (s *Session) addCreatedWatchPath(path string) {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return
	}
	s.addFileWatchRoot(path)
}

func (s *Session) addFileWatchRoot(root string) {
	_ = filepath.WalkDir(root, func(
		path string, d os.DirEntry, err error,
	) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		s.addFileWatchDir(path)
		return nil
	})
}

func (s *Session) addFileWatchDir(path string) {
	path = filepath.Clean(path)
	s.mu.Lock()
	state := s.watcher
	if state == nil {
		s.mu.Unlock()
		return
	}
	if _, ok := state.dirs[path]; ok {
		s.mu.Unlock()
		return
	}
	state.dirs[path] = struct{}{}
	s.mu.Unlock()
	if err := state.watcher.Add(path); err != nil {
		s.mu.Lock()
		delete(state.dirs, path)
		s.mu.Unlock()
	}
}

func (s *Session) clientsWatching(path string) []*Client {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*Client
	for server, regs := range s.watches {
		if !watchRegistrationsMatch(regs, path) {
			continue
		}
		if client := s.clients[server]; client != nil {
			out = append(out, client)
		}
	}
	return out
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

func relativePatternBase(
	base protocol.RelativePatternBaseURI,
) (string, bool) {
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

func watchRegistrationsMatch(
	regs map[string][]fileWatch, path string,
) bool {
	for _, watches := range regs {
		for _, watch := range watches {
			if watch.match(path) {
				return true
			}
		}
	}
	return false
}

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

func fileWatchChangeType(
	op fsnotify.Op,
) (protocol.FileChangeType, bool) {
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
