package lsp

import (
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

func (s *Session) ensureFileWatcher() {
	roots := s.fileWatchRoots()
	if len(roots) == 0 {
		return
	}
	s.watch.Lock()
	if s.watch.watcher == nil {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			s.watch.Unlock()
			if s.editor != nil {
				s.editor.SetStatusMsg(
					"file watching unavailable: " + err.Error(),
				)
			}
			return
		}
		state := &fsWatcher{
			watcher: watcher,
			done:    make(chan struct{}),
			dirs:    map[string]struct{}{},
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
	_ = state.watcher.Close()
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
	s.watch.Lock()
	state := s.watch.watcher
	if state == nil {
		s.watch.Unlock()
		return
	}
	if _, ok := state.dirs[path]; ok {
		s.watch.Unlock()
		return
	}
	state.dirs[path] = struct{}{}
	s.watch.Unlock()
	if err := state.watcher.Add(path); err != nil {
		s.watch.Lock()
		delete(state.dirs, path)
		s.watch.Unlock()
	}
}
