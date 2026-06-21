package action

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/language"
)

type (
	commentSpan struct {
		from int
		to   int
		sep  string
	}

	joinDedupKey struct {
		lineEnd int
		skip    int
	}

	selectionBoundsKey struct {
		from int
		to   int
	}
)

// CloseCurrentView closes the focused view. If the document has unsaved
// changes and there are other views, the close is blocked
func CloseCurrentView(e *view.Editor) {
	doc, _ := e.FocusedDocument()
	if doc != nil && doc.Modified() {
		all := e.AllViews()
		if len(all) > 1 {
			return
		}
	}
	e.CloseCurrentView()
}

// CloseCurrentViewForce closes the focused view unconditionally
func CloseCurrentViewForce(e *view.Editor) { e.CloseCurrentView() }

// HSplit opens the current document in a new horizontal split (stacked)
func HSplit(e *view.Editor) {
	if doc, ok := e.FocusedDocument(); ok {
		e.HSplit(doc.ID())
	}
}

// VSplit opens the current document in a new vertical split (side by side)
func VSplit(e *view.Editor) {
	if doc, ok := e.FocusedDocument(); ok {
		e.VSplit(doc.ID())
	}
}

// TransposeView flips the layout of the split container holding the focused
// view
func TransposeView(e *view.Editor) { e.Transpose() }

// JumpViewLeft moves focus to the nearest split to the left
func JumpViewLeft(e *view.Editor) { e.FocusDirection(view.DirectionLeft) }

// JumpViewRight moves focus to the nearest split to the right
func JumpViewRight(e *view.Editor) { e.FocusDirection(view.DirectionRight) }

// JumpViewUp moves focus to the nearest split above
func JumpViewUp(e *view.Editor) { e.FocusDirection(view.DirectionUp) }

// JumpViewDown moves focus to the nearest split below
func JumpViewDown(e *view.Editor) { e.FocusDirection(view.DirectionDown) }

// SwapViewLeft swaps the focused split with the one to its left
func SwapViewLeft(e *view.Editor) { e.SwapSplitInDirection(view.DirectionLeft) }

// SwapViewRight swaps the focused split with the one to its right
func SwapViewRight(e *view.Editor) {
	e.SwapSplitInDirection(view.DirectionRight)
}

// SwapViewUp swaps the focused split with the one above it
func SwapViewUp(e *view.Editor) { e.SwapSplitInDirection(view.DirectionUp) }

// SwapViewDown swaps the focused split with the one below it
func SwapViewDown(e *view.Editor) { e.SwapSplitInDirection(view.DirectionDown) }

// RotateView cycles focus to the next view in tree order, wrapping around
func RotateView(e *view.Editor) {
	next := e.Tree().Next()
	if next != view.InvalidViewId {
		e.FocusView(next)
	}
}

// CloseOtherViews closes every view except the currently focused one
func CloseOtherViews(e *view.Editor) {
	focused, _ := e.FocusedView()
	for _, v := range e.AllViews() {
		if focused == nil || v.ID() != focused.ID() {
			e.CloseView(v.ID())
		}
	}
}

// GotoLastAccessedFile switches to the most recently accessed alternate file
func GotoLastAccessedFile(e *view.Editor) {
	did, ok := e.PrevDocID()
	if !ok {
		return
	}
	for _, v := range e.AllViews() {
		if v.DocID() == did {
			e.FocusView(v.ID())
			return
		}
	}
}

// GotoLastModifiedFile switches focus to the most recently modified document
// that is not currently focused
func GotoLastModifiedFile(e *view.Editor) {
	curDID := view.InvalidDocumentId
	if v, ok := e.FocusedView(); ok {
		curDID = v.DocID()
	}
	ids := e.LastModifiedDocIDs()
	for _, did := range ids {
		if did == view.InvalidDocumentId || did == curDID {
			continue
		}
		for _, v := range e.AllViews() {
			if v.DocID() == did {
				e.FocusView(v.ID())
				return
			}
		}
	}
}

// RepeatLastMotion replays the most recently recorded repeatable motion
func RepeatLastMotion(e *view.Editor) {
	fn := e.LastMotion()
	if fn == nil {
		return
	}
	n := max(e.Count(), 1)
	for range n {
		fn(e)
	}
}

// GotoColumn moves each cursor to the Nth character column of its current line
func GotoColumn(e *view.Editor) {
	gotoColumn(e, false)
}

// ExtendToColumn extends each selection to the Nth character column
func ExtendToColumn(e *view.Editor) {
	gotoColumn(e, true)
}

func gotoColumn(e *view.Editor, extend bool) {
	col := max(e.Count(), 1)
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		cursor := r.Cursor(doc)
		line, err := doc.CharToLine(cursor)
		if err != nil {
			return r
		}
		lineStart, err := doc.LineToChar(line)
		if err != nil {
			return r
		}
		lineEnd, err := doc.LineEndCharIndex(line)
		if err != nil {
			return r
		}
		return r.PutCursor(doc, min(lineStart+col-1, lineEnd), extend)
	})
}

// Increment increments the integer in each selection range by count
func Increment(e *view.Editor) {
	incrementImpl(e, 1)
}

// Decrement decrements the integer in each selection range by count
func Decrement(e *view.Editor) {
	incrementImpl(e, -1)
}

// InsertTab inserts one indentation unit (tab or spaces) at each cursor
func InsertTab(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	if doc.Readonly() {
		return
	}
	tab := doc.IndentStyle().AsStr()
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	changes := make([]core.Change, 0, len(sel.Ranges()))
	seen := map[int]bool{}
	for _, r := range sel.Ranges() {
		pos := r.Cursor(text)
		if seen[pos] {
			continue
		}
		seen[pos] = true
		changes = append(changes, core.TextChange(pos, pos, tab))
	}
	if len(changes) == 0 {
		return
	}
	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return
	}
	newSel, err := sel.Map(cs)
	if err != nil {
		return
	}
	_ = e.Apply(core.NewTransaction(text).WithChanges(cs).WithSelection(newSel))
}

// SmartTab inserts a tab when all cursors have only whitespace to their left;
// otherwise jumps to the next snippet tabstop or parent node end (no-op when
// those subsystems are absent)
func SmartTab(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	allWhitespace := true
	for _, r := range sel.Ranges() {
		cursor := r.Cursor(text)
		lineNum, err := text.CharToLine(cursor)
		if err != nil {
			continue
		}
		lineStart, err := text.LineToChar(lineNum)
		if err != nil {
			continue
		}
		left, err := text.Slice(lineStart, cursor)
		if err != nil {
			continue
		}
		for _, ch := range left.String() {
			if ch != ' ' && ch != '\t' {
				allWhitespace = false
				break
			}
		}
		if !allWhitespace {
			break
		}
	}
	if !allWhitespace {
		return
	}
	InsertTab(e)
}

// SelectWithinRegex keeps only the parts of each selection that match regex
func SelectWithinRegex(e *view.Editor, pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	v, ok := e.FocusedView()
	if !ok {
		return nil
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return nil
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	var out []core.Range
	primary := 0
	for i, r := range sel.Ranges() {
		from, to := r.From(), r.To()
		slice, err := text.Slice(from, to)
		if err != nil {
			continue
		}
		s := slice.String()
		for _, loc := range re.FindAllStringIndex(s, -1) {
			start := from + byteOffsetToRuneOffset(s, loc[0])
			end := from + byteOffsetToRuneOffset(s, loc[1])
			out = append(out, core.NewRange(start, end))
		}
		if i == sel.PrimaryIndex() && len(out) > 0 {
			primary = len(out) - 1
		}
	}
	if len(out) == 0 {
		return nil
	}
	newSel, err := core.NewSelection(out, primary)
	if err != nil {
		return err
	}
	doc.SetSelectionFor(v.ID(), newSel)
	return nil
}

// SplitSelectionByRegex splits each selection range at every regex match
func SplitSelectionByRegex(e *view.Editor, pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	v, ok := e.FocusedView()
	if !ok {
		return nil
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return nil
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	var out []core.Range
	primary := 0
	for i, r := range sel.Ranges() {
		from, to := r.From(), r.To()
		slice, err := text.Slice(from, to)
		if err != nil {
			continue
		}
		s := slice.String()
		indices := re.FindAllStringIndex(s, -1)
		prev := 0
		for _, loc := range indices {
			if loc[0] > prev {
				out = append(out, core.NewRange(
					from+byteOffsetToRuneOffset(s, prev),
					from+byteOffsetToRuneOffset(s, loc[0]),
				))
			}
			prev = loc[1]
		}
		if prev < len(s) {
			out = append(out, core.NewRange(
				from+byteOffsetToRuneOffset(s, prev), to,
			))
		}
		if i == sel.PrimaryIndex() && len(out) > 0 {
			primary = len(out) - 1
		}
	}
	if len(out) == 0 {
		return nil
	}
	newSel, err := core.NewSelection(out, primary)
	if err != nil {
		return err
	}
	doc.SetSelectionFor(v.ID(), newSel)
	return nil
}

// KeepSelectionsMatching keeps only selection ranges whose text matches regex
func KeepSelectionsMatching(e *view.Editor, pattern string) error {
	return filterSelectionsImpl(e, pattern, false)
}

// RemoveSelectionsMatching removes selection ranges whose text matches regex
func RemoveSelectionsMatching(e *view.Editor, pattern string) error {
	return filterSelectionsImpl(e, pattern, true)
}

// PasteRegisterAtCursor inserts the contents of the given register at each
// cursor position (for use in insert mode)
func PasteRegisterAtCursor(e *view.Editor, reg rune) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	if doc.Readonly() {
		return
	}
	val, ok := e.Registers().First(reg)
	if !ok {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	changes := make([]core.Change, 0, len(ranges))
	seen := map[int]bool{}
	for _, r := range ranges {
		pos := r.Cursor(text)
		if seen[pos] {
			continue
		}
		seen[pos] = true
		changes = append(changes, core.TextChange(pos, pos, val))
	}
	if len(changes) == 0 {
		return
	}
	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return
	}
	newSel, err := sel.Map(cs)
	if err != nil {
		return
	}
	_ = e.Apply(core.NewTransaction(text).WithChanges(cs).WithSelection(newSel))
}

// SortSelections sorts the text of each selection range lexicographically,
// replacing each range with the sorted text. Mirrors :sort in the reference
func SortSelections(e *view.Editor, reverse, insensitive bool) error {
	v, ok := e.FocusedView()
	if !ok {
		return nil
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return nil
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	if len(ranges) < 2 {
		return errors.New(
			"sorting requires multiple selections; hint: split selection first")
	}

	fragments := make([]string, len(ranges))
	for i, r := range ranges {
		start, end := r.From(), r.To()
		if start > end {
			start, end = end, start
		}
		sl, _ := text.Slice(start, end)
		fragments[i] = sl.String()
	}

	cmp := func(a, b string) int {
		ka, kb := a, b
		if insensitive {
			ka = strings.ToLower(a)
			kb = strings.ToLower(b)
		}
		if ka < kb {
			return -1
		}
		if ka > kb {
			return 1
		}
		return 0
	}
	slices.SortStableFunc(fragments, func(a, b string) int {
		c := cmp(a, b)
		if reverse {
			return -c
		}
		return c
	})

	changes := make([]core.Change, len(ranges))
	for i, r := range ranges {
		start, end := r.From(), r.To()
		if start > end {
			start, end = end, start
		}
		changes[i] = core.TextChange(start, end, fragments[i])
	}
	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return err
	}
	newSel, err := sel.Map(cs)
	if err != nil {
		return err
	}
	return e.Apply(
		core.NewTransaction(text).WithChanges(cs).WithSelection(newSel))
}

// ReflowSelections reflows the text of each selection range to the given
// column width. Mirrors :reflow in the reference
func ReflowSelections(e *view.Editor, width int) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()

	changes := make([]core.Change, 0, len(ranges))
	for _, r := range ranges {
		start, end := r.From(), r.To()
		if start > end {
			start, end = end, start
		}
		fragRope, _ := text.Slice(start, end)
		reflowed := core.ReflowHardWrap(fragRope.String(), width)
		changes = append(changes, core.TextChange(start, end, reflowed))
	}
	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return
	}
	newSel, err := sel.Map(cs)
	if err != nil {
		return
	}
	_ = e.Apply(
		core.NewTransaction(text).WithChanges(cs).WithSelection(newSel))
}

// SetLineEnding changes the document line-ending style and rewrites existing
// line endings in the focused buffer to match
func SetLineEnding(e *view.Editor, le core.LineEnding) error {
	if _, ok := e.FocusedView(); !ok {
		return view.ErrNoView
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return view.ErrNoDocument
	}
	text := doc.Text()
	changes := lineEndingChanges(text.String(), le)
	if len(changes) == 0 {
		doc.SetLineEnding(le)
		return nil
	}
	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return err
	}
	if err := e.Apply(core.NewTransaction(text).WithChanges(cs)); err != nil {
		return err
	}
	doc.SetLineEnding(le)
	return nil
}

// CharInfo returns a string describing the grapheme at the primary cursor.
// Format: "<printable>" (U+XXXX ...) [Dec N] Hex xx [+xx ...]
func CharInfo(e *view.Editor) string {
	v, ok := e.FocusedView()
	if !ok {
		return ""
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return ""
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	start := sel.Primary().Cursor(text)
	end := core.NextGraphemeBoundary(text, start)
	if start == end {
		return ""
	}
	gr, err := text.Slice(start, end)
	if err != nil {
		return ""
	}
	grapheme := gr.String()

	var printable strings.Builder
	for _, c := range grapheme {
		switch c {
		case '\000':
			printable.WriteString(`\0`)
		case '\t':
			printable.WriteString(`\t`)
		case '\n':
			printable.WriteString(`\n`)
		case '\r':
			printable.WriteString(`\r`)
		default:
			printable.WriteRune(c)
		}
	}

	var uni strings.Builder
	uni.WriteString(" (")
	for i, c := range grapheme {
		if i != 0 {
			uni.WriteByte(' ')
		}
		_, _ = fmt.Fprintf(&uni, "U+%04x", c)
	}
	uni.WriteByte(')')

	var dec string
	if len(grapheme) == 1 && grapheme[0] < 0x80 {
		dec = fmt.Sprintf(" Dec %d", grapheme[0])
	}

	var hex strings.Builder
	for i, c := range grapheme {
		if i != 0 {
			hex.WriteString(" +")
		}
		for _, by := range []byte(string(c)) {
			_, _ = fmt.Fprintf(&hex, " %02x", by)
		}
	}

	return fmt.Sprintf(`"%s"%s%s Hex%s`,
		printable.String(), uni.String(), dec, hex.String())
}

// MatchBrackets moves each cursor to the matching bracket at that position
// Uses plaintext bracket matching
func MatchBrackets(e *view.Editor) {
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		pos := r.Cursor(doc)
		match, ok := core.FindMatchingBracketPlaintext(doc, pos)
		if !ok {
			return r
		}
		return r.PutCursor(doc, match, false)
	})
}

// YankJoin yanks all selection text joined by a separator to the active
// register (default '"'). Mirrors :yank-join
func YankJoin(e *view.Editor, sep string) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	reg := e.ActiveRegister()
	if reg == 0 {
		reg = defaultYankRegister
	}
	parts := yankFragments(text, sel)
	if len(parts) == 0 {
		return
	}
	e.Registers().Set(reg, strings.Join(parts, sep))
	e.SetMode(view.ModeNormal)
}

func dedentDelete(
	text core.Rope, r core.Range, tabWidth, indentWidth int,
) (core.Deletion, bool) {
	pos := r.Cursor(text)
	line, err := text.CharToLine(pos)
	if err != nil {
		return core.Deletion{}, false
	}
	lineStart, err := text.LineToChar(line)
	if err != nil {
		return core.Deletion{}, false
	}
	if pos == lineStart {
		return core.Deletion{}, false
	}
	// Verify the slice [lineStart, pos) is all whitespace
	width := 0
	for i := lineStart; i < pos; i++ {
		ch, err := text.CharAt(i)
		if err != nil || (ch != ' ' && ch != '\t') {
			return core.Deletion{}, false
		}
		if ch == '\t' {
			width += tabWidth
		} else {
			width++
		}
	}
	// If last char is a tab, delete one tab
	prevCh, err := text.CharAt(pos - 1)
	if err != nil {
		return core.Deletion{}, false
	}
	if prevCh == '\t' {
		return core.Deletion{From: pos - 1, To: pos}, true
	}
	// Otherwise delete enough spaces to reach the previous indent stop
	drop := width % indentWidth
	if drop == 0 {
		drop = indentWidth
	}
	start := pos
	for i := 0; i < drop; i++ {
		ch, err := text.CharAt(start - 1)
		if err != nil || ch != ' ' {
			break
		}
		start--
	}
	return core.Deletion{From: start, To: pos}, true
}

func joinSelectionsImpl(e *view.Editor, withSpace bool) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	if doc.Readonly() {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	lang := language.LoadLanguage(doc.Lang())
	commentTokens := slices.Clone(lang.CommentTokens)
	slices.SortFunc(commentTokens, func(a, b string) int {
		return len(b) - len(a)
	})

	var spans []commentSpan
	dedup := map[joinDedupKey]bool{}

	for _, r := range sel.Ranges() {
		lr, err := r.LineRange(text)
		if err != nil {
			continue
		}
		endLine := lr.To
		if lr.From == lr.To {
			endLine = min(lr.To+1, text.LenLines()-1)
		}
		firstPos, err := text.LineToChar(lr.From)
		if err != nil {
			continue
		}
		firstEnd, err := text.LineEndCharIndex(lr.From)
		if err != nil {
			continue
		}
		firstPos = skipHorizontalWhitespace(text, firstPos, firstEnd)
		currentToken := commentTokenAt(text, commentTokens, firstPos)
		for l := lr.From; l < endLine; l++ {
			span, token, ok := joinLinePair(
				text, commentTokens, currentToken, l)
			if !ok {
				continue
			}
			currentToken = token
			key := joinDedupKey{lineEnd: span.from, skip: span.to}
			if dedup[key] {
				continue
			}
			dedup[key] = true
			spans = append(spans, span)
		}
	}
	if len(spans) == 0 {
		return
	}
	slices.SortFunc(spans, func(a, b commentSpan) int {
		return a.from - b.from
	})

	changes := make([]core.Change, len(spans))
	for i, s := range spans {
		changes[i] = core.TextChange(s.from, s.to, s.sep)
	}
	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return
	}
	newSel, err := sel.Map(cs)
	if err != nil {
		return
	}
	if withSpace {
		if sel, ok := spaceJoinSelection(spans); ok {
			newSel = sel
		}
	}
	_ = e.Apply(core.NewTransaction(text).WithChanges(cs).WithSelection(newSel))
}

func skipHorizontalWhitespace(text core.Rope, from, to int) int {
	for from < to {
		ch, err := text.CharAt(from)
		if err != nil || (ch != ' ' && ch != '\t') {
			return from
		}
		from++
	}
	return from
}

func joinLinePair(
	text core.Rope, tokens []string, currentToken string, l int,
) (commentSpan, string, bool) {
	lineEnd, err := text.LineEndCharIndex(l)
	if err != nil {
		return commentSpan{}, currentToken, false
	}
	nextStart, err := text.LineToChar(l + 1)
	if err != nil {
		return commentSpan{}, currentToken, false
	}
	nextLineEnd, err := text.LineEndCharIndex(l + 1)
	if err != nil {
		return commentSpan{}, currentToken, false
	}
	skip := skipHorizontalWhitespace(text, nextStart, nextLineEnd)
	if token := commentTokenAt(text, tokens, skip); token != "" {
		if token == currentToken {
			skip += len([]rune(token))
			skip = skipHorizontalWhitespace(text, skip, nextLineEnd)
		} else {
			currentToken = token
		}
	}
	sep := " "
	if skip == nextLineEnd {
		sep = ""
	}
	return commentSpan{from: lineEnd, to: skip, sep: sep}, currentToken, true
}

func spaceJoinSelection(spans []commentSpan) (core.Selection, bool) {
	ranges := make([]core.Range, 0, len(spans))
	off := 0
	for _, s := range spans {
		if s.sep == "" {
			off += s.to - s.from
			continue
		}
		ranges = append(ranges, core.PointRange(s.from-off))
		off += s.to - s.from - 1
	}
	if len(ranges) == 0 {
		return core.Selection{}, false
	}
	sel, err := core.NewSelection(ranges, 0)
	return sel, err == nil
}

func commentTokenAt(text core.Rope, tokens []string, pos int) string {
	for _, token := range tokens {
		end := pos + len([]rune(token))
		s, err := text.Slice(pos, end)
		if err == nil && s.String() == token {
			return token
		}
	}
	return ""
}

func rotateSelectionContents(e *view.Editor, forward bool) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	if doc.Readonly() {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	n := len(ranges)
	if n == 0 {
		return
	}
	count := max(e.Count(), 1)
	steps := min(count, n)
	texts := make([]string, n)
	for i, r := range ranges {
		slice, err := text.Slice(r.From(), r.To())
		if err != nil {
			return
		}
		texts[i] = slice.String()
	}
	rotated := make([]string, n)
	var newPrimary int
	p := sel.PrimaryIndex()
	if forward {
		for i := range n {
			rotated[i] = texts[(i-steps+n)%n]
		}
		newPrimary = (p + steps) % n
	} else {
		for i := range n {
			rotated[i] = texts[(i+steps)%n]
		}
		newPrimary = (p + n - steps) % n
	}
	changes := make([]core.Change, n)
	for i, r := range ranges {
		changes[i] = core.TextChange(r.From(), r.To(), rotated[i])
	}
	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return
	}
	newRanges := rangesAfterReplace(ranges, rotated)
	newSel, err := core.NewSelection(newRanges, newPrimary)
	if err != nil {
		return
	}
	_ = e.Apply(core.NewTransaction(text).WithChanges(cs).WithSelection(newSel))
}

func scrollView(e *view.Editor, lines int, up bool) {
	if v, ok := e.FocusedView(); ok {
		scrollViewBy(e, v, max(e.ViewHeight(), 1), lines, up)
	}
}

// ScrollViewLines scrolls a specific view by n lines without changing keyboard
// focus, used for mouse-wheel events over a (possibly unfocused) pane. The
// pane's own height drives the scrolloff so stacked splits scroll correctly
func ScrollViewLines(e *view.Editor, v *view.View, n int, up bool) {
	scrollViewBy(e, v, max(v.Area().Height-1, 1), n, up)
}

func scrollViewBy(e *view.Editor, v *view.View, height, lines int, up bool) {
	doc, ok := e.Document(v.DocID())
	if !ok {
		return
	}
	if lines < 1 {
		lines = 1
	}
	text := doc.Text()
	so := min(e.Options().ScrollOff, max(height-1, 0)/2)

	offset := v.Offset()
	anchorLine, err := text.CharToLine(offset.Anchor)
	if err != nil {
		anchorLine = 0
	}
	nLines := text.LenLines()
	var newAnchorLine int
	if up {
		newAnchorLine = max(anchorLine-lines, 0)
	} else {
		newAnchorLine = min(anchorLine+lines, max(nLines-1, 0))
	}
	newAnchor, err := text.LineToChar(newAnchorLine)
	if err != nil {
		return
	}
	offset.Anchor = newAnchor
	v.SetOffset(offset)

	sel := doc.SelectionFor(v.ID())
	cursor := sel.Primary().Cursor(text)
	cursorLine, err := text.CharToLine(cursor)
	if err != nil {
		return
	}

	if up {
		newCursorLine := max(cursorLine-lines, 0)
		if newCursorLine == cursorLine {
			return
		}
		newCursorChar, err := text.LineToChar(newCursorLine)
		if err != nil {
			return
		}
		newSel := clampSelectionToLine(text, sel, newCursorChar)
		doc.SetSelectionFor(v.ID(), newSel)
	} else {
		topLine := min(newAnchorLine+so, max(nLines-1, 0))
		if cursorLine >= topLine {
			return
		}
		topChar, err := text.LineToChar(topLine)
		if err != nil {
			return
		}
		newSel := clampSelectionToLine(text, sel, topChar)
		doc.SetSelectionFor(v.ID(), newSel)
	}
}

func clampSelectionToLine(
	text core.Rope, sel core.Selection, targetChar int,
) core.Selection {
	line, err := text.CharToLine(targetChar)
	if err != nil {
		return sel
	}
	lineStart, err := text.LineToChar(line)
	if err != nil {
		return sel
	}
	ranges := sel.Ranges()
	newRanges := make([]core.Range, len(ranges))
	copy(newRanges, ranges)
	newRanges[sel.PrimaryIndex()] = core.PointRange(lineStart)
	newSel, err := core.NewSelection(newRanges, sel.PrimaryIndex())
	if err != nil {
		return sel
	}
	return newSel
}

func incrementImpl(e *view.Editor, sign int) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	if doc.Readonly() {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	count := max(e.Count(), 1)
	amount := sign * count
	increaseBy := 0
	if e.ActiveRegister() == '#' {
		increaseBy = sign
	}
	changes := make([]core.Change, 0, len(sel.Ranges()))
	seenKey := map[selectionBoundsKey]bool{}
	for _, r := range sel.Ranges() {
		from, to := r.From(), r.To()
		if from == to {
			from, to = wordBoundsAt(text, from)
		}
		key := selectionBoundsKey{from: from, to: to}
		if seenKey[key] {
			continue
		}
		seenKey[key] = true
		slice, err := text.Slice(from, to)
		if err != nil {
			amount += increaseBy
			continue
		}
		newS, ok := incrementInteger(slice.String(), amount)
		if !ok {
			amount += increaseBy
			continue
		}
		changes = append(changes, core.TextChange(from, to, newS))
		amount += increaseBy
	}
	if len(changes) == 0 {
		return
	}
	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return
	}
	newSel, err := sel.Map(cs)
	if err != nil {
		return
	}
	_ = e.Apply(core.NewTransaction(text).WithChanges(cs).WithSelection(newSel))
}

func filterSelectionsImpl(e *view.Editor, pattern string, remove bool) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	v, ok := e.FocusedView()
	if !ok {
		return nil
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return nil
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	var out []core.Range
	primary := 0
	for i, r := range sel.Ranges() {
		slice, err := text.Slice(r.From(), r.To())
		if err != nil {
			continue
		}
		if re.MatchString(slice.String()) != remove {
			out = append(out, r)
			if i == sel.PrimaryIndex() {
				primary = len(out) - 1
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	newSel, err := core.NewSelection(out, primary)
	if err != nil {
		return err
	}
	doc.SetSelectionFor(v.ID(), newSel)
	return nil
}

func wordBoundsAt(text core.Rope, pos int) (int, int) {
	n := text.LenChars()
	from := pos
	for from > 0 {
		ch, err := text.CharAt(from - 1)
		if err != nil || !isWordChar(ch) {
			break
		}
		from--
	}
	to := pos
	for to < n {
		ch, err := text.CharAt(to)
		if err != nil || !isWordChar(ch) {
			break
		}
		to++
	}
	return from, to
}

func isWordChar(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_'
}

func incrementInteger(s string, delta int) (string, bool) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		n, err := strconv.ParseInt(s[2:], 16, 64)
		if err != nil {
			return "", false
		}
		return s[:2] + strconv.FormatInt(n+int64(delta), 16), true
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return "", false
	}
	return strconv.FormatInt(n+int64(delta), 10), true
}

func rangesAfterReplace(
	ranges []core.Range, replacements []string,
) []core.Range {
	out := make([]core.Range, len(ranges))
	delta := 0
	for i, r := range ranges {
		newFrom := r.From() + delta
		newLen := len([]rune(replacements[i]))
		out[i] = core.NewRange(newFrom, newFrom+newLen)
		delta += newLen - (r.To() - r.From())
	}
	return out
}

func lineEndingChanges(s string, le core.LineEnding) []core.Change {
	var changes []core.Change
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		if runes[i] == '\r' && i+1 < len(runes) && runes[i+1] == '\n' {
			if le != core.LineEndingCRLF {
				changes = append(changes, core.TextChange(
					i, i+2, string(le),
				))
			}
			i++
			continue
		}
		if runes[i] == '\n' && le != core.LineEndingLF {
			changes = append(changes, core.TextChange(i, i+1, string(le)))
		}
	}
	return changes
}
