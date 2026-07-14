package files

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

type (
	docSnap struct{ path, text string }

	filePickerSource struct {
		ui.PickerBase
		dir string
	}

	pickerWalker struct {
		root     string
		rootReal string
		seen     map[string]bool
		done     <-chan struct{}
		visit    pickerVisitor
	}

	pickerFollow struct {
		dir  bool
		file bool
	}

	pickerVisitor func(path, rel string) bool
)

// NewFilePicker opens a file picker rooted at the workspace directory
func NewFilePicker(e *view.Editor) *ui.Picker {
	root, _ := loader.FindWorkspace(e.Cwd())
	return ui.NewPicker(e, newFilePickerSource(root))
}

// NewFilePickerInCWD opens a file picker rooted at the current working
// directory
func NewFilePickerInCWD(e *view.Editor) *ui.Picker {
	return ui.NewPicker(e, newFilePickerSource(e.Cwd()))
}

// NewFilePickerInDir opens a file picker rooted at the given directory
func NewFilePickerInDir(dir string) ui.PickerFunc {
	return func(e *view.Editor) *ui.Picker {
		return ui.NewPicker(e, newFilePickerSource(dir))
	}
}

func (f *filePickerSource) Load(
	e *view.Editor,
) ([]ui.PickerItem, <-chan ui.PickerItem, ui.StopFunc) {
	return startFilePickerFeed(f.dir, pickerListRows(e))
}

func (f *filePickerSource) Accept(
	e *view.Editor, item ui.PickerItem, action ui.PickerAcceptAction,
) {
	path := item.Location.Target.Path
	ui.AcceptPath(e, path, action)
}

func newFilePickerSource(dir string) *filePickerSource {
	return &filePickerSource{
		PickerBase: ui.NewPickerBase("open-file", []string{"path"}, 0, nil),
		dir:        dir,
	}
}

func startFilePickerFeed(
	root string, count int,
) ([]ui.PickerItem, <-chan ui.PickerItem, ui.StopFunc) {
	root = resolvePickerWalkRoot(root)
	done := make(chan struct{})
	var once sync.Once
	cancel := func() { once.Do(func() { close(done) }) }

	ch := make(chan ui.PickerItem, 256)
	go func() {
		defer close(ch)
		walkPickerFiles(root, done, func(path, rel string) bool {
			select {
			case ch <- ui.PickerItem{
				Display:  rel,
				Location: ui.PickerLocation{Target: ui.PickerTarget{Path: path}},
			}:
				return true
			case <-done:
				return false
			}
		})
	}()

	var initial []ui.PickerItem
	for len(initial) < count {
		item, ok := <-ch
		if !ok {
			ui.SortPickerItems(initial)
			return initial, nil, cancel
		}
		initial = append(initial, item)
	}
	for {
		select {
		case item, ok := <-ch:
			if !ok {
				ui.SortPickerItems(initial)
				return initial, nil, cancel
			}
			initial = append(initial, item)
		default:
			ui.SortPickerItems(initial)
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

func pickerListRows(e *view.Editor) int {
	h := e.ViewHeight() + 1
	areaH := max((h-2)*90/100, 0)
	return max(areaH-4, 1)
}

func walkPickerFiles(root string, done <-chan struct{}, visit pickerVisitor) {
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
		ignoreOpts := ui.DefaultPickerIgnoreOptions()
		if ui.SkipPickerPath(ui.SkipPickerPathArgs{
			Rel: rel, Path: path, Entry: entry,
			Ignores: ui.LoadIgnoreFiles(w.root, path, ignoreOpts),
			Opts:    ignoreOpts,
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
