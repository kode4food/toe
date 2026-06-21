package view

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view/config"
	"github.com/kode4food/toe/internal/view/language"
)

type (
	// Document holds the text, history, and per-view state for an open buffer
	Document struct {
		id       DocumentId
		text     core.Rope
		path     string
		modified bool
		// modifiedSinceAccessed is set when changes land and cleared when this
		// document becomes the focused document, enabling goto_last_modified_file
		modifiedSinceAccessed bool
		// hasBOM records that the file was opened with a UTF-8 BOM so it is
		// preserved on save
		hasBOM bool

		selections map[Id]core.Selection
		history    core.History

		// version increments on every text change, including insert-mode
		// changes not yet committed to history
		version int

		// insertAcc accumulates changes made during insert mode so that the whole
		// session is committed as one history revision on exit
		insertAcc *insertAccum

		indent       core.IndentStyle
		tabWidth     int
		lineEnding   core.LineEnding
		editorConfig *config.EditorConfig
		readonly     bool
		lang         string
		// langDef holds the resolved language definition for lang, set whenever
		// lang is, so the render path reads it directly instead of re-parsing
		// the language config every frame
		langDef       *language.Language
		restoreCursor bool
	}

	// insertAccum holds the pre-insert state and the composed changeset for the
	// current insert-mode session
	insertAccum struct {
		oldState core.State
		cs       core.ChangeSet
	}
)

// RestoreCursor reports whether the next exit from insert mode should move the
// cursor back by one grapheme
func (d *Document) RestoreCursor() bool { return d.restoreCursor }

// SetRestoreCursor marks whether the next insert-mode exit should restore the
// cursor one grapheme to the left
func (d *Document) SetRestoreCursor(v bool) { d.restoreCursor = v }

func newDocument(id DocumentId, opts *Options) *Document {
	d := &Document{
		id:         id,
		text:       core.NewRope(""),
		selections: map[Id]core.Selection{},
		history:    core.NewHistory(),
		indent:     core.Tabs(),
		tabWidth:   4,
		lineEnding: defaultLineEnding(opts.DefaultLineEnding),
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
			doc.path = absPath
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
		id:           id,
		text:         rope,
		path:         absPath,
		selections:   map[Id]core.Selection{},
		history:      core.NewHistory(),
		tabWidth:     4,
		lineEnding:   defaultLineEnding(opts.DefaultLineEnding),
		editorConfig: ec,
		hasBOM:       hasBOM,
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

	return doc, nil
}

// ID returns the unique document identifier
func (d *Document) ID() DocumentId { return d.id }

// Text returns the current rope text
func (d *Document) Text() core.Rope { return d.text }

// Path returns the file path, or empty string for scratch buffers
func (d *Document) Path() string { return d.path }

// SetPath sets the file path for this document
func (d *Document) SetPath(path string) { d.path = path }

// Modified reports whether the document has unsaved changes
func (d *Document) Modified() bool { return d.modified }

// Lang returns the language identifier for syntax highlighting
func (d *Document) Lang() string { return d.lang }

// SetLang sets the language identifier and resolves its definition once, so the
// render path reads the cached *config.Language directly instead of re-parsing
func (d *Document) SetLang(lang string) {
	d.lang = lang
	d.langDef = language.LoadLanguage(lang)
}

// TextFormat returns the display-time text layout options for this document
func (d *Document) TextFormat(w int) *language.TextFormat {
	return d.TextFormatForConfig(w, new(defaultOptions()))
}

// TextFormatForConfig returns layout options using the supplied editor options
func (d *Document) TextFormatForConfig(
	w int, opts *Options,
) *language.TextFormat {
	langDef := d.langDef
	if d.editorConfig != nil && d.editorConfig.MaxLineLength != nil {
		cpy := *langDef
		cpy.TextWidth = d.editorConfig.MaxLineLength
		langDef = &cpy
	}
	format := language.TextFormatForConfig(langDef, opts.TextWidth, opts.SoftWrap, w)
	format.TabWidth = d.tabWidth
	return format
}

// Readonly reports whether the document is read-only
func (d *Document) Readonly() bool { return d.readonly }

// IndentStyle returns the active indentation style
func (d *Document) IndentStyle() core.IndentStyle { return d.indent }

// SetIndentStyle updates the indent style for this document
func (d *Document) SetIndentStyle(s core.IndentStyle) { d.indent = s }

// TabWidth returns the display tab width
func (d *Document) TabWidth() int { return d.tabWidth }

// LineEnding returns the document's line-ending style
func (d *Document) LineEnding() core.LineEnding { return d.lineEnding }

// SetLineEnding updates the line ending for this document
func (d *Document) SetLineEnding(le core.LineEnding) { d.lineEnding = le }

// DisplayName returns the short display name for the document
func (d *Document) DisplayName() string { return DocumentDisplayName(d.path) }

// RelativeName returns the path relative to basedir
func (d *Document) RelativeName(basedir string) string {
	return DocumentRelativeName(d.path, basedir)
}

// SelectionFor returns the selection for a given view
func (d *Document) SelectionFor(vid Id) core.Selection {
	if sel, ok := d.selections[vid]; ok {
		return sel
	}
	sel, _ := core.NewSelection([]core.Range{core.PointRange(0)}, 0)
	return sel
}

// SetSelectionFor sets the selection for a view
func (d *Document) SetSelectionFor(vid Id, sel core.Selection) {
	d.selections[vid] = sel
}

// RemoveView cleans up selection state for a closed view
func (d *Document) RemoveView(vid Id) {
	delete(d.selections, vid)
}

// BeginInsertGroup starts insert-mode change accumulation for vid if not
// already active. Subsequent Apply calls accumulate into a single history
// revision until CommitInsertHistory is called
func (d *Document) BeginInsertGroup(vid Id) {
	if d.insertAcc != nil {
		return
	}
	d.insertAcc = &insertAccum{
		oldState: core.State{Doc: d.text, Selection: d.SelectionFor(vid)},
		cs:       core.NewChangeSet(d.text),
	}
}

// CommitInsertHistory flushes any accumulated insert-mode changes as one
// history revision. It is a no-op when no accumulation is active
func (d *Document) CommitInsertHistory(vid Id) {
	acc := d.insertAcc
	d.insertAcc = nil
	if acc == nil || acc.cs.Empty() {
		return
	}
	tx := core.NewTransaction(acc.oldState.Doc).
		WithChanges(acc.cs).
		WithSelection(d.SelectionFor(vid))
	_ = d.history.CommitRevision(tx, acc.oldState)
}

// Apply applies a transaction to the document, recording in history. While an
// insert group is active (BeginInsertGroup was called), changes are accumulated
// and a single revision is committed by CommitInsertHistory
func (d *Document) Apply(tx core.Transaction, vid Id) error {
	newText, err := tx.Apply(d.text)
	if err != nil {
		return err
	}

	if d.insertAcc != nil {
		// Accumulate into the ongoing insert group
		newSel := d.SelectionFor(vid)
		if txSel := tx.Selection(); txSel != nil {
			newSel = *txSel
		}
		d.text = newText
		d.selections[vid] = newSel
		if cs := tx.Changes(); !cs.Empty() {
			d.insertAcc.cs = d.insertAcc.cs.Compose(cs)
			d.mapOtherSelections(vid, cs)
			d.modified = true
			d.modifiedSinceAccessed = true
			d.version++
		}
		return nil
	}

	// Commit the FORWARD tx with the BEFORE state so that Undo can restore it
	beforeSt := core.State{Doc: d.text, Selection: d.SelectionFor(vid)}
	if err := d.history.CommitRevision(tx, beforeSt); err != nil {
		return err
	}

	newSel := beforeSt.Selection
	if txSel := tx.Selection(); txSel != nil {
		newSel = *txSel
	}

	d.text = newText
	d.selections[vid] = newSel
	if cs := tx.Changes(); !cs.Empty() {
		d.mapOtherSelections(vid, cs)
		d.modified = true
		d.modifiedSinceAccessed = true
		d.version++
	}
	return nil
}

// LastEditPos returns the char offset of the most recently committed change
func (d *Document) LastEditPos() int { return d.history.LastEditPos() }

// Revision returns the document version used for render-cache invalidation
func (d *Document) Revision() int { return d.version }

// Undo reverts one history step for the given view
func (d *Document) Undo(vid Id) bool {
	tx, ok := d.history.Undo()
	if !ok {
		return false
	}
	newText, err := tx.Apply(d.text)
	if err != nil {
		return false
	}
	if cs := tx.Changes(); !cs.Empty() {
		d.mapOtherSelections(vid, cs)
	}
	d.text = newText
	d.version++
	if txSel := tx.Selection(); txSel != nil {
		d.selections[vid] = *txSel
	}
	d.modified = !d.history.AtRoot()
	d.modifiedSinceAccessed = true
	return true
}

// Redo reapplies one reverted step for the given view
func (d *Document) Redo(vid Id) bool {
	tx, ok := d.history.Redo()
	if !ok {
		return false
	}
	newText, err := tx.Apply(d.text)
	if err != nil {
		return false
	}
	if cs := tx.Changes(); !cs.Empty() {
		d.mapOtherSelections(vid, cs)
	}
	d.text = newText
	d.version++
	if txSel := tx.Selection(); txSel != nil {
		d.selections[vid] = *txSel
	}
	d.modified = true
	d.modifiedSinceAccessed = true
	return true
}

func (d *Document) mapOtherSelections(vid Id, cs core.ChangeSet) {
	for otherVid, sel := range d.selections {
		if otherVid == vid {
			continue
		}
		if mapped, err := sel.Map(cs); err == nil {
			d.selections[otherVid] = mapped
		}
	}
}
