package action

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

type newlineTarget struct {
	pos int
	off int
}

const searchRegister = '/'

// OpenAbove inserts a new line above each cursor's current line, places
// the cursor at the start of the new line, and enters insert mode
func OpenAbove(e *view.Editor) {
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
	count := max(e.Count(), 1)

	changes := make([]core.Change, 0, len(ranges))
	targets := make([]newlineTarget, 0, len(ranges)*count)
	seen := map[int]bool{}
	for _, r := range ranges {
		cursor := r.Cursor(text)
		line, err := text.CharToLine(cursor)
		if err != nil {
			continue
		}
		var insertPos int
		if line == 0 {
			insertPos = 0
		} else {
			insertPos, err = text.LineEndCharIndex(line - 1)
			if err != nil {
				continue
			}
		}
		if seen[insertPos] {
			continue
		}
		seen[insertPos] = true
		indent, _ := continuedIndent(e, doc, line, cursor)
		var unit string
		var firstOff int
		if line == 0 {
			unit = indent + "\n"
			firstOff = len([]rune(indent))
		} else {
			unit = "\n" + indent
			firstOff = len([]rune(unit))
		}
		changes = append(changes,
			core.TextChange(insertPos, insertPos, strings.Repeat(unit, count)),
		)
		unitLen := len([]rune(unit))
		for i := range count {
			targets = append(targets, newlineTarget{
				pos: insertPos,
				off: i*unitLen + firstOff,
			})
		}
	}
	applyNewlines(e, applyNewlinesArgs{
		text: text, sel: sel, changes: changes, targets: targets,
	})
}

// GotoLine moves (or extends in select mode) the cursor to line n (1-based)
// If n is 0 the command is a no-op. Clamps to the last non-empty line
func GotoLine(e *view.Editor, n int) {
	if n <= 0 {
		return
	}
	SaveSelection(e)
	extend := e.Mode() == view.ModeSelect
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		nLines := doc.LenLines()
		maxLine := nLines - 1
		// If the last line is blank, don't jump to it
		if maxLine > 0 {
			lastLineStart, err := doc.LineToChar(maxLine)
			if err == nil && lastLineStart >= doc.LenChars() {
				maxLine--
			}
		}
		line := min(n-1, maxLine)
		pos, err := doc.LineToChar(line)
		if err != nil {
			return r
		}
		return r.PutCursor(doc, pos, extend)
	})
}

// ReplaceChar replaces every grapheme in each selection with ch and exits
// select mode
func ReplaceChar(e *view.Editor, ch rune) {
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
	replacement := string(ch)

	changes := make([]core.Change, 0, len(ranges))
	for _, r := range ranges {
		if r.Empty() {
			continue
		}
		// Replace each grapheme in the range with ch
		frag, err := r.Fragment(text)
		if err != nil {
			continue
		}
		var b strings.Builder
		for range []rune(frag) {
			b.WriteString(replacement)
		}
		changes = append(changes, core.TextChange(r.From(), r.To(), b.String()))
	}
	applyChangesFrom(e, applyChangesFromArgs{text, sel, ranges, changes})
}

// FindCharArgs holds the parameters for a FindChar operation
type FindCharArgs struct {
	Editor    *view.Editor
	Ch        rune
	Forward   bool
	Inclusive bool
	Extend    bool
}

// FindChar moves (or extends) each cursor to the nth occurrence of ch in the
// given direction. inclusive=true lands on the char (f/F), false stops before/
// after it (t/T). extend=true keeps the anchor (select mode)
func FindChar(args FindCharArgs) {
	n := countOrOne(args.Editor)
	applyMove(args.Editor, func(doc core.Rope, r core.Range) core.Range {
		cursor := r.Cursor(doc)
		cursorHead := core.NextGraphemeBoundary(doc, cursor)

		// Compute first search start, preserving original skip semantics
		var fwd, bwd int
		if args.Forward {
			fwd = cursorHead
			if !args.Inclusive {
				fwd = cursorHead + 1
			}
		} else {
			if args.Inclusive {
				bwd = cursor - 1
			} else if cursor > 0 {
				bwd = cursor - 2
			} else {
				return r
			}
		}

		start := fwd
		search := findCharForward
		if !args.Forward {
			start = bwd
			search = findCharBackward
		}
		found := -1
		for range n {
			found, start = search(doc, start, args.Ch)
			if found == -1 {
				return r
			}
		}

		target := found
		if !args.Inclusive {
			if args.Forward {
				target--
			} else {
				target++
			}
		}

		if args.Extend {
			return r.PutCursor(doc, target, true)
		}
		return core.PointRange(cursor).PutCursor(doc, target, true)
	})
}

// ReplaceWithYanked replaces each selection with the corresponding value from
// the active register (default '"'). Exits select mode
func ReplaceWithYanked(e *view.Editor) {
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
	reg := e.ActiveRegister()
	if reg == 0 {
		reg = defaultYankRegister
	}
	values := e.Registers().Read(reg)
	if len(values) == 0 {
		return
	}
	n := max(e.Count(), 1)
	valueFor := func(i int) string {
		v := values[len(values)-1]
		if i < len(values) {
			v = values[i]
		}
		return strings.Repeat(v, n)
	}

	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()

	// valueFor uses a sequential counter that advances only for non-empty
	// ranges
	valueIdx := 0
	changes := make([]core.Change, 0, len(ranges))
	for _, r := range ranges {
		if r.Empty() {
			continue
		}
		changes = append(changes,
			core.TextChange(r.From(), r.To(), valueFor(valueIdx)),
		)
		valueIdx++
	}
	applyChangesFrom(e, applyChangesFromArgs{text, sel, ranges, changes})
}

// ExtendToLineBounds extends each selection range to cover complete lines
// (from line start to next-line start), preserving direction
func ExtendToLineBounds(e *view.Editor) {
	applyMove(e, func(text core.Rope, r core.Range) core.Range {
		lr, err := r.LineRange(text)
		if err != nil {
			return r
		}
		start, err := text.LineToChar(lr.From)
		if err != nil {
			return r
		}
		nLines := text.LenLines()
		endLine := lr.To + 1
		var end int
		if endLine >= nLines {
			end = text.LenChars()
		} else {
			end, err = text.LineToChar(endLine)
			if err != nil {
				return r
			}
		}
		return core.NewRange(start, end).WithDirection(r.Direction())
	})
}

// ShrinkToLineBounds shrinks each multi-line selection so that it no longer
// includes leading/trailing line endings. Single-line selections are unchanged
func ShrinkToLineBounds(e *view.Editor) {
	applyMove(e, func(text core.Rope, r core.Range) core.Range {
		lr, err := r.LineRange(text)
		if err != nil {
			return r
		}
		if lr.From == lr.To {
			return r
		}
		nLines := text.LenLines()
		start, err := text.LineToChar(lr.From)
		if err != nil {
			return r
		}
		endLine := lr.To + 1
		var end int
		if endLine >= nLines {
			end = text.LenChars()
		} else {
			end, err = text.LineToChar(endLine)
			if err != nil {
				return r
			}
		}
		if start != r.From() {
			nextLine := lr.From + 1
			if nextLine < nLines {
				start, err = text.LineToChar(nextLine)
				if err != nil {
					return r
				}
			}
		}
		if end != r.To() {
			end, err = text.LineToChar(lr.To)
			if err != nil {
				return r
			}
		}
		return core.NewRange(start, end).WithDirection(r.Direction())
	})
}

// RemovePrimarySelection removes the primary selection range. If only one
// range exists, the command is a no-op
func RemovePrimarySelection(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	sel := doc.SelectionFor(v.ID())
	if len(sel.Ranges()) == 1 {
		e.SetStatusMsg("no selections remaining")
		return
	}
	newSel, err := sel.Remove(sel.PrimaryIndex())
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}

// MergeSelections merges all selection ranges into one spanning range
func MergeSelections(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	sel := doc.SelectionFor(v.ID())
	doc.SetSelectionFor(v.ID(), sel.MergeRanges())
}

// MergeConsecutive merges overlapping or adjacent selection ranges
func MergeConsecutive(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	sel := doc.SelectionFor(v.ID())
	doc.SetSelectionFor(v.ID(), sel.MergeConsecutiveRanges())
}

// EnsureForward forces all selection ranges to have a forward
// direction (anchor <= head)
func EnsureForward(e *view.Editor) {
	applyMove(e, func(_ core.Rope, r core.Range) core.Range {
		return r.WithDirection(core.DirectionForward)
	})
}

// SearchSelection stores the joined selection text as the search pattern (no
// word-boundary detection) and sets it in the '/' register
func SearchSelection(e *view.Editor) {
	searchSelectionImpl(e, false)
}

// SearchSelectionWord stores the selection text as the search
// pattern, adding \b word-boundary anchors where the selection touches word
// boundaries
func SearchSelectionWord(e *view.Editor) {
	searchSelectionImpl(e, true)
}

// MakeSearchWordBounded wraps the current search pattern with \b word-boundary
// anchors if they are not already present
func MakeSearchWordBounded(e *view.Editor) {
	pat, ok := e.Registers().First(searchRegister)
	if !ok {
		return
	}
	startAnchored := len(pat) >= 2 && pat[:2] == `\b`
	endAnchored := len(pat) >= 2 && pat[len(pat)-2:] == `\b`
	if startAnchored && endAnchored {
		return
	}
	var out string
	if !startAnchored {
		out += `\b`
	}
	out += pat
	if !endAnchored {
		out += `\b`
	}
	e.Registers().Write(searchRegister, []string{out})
}

// CopyOnNextLine duplicates each selection range to the same column
// on the next line (count times)
func CopyOnNextLine(e *view.Editor) {
	copySelectionOnLine(e, true)
}

// CopyOnPrevLine duplicates each selection range to the same column
// on the previous line (count times)
func CopyOnPrevLine(e *view.Editor) {
	copySelectionOnLine(e, false)
}

// DeleteWordBackward deletes from the cursor to the start of the previous
// word, for use in insert mode (C-w)
func DeleteWordBackward(e *view.Editor) {
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

	changes := make([]core.Change, 0, len(ranges))
	seen := map[int]bool{}
	for _, r := range ranges {
		pos := r.Cursor(text)
		if pos == 0 || seen[pos] {
			continue
		}
		seen[pos] = true
		wordStart := core.MovePrevWordStart(
			text, core.PointRange(pos), 1,
		).From()
		changes = append(changes, core.DeleteChange(wordStart, pos))
	}
	applyDeletesAtCursor(e, applyDeletesArgs{
		text: text, sel: sel, ranges: ranges, changes: changes,
	})
}

// DeleteWordForward deletes from the cursor to the end of the next word,
// for use in insert mode (A-d)
func DeleteWordForward(e *view.Editor) {
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

	changes := make([]core.Change, 0, len(ranges))
	seen := map[int]bool{}
	for _, r := range ranges {
		pos := r.Cursor(text)
		if seen[pos] {
			continue
		}
		seen[pos] = true
		wordEnd := core.MoveNextWordEnd(text, core.PointRange(pos), 1).To()
		if wordEnd <= pos {
			continue
		}
		changes = append(changes, core.DeleteChange(pos, wordEnd))
	}
	applyDeletesAtCursor(e, applyDeletesArgs{
		text: text, sel: sel, ranges: ranges, changes: changes,
	})
}

// KillToLineEnd deletes from the cursor to the end of the line. If the cursor
// is already at the line ending, the newline itself is deleted
func KillToLineEnd(e *view.Editor) {
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

	changes := make([]core.Change, 0, len(ranges))
	seen := map[int]bool{}
	for _, r := range ranges {
		pos := r.Cursor(text)
		if seen[pos] {
			continue
		}
		seen[pos] = true
		line, err := text.CharToLine(pos)
		if err != nil {
			continue
		}
		lineEnd, err := text.LineEndCharIndex(line)
		if err != nil {
			continue
		}
		if pos == lineEnd {
			nextLine := line + 1
			if nextLine < text.LenLines() {
				next, err := text.LineToChar(nextLine)
				if err != nil {
					continue
				}
				changes = append(changes, core.DeleteChange(pos, next))
			}
		} else {
			changes = append(changes, core.DeleteChange(pos, lineEnd))
		}
	}
	applyDeletesAtCursor(e, applyDeletesArgs{
		text: text, sel: sel, ranges: ranges, changes: changes,
	})
}

// KillToLineStart deletes from the cursor to the start of the current line
// If the cursor is at the start, deletes the preceding newline (joins lines)
func KillToLineStart(e *view.Editor) {
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

	changes := make([]core.Change, 0, len(ranges))
	seen := map[int]bool{}
	for _, r := range ranges {
		pos := r.Cursor(text)
		if seen[pos] {
			continue
		}
		seen[pos] = true
		line, err := text.CharToLine(pos)
		if err != nil {
			continue
		}
		lineStart, err := text.LineToChar(line)
		if err != nil {
			continue
		}
		var head int
		if pos == lineStart {
			if line == 0 {
				continue
			}
			prevEnd, err := text.LineEndCharIndex(line - 1)
			if err != nil {
				continue
			}
			head = prevEnd
		} else {
			lineEnd, _ := text.LineEndCharIndex(line)
			firstNonWS := skipHorizontalWhitespace(text, lineStart, lineEnd)
			if firstNonWS < pos {
				head = firstNonWS
			} else {
				head = lineStart
			}
		}
		changes = append(changes, core.DeleteChange(head, pos))
	}
	applyDeletesAtCursor(e, applyDeletesArgs{
		text: text, sel: sel, ranges: ranges, changes: changes,
	})
}

type applyNewlinesArgs struct {
	text    core.Rope
	sel     core.Selection
	changes []core.Change
	targets []newlineTarget
}

func applyNewlines(e *view.Editor, args applyNewlinesArgs) {
	if len(args.changes) == 0 {
		e.SetMode(view.ModeInsert)
		return
	}
	cs, err := core.NewChangeSetFromChanges(args.text, args.changes)
	if err != nil {
		return
	}
	newRanges := make([]core.Range, len(args.targets))
	for i, target := range args.targets {
		pos, err := cs.MapPos(target.pos, core.AssocBefore)
		if err != nil {
			return
		}
		newRanges[i] = core.PointRange(pos + target.off)
	}
	primary := min(args.sel.PrimaryIndex(), len(newRanges)-1)
	newSel, err := core.NewSelection(newRanges, primary)
	if err != nil {
		return
	}
	tx := core.NewTransaction(args.text).WithChanges(cs).WithSelection(newSel)
	_ = e.Apply(tx)
	e.SetMode(view.ModeInsert)
}

type applyChangesFromArgs struct {
	text    core.Rope
	sel     core.Selection
	ranges  []core.Range
	changes []core.Change
}

func applyChangesFrom(e *view.Editor, args applyChangesFromArgs) {
	if len(args.changes) == 0 {
		return
	}
	cs, err := core.NewChangeSetFromChanges(args.text, args.changes)
	if err != nil {
		return
	}
	newRanges := make([]core.Range, len(args.ranges))
	for i, r := range args.ranges {
		mapped, err := cs.MapRange(r)
		if err != nil {
			return
		}
		newRanges[i] = core.PointRange(mapped.From())
	}
	newSel, err := core.NewSelection(newRanges, args.sel.PrimaryIndex())
	if err != nil {
		return
	}
	tx := core.NewTransaction(args.text).WithChanges(cs).WithSelection(newSel)
	_ = e.Apply(tx)
	e.SetMode(view.ModeNormal)
}

type applyDeletesArgs struct {
	text    core.Rope
	sel     core.Selection
	ranges  []core.Range
	changes []core.Change
}

func applyDeletesAtCursor(e *view.Editor, args applyDeletesArgs) {
	if len(args.changes) == 0 {
		return
	}
	cs, err := core.NewChangeSetFromChanges(args.text, args.changes)
	if err != nil {
		return
	}
	newRanges := make([]core.Range, len(args.ranges))
	for i, r := range args.ranges {
		mapped, err := cs.MapRange(r)
		if err != nil {
			return
		}
		newRanges[i] = core.PointRange(mapped.Head)
	}
	newSel, err := core.NewSelection(newRanges, args.sel.PrimaryIndex())
	if err != nil {
		return
	}
	tx := core.NewTransaction(args.text).WithChanges(cs).WithSelection(newSel)
	_ = e.Apply(tx)
}

func searchSelectionImpl(e *view.Editor, wordBoundaries bool) {
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	var parts []string
	for _, r := range sel.Ranges() {
		from, to := r.From(), r.To()
		if from >= to {
			continue
		}
		slice, err := text.Slice(from, to)
		if err != nil {
			continue
		}
		parts = append(parts, regexp.QuoteMeta(slice.String()))
	}
	if len(parts) == 0 {
		return
	}
	pat := strings.Join(parts, "|")
	if wordBoundaries {
		pat = `\b(?:` + pat + `)\b`
	}
	e.Registers().Write(searchRegister, []string{pat})
}

func pasteImpl(e *view.Editor, before bool) {
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
	reg := e.ActiveRegister()
	if reg == 0 {
		reg = defaultYankRegister
	}
	values := e.Registers().Read(reg)
	if len(values) == 0 {
		return
	}

	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()

	linewise := false
	for _, val := range values {
		if len(val) > 0 && val[len(val)-1] == '\n' {
			linewise = true
			break
		}
	}

	valueFor := func(i int) string {
		if i < len(values) {
			return values[i]
		}
		return values[len(values)-1]
	}

	pastePos := make([]int, len(ranges))
	for i := range pastePos {
		pastePos[i] = -1
	}
	changes := make([]core.Change, 0, len(ranges))
	for i, r := range ranges {
		pos, ok := pastePosition(text, r, linewise, before)
		if !ok {
			continue
		}
		pastePos[i] = pos
		changes = append(changes, core.TextChange(pos, pos, valueFor(i)))
	}
	if len(changes) == 0 {
		return
	}

	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return
	}

	newRanges := make([]core.Range, len(ranges))
	for i, r := range ranges {
		if pastePos[i] < 0 {
			newRanges[i] = r
			continue
		}
		newPos, err := cs.MapPos(pastePos[i], core.AssocBeforeSticky)
		if err != nil {
			newRanges[i] = r
			continue
		}
		newRanges[i] = core.PointRange(newPos)
	}
	newSel, err := core.NewSelection(newRanges, sel.PrimaryIndex())
	if err != nil {
		return
	}
	tx := core.NewTransaction(text).WithChanges(cs).WithSelection(newSel)
	_ = e.Apply(tx)
}

// SearchForward executes a forward search with the given pattern, storing it
// in the '/' register, and moves each cursor to the first match
func SearchForward(e *view.Editor, pattern string) error {
	return searchImpl(searchArgs{
		editor: e, pattern: pattern,
		forward: true, wrap: e.Options().SearchWrapAround,
	})
}

// SearchBackward executes a backward search with the given pattern, storing
// it in the '/' register, and moves each cursor to the previous match
func SearchBackward(e *view.Editor, pattern string) error {
	return searchImpl(searchArgs{
		editor: e, pattern: pattern,
		wrap: e.Options().SearchWrapAround,
	})
}

// SearchNext repeats the last search forward, moving the selection
func SearchNext(e *view.Editor) {
	pat, ok := e.Registers().First(searchRegister)
	if !ok {
		return
	}
	_ = searchImpl(searchArgs{
		editor: e, pattern: pat, count: countOrOne(e), forward: true,
		wrap: e.Options().SearchWrapAround,
	})
}

// SearchPrev repeats the last search backward, moving the selection
func SearchPrev(e *view.Editor) {
	pat, ok := e.Registers().First(searchRegister)
	if !ok {
		return
	}
	_ = searchImpl(searchArgs{
		editor: e, pattern: pat, count: countOrOne(e),
		wrap: e.Options().SearchWrapAround,
	})
}

// ExtendSearchNext repeats the last search forward, extending the selection
func ExtendSearchNext(e *view.Editor) {
	pat, ok := e.Registers().First(searchRegister)
	if !ok {
		return
	}
	_ = searchImpl(searchArgs{
		editor: e, pattern: pat, count: countOrOne(e), forward: true,
		wrap: e.Options().SearchWrapAround, extend: true,
	})
}

// ExtendSearchPrev repeats the last search backward, extending the selection
func ExtendSearchPrev(e *view.Editor) {
	pat, ok := e.Registers().First(searchRegister)
	if !ok {
		return
	}
	_ = searchImpl(searchArgs{
		editor: e, pattern: pat, count: countOrOne(e),
		wrap: e.Options().SearchWrapAround, extend: true,
	})
}

type searchArgs struct {
	editor  *view.Editor
	pattern string
	count   int
	forward bool
	wrap    bool
	extend  bool
}

func searchImpl(args searchArgs) error {
	e := args.editor
	pattern := args.pattern
	forward := args.forward
	wrap := args.wrap
	extend := args.extend
	re, err := compileSearchRegexp(pattern, e.Options().SearchSmartCase)
	if err != nil {
		return err
	}
	e.Registers().Write(searchRegister, []string{pattern})

	v, ok := e.FocusedView()
	if !ok {
		return nil
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return nil
	}
	// text and its string form are stable across repeats; only the selection
	// advances, so compile the regex and materialize the text once
	text := doc.Text()
	fullStr := text.String()

	for range max(args.count, 1) {
		sel := doc.SelectionFor(v.ID())
		ranges := sel.Ranges()

		newRanges := make([]core.Range, len(ranges))
		for i, r := range ranges {
			cursor := r.Cursor(text)
			var pos int
			if forward {
				pos = findNextMatch(re, fullStr, cursor+1, wrap)
			} else {
				pos = findPrevMatch(re, fullStr, cursor, wrap)
			}
			if pos < 0 {
				newRanges[i] = r
				continue
			}
			newRanges[i] = r.PutCursor(text, pos, extend)
		}
		newSel, err2 := core.NewSelection(newRanges, sel.PrimaryIndex())
		if err2 != nil {
			return nil
		}
		doc.SetSelectionFor(v.ID(), newSel)
	}
	return nil
}

func compileSearchRegexp(
	pattern string, smartCase bool,
) (*regexp.Regexp, error) {
	if smartCase && !hasUppercase(pattern) {
		pattern = "(?i)" + pattern
	}
	return regexp.Compile(pattern)
}

func hasUppercase(pattern string) bool {
	for _, ch := range pattern {
		if unicode.IsUpper(ch) {
			return true
		}
	}
	return false
}

func findNextMatch(re *regexp.Regexp, text string, from int, wrap bool) int {
	runes := []rune(text)
	if from >= len(runes) {
		if !wrap {
			return -1
		}
		from = 0
	}
	byteFrom := runeOffsetToByteOffset(text, from)
	if idx := re.FindStringIndex(text[byteFrom:]); idx != nil {
		return from + byteOffsetToRuneOffset(text[byteFrom:], idx[0])
	}
	if wrap {
		if idx := re.FindStringIndex(text[:byteFrom]); idx != nil {
			return byteOffsetToRuneOffset(text, idx[0])
		}
	}
	return -1
}

func findPrevMatch(re *regexp.Regexp, text string, before int, wrap bool) int {
	runes := []rune(text)
	if before <= 0 {
		if !wrap {
			return -1
		}
		before = len(runes)
	}
	byteEnd := runeOffsetToByteOffset(text, before)
	all := re.FindAllStringIndex(text[:byteEnd], -1)
	if len(all) > 0 {
		last := all[len(all)-1]
		return byteOffsetToRuneOffset(text, last[0])
	}
	if wrap {
		all2 := re.FindAllStringIndex(text[byteEnd:], -1)
		if len(all2) > 0 {
			last := all2[len(all2)-1]
			return before + byteOffsetToRuneOffset(text[byteEnd:], last[0])
		}
	}
	return -1
}

func runeOffsetToByteOffset(s string, runeOff int) int {
	for i := range s {
		if runeOff == 0 {
			return i
		}
		runeOff--
	}
	return len(s)
}

func byteOffsetToRuneOffset(s string, byteOff int) int {
	return len([]rune(s[:byteOff]))
}

func pastePosition(
	text core.Rope, r core.Range, linewise, before bool,
) (int, bool) {
	if !linewise {
		if before {
			return r.From(), true
		}
		return r.To(), true
	}
	if before {
		line, err := text.CharToLine(r.From())
		if err != nil {
			return 0, false
		}
		pos, err := text.LineToChar(line)
		if err != nil {
			return 0, false
		}
		return pos, true
	}
	line, err := text.CharToLine(r.To())
	if err != nil {
		return 0, false
	}
	next := line + 1
	if next >= text.LenLines() {
		return text.LenChars(), true
	}
	pos, err := text.LineToChar(next)
	if err != nil {
		return 0, false
	}
	return pos, true
}

func findCharForward(doc core.Rope, start int, ch rune) (int, int) {
	for j := start; j < doc.LenChars(); j++ {
		c, err := doc.CharAt(j)
		if err != nil {
			break
		}
		if c == ch {
			return j, j + 1
		}
	}
	return -1, start
}

func findCharBackward(doc core.Rope, start int, ch rune) (int, int) {
	for j := start; j >= 0; j-- {
		c, err := doc.CharAt(j)
		if err != nil {
			break
		}
		if c == ch {
			return j, j - 1
		}
	}
	return -1, start
}
