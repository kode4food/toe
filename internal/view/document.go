package view

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view/config"
	"github.com/kode4food/toe/internal/view/language"
)

type (
	// Document holds metadata for an open buffer; editing and LSP state live in
	// the buf and ls sub-structs respectively
	Document struct {
		id         DocumentId
		accessedAt int64
		hasBOM     bool

		indent        core.IndentStyle
		tabWidth      int
		lineEnding    core.LineEnding
		editorConfig  *config.EditorConfig
		readOnly      bool
		langDef       *language.Language
		restoreCursor bool
		disk          diskSnapshot
		external      ExternalState

		buf bufState
		ls  lsState
	}

	// bufState holds the full editing state of the document. Snapshot fields
	// (path, lang, text, version) are read by async LSP goroutines; editing
	// state (history, selections, insertAcc, dirty flags) is main-goroutine
	// only. The embedded RWMutex lets goroutines snapshot safely
	bufState struct {
		sync.RWMutex
		path       string
		lang       string
		text       core.Rope
		version    int
		history    core.History
		insertAcc  *insertAccum
		selections map[Id]core.Selection
		unsaved    bool
		modified   bool
	}

	// lsState holds all language-server-managed overlays for the document,
	// protected by an embedded RWMutex so async LSP goroutines write safely
	lsState struct {
		sync.RWMutex
		highlights  map[Id][]DocumentHighlight
		links       []DocumentLink
		colors      []DocumentColor
		hints       map[Id][]InlayHint
		diagnostics []Diagnostic
	}

	// insertAccum holds the pre-insert state and the composed changeset for the
	// current insert-mode session
	insertAccum struct {
		oldState core.State
		cs       core.ChangeSet
	}

	// DocumentId is the unique identifier for an open document
	DocumentId int

	// DocumentOpenError describes why a document could not be opened
	DocumentOpenError struct {
		Path string
		Err  error
	}

	// ExternalState describes whether a file-backed document has diverged from
	// the last disk snapshot toe loaded or wrote
	ExternalState int
)

const (
	// InvalidDocumentId is the zero value, indicating no document
	InvalidDocumentId DocumentId = 0
	// ScratchBufferName is the display name used for unnamed scratch documents
	ScratchBufferName = "[scratch]"
)

const (
	ExternalStateClean   ExternalState = iota // no external disk change pending
	ExternalStateChanged                      // changed while buffer dirty
	ExternalStateDeleted                      // backing file removed while open
)

// RestoreCursor reports whether the next exit from insert mode should move the
// cursor back by one grapheme
func (d *Document) RestoreCursor() bool {
	return d.restoreCursor
}

// SetRestoreCursor marks whether the next insert-mode exit should restore the
// cursor one grapheme to the left
func (d *Document) SetRestoreCursor(v bool) {
	d.restoreCursor = v
}

// ID returns the unique document identifier
func (d *Document) ID() DocumentId {
	return d.id
}

// AccessedAt returns the monotonic focus/access sequence for MRU ordering
func (d *Document) AccessedAt() int64 {
	return d.accessedAt
}

// Text returns the current rope text
func (d *Document) Text() core.Rope {
	d.buf.RLock()
	defer d.buf.RUnlock()
	return d.buf.text
}

// Path returns the file path, or empty string for scratch buffers
func (d *Document) Path() string {
	d.buf.RLock()
	defer d.buf.RUnlock()
	return d.buf.path
}

// SetPath sets the file path for this document
func (d *Document) SetPath(path string) {
	d.buf.Lock()
	d.buf.path = path
	d.buf.Unlock()
}

// Modified reports whether the document has unsaved changes
func (d *Document) Modified() bool {
	return d.buf.unsaved
}

// ExternalState reports any unresolved change made to the backing file by
// another process
func (d *Document) ExternalState() ExternalState {
	return d.external
}

// Lang returns the language identifier for syntax highlighting
func (d *Document) Lang() string {
	d.buf.RLock()
	defer d.buf.RUnlock()
	return d.buf.lang
}

// SetLang sets the language identifier and resolves its definition once so
// the render path reads the cached *language.Language directly
func (d *Document) SetLang(lang string) {
	d.langDef = language.LoadLanguage(lang)
	d.buf.Lock()
	d.buf.lang = lang
	d.buf.Unlock()
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
	format := language.TextFormatForConfig(
		langDef, opts.TextWidth, opts.SoftWrap, w,
	)
	format.TabWidth = d.tabWidth
	return format
}

// ReadOnly reports whether the document is read-only
func (d *Document) ReadOnly() bool {
	return d.readOnly
}

// SetReadOnly marks the document as read-only or writable
func (d *Document) SetReadOnly(v bool) {
	d.readOnly = v
}

// IndentStyle returns the active indentation style
func (d *Document) IndentStyle() core.IndentStyle {
	return d.indent
}

// SetIndentStyle updates the indent style for this document
func (d *Document) SetIndentStyle(s core.IndentStyle) {
	d.indent = s
}

// TabWidth returns the display tab width
func (d *Document) TabWidth() int {
	return d.tabWidth
}

// LineEnding returns the document's line-ending style
func (d *Document) LineEnding() core.LineEnding {
	return d.lineEnding
}

// SetLineEnding updates the line ending for this document
func (d *Document) SetLineEnding(le core.LineEnding) {
	d.lineEnding = le
}

// DisplayName returns the short display name for the document
func (d *Document) DisplayName() string {
	return DocumentDisplayName(d.Path())
}

// RelativeName returns the path relative to basedir
func (d *Document) RelativeName(basedir string) string {
	return DocumentRelativeName(d.Path(), basedir)
}

// SelectionFor returns the selection for a given view
func (d *Document) SelectionFor(vid Id) core.Selection {
	if sel, ok := d.buf.selections[vid]; ok {
		return sel
	}
	sel, _ := core.NewSelection([]core.Range{core.PointRange(0)}, 0)
	return sel
}

// SetSelectionFor sets the selection for a view
func (d *Document) SetSelectionFor(vid Id, sel core.Selection) {
	d.buf.selections[vid] = sel
}

// RemoveView cleans up selection and LSP state for a closed view
func (d *Document) RemoveView(vid Id) {
	delete(d.buf.selections, vid)
	d.ls.Lock()
	delete(d.ls.highlights, vid)
	delete(d.ls.hints, vid)
	d.ls.Unlock()
}

// BeginInsertGroup starts insert-mode change accumulation for vid if not
// already active. Subsequent Apply calls accumulate into a single history
// revision until CommitInsertHistory is called
func (d *Document) BeginInsertGroup(vid Id) {
	if d.buf.insertAcc != nil {
		return
	}
	d.buf.insertAcc = &insertAccum{
		oldState: core.State{
			Doc:       d.buf.text,
			Selection: d.SelectionFor(vid),
		},
		cs: core.NewChangeSet(d.buf.text),
	}
}

// CommitInsertHistory flushes any accumulated insert-mode changes as one
// history revision. It is a no-op when no accumulation is active
func (d *Document) CommitInsertHistory(vid Id) {
	acc := d.buf.insertAcc
	d.buf.insertAcc = nil
	if acc == nil || acc.cs.Empty() {
		return
	}
	tx := core.NewTransaction(acc.oldState.Doc).
		WithChanges(acc.cs).
		WithSelection(d.SelectionFor(vid))
	_ = d.buf.history.CommitRevision(tx, acc.oldState)
}

// Apply applies a transaction to the document, recording in history. While an
// insert group is active (BeginInsertGroup was called), changes are accumulated
// and a single revision is committed by CommitInsertHistory
func (d *Document) Apply(tx core.Transaction, vid Id) error {
	newText, err := tx.Apply(d.buf.text)
	if err != nil {
		return err
	}

	if d.buf.insertAcc != nil {
		// Accumulate into the ongoing insert group
		newSel := d.SelectionFor(vid)
		if txSel := tx.Selection(); txSel != nil {
			newSel = *txSel
		}
		cs := tx.Changes()
		d.buf.Lock()
		d.buf.text = newText
		d.buf.selections[vid] = newSel
		if !cs.Empty() {
			d.buf.insertAcc.cs = d.buf.insertAcc.cs.Compose(cs)
			d.mapOtherSelections(vid, cs)
			d.buf.unsaved = true
			d.buf.modified = true
			d.buf.version++
		}
		d.buf.Unlock()
		return nil
	}

	// Commit the FORWARD tx with the BEFORE state so that Undo can restore it
	beforeSt := core.State{Doc: d.buf.text, Selection: d.SelectionFor(vid)}
	if err := d.buf.history.CommitRevision(tx, beforeSt); err != nil {
		return err
	}

	newSel := beforeSt.Selection
	if txSel := tx.Selection(); txSel != nil {
		newSel = *txSel
	}
	cs := tx.Changes()
	d.buf.Lock()
	d.buf.text = newText
	d.buf.selections[vid] = newSel
	if !cs.Empty() {
		d.mapOtherSelections(vid, cs)
		d.buf.unsaved = true
		d.buf.modified = true
		d.buf.version++
	}
	d.buf.Unlock()
	return nil
}

// LastEditPos returns the char offset of the most recently committed change
func (d *Document) LastEditPos() int {
	return d.buf.history.LastEditPos()
}

// Revision returns the document version used for render-cache invalidation
func (d *Document) Revision() int {
	d.buf.RLock()
	defer d.buf.RUnlock()
	return d.buf.version
}

// Undo reverts one history step for the given view
func (d *Document) Undo(vid Id) bool {
	tx, ok := d.buf.history.Undo()
	if !ok {
		return false
	}
	newText, err := tx.Apply(d.buf.text)
	if err != nil {
		return false
	}
	cs := tx.Changes()
	d.buf.Lock()
	if !cs.Empty() {
		d.mapOtherSelections(vid, cs)
	}
	d.buf.text = newText
	d.buf.version++
	if txSel := tx.Selection(); txSel != nil {
		d.buf.selections[vid] = *txSel
	}
	d.buf.unsaved = !d.buf.history.AtRoot()
	d.buf.modified = true
	d.buf.Unlock()
	return true
}

// Redo reapplies one reverted step for the given view
func (d *Document) Redo(vid Id) bool {
	tx, ok := d.buf.history.Redo()
	if !ok {
		return false
	}
	newText, err := tx.Apply(d.buf.text)
	if err != nil {
		return false
	}
	cs := tx.Changes()
	d.buf.Lock()
	if !cs.Empty() {
		d.mapOtherSelections(vid, cs)
	}
	d.buf.text = newText
	d.buf.version++
	if txSel := tx.Selection(); txSel != nil {
		d.buf.selections[vid] = *txSel
	}
	d.buf.unsaved = true
	d.buf.modified = true
	d.buf.Unlock()
	return true
}

func (d *DocumentOpenError) Error() string {
	return fmt.Sprintf("open %s: %v", d.Path, d.Err)
}

func (d *DocumentOpenError) Unwrap() error {
	return d.Err
}

func (d *Document) mapOtherSelections(vid Id, cs core.ChangeSet) {
	for otherVid, sel := range d.buf.selections {
		if otherVid == vid {
			continue
		}
		if mapped, err := sel.Map(cs); err == nil {
			d.buf.selections[otherVid] = mapped
		}
	}
}

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

func setOverlaySlice[T any](ls *lsState, field *[]T, items []T) {
	ls.Lock()
	defer ls.Unlock()
	if len(items) == 0 {
		*field = nil
		return
	}
	*field = slices.Clone(items)
}

func clearOverlaySlice[T any](ls *lsState, field *[]T) {
	ls.Lock()
	defer ls.Unlock()
	*field = nil
}

func getOverlaySlice[T any](ls *lsState, field *[]T) []T {
	ls.RLock()
	defer ls.RUnlock()
	return slices.Clone(*field)
}

func setOverlayMap[T any](ls *lsState, m map[Id][]T, vid Id, items []T) {
	ls.Lock()
	defer ls.Unlock()
	if len(items) == 0 {
		delete(m, vid)
		return
	}
	m[vid] = slices.Clone(items)
}

func clearOverlayMap[T any](ls *lsState, m map[Id][]T, vid Id) {
	ls.Lock()
	defer ls.Unlock()
	delete(m, vid)
}

func clearAllOverlayMap[T any](ls *lsState, m map[Id][]T) {
	ls.Lock()
	defer ls.Unlock()
	clear(m)
}

func getOverlayMap[T any](ls *lsState, m map[Id][]T, vid Id) []T {
	ls.RLock()
	defer ls.RUnlock()
	return slices.Clone(m[vid])
}
