package ui

import (
	"cmp"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

type (
	// Picker holds the runtime state for an open picker overlay
	Picker struct {
		source PickerSource

		list      listState
		preview   previewState
		load      loadState
		fileIcons map[string]pickerFileMarker
	}

	listState struct {
		listScroll
		items           []*PickerItem
		sections        []*PickerItem
		matched         []pickerMatch
		matchedSections int
		scores          map[pickerScoreKey]*MatchResult
		query           string
	}

	previewState struct {
		vScroll       int
		hScroll       int
		scrollFor     int
		cache         previewCache
		diffBaseCache map[diffBaseKey]core.Rope
	}

	loadState struct {
		feedCmd     tea.Cmd
		cancel      StopFunc
		loading     bool
		dynamicGen  int
		dynamicStop StopFunc
		refreshGen  int
		pending     map[string]struct{}

		wantTarget PickerTarget
		wantSet    bool
	}

	// SnapshotPickerSource returns its whole row set synchronously, so a
	// refresh can replace the list without a visible rebuild
	SnapshotPickerSource interface {
		PickerSource
		Items(e *view.Editor) []*PickerItem
	}

	// PickerFunc constructs a Picker from the editor
	PickerFunc func(e *view.Editor) *Picker

	// StopFunc cancels an in-progress feed or search
	StopFunc func()

	// PickerLoad is what a source returns from Load: the rows known up front,
	// an optional channel of rows discovered asynchronously, and the cancel for
	// that feed
	PickerLoad struct {
		Items []*PickerItem
		Feed  <-chan *PickerItem
		Stop  StopFunc
	}

	// PickerMatcher matches picker items against a prepared query
	PickerMatcher func(*PickerItem) (MatchResult, bool)

	// PickerSource is implemented by every picker data source
	PickerSource interface {
		ID() string
		Title() string
		Columns() []string
		MatchColumn() int
		ColumnProportions() []int
		Load(*view.Editor) PickerLoad
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
		// ItemsForPath returns the current rows for path, empty when the source
		// no longer contains it
		ItemsForPath(e *view.Editor, path string) []*PickerItem
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

	// PickerKeySource extends PickerSource with source-specific bindings,
	// consulted after the picker's own keys and before a typable key extends
	// the query
	PickerKeySource interface {
		PickerSource
		// HandleKey acts on a key, reporting whether it did. Any overlay it
		// returns is pushed above the picker, such as a confirmation
		HandleKey(
			*view.Editor, *PickerItem, command.KeyEvent,
		) (BufferOverlayComponent, bool)
	}

	// NavigablePickerSource extends PickerSource for pickers that can drill
	// into sub-pickers. Navigate returns a PickerFunc to replace the current
	// picker, or nil to fall through to Accept
	NavigablePickerSource interface {
		PickerSource
		Navigate(*view.Editor, *PickerItem) PickerFunc
	}

	// PickerItem is a single row shown in the picker list. A Section row labels
	// the group its Group ordinal names. It never matches a query and the
	// cursor skips it
	PickerItem struct {
		Display     string
		Columns     []string
		StyleScopes []string
		SecFrom     int
		SortKey     string

		Group     int
		Section   bool
		Directory bool

		Preview  PreviewRenderer
		Location PickerLocation
		Payload  any

		hunks       func() []view.DiffHunk
		DiffPreview bool
		DiffKind    view.FileChangeKind
		BasePath    string
	}

	// PickerItemSlab is a Slab specialized for PickerItem
	PickerItemSlab = core.Slab[PickerItem]

	// PreviewRenderer renders a picker item's preview at the given size
	PreviewRenderer func(geom.Size) string

	// PickerLocation holds a target and an optional line range
	PickerLocation struct {
		Target PickerTarget
		Lines  *core.Span
	}

	// PickerTarget identifies a document by path or in-memory ID. Variant
	// separates two rows naming the same document, as the changed-file picker's
	// staged and unstaged rows do
	PickerTarget struct {
		Path    string
		ID      view.DocumentId
		Variant int
	}

	// PickerAcceptAction is what accepting an item does with the pane it
	// opens into, such as replacing it or splitting
	PickerAcceptAction int

	// MatchResult is a source's verdict on one item: its rank against the
	// query, and the rune offsets to highlight in the matched column
	MatchResult struct {
		Score   int
		Indices []int
	}

	// PickerBase is an optional starting point a source can embed for default
	// id, title, column, and fuzzy-match behavior. A source is free to
	// implement those methods itself instead
	PickerBase struct {
		Ident       string
		Label       string
		Cols        []string
		MatchCol    int
		Proportions []int
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
// immediately. The returned feedCmd (if any) must be dispatched by the caller
// after mounting the component
func NewPicker(e *view.Editor, source PickerSource) *Picker {
	p := &Picker{
		source: source,
		list: listState{
			scores: map[pickerScoreKey]*MatchResult{},
		},
		preview: previewState{
			cache:         previewCache{},
			diffBaseCache: map[diffBaseKey]core.Rope{},
		},
		load: loadState{
			cancel: func() {},
		},
	}
	p.load.feedCmd = p.loadItems(e)
	return p
}

// DiffHunks returns the item's diff hunks, computing them on first use
func (p *PickerItem) DiffHunks() []view.DiffHunk {
	if p.hunks == nil {
		return nil
	}
	return p.hunks()
}

// TargetLines returns the line range the item opens at, taken from its first
// diff hunk when the source supplies hunks instead
func (p *PickerItem) TargetLines() *core.Span {
	if p.Location.Lines == nil && p.hunks != nil {
		p.Location.Lines = firstChangeLines(p.DiffHunks())
	}
	return p.Location.Lines
}

// Valid reports whether the target refers to a real document or path
func (p PickerTarget) Valid() bool {
	return p.Path != "" || p.ID != view.InvalidDocumentId
}

// ID is the picker's identifier, used to restore the last picker
func (p PickerBase) ID() string {
	return p.Ident
}

// Title is the picker's display title, drawn in the frame's top border
func (p PickerBase) Title() string {
	return p.Label
}

// Columns are the picker's column headings
func (p PickerBase) Columns() []string {
	return p.Cols
}

// MatchColumn is the column the filter query matches against
func (p PickerBase) MatchColumn() int {
	return p.MatchCol
}

// ColumnProportions are the relative widths of the columns
func (p PickerBase) ColumnProportions() []int {
	if len(p.Proportions) == len(p.Cols) {
		for _, proportion := range p.Proportions {
			if proportion > 0 {
				return p.Proportions
			}
		}
	}
	return defaultColumnProportions(len(p.Cols))
}

// PrepareMatcher prepares a query for matching multiple items
func (p PickerBase) PrepareMatcher(query string) PickerMatcher {
	f := newMatcher(query, p.Cols, p.MatchCol)
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
		p.load.wantSet = false
	}
}

func (p *Picker) awaitingQuery() bool {
	_, ok := p.source.(DynamicPickerSource)
	return ok && p.list.query == ""
}

func (p *Picker) loadItems(e *view.Editor) tea.Cmd {
	load := p.source.Load(e)
	items := load.Items
	feed := load.Feed
	p.load.cancel = load.Stop
	_, static := p.source.(StaticPickerSource)
	p.list.sections = nil
	p.list.items = p.takeSections(items)
	if static {
		p.refilter()
	} else {
		p.resetMatchesFromItems()
	}
	if feed == nil {
		p.load.loading = false
		return nil
	}
	p.load.loading = true
	done := make(chan struct{})
	closeDone := sync.OnceFunc(func() { close(done) })
	oldStop := p.load.cancel
	p.load.cancel = func() { oldStop(); closeDone() }
	return drainPickerFeed(feed, done)
}

func (p *Picker) reload(e *view.Editor) tea.Cmd {
	p.load.cancel()
	// a reload landing mid-refill would otherwise capture whatever row the
	// half-filled list is sitting on, losing the row the user chose
	if !p.load.wantSet {
		p.load.wantTarget, p.load.wantSet = p.selectedTarget()
	}
	cmd := p.loadItems(e)
	p.applyWantedSelection()
	return cmd
}

func (p *Picker) applyWantedSelection() {
	if p.load.wantSet && p.selectTarget(p.load.wantTarget) {
		p.load.wantSet = false
	}
}

func (p *Picker) selectTarget(target PickerTarget) bool {
	for i, m := range p.list.matched {
		if !m.item.Section && m.item.Location.Target == target {
			p.list.cursor = i
			return true
		}
	}
	return false
}

func (p *Picker) scheduleFileRefresh(path string) tea.Cmd {
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
		// a directory event carries coalesced changes with unknown members, and
		// Git state changes can regroup or remove rows
		if isGitStatePath(path) {
			p.clearPreviewCache()
			p.refreshItems(e)
			continue
		}
		if info, err := os.Lstat(path); err == nil && info.IsDir() {
			return p.reload(e)
		}
		p.reconcilePath(path, src.ItemsForPath(e, path))
	}
	return nil
}

func (p *Picker) refreshItems(e *view.Editor) {
	src, ok := p.source.(SnapshotPickerSource)
	if !ok {
		return
	}
	items := src.Items(e)
	if len(items) == 0 {
		return
	}
	target, hadSelection := p.selectedTarget()
	p.list.sections = nil
	p.list.items = p.takeSections(items)
	SortPickerItems(p.list.items)
	p.rematchPreservingSelection(target, hadSelection)
}

// a path holds as many rows as the source reports for it, so the whole set is
// replaced at once rather than matched up row by row
func (p *Picker) reconcilePath(path string, items []*PickerItem) {
	atPath := itemsAtPath(path)
	if len(items) == 0 && !slices.ContainsFunc(p.list.items, atPath) {
		return
	}
	target, hadSelection := p.selectedTarget()
	kept := slices.DeleteFunc(p.list.items, atPath)
	p.list.items = append(kept, items...)
	SortPickerItems(p.list.items)
	p.rematchPreservingSelection(target, hadSelection)
}

func (p *Picker) takeSections(items []*PickerItem) []*PickerItem {
	out := items[:0]
	for _, item := range items {
		if item.Section {
			p.list.sections = append(p.list.sections, item)
			continue
		}
		out = append(out, item)
	}
	return out
}

func (p *Picker) addItems(items []*PickerItem) {
	items = p.takeSections(items)
	if len(items) == 0 {
		return
	}
	if len(p.list.sections) > 0 {
		target, hadSelection := p.selectedTarget()
		p.list.items = append(p.list.items, items...)
		SortPickerItems(p.list.items)
		p.rematchPreservingSelection(target, hadSelection)
		return
	}
	start := len(p.list.items)
	p.list.items = append(p.list.items, items...)
	p.appendMatches(items, start)
}

func (p *Picker) appendMatches(items []*PickerItem, startIndex int) {
	src, _ := p.source.(StaticPickerSource)
	if src == nil {
		p.list.matched = unscoredItems(items, startIndex, p.list.matched)
	} else {
		match := p.prepareMatcher(src)
		p.list.matched = p.scoreItems(match, items, startIndex, p.list.matched)
	}
	p.applyWantedSelection()
	p.ensureSelectable()
	p.clampScroll()
}

func (p *Picker) finishLoad() {
	p.load.loading = false
	if _, ok := p.source.(StaticPickerSource); !ok {
		return
	}
	// cursor 0 is the append-order default, not a row anyone picked
	target, hadSelection := p.selectedTarget()
	SortPickerItems(p.list.items)
	p.rematchPreservingSelection(target, hadSelection && p.list.cursor != 0)
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
	p.applyWantedSelection()
	p.ensureSelectable()
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
	if p.selectTarget(target) {
		return
	}
	if p.list.cursor >= len(p.list.matched) {
		p.list.cursor = max(0, len(p.list.matched)-1)
	}
}

func (p *Picker) addDynamicItems(items []*PickerItem) {
	if len(items) == 0 {
		return
	}
	p.list.items = append(p.list.items, p.takeSections(items)...)
	p.resetMatchesFromItems()
}

func (p *Picker) resetMatchesFromItems() {
	p.list.matched = p.list.matched[:0]
	for i, item := range p.list.items {
		p.list.matched = append(
			p.list.matched, pickerMatch{item: item, itemIndex: i},
		)
	}
	p.insertSections()
	if p.list.cursor >= len(p.list.matched) {
		p.list.cursor = max(0, len(p.list.matched)-1)
	}
	p.ensureSelectable()
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
	p.preview.vScroll = 0
	p.load.dynamicGen++
	gen := p.load.dynamicGen
	q := p.list.query
	if q == "" {
		p.load.loading = false
		return nil
	}
	p.load.loading = true
	return func() tea.Msg {
		time.Sleep(pickerDynamicDelay)
		return pickerDynamicTriggerMsg{gen: gen, query: q}
	}
}

func (p *Picker) clearPreviewCache() {
	clear(p.preview.cache)
	clear(p.preview.diffBaseCache)
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
		v := acceptDocumentID(e, doc.ID(), action)
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

// SortPickerItems sorts items by sort key, falling back to display text, the
// default ordering for static picker sources
func SortPickerItems(items []*PickerItem) {
	slices.SortStableFunc(items, func(a, b *PickerItem) int {
		return cmp.Compare(pickerSortText(a), pickerSortText(b))
	})
}

// PickerNamePath renders a path name-first, trailing the directory holding it,
// and returns the rune offset where that directory begins
func PickerNamePath(rel string) (string, int) {
	dir := filepath.Dir(rel)
	if dir == "" || dir == "." {
		return rel, 0
	}
	name := filepath.Base(rel)
	return name + " " + dir, len([]rune(name)) + 1
}

// PickerTrailingPath trails a path behind a row's own text, and returns the
// rune offset where that path begins
func PickerTrailingPath(text, rel string) (string, int) {
	if text == "" {
		return rel, 0
	}
	return text + " " + rel, len([]rune(text)) + 1
}

func pickerSortText(item *PickerItem) string {
	if item.SortKey != "" {
		return item.SortKey
	}
	return item.Display
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

func itemsAtPath(path string) func(*PickerItem) bool {
	key := loader.CanonicalPath(path)
	return func(it *PickerItem) bool {
		target := it.Location.Target.Path
		return target == path || loader.CanonicalPath(target) == key
	}
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
