package ui

import (
	"os"
	"path/filepath"
	"sort"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type (
	fileExplorerSource struct {
		pickerMeta
		root     string
		dirStyle lipgloss.Style
	}

	explorerEntry struct {
		path  string
		isDir bool
	}
)

// FileExplorer opens a directory listing picker rooted at the editor's cwd
func FileExplorer(e *view.Editor) *Picker {
	return NewPicker(e, newFileExplorerSource(e.Cwd()))
}

// FileExplorerInBufferDir opens a directory listing picker rooted at the
// focused buffer's directory, falling back to cwd for scratch buffers
func FileExplorerInBufferDir(e *view.Editor) *Picker {
	dir := e.Cwd()
	if doc, ok := e.FocusedDocument(); ok && doc.Path() != "" {
		dir = filepath.Dir(doc.Path())
	}
	return NewPicker(e, newFileExplorerSource(dir))
}

func newFileExplorerSource(root string) *fileExplorerSource {
	return &fileExplorerSource{
		pickerMeta: pickerMeta{columns: []string{"name"}},
		root:       root,
	}
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

func (f *fileExplorerSource) Accept(e *view.Editor, item PickerItem) {
	entry, ok := item.Payload.(explorerEntry)
	if !ok || entry.isDir {
		return
	}
	_, _ = e.OpenFile(entry.path)
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
		return NewPicker(e, newFileExplorerSource(dir))
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
		if entry.IsDir() {
			dirs = append(dirs, explorerEntry{full, true})
		} else {
			files = append(files, explorerEntry{full, false})
		}
	}
	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].path < dirs[j].path
	})
	sort.Slice(files, func(i, j int) bool {
		return files[i].path < files[j].path
	})

	var items []PickerItem
	parent, _ := filepath.Abs(filepath.Join(f.root, ".."))
	if parent != f.root {
		items = append(items, f.makeDirItem("../", parent))
	}
	for _, e := range dirs {
		items = append(items, f.makeDirItem(filepath.Base(e.path)+"/", e.path))
	}
	for _, e := range files {
		items = append(items, f.makeFileItem(e.path))
	}
	return items
}

func (f *fileExplorerSource) makeDirItem(display, path string) PickerItem {
	dirStyle := f.dirStyle
	return PickerItem{
		Display: display,
		SortKey: display,
		Style:   dirStyle,
		Payload: explorerEntry{path, true},
		Preview: func(w, h int) string {
			return explorerDirPreview(path, w, h, dirStyle)
		},
	}
}

func (f *fileExplorerSource) makeFileItem(path string) PickerItem {
	return PickerItem{
		Display:  filepath.Base(path),
		SortKey:  filepath.Base(path),
		Payload:  explorerEntry{path, false},
		Location: PickerLocation{Target: PickerTarget{Path: path}},
	}
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
	sort.Strings(dirs)
	sort.Strings(files)

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
