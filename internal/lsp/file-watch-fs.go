package lsp

import (
	"cmp"
	"maps"
	"path/filepath"
	"slices"

	"github.com/rjeczalik/notify"

	"github.com/kode4food/toe/internal/i18n"
)

func (s *Session) ensureFileWatcher() {
	roots := s.fileWatchRoots()
	if len(roots) == 0 {
		return
	}
	s.watch.Lock()
	if s.watch.watcher == nil {
		state := &fsWatcher{
			events: make(chan notify.EventInfo, 256),
			done:   make(chan struct{}),
			roots:  map[string]struct{}{},
		}
		s.watch.watcher = state
		go s.runFileWatcher(state)
	}
	s.watch.Unlock()
	for _, root := range roots {
		s.addFileWatchRoot(root)
	}
}

func (s *Session) closeFileWatcher() {
	s.watch.Lock()
	state := s.watch.watcher
	s.watch.watcher = nil
	s.watch.Unlock()
	if state == nil {
		return
	}
	close(state.done)
	notify.Stop(state.events)
}

func (s *Session) fileWatchRoots() []string {
	seen := map[string]struct{}{}
	var out []string
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
	for _, root := range s.servers.allRoots() {
		add(root)
	}
	return out
}

func (s *Session) runFileWatcher(state *fsWatcher) {
	for {
		select {
		case ev, ok := <-state.events:
			if !ok {
				return
			}
			s.handleFileWatchEvent(ev)
		case <-state.done:
			return
		}
	}
}

func (s *Session) handleFileWatchEvent(ev notify.EventInfo) {
	kind, ok := fileWatchChangeType(ev.Event())
	if !ok {
		return
	}
	event := fileWatchEvent{path: filepath.Clean(ev.Path()), kind: kind}
	if s.queueWatchEvent(event) {
		key := refreshKey{kind: refreshWatchEvents}
		s.refresh.Do(key, s.flushWatchEvents)
	}
}

// queueWatchEvent reports whether the event opened a new batch; only that event
// schedules a flush, so a steady stream cannot postpone it
func (s *Session) queueWatchEvent(event fileWatchEvent) bool {
	s.watch.Lock()
	defer s.watch.Unlock()
	opened := len(s.watch.queued) == 0
	if s.watch.queued == nil {
		s.watch.queued = map[string]fileWatchEvent{}
	}
	s.watch.queued[event.path] = event
	return opened
}

func (s *Session) flushWatchEvents() {
	s.watch.Lock()
	batch := slices.Collect(maps.Values(s.watch.queued))
	clear(s.watch.queued)
	s.watch.Unlock()
	slices.SortFunc(batch, func(a, b fileWatchEvent) int {
		return cmp.Compare(a.path, b.path)
	})
	byClient := map[*Client][]fileWatchEvent{}
	for _, event := range batch {
		for _, client := range s.clientsWatching(event.path) {
			byClient[client] = append(byClient[client], event)
		}
	}
	for client, events := range byClient {
		_ = client.DidChangeWatchedFiles(s.ctx, events)
	}
}

func (s *Session) addFileWatchRoot(root string) {
	root = filepath.Clean(root)
	s.watch.Lock()
	state := s.watch.watcher
	if state == nil {
		s.watch.Unlock()
		return
	}
	if _, ok := state.roots[root]; ok {
		s.watch.Unlock()
		return
	}
	state.roots[root] = struct{}{}
	s.watch.Unlock()
	err := notify.Watch(filepath.Join(root, "..."), state.events, notify.All)
	if err == nil {
		return
	}
	s.watch.Lock()
	delete(state.roots, root)
	s.watch.Unlock()
	s.editor.SetStatusMsg(i18n.Text(
		i18n.StatusFileWatchUnavailable, i18n.Vars{"error": err},
	))
}
