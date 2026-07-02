package ui

import (
	"cmp"
	"os"
	"path/filepath"
	"slices"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type (
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
		pickerMeta
		root     string
		opts     FileExplorerOptions
		dirStyle lipgloss.Style
	}

	explorerEntry struct {
		path  string
		isDir bool
	}
)

func NewFileExplorer(e *view.Editor, opts FileExplorerOptions) *Picker {
	return NewPicker(e, newFileExplorerSource(e.Cwd(), opts))
}

func NewBufferDirExplorer(e *view.Editor, opts FileExplorerOptions) *Picker {
	dir := e.Cwd()
	if doc, ok := e.FocusedDocument(); ok && doc.Path() != "" {
		dir = filepath.Dir(doc.Path())
	}
	return NewPicker(e, newFileExplorerSource(dir, opts))
}

func DefaultFileExplorerOptions() FileExplorerOptions {
	return FileExplorerOptions{FlattenDirs: true}
}

func (f *fileExplorerSource) Title() string {
	return filepath.Base(f.root)
}

func (f *fileExplorerSource) Load(
	e *view.Editor,
) ([]PickerItem, <-chan PickerItem, StopFunc) {
	if th, _, err := theme.Load(e.Options().Theme); err == nil {
		f.dirStyle = th.Get("ui.text.directory")
	}
	return f.readDir(), nil, func() {}
}

func (f *fileExplorerSource) Match(
	query string, item PickerItem,
) (int, []int, bool) {
	return fuzzyMatchItem(query, item, f.Columns(), f.Primary())
}

func (f *fileExplorerSource) Accept(
	e *view.Editor, item PickerItem, action PickerAcceptAction,
) {
	path := item.Location.Target.Path
	if path == "" {
		return
	}
	acceptPath(e, path, action)
}

func (f *fileExplorerSource) Navigate(
	_ *view.Editor, item PickerItem,
) PickerFunc {
	entry, ok := item.Payload.(explorerEntry)
	if !ok || !entry.isDir {
		return nil
	}
	dir, _ := filepath.Abs(entry.path)
	return func(e *view.Editor) *Picker {
		return NewPicker(e, newFileExplorerSource(dir, f.opts))
	}
}

func newFileExplorerSource(
	root string, opts FileExplorerOptions,
) *fileExplorerSource {
	return &fileExplorerSource{
		pickerMeta: pickerMeta{columns: []string{"name"}},
		root:       root,
		opts:       opts,
	}
}

func (f *fileExplorerSource) readDir() []PickerItem {
	entries, err := os.ReadDir(f.root)
	if err != nil {
		return nil
	}
	var dirs, files []explorerEntry
	for _, entry := range entries {
		full := filepath.Join(f.root, entry.Name())
		rel := filepath.ToSlash(entry.Name())
		ignoreOpts := explorerIgnoreOptions(f.opts)
		if skipPickerPath(skipPickerPathArgs{
			rel: rel, path: full, entry: entry,
			ignores: loadIgnoreFiles(f.root, full, ignoreOpts),
			opts:    ignoreOpts,
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

	var items []PickerItem
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

func explorerIgnoreOptions(cfg FileExplorerOptions) pickerIgnoreOptions {
	return pickerIgnoreOptions{
		hidden: cfg.Hidden, parents: cfg.Parents, ignore: cfg.Ignore,
		gitIgnore: cfg.GitIgnore, gitGlobal: cfg.GitGlobal,
		gitExclude: cfg.GitExclude,
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

func (f *fileExplorerSource) makeDirItem(display, path string) PickerItem {
	dirStyle := f.dirStyle
	return PickerItem{
		Display: display,
		SortKey: display,
		Style:   dirStyle,
		Payload: explorerEntry{
			path:  path,
			isDir: true,
		},
		Preview: func(w, h int) string {
			return explorerDirPreview(path, w, h, dirStyle)
		},
	}
}

func (f *fileExplorerSource) makeFileItem(path string) PickerItem {
	return PickerItem{
		Display:  filepath.Base(path),
		SortKey:  filepath.Base(path),
		Location: PickerLocation{Target: PickerTarget{Path: path}},
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

func explorerDirPreview(dir string, w, h int, dirStyle lipgloss.Style) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var dirs, files []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name()+"/")
		} else {
			files = append(files, e.Name())
		}
	}
	slices.Sort(dirs)
	slices.Sort(files)

	all := append(dirs, files...)
	if len(all) > h {
		all = all[:h]
	}
	if len(all) == 0 || w <= 0 {
		return ""
	}
	buf := tui.NewBuffer(w, len(all))
	dirTUI := lipglossToTUIStyle(
		lipgloss.NewStyle().Foreground(dirStyle.GetForeground()),
	)
	for i, name := range all {
		st := tui.Style{}
		if i < len(dirs) {
			st = dirTUI
		}
		buf.SetString(0, i, ansi.Truncate(name, w, ""), st)
	}
	return buf.RenderToANSI()
}
