package ui

import (
	"cmp"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/view"
)

type (
	docSnap struct{ path, text string }

	filePickerSource struct {
		pickerMeta
		dir string
	}

	pickerWalker struct {
		root     string
		rootReal string
		seen     map[string]bool
		done     <-chan struct{}
		visit    func(path, rel string) bool
	}

	pickerFollow struct {
		dir  bool
		file bool
	}
)

const (
	pickerFeedBatchSize = 256
	pickerFeedFlushWait = 40 * time.Millisecond
	pickerMaxPreview    = 10 * 1024 * 1024
)

func (f *filePickerSource) Load(
	e *view.Editor,
) ([]PickerItem, <-chan PickerItem, StopFunc) {
	return startFilePickerFeed(f.dir, pickerListRows(e))
}

func (f *filePickerSource) Match(
	query string, item PickerItem,
) (int, []int, bool) {
	return fuzzyMatchItem(query, item, f.Columns(), f.Primary())
}

func (f *filePickerSource) Accept(
	e *view.Editor, item PickerItem, action PickerAcceptAction,
) {
	path := item.Location.Target.Path
	acceptPath(e, path, action)
}

func newFilePickerSource(dir string) *filePickerSource {
	return &filePickerSource{
		pickerMeta: pickerMeta{
			title:   "Open file",
			columns: []string{"path"},
		},
		dir: dir,
	}
}

func startFilePickerFeed(
	root string, count int,
) ([]PickerItem, <-chan PickerItem, StopFunc) {
	root = resolvePickerWalkRoot(root)
	done := make(chan struct{})
	var once sync.Once
	cancel := func() { once.Do(func() { close(done) }) }

	ch := make(chan PickerItem, 256)
	go func() {
		defer close(ch)
		walkPickerFiles(root, done, func(path, rel string) bool {
			select {
			case ch <- PickerItem{
				Display:  rel,
				Location: PickerLocation{Target: PickerTarget{Path: path}},
			}:
				return true
			case <-done:
				return false
			}
		})
	}()

	var initial []PickerItem
	for len(initial) < count {
		item, ok := <-ch
		if !ok {
			sortPickerItems(initial)
			return initial, nil, cancel
		}
		initial = append(initial, item)
	}
	for {
		select {
		case item, ok := <-ch:
			if !ok {
				sortPickerItems(initial)
				return initial, nil, cancel
			}
			initial = append(initial, item)
		default:
			sortPickerItems(initial)
			return initial, ch, cancel
		}
	}
}

func resolvePickerWalkRoot(root string) string {
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return root
	}
	if fi, err := os.Stat(resolved); err == nil && fi.IsDir() {
		return resolved
	}
	return root
}

func sortPickerItems(items []PickerItem) {
	slices.SortStableFunc(items, func(a, b PickerItem) int {
		return cmp.Compare(a.Display, b.Display)
	})
}

func looksBinary(data []byte) bool {
	return slices.Contains(data[:min(len(data), 1024)], 0)
}

func pickerListRows(e *view.Editor) int {
	h := e.ViewHeight() + 1
	areaH := max((h-2)*90/100, 0)
	return max(areaH-4, 1)
}

func walkPickerFiles(
	root string, done <-chan struct{}, visit func(path, rel string) bool,
) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		rootAbs = root
	}
	rootReal, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		rootReal = rootAbs
	}
	w := &pickerWalker{
		root: root, rootReal: rootReal, seen: map[string]bool{},
		done: done, visit: visit,
	}
	w.walkDir(root)
}

func (w *pickerWalker) walkDir(dir string) bool {
	realDir, err := filepath.EvalSymlinks(dir)
	if err == nil {
		if w.seen[realDir] {
			return true
		}
		w.seen[realDir] = true
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return true
	}
	for _, entry := range entries {
		select {
		case <-w.done:
			return false
		default:
		}
		path := filepath.Join(dir, entry.Name())
		rel, err := filepath.Rel(w.root, path)
		if err != nil {
			continue
		}
		rel = filepath.ToSlash(rel)
		ignoreOpts := defaultPickerIgnoreOptions()
		if skipPickerPath(skipPickerPathArgs{
			rel: rel, path: path, entry: entry,
			ignores: loadIgnoreFiles(w.root, path, ignoreOpts),
			opts:    ignoreOpts,
		}) {
			continue
		}
		if !w.walkEntry(path, rel, entry) {
			return false
		}
	}
	return true
}

func (w *pickerWalker) walkEntry(path, rel string, entry os.DirEntry) bool {
	follow, err := pickerFollowEntry(path, w.rootReal, entry)
	if err != nil || (!follow.dir && !follow.file) {
		return true
	}
	if follow.dir {
		return w.walkDir(path)
	}
	if follow.file {
		return w.visit(path, rel)
	}
	return true
}

// FilePicker opens a file picker rooted at the workspace directory
func FilePicker(e *view.Editor) *Picker {
	root, _ := loader.FindWorkspace(e.Cwd())
	return NewPicker(e, newFilePickerSource(root))
}

// FilePickerInCWD opens a file picker rooted at the current working directory
func FilePickerInCWD(e *view.Editor) *Picker {
	return NewPicker(e, newFilePickerSource(e.Cwd()))
}

// FilePickerInDir opens a file picker rooted at the given directory
func FilePickerInDir(dir string) PickerFunc {
	return func(e *view.Editor) *Picker {
		return NewPicker(e, newFilePickerSource(dir))
	}
}

func pickerFollowEntry(
	path, rootReal string, entry os.DirEntry,
) (pickerFollow, error) {
	info, err := os.Stat(path)
	if err != nil {
		return pickerFollow{}, err
	}
	if entry.Type()&os.ModeSymlink != 0 {
		realPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			return pickerFollow{}, err
		}
		if pathWithinRoot(realPath, rootReal) {
			return pickerFollow{}, nil
		}
	}
	return pickerFollow{dir: info.IsDir(), file: info.Mode().IsRegular()}, nil
}

func pathWithinRoot(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, "../")
}
