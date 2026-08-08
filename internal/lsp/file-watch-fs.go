package lsp

import (
	"path/filepath"

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
	path := filepath.Clean(ev.Path())
	if kind, ok := fileWatchChangeType(ev.Event()); ok {
		s.didChangeWatchedFileEvent(fileWatchEvent{path: path, kind: kind})
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
	if s.editor != nil {
		s.editor.SetStatusMsg(i18n.Text(
			i18n.StatusFileWatchUnavailable, i18n.Vars{"error": err},
		))
	}
}
