package files

import (
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
		IgnoreFiles    bool
		FlattenDirs    bool
	}

	fileExplorerSource struct {
		ui.PickerBase
		root string
		opts FileExplorerOptions
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

// DefaultFileExplorerOptions returns the explorer's out-of-the-box behavior
func DefaultFileExplorerOptions() FileExplorerOptions {
	return FileExplorerOptions{FlattenDirs: true}
}

func newFileExplorerSource(
	root string, opts FileExplorerOptions,
) *fileExplorerSource {
	return &fileExplorerSource{
		PickerBase: ui.PickerBase{
			Ident: "file-explorer",
			Label: "File Explorer",
			Cols:  []string{"name"},
		},
		root: root,
		opts: opts,
	}
}

// Load lists the entries of the current directory
func (f *fileExplorerSource) Load(*view.Editor) ui.PickerLoad {
	return ui.PickerLoad{Items: f.readDir(), Stop: func() {}}
}

// Accept opens the chosen file, or descends into a directory
func (f *fileExplorerSource) Accept(
	e *view.Editor, item *ui.PickerItem, action ui.PickerAcceptAction,
) {
	path := item.Location.Target.Path
	if path == "" {
		return
	}
	ui.AcceptPath(e, path, action)
}

// Navigate moves the explorer to another directory
func (f *fileExplorerSource) Navigate(
	_ *view.Editor, item *ui.PickerItem,
) ui.PickerFunc {
	path := item.Location.Target.Path
	if !item.Directory || path == "" {
		return nil
	}
	dir, _ := filepath.Abs(path)
	return func(e *view.Editor) *ui.Picker {
		return ui.NewPicker(e, newFileExplorerSource(dir, f.opts))
	}
}

func (f *fileExplorerSource) readDir() []*ui.PickerItem {
	entries, err := os.ReadDir(f.root)
	if err != nil {
		return nil
	}
	var dirs, files []string
	for _, entry := range entries {
		full := filepath.Join(f.root, entry.Name())
		rel := filepath.ToSlash(entry.Name())
		ignoreOpts := explorerIgnoreOptions(f.opts)
		if ui.SkipPickerPath(ui.SkipPickerPathArgs{
			Rel:   rel,
			Path:  full,
			Entry: entry,
			Ignores: ui.LoadIgnoreFiles(ui.IgnoreTarget{
				Root: f.root,
				Path: full,
			}, ignoreOpts),
			Opts: ignoreOpts,
		}) {
			continue
		}
		if pickerDirEntryIsDir(entry, full, f.opts.FollowSymlinks) {
			if f.opts.FlattenDirs {
				full = flattenExplorerDir(full, f.opts.FollowSymlinks)
			}
			dirs = append(dirs, full)
		} else {
			files = append(files, full)
		}
	}
	slices.Sort(dirs)
	slices.Sort(files)

	var items []*ui.PickerItem
	var slab ui.PickerItemSlab
	parent, _ := filepath.Abs(filepath.Join(f.root, ".."))
	if parent != f.root {
		items = append(items, f.makeDirItem(makeDirItemArgs{
			slab:    &slab,
			display: "../",
			path:    parent,
		}))
	}
	for _, path := range dirs {
		rel, err := filepath.Rel(f.root, path)
		if err != nil {
			rel = filepath.Base(path)
		}
		items = append(items, f.makeDirItem(makeDirItemArgs{
			slab:    &slab,
			display: filepath.ToSlash(rel) + "/",
			path:    path,
		}))
	}
	for _, path := range files {
		items = append(items, f.makeFileItem(&slab, path))
	}
	return items
}

type makeDirItemArgs struct {
	slab    *ui.PickerItemSlab
	display string
	path    string
}

func (f *fileExplorerSource) makeDirItem(args makeDirItemArgs) *ui.PickerItem {
	slab := args.slab
	display := args.display
	path := args.path
	return slab.Add(ui.PickerItem{
		Display:     display,
		SortKey:     display,
		StyleScopes: []string{explorerDirScope},
		Directory:   true,
		Location:    ui.PickerLocation{Target: ui.PickerTarget{Path: path}},
	})
}

func (f *fileExplorerSource) makeFileItem(
	slab *ui.PickerItemSlab, path string,
) *ui.PickerItem {
	return slab.Add(ui.PickerItem{
		Display:  filepath.Base(path),
		SortKey:  filepath.Base(path),
		Location: ui.PickerLocation{Target: ui.PickerTarget{Path: path}},
	})
}

func focusedPaneDir(e *view.Editor) string {
	if p := e.FocusedPane(); p != nil {
		if path := p.Path(); path != "" {
			return filepath.Dir(path)
		}
	}
	return e.Cwd()
}

func explorerIgnoreOptions(cfg FileExplorerOptions) ui.PickerIgnoreOptions {
	return ui.PickerIgnoreOptions{
		Hidden:      cfg.Hidden,
		Parents:     cfg.Parents,
		IgnoreFiles: cfg.IgnoreFiles,
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
