package ui

import (
	"cmp"
	"errors"
	"os"
	"slices"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/view"
)

type (
	// Picker holds the runtime state for an open picker overlay
	Picker struct {
		source PickerSource

		list    listState
		preview previewState
		load    loadState
	}

	listState struct {
		items   []PickerItem
		matched []pickerMatch
		scores  map[pickerScoreKey]*MatchResult
		query   string
		cursor  int
		scroll  int
		height  int
	}

	previewState struct {
		scroll        int
		scrollFor     int
		cache         previewCache
		diffBaseCache map[string]core.Rope
	}

	loadState struct {
		feedCmd        tea.Cmd
		cancel         StopFunc
		dynamicGen     int
		dynamicStop    StopFunc
		dynamicPending bool
		refreshGen     int
		pending        map[string]struct{}
	}

	// PickerFunc constructs a Picker from the editor
	PickerFunc func(e *view.Editor) *Picker

	// StopFunc cancels an in-progress feed or search
	StopFunc func()

	// PickerMatcher matches picker items against a prepared query
	PickerMatcher func(*PickerItem) (MatchResult, bool)

	// PickerSource is implemented by every picker data source
	PickerSource interface {
		ID() string
		Columns() []string
		MatchColumn() int
		ColumnProportions() []int
		Load(*view.Editor) ([]PickerItem, <-chan PickerItem, StopFunc)
		Accept(*view.Editor, *PickerItem, PickerAcceptAction)
	}

	// PickerPreviewSkipper marks picker sources that never render previews
	PickerPreviewSkipper interface {
		SkipPreview()
	}

	// FileBackedPickerSource is a picker whose rows are workspace files, each
	// reconciled one changed path at a time
	FileBackedPickerSource interface {
		PickerSource
		// ItemForPath returns the current row for path and whether the source
		// contains it
		ItemForPath(e *view.Editor, path string) (PickerItem, bool)
	}

	// StaticPickerSource extends PickerSource with fuzzy-match filtering
	StaticPickerSource interface {
		PickerSource
		PrepareMatcher(query string) PickerMatcher
	}

	// DynamicPickerSource extends PickerSource with query-driven search
	DynamicPickerSource interface {
		PickerSource
		Search(query string)
	}

	// NavigablePickerSource extends PickerSource for pickers that can drill
	// into sub-pickers. Navigate returns a PickerFunc to replace the current
	// picker, or nil to fall through to Accept
	NavigablePickerSource interface {
		PickerSource
		Navigate(*view.Editor, *PickerItem) PickerFunc
	}

	// PickerItem is a single row shown in the picker list
	PickerItem struct {
		Display     string
		Columns     []string
		StyleScopes []string
		SortKey     string
		Preview     PreviewRenderer
		Location    PickerLocation
		Payload     any
		DiffHunks   []view.DiffHunk
		DiffPreview bool
		DiffKind    view.FileChangeKind
		BasePath    string
	}

	// PreviewRenderer renders a picker item's preview at the given size
	PreviewRenderer func(geom.Size) string

	// PickerLocation holds a target and an optional line range
	PickerLocation struct {
		Target PickerTarget
		Lines  *PickerLineRange
	}

	PickerLineRange struct {
		From int
		To   int
	}

	// PickerTarget identifies a document by path or in-memory ID
	PickerTarget struct {
		Path string
		ID   view.DocumentId
	}

	PickerAcceptAction int

	// MatchResult is a source's verdict on one item: its rank against the
	// query, and the rune offsets to highlight in the matched column
	MatchResult struct {
		Score   int
		Indices []int
	}

	// PickerBase is an optional starting point a source can embed for default
	// id, column, and fuzzy-match behavior; a source is free to implement
	// those methods itself instead
	PickerBase struct {
		id          string
		columns     []string
		matchColumn int
		proportions []int
	}

	pickerMatch struct {
		item      *PickerItem
		itemIndex int
		result    MatchResult
	}

	pickerScoreKey struct {
		query string
		text  string
	}

	pickerDynamicTriggerMsg struct {
		gen   int
		query string
	}

	pickerRefreshMsg struct {
		gen int
	}
)

const pickerDynamicDelay = 275 * time.Millisecond

const (
	PickerAcceptReplace PickerAcceptAction = iota
	PickerAcceptHorizontalSplit
	PickerAcceptVerticalSplit
)

// NewPicker constructs a Picker for the given source, triggering Load
// immediately. The returned feedCmd (if any) must be dispatched by the
// caller after mounting the component
func NewPicker(e *view.Editor, source PickerSource) *Picker {
	p := &Picker{
		source: source,
		list: listState{
			scores: map[pickerScoreKey]*MatchResult{},
		},
		preview: previewState{
			cache:         previewCache{},
			diffBaseCache: map[string]core.Rope{},
		},
		load: loadState{
			cancel: func() {},
		},
	}
	p.load.feedCmd = p.loadItems(e)
	return p
}

// Valid reports whether the target refers to a real document or path
func (p PickerTarget) Valid() bool {
	return p.Path != "" || p.ID != view.InvalidDocumentId
}

// NewPickerBase builds the fixed metadata a source embeds: kebab-case id,
// column headers, the column matched against, and each column's flex weight
func NewPickerBase(
	id string, columns []string, matchColumn int, proportions []int,
) PickerBase {
	return PickerBase{
		id:          id,
		columns:     columns,
		matchColumn: matchColumn,
		proportions: proportions,
	}
}

func (p PickerBase) ID() string {
	return p.id
}

func (p PickerBase) Columns() []string {
	return p.columns
}

func (p PickerBase) MatchColumn() int {
	return p.matchColumn
}

func (p PickerBase) ColumnProportions() []int {
	if len(p.proportions) == len(p.columns) {
		for _, proportion := range p.proportions {
			if proportion > 0 {
				return p.proportions
			}
		}
	}
	return defaultColumnProportions(len(p.columns))
}

// PrepareMatcher prepares a query for matching multiple items
func (p PickerBase) PrepareMatcher(query string) PickerMatcher {
	f := newMatcher(query, p.columns, p.matchColumn)
	return f.match
}

// MatchCount reports how many items currently match the query
func (p *Picker) MatchCount() int {
	return len(p.list.matched)
}

// SelectIndex moves the cursor to i when it is a valid match index
func (p *Picker) SelectIndex(i int) {
	if i >= 0 && i < len(p.list.matched) {
		p.list.cursor = i
	}
}

func (p *Picker) loadItems(e *view.Editor) tea.Cmd {
	items, feed, stop := p.source.Load(e)
	p.load.cancel = stop
	_, static := p.source.(StaticPickerSource)
	p.list.items = items
	if static {
		p.refilter()
	} else {
		p.list.matched = make([]pickerMatch, len(items))
		for i := range items {
			p.list.matched[i] = pickerMatch{item: &items[i], itemIndex: i}
		}
	}
	if feed == nil {
		return nil
	}
	done := make(chan struct{})
	closeDone := sync.OnceFunc(func() { close(done) })
	oldStop := p.load.cancel
	p.load.cancel = func() { oldStop(); closeDone() }
	return drainPickerFeed(feed, done)
}

func (p *Picker) reload(e *view.Editor) tea.Cmd {
	p.load.cancel()
	target, hadSelection := p.selectedTarget()
	cmd := p.loadItems(e)
	if hadSelection {
		p.restoreSelection(target)
	}
	return cmd
}

func (p *Picker) scheduleFileRefresh(path string) tea.Cmd {
	if _, ok := p.source.(DynamicPickerSource); ok {
		return p.dynamicTriggerCmd()
	}
	if _, ok := p.source.(FileBackedPickerSource); !ok {
		return nil
	}
	if p.load.pending == nil {
		p.load.pending = map[string]struct{}{}
	}
	p.load.pending[path] = struct{}{}
	p.load.refreshGen++
	gen := p.load.refreshGen
	return func() tea.Msg {
		time.Sleep(pickerDynamicDelay)
		return pickerRefreshMsg{gen: gen}
	}
}

func (p *Picker) flushFileChanges(e *view.Editor) tea.Cmd {
	pending := p.load.pending
	p.load.pending = nil
	src, ok := p.source.(FileBackedPickerSource)
	if !ok {
		return nil
	}
	for path := range pending {
		// a directory event carries coalesced changes with unknown members;
		// a full reload covers the batch
		if info, err := os.Lstat(path); err == nil && info.IsDir() {
			return p.reload(e)
		}
		item, exists := src.ItemForPath(e, path)
		p.reconcilePath(path, item, exists)
	}
	return nil
}

func (p *Picker) reconcilePath(path string, item PickerItem, exists bool) {
	target, hadSelection := p.selectedTarget()
	idx := p.findItemIndexByPath(path)
	switch {
	case exists && idx >= 0:
		p.list.items[idx] = item
	case exists:
		p.list.items = append(p.list.items, item)
		SortPickerItems(p.list.items)
	case idx >= 0:
		p.list.items = slices.Delete(p.list.items, idx, idx+1)
	default:
		return
	}
	p.rematchPreservingSelection(target, hadSelection)
}

func (p *Picker) findItemIndexByPath(path string) int {
	exact := slices.IndexFunc(p.list.items, func(it PickerItem) bool {
		return it.Location.Target.Path == path
	})
	if exact >= 0 {
		return exact
	}
	key := loader.CanonicalPath(path)
	return slices.IndexFunc(p.list.items, func(it PickerItem) bool {
		return loader.CanonicalPath(it.Location.Target.Path) == key
	})
}

func (p *Picker) addItems(items []PickerItem) {
	if len(items) == 0 {
		return
	}
	target, hadSelection := p.selectedTarget()
	p.list.items = append(p.list.items, items...)
	SortPickerItems(p.list.items)
	p.rematchPreservingSelection(target, hadSelection)
}

func (p *Picker) rematchPreservingSelection(
	target PickerTarget, hadSelection bool,
) {
	p.rebuildMatches()
	if hadSelection {
		p.restoreSelection(target)
	} else if p.list.cursor >= len(p.list.matched) {
		p.list.cursor = max(0, len(p.list.matched)-1)
	}
	p.clampScroll()
}

func (p *Picker) selectedTarget() (PickerTarget, bool) {
	if p.list.cursor < 0 || p.list.cursor >= len(p.list.matched) {
		return PickerTarget{}, false
	}
	target := p.list.matched[p.list.cursor].item.Location.Target
	return target, target.Valid()
}

func (p *Picker) restoreSelection(target PickerTarget) {
	for i, m := range p.list.matched {
		if m.item.Location.Target == target {
			p.list.cursor = i
			return
		}
	}
	if p.list.cursor >= len(p.list.matched) {
		p.list.cursor = max(0, len(p.list.matched)-1)
	}
}

func (p *Picker) addDynamicItems(items []PickerItem) {
	if len(items) == 0 {
		return
	}
	p.list.items = append(p.list.items, items...)
	p.list.matched = make([]pickerMatch, len(p.list.items))
	for i := range p.list.items {
		p.list.matched[i] = pickerMatch{item: &p.list.items[i], itemIndex: i}
	}
	if p.list.cursor >= len(p.list.matched) {
		p.list.cursor = max(0, len(p.list.matched)-1)
	}
}

func (p *Picker) dynamicTriggerCmd() tea.Cmd {
	_, isDynamic := p.source.(DynamicPickerSource)
	if !isDynamic {
		return nil
	}
	if p.load.dynamicStop != nil {
		p.load.dynamicStop()
		p.load.dynamicStop = nil
	}
	p.list.items = nil
	p.list.matched = nil
	p.list.cursor = 0
	p.list.scroll = 0
	p.preview.scroll = 0
	p.load.dynamicGen++
	gen := p.load.dynamicGen
	q := p.list.query
	if q == "" {
		p.load.dynamicPending = false
		return nil
	}
	p.load.dynamicPending = true
	return func() tea.Msg {
		time.Sleep(pickerDynamicDelay)
		return pickerDynamicTriggerMsg{gen: gen, query: q}
	}
}

func (p *Picker) clearPreviewCache() {
	clear(p.preview.cache)
	clear(p.preview.diffBaseCache)
}

// AcceptDocumentID opens the document by id, splitting per action, and
// returns the view now showing it
func AcceptDocumentID(
	e *view.Editor, id view.DocumentId, action PickerAcceptAction,
) *view.View {
	switch action {
	case PickerAcceptHorizontalSplit:
		return e.HSplit(id)
	case PickerAcceptVerticalSplit:
		return e.VSplit(id)
	default:
		return e.ShowDocument(id)
	}
}

// OpenPath opens a text document, image pane, or binary dump at path
func OpenPath(
	e *view.Editor, path string, action PickerAcceptAction,
) (*view.View, bool, error) {
	if path == "" {
		return nil, false, view.ErrDocumentNoPath
	}
	doc, err := e.SwitchOrOpenDoc(path)
	if err == nil {
		v := AcceptDocumentID(e, doc.ID(), action)
		return v, v != nil, nil
	}
	if !errors.Is(err, core.ErrBinaryFile) {
		return nil, false, err
	}
	var pane view.Pane
	if isImagePath(path) {
		pane, err = NewImagePane(e, path)
	} else {
		pane, err = NewBinaryPane(e, path)
	}
	if err != nil {
		return nil, false, err
	}
	if !acceptPickerPane(e, pane, action) {
		return nil, false, view.ErrNoView
	}
	return nil, true, nil
}

// AcceptPath opens the file at path (switching to it if already open),
// splitting per action, and returns the view now showing it
func AcceptPath(
	e *view.Editor, path string, action PickerAcceptAction,
) (*view.View, bool) {
	if path == "" {
		return nil, false
	}
	v, ok, err := OpenPath(e, path, action)
	if err != nil {
		e.SetStatusMsg(i18n.ErrorText(err))
		return nil, false
	}
	return v, ok
}

// AlignAcceptedView scrolls the view so the accepted document's cursor is
// visible after a picker jump
func AlignAcceptedView(e *view.Editor, v *view.View, doc *view.Document) {
	sel := doc.SelectionFor(v.ID())
	v.EnsureCursorVisible(
		doc.Text(), sel, max(v.Area().Height, e.ViewHeight()),
		e.Options().ScrollOff, nil,
	)
	v.EnsureCursorVisibleHorizontal(
		doc.Text(), sel, e.ViewContentWidth(), doc.TabWidth(),
		e.Options().ScrollOff,
	)
}

func alignAcceptedSelection(
	e *view.Editor, v *view.View, doc *view.Document,
) {
	text := doc.Text()
	primary := doc.SelectionFor(v.ID()).Primary()
	from := primary.From()
	fromLine, err := text.CharToLine(from)
	if err != nil {
		return
	}
	to := primary.To()
	if to > from {
		to--
	}
	toLine, err := text.CharToLine(to)
	if err != nil {
		return
	}
	height := v.Area().Height - 1
	if height <= 0 {
		height = max(e.ViewHeight(), 1)
	}
	opts := e.Options()
	width := max(v.Area().Width-gutterWidthFor(text, opts.Gutters), 0)
	format := doc.TextFormatForConfig(width, opts)
	anchor := (&selectionViewport{
		text:      text,
		format:    format,
		from:      fromLine,
		to:        toLine,
		height:    height,
		scrolloff: opts.ScrollOff,
	}).anchor()
	at, err := text.LineToChar(anchor.line)
	if err != nil {
		return
	}
	offset := v.Offset()
	offset.Anchor = at
	offset.VerticalOffset = anchor.offset
	v.SetOffset(offset)
}

func acceptPickerPane(
	e *view.Editor, pane view.Pane, action PickerAcceptAction,
) bool {
	switch action {
	case PickerAcceptHorizontalSplit:
		if !e.SplitPane(pane, view.LayoutHorizontal) {
			return false
		}
	case PickerAcceptVerticalSplit:
		if !e.SplitPane(pane, view.LayoutVertical) {
			return false
		}
	default:
		id := e.Tree().Focus()
		if e.Tree().Get(id) == nil {
			return false
		}
		old := e.ReplacePane(id, pane)
		e.DiscardPane(old)
	}
	return true
}

func defaultColumnProportions(n int) []int {
	proportions := make([]int, n)
	if n > 0 {
		proportions[0] = 1
	}
	return proportions
}

func previewEnabled(source PickerSource) bool {
	_, skip := source.(PickerPreviewSkipper)
	return !skip
}

func wantsFileWatchTree(source PickerSource) bool {
	if _, ok := source.(FileBackedPickerSource); ok {
		return true
	}
	_, ok := source.(DynamicPickerSource)
	return ok
}

// SortPickerItems sorts items by display text, the default ordering for
// static picker sources
func SortPickerItems(items []PickerItem) {
	slices.SortStableFunc(items, func(a, b PickerItem) int {
		return cmp.Compare(a.Display, b.Display)
	})
}
