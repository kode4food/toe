package view

import (
	"fmt"
	"sync"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view/config"
	"github.com/kode4food/toe/internal/view/language"
)

type (
	// Document holds an open buffer and its editing, file, view, and render
	// state
	Document struct {
		identity identityState
		content  contentState
		edits    editState
		views    viewState
		format   formatState
		file     fileState
		overlays overlayState
		render   renderState
	}

	identityState struct {
		id         DocumentId
		accessedAt int64
	}

	contentState struct {
		sync.RWMutex
		path    string
		lang    string
		text    core.Rope
		version int
		pending *pendingLoad
	}

	editState struct {
		history            core.History
		insertAcc          *insertAccum
		savePoint          int
		changedSinceAccess bool
	}

	viewState struct {
		selections       map[Id]core.Selection
		lastSelection    core.Selection
		searchHighlights map[Id]bool
		restoreCursor    bool
	}

	formatState struct {
		hasBOM       bool
		indent       core.IndentStyle
		tabWidth     int
		lineEnding   core.LineEnding
		editorConfig *config.EditorConfig
		language     *language.Language
	}

	fileState struct {
		snapshot diskSnapshot
		external ExternalState
		readOnly bool
	}

	// overlayState holds all language-server-managed overlays for the document,
	// protected by an embedded RWMutex so async LSP goroutines write safely
	overlayState struct {
		sync.RWMutex
		highlights  map[Id][]DocumentHighlight
		links       []DocumentLink
		colors      []DocumentColor
		hints       map[Id][]InlayHint
		diagnostics []Diagnostic
	}

	// renderState tracks, per view, whether anything render-relevant has
	// changed since ConsumeDirty last checked
	renderState struct {
		sync.Mutex
		dirty map[Id]bool
	}

	// pendingLoad defers reading a restored file until its content is first
	// accessed
	pendingLoad struct {
		opts *Options
		lang string
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

	// EncodingUTF8 and EncodingUTF8BOM are the text-encoding display names
	// used in config and status display
	EncodingUTF8    = "utf-8"
	EncodingUTF8BOM = "utf-8-bom"
)

const (
	ExternalStateClean   ExternalState = iota // no external disk change pending
	ExternalStateChanged                      // changed while buffer dirty
	ExternalStateDeleted                      // backing file removed while open
)

// RestoreCursor reports whether the next exit from insert mode should move the
// cursor back by one grapheme
func (d *Document) RestoreCursor() bool {
	return d.views.restoreCursor
}

// SetRestoreCursor marks whether the next insert-mode exit should restore the
// cursor one grapheme to the left
func (d *Document) SetRestoreCursor(v bool) {
	d.views.restoreCursor = v
}

// ID returns the unique document identifier
func (d *Document) ID() DocumentId {
	return d.identity.id
}

// AccessedAt returns the monotonic focus/access sequence for MRU ordering
func (d *Document) AccessedAt() int64 {
	return d.identity.accessedAt
}

// Text returns the current rope text
func (d *Document) Text() core.Rope {
	d.ensureLoaded()
	d.content.RLock()
	defer d.content.RUnlock()
	return d.content.text
}

// Path returns the file path, or empty string for scratch buffers
func (d *Document) Path() string {
	d.content.RLock()
	defer d.content.RUnlock()
	return d.content.path
}

// SetPath sets the file path for this document
func (d *Document) SetPath(path string) {
	d.content.Lock()
	d.content.path = path
	d.content.Unlock()
}

// Modified reports whether the document has unsaved changes
func (d *Document) Modified() bool {
	return d.edits.history.CurrentRevision() != d.edits.savePoint
}

// Loaded reports whether the backing file has been read; a restored session
// buffer stays unloaded until its content is first accessed
func (d *Document) Loaded() bool {
	d.content.RLock()
	defer d.content.RUnlock()
	return d.content.pending == nil
}

// ExternalState reports any unresolved change made to the backing file by
// another process
func (d *Document) ExternalState() ExternalState {
	return d.file.external
}

// Lang returns the language identifier for syntax highlighting
func (d *Document) Lang() string {
	d.content.RLock()
	defer d.content.RUnlock()
	return d.content.lang
}

// SetLang sets the language identifier and resolves its definition once so
// the render path reads the cached *language.Language directly
func (d *Document) SetLang(lang string) {
	d.format.language = language.LoadLanguage(lang)
	d.content.Lock()
	d.content.lang = lang
	d.content.Unlock()
	d.MarkDirty()
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
	langDef := d.format.language
	if d.format.editorConfig != nil &&
		d.format.editorConfig.MaxLineLength != nil {
		cpy := *langDef
		cpy.TextWidth = d.format.editorConfig.MaxLineLength
		langDef = &cpy
	}
	format := language.TextFormatForConfig(
		langDef, opts.TextWidth, opts.SoftWrap, w,
	)
	format.TabWidth = d.format.tabWidth
	return format
}

// ReadOnly reports whether the document is read-only
func (d *Document) ReadOnly() bool {
	return d.file.readOnly
}

// SetReadOnly marks the document as read-only or writable
func (d *Document) SetReadOnly(v bool) {
	d.file.readOnly = v
}

// IndentStyle returns the active indentation style
func (d *Document) IndentStyle() core.IndentStyle {
	d.ensureLoaded()
	return d.format.indent
}

// SetIndentStyle updates the indent style for this document
func (d *Document) SetIndentStyle(s core.IndentStyle) {
	d.format.indent = s
	d.MarkDirty()
}

// TabWidth returns the display tab width
func (d *Document) TabWidth() int {
	d.ensureLoaded()
	return d.format.tabWidth
}

// LineEnding returns the document's line-ending style
func (d *Document) LineEnding() core.LineEnding {
	d.ensureLoaded()
	return d.format.lineEnding
}

// HasBOM reports whether the document was loaded with a UTF-8 BOM, which is
// preserved on save
func (d *Document) HasBOM() bool {
	d.ensureLoaded()
	return d.format.hasBOM
}

// SetLineEnding updates the line ending for this document
func (d *Document) SetLineEnding(le core.LineEnding) {
	d.format.lineEnding = le
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
	if sel, ok := d.views.selections[vid]; ok {
		return sel
	}
	return d.Selection()
}

// Selection returns the buffer's canonical cursor, independent of any view
func (d *Document) Selection() core.Selection {
	if len(d.views.lastSelection.Ranges()) > 0 {
		return d.views.lastSelection
	}
	return core.PointSelection(0)
}

// SetSelectionFor sets the selection for a view. Changing the selection clears
// any search-match highlighting for that view
func (d *Document) SetSelectionFor(vid Id, sel core.Selection) {
	if old, ok := d.views.selections[vid]; !ok || !old.Equal(sel) {
		d.markViewDirty(vid)
	}
	d.views.selections[vid] = sel
	delete(d.views.searchHighlights, vid)
}

// ShowSearchHighlights marks a view's search matches as visible. Search actions
// call this after moving the selection to their match
func (d *Document) ShowSearchHighlights(vid Id) {
	if d.views.searchHighlights == nil {
		d.views.searchHighlights = make(map[Id]bool)
	}
	d.views.searchHighlights[vid] = true
}

// SearchHighlightsActive reports whether search matches should be highlighted
// for a view
func (d *Document) SearchHighlightsActive(vid Id) bool {
	return d.views.searchHighlights[vid]
}

// RemoveView cleans up selection and LSP state for a closed view
func (d *Document) RemoveView(vid Id) {
	d.rememberSelection(vid)
	delete(d.views.selections, vid)
	delete(d.views.searchHighlights, vid)
	d.overlays.Lock()
	delete(d.overlays.highlights, vid)
	delete(d.overlays.hints, vid)
	d.overlays.Unlock()
	d.render.Lock()
	delete(d.render.dirty, vid)
	d.render.Unlock()
}

// LastEditPos returns the char offset of the most recently committed change
func (d *Document) LastEditPos() int {
	return d.edits.history.LastEditPos()
}

// Revision returns the document version used for render-cache invalidation
func (d *Document) Revision() int {
	d.content.RLock()
	defer d.content.RUnlock()
	return d.content.version
}

func (d *Document) MarkDirty() {
	d.render.Lock()
	defer d.render.Unlock()
	if d.render.dirty == nil {
		d.render.dirty = map[Id]bool{}
	}
	// render.dirty (not views.selections, empty until a view's own cursor
	// first moves) is the reliable registry: ConsumeDirty runs for every
	// visible view on every render regardless of selection state
	for vid := range d.render.dirty {
		d.render.dirty[vid] = true
	}
}

// ConsumeDirty reports whether vid's rendered state changed since the last
// call for vid, clearing the flag. A vid never seen before is dirty
func (d *Document) ConsumeDirty(vid Id) bool {
	d.render.Lock()
	defer d.render.Unlock()
	wasDirty, ok := d.render.dirty[vid]
	if d.render.dirty == nil {
		d.render.dirty = map[Id]bool{}
	}
	d.render.dirty[vid] = false
	return !ok || wasDirty
}

func (d *DocumentOpenError) Error() string {
	return fmt.Sprintf("open %s: %v", d.Path, d.Err)
}

func (d *DocumentOpenError) Unwrap() error {
	return d.Err
}

func (d *Document) markViewDirty(vid Id) {
	d.render.Lock()
	defer d.render.Unlock()
	if d.render.dirty == nil {
		d.render.dirty = map[Id]bool{}
	}
	d.render.dirty[vid] = true
}

func (d *Document) rememberSelection(vid Id) {
	if sel, ok := d.views.selections[vid]; ok {
		d.views.lastSelection = sel
	}
}
