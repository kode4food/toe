package ui

import (
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	tea "charm.land/bubbletea/v2"
	"github.com/rjeczalik/notify"

	"github.com/kode4food/toe/internal/view"
)

type (
	fileWatcher struct {
		mu sync.Mutex

		dirs       map[string]*watchRegistration
		wantedDirs map[string]int

		tree           *watchRegistration
		treeRoot       string
		wantedTreeRoot string

		events chan fileWatchEvent
		done   chan struct{}

		enabled atomic.Bool
		closed  bool
	}

	fileWatchEvent struct {
		path         string
		registration *watchRegistration
	}

	watchRegistration struct {
		events chan notify.EventInfo
		done   chan struct{}
		active atomic.Bool
	}

	externalFileChangedMsg struct {
		path string
	}
)

func newFileWatcher() *fileWatcher {
	w := &fileWatcher{
		dirs:       map[string]*watchRegistration{},
		wantedDirs: map[string]int{},
		events:     make(chan fileWatchEvent, 64),
		done:       make(chan struct{}),
	}
	w.enabled.Store(true)
	return w
}

func (w *fileWatcher) sync(e *view.Editor) {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return
	}
	wanted := map[string]int{}
	for _, doc := range e.AllDocuments() {
		addWatchDir(wanted, doc.Path())
	}
	rangeImagePanes(e, func(img *ImagePane) {
		addWatchDir(wanted, img.Path())
	})
	w.wantedDirs = wanted
	w.enabled.Store(e.Options().FileWatch)
	w.reconcileDirs()
	w.reconcileTree()
}

func (w *fileWatcher) reconcileDirs() {
	for dir := range w.dirs {
		if !w.enabled.Load() || w.wantedDirs[dir] == 0 {
			w.unwatchDir(dir)
		}
	}
	if !w.enabled.Load() {
		return
	}
	for dir := range w.wantedDirs {
		if _, ok := w.dirs[dir]; !ok {
			w.watchDir(dir)
		}
	}
}

func (w *fileWatcher) watchDir(dir string) {
	reg, ok := w.startWatch(dir)
	if !ok {
		return
	}
	w.dirs[dir] = reg
}

func (w *fileWatcher) unwatchDir(dir string) {
	reg := w.dirs[dir]
	delete(w.dirs, dir)
	stopWatch(reg)
}

func (w *fileWatcher) setTreeWanted(e *view.Editor, want bool) {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return
	}
	w.wantedTreeRoot = ""
	if want {
		w.wantedTreeRoot = e.Cwd()
		resolved, err := filepath.EvalSymlinks(w.wantedTreeRoot)
		if err == nil {
			w.wantedTreeRoot = resolved
		}
	}
	w.reconcileTree()
}

func (w *fileWatcher) reconcileTree() {
	root := w.wantedTreeRoot
	if !w.enabled.Load() {
		root = ""
	}
	if w.treeRoot == root {
		return
	}
	w.stopTreeWatch()
	if root == "" {
		return
	}
	tree := filepath.Join(root, "...")
	reg, ok := w.startWatch(tree)
	if !ok {
		return
	}
	w.tree = reg
	w.treeRoot = root
}

func (w *fileWatcher) stopTreeWatch() {
	if w.tree == nil {
		return
	}
	stopWatch(w.tree)
	w.tree = nil
	w.treeRoot = ""
}

func (w *fileWatcher) close() {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return
	}
	w.closed = true
	w.enabled.Store(false)
	close(w.done)
	w.stopTreeWatch()
	for dir := range w.dirs {
		w.unwatchDir(dir)
	}
	w.wantedDirs = nil
	w.wantedTreeRoot = ""
}

func (w *fileWatcher) nextCmd(e *view.Editor) tea.Cmd {
	if w == nil {
		return nil
	}
	w.sync(e)
	return func() tea.Msg {
		for {
			select {
			case ev := <-w.events:
				if w.enabled.Load() &&
					ev.registration.active.Load() {
					return externalFileChangedMsg{path: ev.path}
				}
			case <-w.done:
				return nil
			}
		}
	}
}

func (w *fileWatcher) startWatch(path string) (*watchRegistration, bool) {
	reg := &watchRegistration{
		events: make(chan notify.EventInfo, 256),
		done:   make(chan struct{}),
	}
	if err := notify.Watch(path, reg.events, notify.All); err != nil {
		return nil, false
	}
	reg.active.Store(true)
	go w.drain(reg)
	return reg, true
}

func (w *fileWatcher) drain(reg *watchRegistration) {
	for {
		var ev notify.EventInfo
		select {
		case ev = <-reg.events:
		case <-reg.done:
			return
		case <-w.done:
			return
		}
		if !w.enabled.Load() {
			continue
		}
		if !isFileWatchOp(ev.Event()) || isFileWatchPathExcluded(ev.Path()) {
			continue
		}
		select {
		case w.events <- fileWatchEvent{
			path:         ev.Path(),
			registration: reg,
		}:
		case <-reg.done:
			return
		case <-w.done:
			return
		}
	}
}

func addWatchDir(dirs map[string]int, path string) {
	if path != "" {
		dirs[filepath.Dir(path)]++
	}
}

func stopWatch(reg *watchRegistration) {
	reg.active.Store(false)
	notify.Stop(reg.events)
	close(reg.done)
}

func isFileWatchOp(ev notify.Event) bool {
	return ev&(notify.Create|notify.Write|notify.Remove|notify.Rename) != 0
}

func isFileWatchPathExcluded(path string) bool {
	if isGitIndexPath(path) {
		return false
	}
	sep := string(filepath.Separator)
	return strings.Contains(path, sep+".git"+sep) ||
		strings.HasSuffix(path, sep+".git")
}

func isGitIndexPath(path string) bool {
	sep := string(filepath.Separator)
	return strings.HasSuffix(path, sep+".git"+sep+"index")
}
