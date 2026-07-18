package files

import (
	"cmp"
	"os"
	"path/filepath"
	"slices"

	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

type (
	// FileExplorerOptions controls which entries a file explorer lists
	FileExplorerOptions struct {
		Hidden         bool
		FollowSymlinks bool
		Parents        bool
		Ignore         bool
		GitIgnore      bool
		GitGlobal      bool
		GitExclude     bool
		FlattenDirs    bool
	}

	fileExplorerSource struct {
		ui.PickerBase
		root string
		opts FileExplorerOptions
	}

	explorerEntry struct {
		path  string
		isDir bool
	}
)

const explorerDirScope = "ui.text.directory"

// NewFileExplorer opens a file explorer rooted at the editor's working
// directory
func NewFileExplorer(e *view.Editor, opts FileExplorerOptions) *ui.Picker {
	return ui.NewPicker(e, newFileExplorerSource(e.Cwd(), opts))
}

// NewFocusedPaneDirExplorer opens a file explorer rooted at the focused pane's
// path directory, falling back to the working directory
func NewFocusedPaneDirExplorer(
	e *view.Editor, opts FileExplorerOptions,
) *ui.Picker {
	return ui.NewPicker(e, newFileExplorerSource(focusedPaneDir(e), opts))
}

func focusedPaneDir(e *view.Editor) string {
	if p := e.FocusedPane(); p != nil {
		if path := p.Path(); path != "" {
			return filepath.Dir(path)
		}
	}
	return e.Cwd()
}

// DefaultFileExplorerOptions returns the explorer's out-of-the-box behavior
func DefaultFileExplorerOptions() FileExplorerOptions {
	return FileExplorerOptions{FlattenDirs: true}
}

func (f *fileExplorerSource) Load(
	_ *view.Editor,
) ([]ui.PickerItem, <-chan ui.PickerItem, ui.StopFunc) {
	return f.readDir(), nil, func() {}
}

func (f *fileExplorerSource) Accept(
	e *view.Editor, item ui.PickerItem, action ui.PickerAcceptAction,
) {
	path := item.Location.Target.Path
	if path == "" {
		return
	}
	ui.AcceptPath(e, path, action)
}

func (f *fileExplorerSource) Navigate(
	_ *view.Editor, item ui.PickerItem,
) ui.PickerFunc {
	entry, ok := item.Payload.(explorerEntry)
	if !ok || !entry.isDir {
		return nil
	}
	dir, _ := filepath.Abs(entry.path)
	return func(e *view.Editor) *ui.Picker {
		return ui.NewPicker(e, newFileExplorerSource(dir, f.opts))
	}
}

func newFileExplorerSource(
	root string, opts FileExplorerOptions,
) *fileExplorerSource {
	return &fileExplorerSource{
		PickerBase: ui.NewPickerBase("file-explorer", []string{"name"}, 0, nil),
		root:       root,
		opts:       opts,
	}
}

func (f *fileExplorerSource) readDir() []ui.PickerItem {
	entries, err := os.ReadDir(f.root)
	if err != nil {
		return nil
	}
	var dirs, files []explorerEntry
	for _, entry := range entries {
		full := filepath.Join(f.root, entry.Name())
		rel := filepath.ToSlash(entry.Name())
		ignoreOpts := explorerIgnoreOptions(f.opts)
		if ui.SkipPickerPath(ui.SkipPickerPathArgs{
			Rel: rel, Path: full, Entry: entry,
			Ignores: ui.LoadIgnoreFiles(f.root, full, ignoreOpts),
			Opts:    ignoreOpts,
		}) {
			continue
		}
		if pickerDirEntryIsDir(entry, full, f.opts.FollowSymlinks) {
			if f.opts.FlattenDirs {
				full = flattenExplorerDir(full, f.opts.FollowSymlinks)
			}
			dirs = append(dirs, explorerEntry{
				path:  full,
				isDir: true,
			})
		} else {
			files = append(files, explorerEntry{
				path:  full,
				isDir: false,
			})
		}
	}
	slices.SortFunc(dirs, func(a, b explorerEntry) int {
		return cmp.Compare(a.path, b.path)
	})
	slices.SortFunc(files, func(a, b explorerEntry) int {
		return cmp.Compare(a.path, b.path)
	})

	var items []ui.PickerItem
	parent, _ := filepath.Abs(filepath.Join(f.root, ".."))
	if parent != f.root {
		items = append(items, f.makeDirItem("../", parent))
	}
	for _, e := range dirs {
		rel, err := filepath.Rel(f.root, e.path)
		if err != nil {
			rel = filepath.Base(e.path)
		}
		items = append(items, f.makeDirItem(filepath.ToSlash(rel)+"/", e.path))
	}
	for _, e := range files {
		items = append(items, f.makeFileItem(e.path))
	}
	return items
}

func explorerIgnoreOptions(cfg FileExplorerOptions) ui.PickerIgnoreOptions {
	return ui.PickerIgnoreOptions{
		Hidden: cfg.Hidden, Parents: cfg.Parents, Ignore: cfg.Ignore,
		GitIgnore: cfg.GitIgnore, GitGlobal: cfg.GitGlobal,
		GitExclude: cfg.GitExclude,
	}
}

func pickerDirEntryIsDir(
	entry os.DirEntry, path string, followSymlinks bool,
) bool {
	if entry.IsDir() {
		return true
	}
	if !followSymlinks || entry.Type()&os.ModeSymlink == 0 {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func (f *fileExplorerSource) makeDirItem(display, path string) ui.PickerItem {
	return ui.PickerItem{
		Display:     display,
		SortKey:     display,
		StyleScopes: []string{explorerDirScope},
		Payload: explorerEntry{
			path:  path,
			isDir: true,
		},
		Location: ui.PickerLocation{Target: ui.PickerTarget{Path: path}},
	}
}

func (f *fileExplorerSource) makeFileItem(path string) ui.PickerItem {
	return ui.PickerItem{
		Display:  filepath.Base(path),
		SortKey:  filepath.Base(path),
		Location: ui.PickerLocation{Target: ui.PickerTarget{Path: path}},
	}
}

func flattenExplorerDir(path string, followSymlinks bool) string {
	for {
		next, ok := singleChildExplorerDir(path, followSymlinks)
		if !ok {
			return path
		}
		path = next
	}
}

func singleChildExplorerDir(path string, followSymlinks bool) (string, bool) {
	entries, err := os.ReadDir(path)
	if err != nil || len(entries) != 1 {
		return "", false
	}
	next := filepath.Join(path, entries[0].Name())
	if !pickerDirEntryIsDir(entries[0], next, followSymlinks) {
		return "", false
	}
	return next, true
}
