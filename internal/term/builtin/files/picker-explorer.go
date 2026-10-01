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
		IgnoreFiles    bool
		FlattenDirs    bool
	}

	fileExplorerSource struct {
		ui.PickerBase
		root string
		opts FileExplorerOptions
	}
)

const (
	explorerDirScope = "ui.text.directory"
	explorerParent   = "../"
)

const (
	explorerParentRank = iota
	explorerDirRank
	explorerFileRank
)

// NewFileExplorer opens a file explorer rooted at the editor's working
// directory
func NewFileExplorer(e *view.Editor, opts FileExplorerOptions) *ui.Picker {
	return ui.NewPicker(e, newFileExplorerSource(e, e.Cwd(), opts))
}

// NewFocusedPaneDirExplorer opens a file explorer rooted at the focused pane's
// path directory, falling back to the working directory
func NewFocusedPaneDirExplorer(
	e *view.Editor, opts FileExplorerOptions,
) *ui.Picker {
	return ui.NewPicker(e, newFileExplorerSource(e, focusedPaneDir(e), opts))
}

// DefaultFileExplorerOptions returns the explorer's out-of-the-box behavior
func DefaultFileExplorerOptions() FileExplorerOptions {
	return FileExplorerOptions{FlattenDirs: true}
}

func newFileExplorerSource(
	e *view.Editor, root string, opts FileExplorerOptions,
) *fileExplorerSource {
	return &fileExplorerSource{
		Editor: e,
		Ident:  "file-explorer",
		Label:  "File Explorer",
		Cols:   []string{"name"},
		Scope:  filepath.Clean(root),
		Order:  compareExplorerItems,
		root:   root,
		opts:   opts,
	}
}

// Load lists the entries of the current directory
func (f *fileExplorerSource) Load() ui.PickerLoad {
	items, _ := f.readDir()
	return ui.PickerLoad{Items: items, Stop: func() {}}
}

// Items returns the current directory entries for an existing picker
func (f *fileExplorerSource) Items() ([]*ui.PickerItem, bool) {
	return f.readDir()
}

// Accept opens the chosen file, or descends into a directory
func (f *fileExplorerSource) Accept(
	item *ui.PickerItem, action ui.PickerAcceptAction,
) {
	path := item.Location.Target.Path
	if path == "" {
		return
	}
	ui.GotoPath(f.Editor, path, nil, action)
}

// Navigate moves the explorer to another directory
func (f *fileExplorerSource) Navigate(item *ui.PickerItem) ui.PickerFunc {
	path := item.Location.Target.Path
	if !item.Directory || path == "" {
		return nil
	}
	dir, _ := filepath.Abs(path)
	return func(e *view.Editor) *ui.Picker {
		return ui.NewPicker(e, newFileExplorerSource(e, dir, f.opts))
	}
}

func (f *fileExplorerSource) readDir() ([]*ui.PickerItem, bool) {
	entries, err := os.ReadDir(f.root)
	if err != nil {
		return nil, false
	}
	var items []*ui.PickerItem
	var slab ui.PickerItemSlab
	parent, _ := filepath.Abs(filepath.Join(f.root, ".."))
	if parent != f.root {
		items = append(items, f.makeDirItem(makeDirItemArgs{
			slab:    &slab,
			display: explorerParent,
			path:    parent,
		}))
	}
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
		if !pickerDirEntryIsDir(entry, full, f.opts.FollowSymlinks) {
			items = append(items, f.makeFileItem(&slab, full))
			continue
		}
		if f.opts.FlattenDirs {
			full = flattenExplorerDir(full, f.opts.FollowSymlinks)
		}
		dirRel, err := filepath.Rel(f.root, full)
		if err != nil {
			dirRel = filepath.Base(full)
		}
		items = append(items, f.makeDirItem(makeDirItemArgs{
			slab:    &slab,
			display: filepath.ToSlash(dirRel) + "/",
			path:    full,
		}))
	}
	slices.SortStableFunc(items, compareExplorerItems)
	return items, true
}

type makeDirItemArgs struct {
	slab    *ui.PickerItemSlab
	display string
	path    string
}

func (f *fileExplorerSource) makeDirItem(args makeDirItemArgs) *ui.PickerItem {
	return args.slab.Add(ui.PickerItem{
		Display:     args.display,
		Columns:     []string{args.display},
		Content:     args.display,
		StyleScopes: []string{explorerDirScope},
		Directory:   true,
		Location: ui.PickerLocation{
			Target: ui.PickerTarget{Path: args.path},
		},
	})
}

func (f *fileExplorerSource) makeFileItem(
	slab *ui.PickerItemSlab, path string,
) *ui.PickerItem {
	name := filepath.Base(path)
	return slab.Add(ui.PickerItem{
		Display:  name,
		Columns:  []string{name},
		Content:  name,
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

func compareExplorerItems(a, b *ui.PickerItem) int {
	if c := cmp.Compare(explorerRank(a), explorerRank(b)); c != 0 {
		return c
	}
	return cmp.Compare(a.Content, b.Content)
}

func explorerRank(item *ui.PickerItem) int {
	switch {
	case item.Content == explorerParent:
		return explorerParentRank
	case item.Directory:
		return explorerDirRank
	}
	return explorerFileRank
}
