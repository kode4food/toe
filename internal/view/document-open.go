package view

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view/config"
)

func newDocument(id DocumentId, opts *Options) *Document {
	d := &Document{
		identity: identityState{id: id},
		content: contentState{
			text: core.NewRope(""),
		},
		edits: editState{
			history: core.NewHistory(),
		},
		views: viewState{
			selections: map[Id]core.Selection{},
		},
		format: formatState{
			indent:     core.Tabs(),
			tabWidth:   4,
			lineEnding: defaultLineEnding(opts.DefaultLineEnding),
		},
		overlays: overlayState{
			highlights: map[Id][]DocumentHighlight{},
			hints:      map[Id][]InlayHint{},
		},
	}
	d.SetLang(DefaultLanguage)
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
	data, err := core.LoadText(absPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			doc := newDocument(id, opts)
			doc.content.path = absPath
			doc.format.editorConfig = ec
			doc.SetLang(detectLang(absPath, ""))
			lang := doc.format.language
			if ec != nil && ec.LineEnding != nil {
				doc.format.lineEnding = *ec.LineEnding
			}
			if ec != nil && ec.IndentStyle != nil {
				doc.format.indent = *ec.IndentStyle
			} else if lang.Indent.Unit != "" {
				doc.format.indent = core.ParseIndentStyle(lang.Indent.Unit)
			}
			if ec != nil && ec.TabWidth != nil {
				doc.format.tabWidth = *ec.TabWidth
			} else if lang.Indent.TabWidth != nil {
				doc.format.tabWidth = *lang.Indent.TabWidth
			}
			doc.refreshDiskSnapshot()
			return doc, nil
		}
		return nil, &DocumentOpenError{Path: path, Err: err}
	}

	hasBOM := hasBOMBytes(data)
	if hasBOM {
		data = data[3:]
	}
	rope := core.NewRope(string(data))
	doc := &Document{
		identity: identityState{id: id},
		content: contentState{
			path: absPath,
			text: rope,
		},
		edits: editState{
			history: core.NewHistory(),
		},
		views: viewState{
			selections: map[Id]core.Selection{},
		},
		format: formatState{
			tabWidth:     4,
			lineEnding:   defaultLineEnding(opts.DefaultLineEnding),
			editorConfig: ec,
			hasBOM:       hasBOM,
		},
		overlays: overlayState{
			highlights: map[Id][]DocumentHighlight{},
			hints:      map[Id][]InlayHint{},
		},
	}
	doc.SetLang(detectLang(absPath, string(data)))

	lang := doc.format.language
	if ec != nil && ec.IndentStyle != nil {
		doc.format.indent = *ec.IndentStyle
	} else if style, ok := core.AutoDetect(rope); ok {
		doc.format.indent = style
	} else if lang.Indent.Unit != "" {
		doc.format.indent = core.ParseIndentStyle(lang.Indent.Unit)
	} else {
		doc.format.indent = core.Tabs()
	}
	if ec != nil && ec.TabWidth != nil {
		doc.format.tabWidth = *ec.TabWidth
	} else if lang.Indent.TabWidth != nil {
		doc.format.tabWidth = *lang.Indent.TabWidth
	}
	if ec != nil && ec.LineEnding != nil {
		doc.format.lineEnding = *ec.LineEnding
	} else if le, ok := core.AutoDetectLineEndingString(string(data)); ok {
		doc.format.lineEnding = le
	}
	doc.refreshDiskSnapshot()

	return doc, nil
}

func newPendingDocument(
	id DocumentId, absPath, lang string, opts *Options,
) *Document {
	d := newDocument(id, opts)
	d.content.path = absPath
	d.content.pending = &pendingLoad{opts: opts, lang: lang}
	if lang != "" {
		d.SetLang(lang)
	}
	return d
}

// ensureLoaded reads the backing file the first time a pending buffer's content
// is touched, copying the content-derived state onto the placeholder
func (d *Document) ensureLoaded() {
	d.content.RLock()
	pending := d.content.pending != nil
	d.content.RUnlock()
	if !pending {
		return
	}
	d.content.Lock()
	defer d.content.Unlock()
	p := d.content.pending
	if p == nil {
		return
	}
	loaded, err := openDocument(d.identity.id, d.content.path, p.opts)
	if err != nil {
		return
	}
	d.content.pending = nil
	d.format = loaded.format
	d.file = loaded.file
	d.content.text = loaded.content.text
	d.content.version = loaded.content.version
	d.edits.history = loaded.edits.history
	if p.lang == "" {
		d.content.lang = loaded.content.lang
		d.format.language = loaded.format.language
	}
	d.views.lastSelection = clampSelection(
		d.views.lastSelection, d.content.text.LenChars(),
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
