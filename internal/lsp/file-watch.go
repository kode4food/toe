package lsp

import (
	"context"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/go-json-experiment/json"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

type (
	// watchState owns the fsnotify watcher and each server's registered
	// file-watch patterns
	watchState struct {
		sync.RWMutex
		registrations map[string]map[string][]fileWatch
		watcher       *fsWatcher
	}

	fsWatcher struct {
		watcher *fsnotify.Watcher
		done    chan struct{}
		dirs    map[string]struct{}
	}

	fileWatchEvent struct {
		path string
		kind protocol.FileChangeType
	}

	fileWatch struct {
		pattern string
		base    string
	}
)

func (c *Client) DidChangeWatchedFile(ctx context.Context, path string) error {
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
	s.watch.Lock()
	defer s.watch.Unlock()
	for _, unreg := range params.Unregisterations {
		if unreg.Method != protocol.MethodWorkspaceDidChangeWatchedFiles {
			continue
		}
		delete(s.watch.registrations[server], unreg.ID)
		if len(s.watch.registrations[server]) == 0 {
			delete(s.watch.registrations, server)
		}
	}
}

func (s *Session) registerWatches(server, id string, watches []fileWatch) {
	s.watch.Lock()
	if s.watch.registrations[server] == nil {
		s.watch.registrations[server] = map[string][]fileWatch{}
	}
	s.watch.registrations[server][id] = watches
	s.watch.Unlock()
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

func (s *Session) clientsWatching(path string) []*Client {
	var servers []string
	s.watch.RLock()
	for server, regs := range s.watch.registrations {
		if watchRegistrationsMatch(regs, path) {
			servers = append(servers, server)
		}
	}
	s.watch.RUnlock()
	var out []*Client
	for _, server := range servers {
		if client, ok := s.servers.client(server); ok {
			out = append(out, client)
		}
	}
	return out
}

func (w *watchState) reset() {
	w.Lock()
	defer w.Unlock()
	w.registrations = map[string]map[string][]fileWatch{}
}
