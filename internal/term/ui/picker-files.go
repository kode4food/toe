package ui

import (
	"cmp"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/kode4food/toe/internal/view"
)

type (
	docSnap struct{ path, text string }

	filePickerSource struct {
		pickerMeta
		dir string
	}
)

const (
	pickerFeedBatchSize = 256
	pickerFeedFlushWait = 40 * time.Millisecond
	pickerMaxPreview    = 10 * 1024 * 1024
)

// FilePicker opens a file picker rooted at the editor's working directory
func FilePicker(e *view.Editor) *Picker {
	return NewPicker(e, newFilePickerSource(e.Cwd()))
}

// FilePickerInDir opens a file picker rooted at the given directory
func FilePickerInDir(dir string) PickerFunc {
	return func(e *view.Editor) *Picker {
		return NewPicker(e, newFilePickerSource(dir))
	}
}

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

func (f *filePickerSource) Accept(e *view.Editor, item PickerItem) {
	path, _ := item.Payload.(string)
	if path != "" {
		_, _ = e.OpenFile(path)
	}
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

func hasPreview(items []PickerItem) bool {
	for _, item := range items {
		if item.Preview != nil || item.Location.Target.Valid() {
			return true
		}
	}
	return false
}

func startFilePickerFeed(
	root string, count int,
) ([]PickerItem, <-chan PickerItem, StopFunc) {
	done := make(chan struct{})
	var once sync.Once
	cancel := func() { once.Do(func() { close(done) }) }

	ch := make(chan PickerItem, 256)
	go func() {
		defer close(ch)
		_ = filepath.WalkDir(root, func(
			path string, d os.DirEntry, err error,
		) error {
			if err != nil || path == root {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return nil
			}
			rel = filepath.ToSlash(rel)
			if skipPickerPath(rel, d, loadIgnoreFiles(root, rel)) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				return nil
			}
			select {
			case ch <- PickerItem{
				Display:  rel,
				Location: PickerLocation{Target: PickerTarget{Path: path}},
				Payload:  path,
			}:
			case <-done:
				return filepath.SkipAll
			}
			return nil
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
