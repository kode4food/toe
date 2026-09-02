package ui

import (
	"cmp"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/term/highlight"
	"github.com/kode4food/toe/internal/term/syntax"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/language"
)

type (
	previewCache map[previewCacheKey]previewCacheEntry

	previewCacheKey struct {
		id     view.DocumentId
		path   string
		staged bool
	}

	previewCacheEntry interface {
		renderInto(*previewCtx, *tui.Buffer, geom.Point)
	}

	previewDocEntry struct {
		rev   int
		lang  string
		rope  core.Rope
		spans []highlight.Span
	}

	previewDirEntry struct {
		path string
		rows []previewDirRow
	}

	noPreviewEntry string

	previewImageEntry struct {
		image *Image
		id    uint32
		path  string
	}

	previewBinaryEntry struct {
		path string
		size int64
	}

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
		rope: core.NewRope(text),
		spans: previewSpans(previewSpansArgs{
			cache: sc,
			text:  text,
			lang:  lang,
		}),
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

// the index text is read through version control rather than off disk, so it is
// highlighted and cached apart from the working file
func (p previewCache) indexText(
	sc *syntax.Cache, vc view.VersionControl, path string,
) *previewDocEntry {
	key := previewIndexKey(path)
	if entry, ok := p[key].(*previewDocEntry); ok {
		return entry
	}
	text := highlight.NormalizeNewlines(vc.IndexText(path))
	lang := language.DetectLanguage(language.DetectLanguageArgs{
		Path:    path,
		Content: text,
		Default: view.DefaultLanguage,
	})
	entry := &previewDocEntry{
		rope: core.NewRope(text),
		spans: previewSpans(previewSpansArgs{
			cache: sc,
			text:  text,
			lang:  lang,
		}),
		lang: lang,
	}
	p[key] = entry
	return entry
}

func (p previewCache) invalidatePath(path string) {
	delete(p, previewPathKey(path))
	delete(p, previewIndexKey(path))
}

func previewDocKey(id view.DocumentId) previewCacheKey {
	return previewCacheKey{id: id}
}

func previewPathKey(path string) previewCacheKey {
	return previewCacheKey{path: path}
}

func previewIndexKey(path string) previewCacheKey {
	return previewCacheKey{path: path, staged: true}
}

func loadPathPreview(sc *syntax.Cache, path string) previewCacheEntry {
	info, err := os.Stat(path)
	if err != nil {
		return noPreviewEntry("<File not found>")
	}
	if info.IsDir() {
		return &previewDirEntry{path: path, rows: previewDirRows(path)}
	}
	if info.Size() > PickerMaxPreview {
		if !isImagePath(path) && pathLooksBinary(path) {
			return &previewBinaryEntry{path: path, size: info.Size()}
		}
		return noPreviewEntry("<File too large to preview>")
	}
	data, err := core.LoadText(path)
	if err != nil {
		if errors.Is(err, core.ErrBinaryFile) {
			return binaryPreview(path)
		}
		return noPreviewEntry("<File not found>")
	}
	text := highlight.NormalizeNewlines(string(data))
	lang := language.DetectLanguage(language.DetectLanguageArgs{
		Path:    path,
		Content: text,
		Default: view.DefaultLanguage,
	})
	return &previewDocEntry{
		rope: core.NewRope(text),
		spans: previewSpans(previewSpansArgs{
			cache: sc,
			text:  text,
			lang:  lang,
		}),
		lang: lang,
	}
}

func binaryPreview(path string) previewCacheEntry {
	if isImagePath(path) {
		if img, err := LoadImage(path); err == nil {
			abs := path
			if value, err := filepath.Abs(path); err == nil {
				abs = value
			}
			return &previewImageEntry{
				image: img,
				id: kittyImageID(kittyImageIDArgs{
					content: img.ContentID(),
					surface: 0,
					preview: true,
				}),
				path: abs,
			}
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		return noPreviewEntry("<File not found>")
	}
	return &previewBinaryEntry{path: path, size: info.Size()}
}

func pathLooksBinary(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	var sample [core.BinarySampleSize]byte
	n, err := f.Read(sample[:])
	if err != nil && err != io.EOF {
		return false
	}
	return core.LooksBinary(sample[:n])
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
