package files

import (
	"io/fs"
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

	// walkedFile is a file the walk reached, named both absolutely and
	// relative to the walk root
	walkedFile struct {
		path string
		rel  string
	}

	// rootedPath is a path together with the root it is resolved against
	rootedPath struct {
		path string
		root string
	}

	pickerVisitor func(file walkedFile) bool
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

func newFilePickerSource(dir string) *filePickerSource {
	return &filePickerSource{
		PickerBase: ui.PickerBase{
			Ident: "open-file",
			Label: "Open File",
			Cols:  []string{""},
		},
		dir: dir,
	}
}

// Load walks the workspace for files, honouring ignore rules
func (f *filePickerSource) Load(e *view.Editor) ui.PickerLoad {
	return startFilePickerFeed(f.dir, pickerListRows(e))
}

// ItemForPath returns the row for a regular file at path and whether the walk
// includes it. Symlinks resolve on a full reload
func (f *filePickerSource) ItemForPath(
	_ *view.Editor, path string,
) (*ui.PickerItem, bool) {
	root := resolvePickerWalkRoot(f.dir)
	if !pathWithinRoot(rootedPath{path: path, root: root}) {
		return nil, false
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, false
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return nil, false
	}
	rel = filepath.ToSlash(rel)
	ignoreOpts := ui.DefaultPickerIgnoreOptions()
	if ui.SkipPickerPath(ui.SkipPickerPathArgs{
		Rel:   rel,
		Path:  path,
		Entry: fs.FileInfoToDirEntry(info),
		Ignores: ui.LoadIgnoreFiles(ui.IgnoreTarget{
			Root: root,
			Path: path,
		}, ignoreOpts),
		Opts: ignoreOpts,
	}) {
		return nil, false
	}
	lbl, sec := ui.PickerNamePath(rel)
	return &ui.PickerItem{
		Display:  lbl,
		Columns:  []string{lbl},
		SortKey:  rel,
		SecFrom:  sec,
		Location: ui.PickerLocation{Target: ui.PickerTarget{Path: path}},
	}, true
}

// Accept opens the chosen file
func (f *filePickerSource) Accept(
	e *view.Editor, item *ui.PickerItem, action ui.PickerAcceptAction,
) {
	ui.GotoPath(e, item.Location.Target.Path, nil, action)
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
			Rel:   rel,
			Path:  path,
			Entry: entry,
			Ignores: ui.LoadIgnoreFiles(ui.IgnoreTarget{
				Root: w.root,
				Path: path,
			}, ignoreOpts),
			Opts: ignoreOpts,
		}) {
			continue
		}
		if !w.walkEntry(walkedFile{path: path, rel: rel}, entry) {
			return false
		}
	}
	return true
}

func (w *pickerWalker) walkEntry(file walkedFile, entry os.DirEntry) bool {
	follow, err := pickerFollowEntry(rootedPath{
		path: file.path,
		root: w.rootReal,
	}, entry)
	if err != nil || (!follow.dir && !follow.file) {
		return true
	}
	if follow.dir {
		return w.walkDir(file.path)
	}
	if follow.file {
		return w.visit(file)
	}
	return true
}

func startFilePickerFeed(root string, count int) ui.PickerLoad {
	root = resolvePickerWalkRoot(root)
	done := make(chan struct{})
	var once sync.Once
	cancel := func() { once.Do(func() { close(done) }) }

	ch := make(chan *ui.PickerItem, 256)
	go func() {
		defer close(ch)
		var slab ui.PickerItemSlab
		walkPickerFiles(root, done, func(file walkedFile) bool {
			lbl, sec := ui.PickerNamePath(file.rel)
			select {
			case ch <- slab.Add(ui.PickerItem{
				Display: lbl,
				Columns: []string{lbl},
				SortKey: file.rel,
				SecFrom: sec,
				Location: ui.PickerLocation{
					Target: ui.PickerTarget{Path: file.path},
				},
			}):
				return true
			case <-done:
				return false
			}
		})
	}()

	var initial []*ui.PickerItem
	for len(initial) < count {
		item, ok := <-ch
		if !ok {
			ui.SortPickerItems(initial)
			return ui.PickerLoad{Items: initial, Stop: cancel}
		}
		initial = append(initial, item)
	}
	for {
		select {
		case item, ok := <-ch:
			if !ok {
				ui.SortPickerItems(initial)
				return ui.PickerLoad{Items: initial, Stop: cancel}
			}
			initial = append(initial, item)
		default:
			ui.SortPickerItems(initial)
			return ui.PickerLoad{Items: initial, Feed: ch, Stop: cancel}
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

func pickerFollowEntry(
	at rootedPath, entry os.DirEntry,
) (pickerFollow, error) {
	info, err := os.Stat(at.path)
	if err != nil {
		return pickerFollow{}, err
	}
	if entry.Type()&os.ModeSymlink != 0 {
		realPath, err := filepath.EvalSymlinks(at.path)
		if err != nil {
			return pickerFollow{}, err
		}
		if pathWithinRoot(rootedPath{path: realPath, root: at.root}) {
			return pickerFollow{}, nil
		}
	}
	return pickerFollow{dir: info.IsDir(), file: info.Mode().IsRegular()}, nil
}

func pathWithinRoot(at rootedPath) bool {
	rel, err := filepath.Rel(at.root, at.path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, "../")
}
