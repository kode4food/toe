package view

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view/config"
)

var (
	// ErrBinaryFile is returned when opening a file whose content looks binary
	ErrBinaryFile = errors.New("binary file")
)

func newDocument(id DocumentId, opts *Options) *Document {
	d := &Document{
		id:         id,
		indent:     core.Tabs(),
		tabWidth:   4,
		lineEnding: defaultLineEnding(opts.DefaultLineEnding),
		buf: bufState{
			text:       core.NewRope(""),
			history:    core.NewHistory(),
			selections: map[Id]core.Selection{},
		},
		ls: lsState{
			highlights: map[Id][]DocumentHighlight{},
			hints:      map[Id][]InlayHint{},
		},
	}
	d.SetLang("text")
	return d
}

func openDocument(
	id DocumentId, path string, opts *Options,
) (*Document, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, &DocumentOpenError{Path: path, Err: err}
	}
	var ec *config.EditorConfig
	if opts.EditorConfig {
		ec = config.FindEditorConfig(absPath)
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			doc := newDocument(id, opts)
			doc.buf.path = absPath
			doc.editorConfig = ec
			doc.SetLang(detectLang(absPath, ""))
			lang := doc.langDef
			if ec != nil && ec.LineEnding != nil {
				doc.lineEnding = *ec.LineEnding
			}
			if ec != nil && ec.IndentStyle != nil {
				doc.indent = *ec.IndentStyle
			} else if lang.Indent.Unit != "" {
				doc.indent = core.ParseIndentStyle(lang.Indent.Unit)
			}
			if ec != nil && ec.TabWidth != nil {
				doc.tabWidth = *ec.TabWidth
			} else if lang.Indent.TabWidth != nil {
				doc.tabWidth = *lang.Indent.TabWidth
			}
			doc.refreshDiskSnapshot()
			return doc, nil
		}
		return nil, &DocumentOpenError{Path: path, Err: err}
	}
	if core.LooksBinary(data) {
		return nil, &DocumentOpenError{Path: path, Err: ErrBinaryFile}
	}

	hasBOM := hasBOMBytes(data)
	if hasBOM {
		data = data[3:]
	}
	rope := core.NewRope(string(data))
	doc := &Document{
		id:           id,
		tabWidth:     4,
		lineEnding:   defaultLineEnding(opts.DefaultLineEnding),
		editorConfig: ec,
		hasBOM:       hasBOM,
		buf: bufState{
			path:       absPath,
			text:       rope,
			history:    core.NewHistory(),
			selections: map[Id]core.Selection{},
		},
		ls: lsState{
			highlights: map[Id][]DocumentHighlight{},
			hints:      map[Id][]InlayHint{},
		},
	}
	doc.SetLang(detectLang(absPath, string(data)))

	lang := doc.langDef
	if ec != nil && ec.IndentStyle != nil {
		doc.indent = *ec.IndentStyle
	} else if style, ok := core.AutoDetect(rope); ok {
		doc.indent = style
	} else if lang.Indent.Unit != "" {
		doc.indent = core.ParseIndentStyle(lang.Indent.Unit)
	} else {
		doc.indent = core.Tabs()
	}
	if ec != nil && ec.TabWidth != nil {
		doc.tabWidth = *ec.TabWidth
	} else if lang.Indent.TabWidth != nil {
		doc.tabWidth = *lang.Indent.TabWidth
	}
	if ec != nil && ec.LineEnding != nil {
		doc.lineEnding = *ec.LineEnding
	} else if le, ok := core.AutoDetectLineEndingString(string(data)); ok {
		doc.lineEnding = le
	}
	doc.refreshDiskSnapshot()

	return doc, nil
}

func newPendingDocument(
	id DocumentId, absPath, lang string, opts *Options,
) *Document {
	d := newDocument(id, opts)
	d.buf.path = absPath
	d.pending = &pendingLoad{opts: opts, lang: lang}
	if lang != "" {
		d.SetLang(lang)
	}
	return d
}

// ensureLoaded reads the backing file the first time a pending buffer's content
// is touched, copying the content-derived state onto the placeholder
func (d *Document) ensureLoaded() {
	d.buf.RLock()
	pending := d.pending != nil
	d.buf.RUnlock()
	if !pending {
		return
	}
	d.buf.Lock()
	defer d.buf.Unlock()
	p := d.pending
	if p == nil {
		return
	}
	loaded, err := openDocument(d.id, d.buf.path, p.opts)
	if err != nil {
		return
	}
	d.pending = nil
	d.hasBOM = loaded.hasBOM
	d.indent = loaded.indent
	d.tabWidth = loaded.tabWidth
	d.lineEnding = loaded.lineEnding
	d.editorConfig = loaded.editorConfig
	d.disk = loaded.disk
	d.external = loaded.external
	d.buf.text = loaded.buf.text
	d.buf.version = loaded.buf.version
	d.buf.history = loaded.buf.history
	if p.lang == "" {
		d.buf.lang = loaded.buf.lang
		d.langDef = loaded.langDef
	}
	d.buf.lastSel = clampSelection(
		d.buf.lastSel, d.buf.text.LenChars(),
	)
}

func clampSelection(sel core.Selection, maxChars int) core.Selection {
	ranges := sel.Ranges()
	if len(ranges) == 0 {
		return core.PointSelection(0)
	}
	clamped := make([]core.Range, len(ranges))
	for i, r := range ranges {
		clamped[i] = core.NewRange(
			min(max(r.Anchor, 0), maxChars),
			min(max(r.Head, 0), maxChars),
		)
	}
	out, err := core.NewSelection(clamped, sel.PrimaryIndex())
	if err != nil {
		return core.PointSelection(0)
	}
	return out
}
