package view

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view/config"
	"github.com/kode4food/toe/internal/view/language"
)

func newDocument(
	id DocumentId, opts *Options, registry *language.Registry,
) *Document {
	d := &Document{
		identity: identityState{id: id},
		content: contentState{
			name: ScratchBufferName,
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
			registry:   registry,
		},
		overlays: overlayState{
			highlights: map[Id][]DocumentHighlight{},
			hints:      map[Id][]InlayHint{},
		},
	}
	d.SetLang(DefaultLanguage)
	return d
}

type newPendingDocumentArgs struct {
	id      DocumentId
	absPath string
	lang    string
	opts    *Options
	langs   *language.Registry
}

func newPendingDocument(args newPendingDocumentArgs) *Document {
	lang := args.lang
	opts := args.opts
	d := newDocument(args.id, opts, args.langs)
	d.SetPath(args.absPath)
	d.content.pending = &pendingLoad{opts: opts, lang: lang}
	if lang != "" {
		d.SetLang(lang)
	}
	return d
}

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
	loaded, err := openDocument(
		d.identity.id, d.content.path, p.opts, d.format.registry,
	)
	if err != nil {
		return
	}
	d.content.pending = nil
	d.format = loaded.format
	d.file = loaded.file
	if !d.file.snapshot.exists {
		d.file.external = ExternalStateMissing
	}
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

func openDocument(
	id DocumentId, path string, opts *Options, registry *language.Registry,
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
			doc := newDocument(id, opts, registry)
			doc.SetPath(absPath)
			doc.format.editorConfig = ec
			doc.SetLang(registry.Detect(
				language.DetectArgs{
					Path:    absPath,
					Default: DefaultLanguage,
				},
			))
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
			name: filepath.Base(absPath),
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
			registry:     registry,
		},
		overlays: overlayState{
			highlights: map[Id][]DocumentHighlight{},
			hints:      map[Id][]InlayHint{},
		},
	}
	doc.SetLang(registry.Detect(language.DetectArgs{
		Path:    absPath,
		Content: string(data),
		Default: DefaultLanguage,
	}))

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

func clampSelection(sel core.Selection, maxChars int) core.Selection {
	ranges := sel.Ranges()
	if len(ranges) == 0 {
		return core.PointSelection(0)
	}
	clamped := make([]core.Range, len(ranges))
	for i, r := range ranges {
		clamped[i] = core.Range{
			Anchor: min(max(r.Anchor, 0), maxChars),
			Head:   min(max(r.Head, 0), maxChars),
		}
	}
	if out, err := core.NewSelection(clamped, sel.PrimaryIndex()); err == nil {
		return out
	}
	return core.PointSelection(0)
}
