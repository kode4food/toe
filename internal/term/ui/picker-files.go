package ui

import (
	"cmp"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	gitignore "github.com/sabhiram/go-gitignore"

	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/config"
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

func newFilePickerSource(dir string) *filePickerSource {
	return &filePickerSource{
		pickerMeta: pickerMeta{
			title:   "Open file",
			columns: []string{"path"},
		},
		dir: dir,
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

func drainPickerFeed(ch <-chan PickerItem, done <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		batch := make([]PickerItem, 0, pickerFeedBatchSize)
		var flush <-chan time.Time
		for {
			select {
			case item, ok := <-ch:
				if !ok {
					return pickerFeedMsg{items: batch}
				}
				batch = append(batch, item)
				if len(batch) == 1 {
					flush = time.After(pickerFeedFlushWait)
				}
				if len(batch) >= pickerFeedBatchSize {
					return pickerFeedMsg{items: batch, feed: ch, done: done}
				}
			case <-flush:
				return pickerFeedMsg{items: batch, feed: ch, done: done}
			case <-done:
				return pickerFeedMsg{items: batch}
			}
		}
	}
}

func drainDynamicFeed(gen int, ch <-chan PickerItem) tea.Cmd {
	return func() tea.Msg {
		batch := make([]PickerItem, 0, pickerFeedBatchSize)
		var flush <-chan time.Time
		for {
			select {
			case item, ok := <-ch:
				if !ok {
					return pickerDynamicFeedMsg{gen: gen, items: batch}
				}
				batch = append(batch, item)
				if len(batch) == 1 {
					flush = time.After(pickerFeedFlushWait)
				}
				if len(batch) >= pickerFeedBatchSize {
					return pickerDynamicFeedMsg{
						gen: gen, items: batch, feed: ch,
					}
				}
			case <-flush:
				return pickerDynamicFeedMsg{gen: gen, items: batch, feed: ch}
			}
		}
	}
}

func sortPickerItems(items []PickerItem) {
	slices.SortStableFunc(items, func(a, b PickerItem) int {
		return cmp.Compare(a.Display, b.Display)
	})
}

func skipPickerPath(rel string, d os.DirEntry, ignores []pickerIgnore) bool {
	name := d.Name()
	if strings.HasPrefix(name, ".") {
		return true
	}
	if excludedPickerType(name) {
		return true
	}
	for _, ig := range ignores {
		sub, ok := ignorePathForBase(rel, ig.base)
		if ok && ig.ig.MatchesPath(sub) {
			return true
		}
	}
	return false
}

func ignorePathForBase(rel, base string) (string, bool) {
	if base == "" {
		return rel, true
	}
	if rel == base {
		return "", true
	}
	sub, ok := strings.CutPrefix(rel, base+"/")
	return sub, ok
}

func loadIgnoreFiles(root, rel string) []pickerIgnore {
	var ignores []pickerIgnore
	for _, base := range ignoreBases(rel) {
		dir := filepath.Join(root, filepath.FromSlash(base))
		for _, name := range []string{
			".ignore",
			".gitignore",
			filepath.Join(loader.WorkspaceDirName, "ignore"),
		} {
			if ig, ok := compileIgnore(filepath.Join(dir, name)); ok {
				ignores = append(ignores, pickerIgnore{base, ig})
			}
		}
		if base == "" {
			if ig, ok := compileIgnore(
				filepath.Join(root, ".git", "info", "exclude"),
			); ok {
				ignores = append(ignores, pickerIgnore{base, ig})
			}
			if ig, ok := compileIgnore(config.IgnorePath()); ok {
				ignores = append(ignores, pickerIgnore{base, ig})
			}
		}
	}
	return ignores
}

func ignoreBases(rel string) []string {
	dir := filepath.ToSlash(filepath.Dir(rel))
	if dir == "." {
		return []string{""}
	}
	parts := strings.Split(dir, "/")
	bases := make([]string, 1, len(parts)+1)
	for i := range parts {
		bases = append(bases, strings.Join(parts[:i+1], "/"))
	}
	return bases
}

func compileIgnore(path string) (*gitignore.GitIgnore, bool) {
	ig, err := gitignore.CompileIgnoreFile(path)
	return ig, err == nil
}

func excludedPickerType(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".zip", ".gz", ".bz2", ".zst", ".lzo", ".sz", ".tgz", ".tbz2",
		".lz", ".lz4", ".lzma", ".z", ".xz", ".7z", ".rar", ".cab":
		return true
	default:
		return false
	}
}

func looksBinary(data []byte) bool {
	return slices.Contains(data[:min(len(data), 1024)], 0)
}

func pickerListRows(e *view.Editor) int {
	h := e.ViewHeight() + 1
	areaH := max((h-2)*90/100, 0)
	return max(areaH-4, 1)
}
