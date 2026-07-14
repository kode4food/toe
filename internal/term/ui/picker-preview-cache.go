package ui

import (
	"cmp"
	"os"
	"slices"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/highlight"
	"github.com/kode4food/toe/internal/term/syntax"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type (
	previewCache map[previewCacheKey]previewCacheEntry

	previewCacheKey struct {
		id   view.DocumentId
		path string
	}

	previewCacheEntry interface {
		renderInto(ctx *previewCtx, buf *tui.Buffer, x, y int)
	}

	previewDocEntry struct {
		rev   int
		lang  string
		rope  core.Rope
		spans []highlight.Span
	}

	previewDirEntry struct {
		rows []previewDirRow
	}

	noPreviewEntry string

	previewDirRow struct {
		name string
		dir  bool
	}
)

// PickerMaxPreview is the largest file size a picker will preview inline
const PickerMaxPreview = 10 * 1024 * 1024

func (p previewCache) doc(
	sc *syntax.Cache, doc *view.Document,
) *previewDocEntry {
	lang := doc.Lang()
	rev := doc.Revision()
	key := previewDocKey(doc.ID())
	entry, ok := p[key].(*previewDocEntry)
	if ok && entry.rev == rev && entry.lang == lang {
		return entry
	}
	text := highlight.NormalizeNewlines(doc.Text().String())
	entry = &previewDocEntry{
		rev: rev, lang: lang,
		rope:  core.NewRope(text),
		spans: previewSpans(sc, text, lang),
	}
	p[key] = entry
	return entry
}

func (p previewCache) path(sc *syntax.Cache, path string) previewCacheEntry {
	key := previewPathKey(path)
	entry, ok := p[key]
	if ok {
		return entry
	}
	entry = loadPathPreview(sc, path)
	p[key] = entry
	return entry
}

func previewDocKey(id view.DocumentId) previewCacheKey {
	return previewCacheKey{id: id}
}

func previewPathKey(path string) previewCacheKey {
	return previewCacheKey{path: path}
}

func loadPathPreview(sc *syntax.Cache, path string) previewCacheEntry {
	info, err := os.Stat(path)
	if err != nil {
		return noPreviewEntry("<File not found>")
	}
	if info.IsDir() {
		return &previewDirEntry{rows: previewDirRows(path)}
	}
	if info.Size() > PickerMaxPreview {
		return noPreviewEntry("<File too large to preview>")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return noPreviewEntry("<File not found>")
	}
	if LooksBinary(data) {
		return noPreviewEntry("<Binary file>")
	}
	text := highlight.NormalizeNewlines(string(data))
	lang := highlight.DetectLanguage(path, text)
	return &previewDocEntry{
		rope: core.NewRope(text), spans: previewSpans(sc, text, lang),
		lang: lang,
	}
}

func previewDirRows(path string) []previewDirRow {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}
	var dirs, files []previewDirRow
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			dirs = append(dirs, previewDirRow{name: name + "/", dir: true})
		} else {
			files = append(files, previewDirRow{name: name})
		}
	}
	slices.SortFunc(dirs, func(a, b previewDirRow) int {
		return cmp.Compare(a.name, b.name)
	})
	slices.SortFunc(files, func(a, b previewDirRow) int {
		return cmp.Compare(a.name, b.name)
	})
	return append(dirs, files...)
}

// LooksBinary reports whether data appears to be non-text content, using
// the presence of a NUL byte in the first 1KB as the heuristic
func LooksBinary(data []byte) bool {
	return slices.Contains(data[:min(len(data), 1024)], 0)
}
