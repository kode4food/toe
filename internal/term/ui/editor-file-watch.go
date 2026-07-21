package ui

import (
	"path/filepath"
	"sync"

	tea "charm.land/bubbletea/v2"
	"github.com/fsnotify/fsnotify"

	"github.com/kode4food/toe/internal/view"
)

type (
	editorFileWatcher struct {
		mu      sync.Mutex
		watcher *fsnotify.Watcher
		events  chan string
		dirs    map[string]struct{}
	}

	externalFileChangedMsg struct {
		path string
	}
)

func newEditorFileWatcher() *editorFileWatcher {
	return &editorFileWatcher{
		events: make(chan string, 64),
		dirs:   map[string]struct{}{},
	}
}

func (e *EditorComponent) syncFileWatcher(cx *Context) {
	if e.fileWatcher != nil {
		e.fileWatcher.sync(cx.Editor)
	}
}

func (e *EditorComponent) fileWatchCmd(cx *Context) tea.Cmd {
	if e.fileWatcher == nil {
		return nil
	}
	return e.fileWatcher.nextCmd(cx.Editor)
}

func (w *editorFileWatcher) sync(e *view.Editor) {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, doc := range e.AllDocuments() {
		w.watchPath(doc.Path())
	}
	rangeImagePanes(e, func(img *ImagePane) {
		w.watchPath(img.Path())
	})
}

// watchPath registers path's directory; the caller must hold w.mu
func (w *editorFileWatcher) watchPath(path string) {
	if path == "" {
		return
	}
	dir := filepath.Dir(path)
	if _, ok := w.dirs[dir]; ok {
		return
	}
	// The inotify instance is a limited per-user resource; open it only
	// once there is a directory to watch
	if w.watcher == nil {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return
		}
		w.watcher = watcher
		go w.run()
	}
	if err := w.watcher.Add(dir); err != nil {
		return
	}
	w.dirs[dir] = struct{}{}
}

func (w *editorFileWatcher) nextCmd(e *view.Editor) tea.Cmd {
	if w == nil {
		return nil
	}
	w.sync(e)
	return func() tea.Msg {
		return externalFileChangedMsg{path: <-w.events}
	}
}

func (w *editorFileWatcher) run() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if !fileWatchOp(event.Op) {
				continue
			}
			w.events <- event.Name
		case _, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
		}
	}
}

func fileWatchOp(op fsnotify.Op) bool {
	return op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) != 0
}
