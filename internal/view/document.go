package view

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
		pending       *pendingLoad

		buf bufState
		ls  lsState

		track renderTrack
	}

	// pendingLoad marks a session buffer whose file has not been read yet. The
	// path and language are known; content and everything derived from it load
	// on first access
	pendingLoad struct {
		opts *Options
		lang string
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
		lastSel    core.Selection
		searchHL   map[Id]bool
		unsaved    bool
		savePoint  int
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

	// renderTrack tracks, per view, whether anything render-relevant has
	// changed since ConsumeDirty last checked
	renderTrack struct {
		sync.Mutex
		dirty map[Id]bool
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
	d.ensureLoaded()
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
	return d.buf.history.CurrentRevision() != d.buf.savePoint
}

// Loaded reports whether the backing file has been read; a restored session
// buffer stays unloaded until its content is first accessed
func (d *Document) Loaded() bool {
	d.buf.RLock()
	defer d.buf.RUnlock()
	return d.pending == nil
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
	d.ensureLoaded()
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
	d.ensureLoaded()
	return d.indent
}

// SetIndentStyle updates the indent style for this document
func (d *Document) SetIndentStyle(s core.IndentStyle) {
	d.indent = s
}

// TabWidth returns the display tab width
func (d *Document) TabWidth() int {
	d.ensureLoaded()
	return d.tabWidth
}

// LineEnding returns the document's line-ending style
func (d *Document) LineEnding() core.LineEnding {
	d.ensureLoaded()
	return d.lineEnding
}

// HasBOM reports whether the document was loaded with a UTF-8 BOM, which is
// preserved on save
func (d *Document) HasBOM() bool {
	d.ensureLoaded()
	return d.hasBOM
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
	return d.Selection()
}

// Selection returns the buffer's canonical cursor, independent of any view
func (d *Document) Selection() core.Selection {
	if len(d.buf.lastSel.Ranges()) > 0 {
		return d.buf.lastSel
	}
	return core.PointSelection(0)
}

// SetSelectionFor sets the selection for a view. Changing the selection clears
// any search-match highlighting for that view, matching helix: a search shows
// its matches until the selection next moves
func (d *Document) SetSelectionFor(vid Id, sel core.Selection) {
	if old, ok := d.buf.selections[vid]; !ok || !old.Equal(sel) {
		d.markDirty(vid)
	}
	d.buf.selections[vid] = sel
	delete(d.buf.searchHL, vid)
}

// ShowSearchHighlights marks a view's search matches as visible. Search actions
// call this after moving the selection to their match
func (d *Document) ShowSearchHighlights(vid Id) {
	if d.buf.searchHL == nil {
		d.buf.searchHL = make(map[Id]bool)
	}
	d.buf.searchHL[vid] = true
}

// SearchHighlightsActive reports whether search matches should be highlighted
// for a view
func (d *Document) SearchHighlightsActive(vid Id) bool {
	return d.buf.searchHL[vid]
}

// RemoveView cleans up selection and LSP state for a closed view
func (d *Document) RemoveView(vid Id) {
	d.rememberSelection(vid)
	delete(d.buf.selections, vid)
	delete(d.buf.searchHL, vid)
	d.ls.Lock()
	delete(d.ls.highlights, vid)
	delete(d.ls.hints, vid)
	d.ls.Unlock()
	d.track.Lock()
	delete(d.track.dirty, vid)
	d.track.Unlock()
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
	d.ensureLoaded()
	newText, err := tx.Apply(d.buf.text)
	if err != nil {
		return err
	}

	if d.buf.insertAcc != nil {
		cs := tx.Changes()
		newSel := d.resolveAppliedSelection(vid, tx, cs)
		oldSel := d.buf.selections[vid]
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
		if !cs.Empty() {
			d.markAllDirty()
		} else if !oldSel.Equal(newSel) {
			d.markDirty(vid)
		}
		return nil
	}

	cs := tx.Changes()
	newSel := d.resolveAppliedSelection(vid, tx, cs)
	oldSel := d.buf.selections[vid]
	if !cs.Empty() {
		// Commit the FORWARD tx with the BEFORE state so Undo can restore it
		beforeSt := core.State{Doc: d.buf.text, Selection: d.SelectionFor(vid)}
		if err := d.buf.history.CommitRevision(tx, beforeSt); err != nil {
			return err
		}
	}
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
	if !cs.Empty() {
		d.markAllDirty()
	} else if !oldSel.Equal(newSel) {
		d.markDirty(vid)
	}
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

// ConsumeDirty reports whether vid's rendered state changed since the last
// call for vid, clearing the flag. A vid never seen before is dirty
func (d *Document) ConsumeDirty(vid Id) bool {
	d.track.Lock()
	defer d.track.Unlock()
	wasDirty, ok := d.track.dirty[vid]
	if d.track.dirty == nil {
		d.track.dirty = map[Id]bool{}
	}
	d.track.dirty[vid] = false
	return !ok || wasDirty
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
	d.buf.unsaved = d.Modified()
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
	d.buf.unsaved = d.Modified()
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

func (d *Document) markDirty(vid Id) {
	d.track.Lock()
	defer d.track.Unlock()
	if d.track.dirty == nil {
		d.track.dirty = map[Id]bool{}
	}
	d.track.dirty[vid] = true
}

func (d *Document) markAllDirty() {
	d.track.Lock()
	defer d.track.Unlock()
	if d.track.dirty == nil {
		d.track.dirty = map[Id]bool{}
	}
	// track.dirty (not buf.selections, empty until a view's own cursor
	// first moves) is the reliable registry: ConsumeDirty runs for every
	// visible view on every render regardless of selection state
	for vid := range d.track.dirty {
		d.track.dirty[vid] = true
	}
}

func (d *Document) rememberSelection(vid Id) {
	if sel, ok := d.buf.selections[vid]; ok {
		d.buf.lastSel = sel
	}
}

func (d *Document) resolveAppliedSelection(
	vid Id, tx core.Transaction, cs core.ChangeSet,
) core.Selection {
	if txSel := tx.Selection(); txSel != nil {
		return *txSel
	}
	cur := d.SelectionFor(vid)
	if mapped, err := cur.Map(cs); err == nil {
		return mapped
	}
	return cur
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
