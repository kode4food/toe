package lsp

import (
	"context"
	"sync"

	"github.com/rjeczalik/notify"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

type (
	// watchState owns the notify watcher and each server's registered
	// file-watch patterns
	watchState struct {
		sync.RWMutex
		registrations map[string]map[string][]fileWatch
		watcher       *fsWatcher
	}

	fsWatcher struct {
		events chan notify.EventInfo
		done   chan struct{}
		roots  map[string]struct{}
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

// DidChangeWatchedFile notifies the server that a watched file changed
func (c *Client) DidChangeWatchedFile(ctx context.Context, path string) error {
	return c.DidChangeWatchedFiles(ctx, []fileWatchEvent{{
		path: path,
		kind: protocol.FileChangeTypeChanged,
	}})
}

// DidChangeWatchedFiles notifies the server that watched files changed
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
		if err := protocol.Unmarshal(reg.RegisterOptions, &opts); err != nil {
			return err
		}
		s.registerWatches(registerWatchesArgs{
			server:  server,
			id:      reg.ID,
			watches: fileWatches(opts),
		})
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

type registerWatchesArgs struct {
	server  string
	id      string
	watches []fileWatch
}

func (s *Session) registerWatches(args registerWatchesArgs) {
	s.watch.Lock()
	if s.watch.registrations[args.server] == nil {
		s.watch.registrations[args.server] = map[string][]fileWatch{}
	}
	s.watch.registrations[args.server][args.id] = args.watches
	s.watch.Unlock()
	if len(args.watches) > 0 {
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
		if client := s.servers.client(server); client != nil {
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
