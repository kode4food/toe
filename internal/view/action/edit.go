package action

import (
	"strings"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/language"
)

type (
	insertEntry struct {
		del  core.Deletion
		newR core.Range
		pair bool
	}

	rangeKind int
)

const (
	kindNormal rangeKind = iota
	kindAutoPair
	kindDup
)

const defaultYankRegister = '"'

// SelectAll selects the entire document in the focused view
func SelectAll(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	n := doc.Text().LenChars()
	sel, err := core.NewSelection([]core.Range{core.NewRange(0, n)}, 0)
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), sel)
}

// CollapseSelection collapses every selection to its cursor position
func CollapseSelection(e *view.Editor) {
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
	for i, r := range ranges {
		pos := r.Cursor(text)
		ranges[i] = core.PointRange(pos)
	}
	newSel, err := core.NewSelection(ranges, sel.PrimaryIndex())
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}

// FlipSelections swaps anchor and head for every selection range
func FlipSelections(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	for i, r := range ranges {
		ranges[i] = r.Flip()
	}
	newSel, err := core.NewSelection(ranges, sel.PrimaryIndex())
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}

// KeepPrimarySelection discards all but the primary selection range
func KeepPrimarySelection(e *view.Editor) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	sel := doc.SelectionFor(v.ID())
	primary := sel.Primary()
	newSel, err := core.NewSelection(
		[]core.Range{core.NewRange(primary.Anchor, primary.Head)}, 0,
	)
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}

// ExtendLineBellow selects the current line(s) inclusive of the trailing
// newline. If the range already spans the full line, extends one more line
func ExtendLineBellow(e *view.Editor) {
	applyMove(e, func(doc core.Rope, r core.Range) core.Range {
		b, ok := resolveLineBounds(doc, r)
		if !ok {
			return r
		}
		if r.From() == b.start && r.To() == b.end {
			var nextEnd int
			var err error
			if b.endLine+2 >= doc.LenLines() {
				nextEnd = doc.LenChars()
			} else {
				nextEnd, err = doc.LineToChar(b.endLine + 2)
				if err != nil {
					return r
				}
			}
			return core.NewRange(b.start, nextEnd)
		}
		return core.NewRange(b.start, b.end)
	})
}

// SelectLineBelow selects current line(s) extending downward, direction-aware:
// forward selections grow at the head; backward selections shrink at the head
func SelectLineBelow(e *view.Editor) {
	selectLineImpl(e, false)
}

// SelectLineAbove selects current line(s) extending upward, direction-aware:
// backward selections grow at the head; forward selections shrink at the head
func SelectLineAbove(e *view.Editor) {
	selectLineImpl(e, true)
}

// DeleteSelection deletes all selections and enters normal mode
func DeleteSelection(e *view.Editor) {
	DeleteSelectionNoyank(e)
}

// InsertChar inserts a character at every cursor position in insert mode
// Auto-pairs are applied when the character matches an opener or closer
func InsertChar(e *view.Editor, ch rune) {
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
	staged := make([]core.Range, len(ranges))
	kinds := make([]rangeKind, len(ranges))
	seen := map[int]bool{}
	pairs, pairEnabled := autoPairsForDocument(e, doc)

	for i, r := range ranges {
		pos := r.Cursor(text)
		if seen[pos] {
			staged[i] = r
			kinds[i] = kindDup
			continue
		}
		seen[pos] = true
		if pairEnabled {
			change, newR, ok := core.HookInsert(text, r, ch, pairs)
			if ok {
				changes = append(changes, change)
				staged[i] = newR
				kinds[i] = kindAutoPair
				continue
			}
		}
		changes = append(changes, core.TextChange(pos, pos, string(ch)))
		staged[i] = r
		kinds[i] = kindNormal
	}

	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return
	}

	newRanges := make([]core.Range, len(ranges))
	for i, r := range staged {
		switch kinds[i] {
		case kindAutoPair:
			newRanges[i] = r
		default:
			mapped, err := cs.MapRange(r)
			if err != nil {
				return
			}
			newRanges[i] = core.PointRange(mapped.Head)
		}
	}
	newSel, err := core.NewSelection(newRanges, sel.PrimaryIndex())
	if err != nil {
		return
	}
	tx := core.NewTransaction(text).WithChanges(cs).WithSelection(newSel)
	_ = e.Apply(tx)
}

// InsertNewline inserts a newline at every cursor in insert mode, then
// replicates the previous line's leading whitespace (auto-indent)
func InsertNewline(e *view.Editor) {
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
	targets := make([]int, len(ranges))
	targetOffs := make([]int, len(ranges))
	seen := map[int]bool{}
	for i, r := range ranges {
		pos := r.Cursor(text)
		if seen[pos] {
			targets[i] = pos
			continue
		}
		seen[pos] = true
		line, err := text.CharToLine(pos)
		if err != nil {
			targets[i] = pos
			continue
		}
		lineStart, err := text.LineToChar(line)
		if err != nil {
			targets[i] = pos
			continue
		}
		// Find the last non-whitespace char on the current line up to pos
		firstTrailingWS := -1
		for i := pos - 1; i >= lineStart; i-- {
			ch, err := text.CharAt(i)
			if err != nil {
				break
			}
			if ch != ' ' && ch != '\t' {
				firstTrailingWS = i + 1
				break
			}
		}
		if firstTrailingWS < 0 {
			// Entire line up to pos is whitespace: insert bare newline at
			// line start, leaving old whitespace on the new line
			changes = append(changes,
				core.TextChange(lineStart, lineStart, "\n"),
			)
			targets[i] = lineStart
			targetOffs[i] = 1
		} else if firstTrailingWS < pos {
			// Trim trailing whitespace then insert newline with indent
			indent, continued := continuedIndent(e, doc, line, pos)
			insert, off := newlineInsertForCursor(newlineInsertArgs{
				editor: e, doc: doc, r: r,
				indent: indent, continued: continued,
			})
			changes = append(changes,
				core.TextChange(firstTrailingWS, pos, insert))
			targets[i] = firstTrailingWS
			targetOffs[i] = off
		} else {
			// No trailing whitespace: plain newline with indent
			indent, continued := continuedIndent(e, doc, line, pos)
			insert, off := newlineInsertForCursor(newlineInsertArgs{
				editor: e, doc: doc, r: r,
				indent: indent, continued: continued,
			})
			changes = append(changes, core.TextChange(pos, pos, insert))
			targets[i] = pos
			targetOffs[i] = off
		}
	}
	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return
	}
	newRanges := make([]core.Range, len(ranges))
	for i := range ranges {
		pos, err := cs.MapPos(targets[i], core.AssocBefore)
		if err != nil {
			return
		}
		newRanges[i] = core.PointRange(pos + targetOffs[i])
	}
	newSel, err := core.NewSelection(newRanges, sel.PrimaryIndex())
	if err != nil {
		return
	}
	_ = e.Apply(core.NewTransaction(text).WithChanges(cs).WithSelection(newSel))
}

// DeleteCharBackward deletes the character before each cursor in insert mode
// Dedents when the cursor is at the end of leading whitespace; otherwise
// deletes an auto-pair if applicable, or one grapheme backward
func DeleteCharBackward(e *view.Editor) {
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
	tabWidth := doc.TabWidth()
	indentWidth := doc.IndentStyle().IndentWidth(tabWidth)

	entries := make([]insertEntry, 0, len(ranges))
	seen := map[int]bool{}
	pairs, pairEnabled := autoPairsForDocument(e, doc)

	for _, r := range ranges {
		pos := r.Cursor(text)
		if pos == 0 || seen[pos] {
			entries = append(entries, insertEntry{})
			continue
		}
		seen[pos] = true
		// Dedent: if everything from line start to cursor is whitespace,
		// delete one indent unit
		if del, ok := dedentDelete(text, r, tabWidth, indentWidth); ok {
			entries = append(entries, insertEntry{del: del})
			continue
		}
		if pairEnabled {
			del, newR, ok := core.HookDelete(text, r, pairs)
			if ok {
				entries = append(entries, insertEntry{
					del:  del,
					newR: newR,
					pair: true,
				})
				continue
			}
		}
		prev := core.NthPrevGraphemeBoundary(text, pos, 1)
		entries = append(entries, insertEntry{
			del: core.Deletion{From: prev, To: pos},
		})
	}

	changes := make([]core.Change, 0, len(entries))
	for _, en := range entries {
		if en.del.From == en.del.To {
			continue
		}
		changes = append(changes, core.DeleteChange(en.del.From, en.del.To))
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
		en := entries[i]
		if en.pair && (en.del.From != en.del.To) {
			mapped, err := cs.MapRange(en.newR)
			if err != nil {
				return
			}
			newRanges[i] = mapped
		} else {
			mapped, err := cs.MapRange(r)
			if err != nil {
				return
			}
			newRanges[i] = core.PointRange(mapped.Anchor)
		}
	}
	newSel, err := core.NewSelection(newRanges, sel.PrimaryIndex())
	if err != nil {
		return
	}
	tx := core.NewTransaction(text).WithChanges(cs).WithSelection(newSel)
	_ = e.Apply(tx)
}

// DeleteCharForward deletes the character under each cursor
func DeleteCharForward(e *view.Editor) {
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
		if pos >= text.LenChars() || seen[pos] {
			continue
		}
		seen[pos] = true
		changes = append(changes, core.DeleteChange(pos, pos+1))
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
		mapped, err := cs.MapRange(r)
		if err != nil {
			return
		}
		newRanges[i] = core.PointRange(mapped.Anchor)
	}
	newSel, err := core.NewSelection(newRanges, sel.PrimaryIndex())
	if err != nil {
		return
	}
	tx := core.NewTransaction(text).WithChanges(cs).WithSelection(newSel)
	_ = e.Apply(tx)
}

// Yank copies the text of every selection range to the active register
// (defaulting to '"') and exits select mode
func Yank(e *view.Editor) {
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
	e.Registers().Write(reg, yankFragments(text, sel))
	e.SetMode(view.ModeNormal)
}

// PasteAfter pastes the active register's contents after each selection
func PasteAfter(e *view.Editor) {
	pasteImpl(e, false)
	e.SetMode(view.ModeNormal)
}

// PasteBefore pastes the active register's contents before each selection
func PasteBefore(e *view.Editor) {
	pasteImpl(e, true)
	e.SetMode(view.ModeNormal)
}

// ChangeSelection deletes the selection and enters insert mode
// For linewise (whole-line) selections, opens a blank line above
func ChangeSelection(e *view.Editor) {
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
	linewise := selectionIsLinewise(text, sel)

	ranges := sel.Ranges()
	reg := e.ActiveRegister()
	if reg == 0 {
		reg = defaultYankRegister
	}

	// Yank first, then delete
	values := make([]string, 0, len(ranges))
	for _, r := range ranges {
		frag, err := r.MinWidth1(text).Slice(text)
		if err != nil {
			continue
		}
		values = append(values, frag.String())
	}
	e.Registers().Write(reg, values)

	if !applyDeletions(e, applyDeletionsArgs{text, sel, ranges}) {
		return
	}
	if linewise {
		OpenAbove(e)
		return
	}
	e.SetMode(view.ModeInsert)
}

// SplitSelectionOnNewline splits each selection range on line boundaries,
// producing one sub-range per line (excluding the line ending itself)
func SplitSelectionOnNewline(e *view.Editor) {
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

	var newRanges []core.Range
	for _, r := range sel.Ranges() {
		if r.From() == r.To() {
			newRanges = append(newRanges, r)
			continue
		}
		from := r.From()
		to := r.To()
		pos := from
		for pos < to {
			line, err := text.CharToLine(pos)
			if err != nil {
				break
			}
			lineEnd, err := text.LineEndCharIndex(line)
			if err != nil {
				break
			}
			lineTotal, err := text.LineToChar(line + 1)
			if err != nil {
				// Last line (no newline at end)
				end := min(to, text.LenChars())
				if pos < end {
					newRanges = append(newRanges, core.NewRange(pos, end))
				}
				break
			}
			// lineEnd = newline char pos; lineTotal = start of next line
			end := min(lineEnd, to)
			if pos < end {
				newRanges = append(newRanges, core.NewRange(pos, end))
			}
			pos = lineTotal
		}
	}

	if len(newRanges) == 0 {
		return
	}
	newSel, err := core.NewSelection(newRanges, 0)
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}

// DeleteSelectionNoyank deletes each selection without yanking first
func DeleteSelectionNoyank(e *view.Editor) {
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
	if !applyDeletions(e, applyDeletionsArgs{text, sel, sel.Ranges()}) {
		return
	}
	e.SetMode(view.ModeNormal)
}

// ChangeSelectionNoyank deletes each selection without yanking and enters
// insert mode
func ChangeSelectionNoyank(e *view.Editor) {
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
	linewise := selectionIsLinewise(text, sel)
	if !applyDeletions(e, applyDeletionsArgs{text, sel, sel.Ranges()}) {
		return
	}
	if linewise {
		OpenAbove(e)
		return
	}
	e.SetMode(view.ModeInsert)
}

type resolveLineBoundsRes struct {
	startLine, endLine int
	start, end         int
}

func resolveLineBounds(
	doc core.Rope, r core.Range,
) (resolveLineBoundsRes, bool) {
	lr, err := r.LineRange(doc)
	if err != nil {
		return resolveLineBoundsRes{}, false
	}
	startLine, endLine := lr.From, lr.To
	start, err := doc.LineToChar(startLine)
	if err != nil {
		return resolveLineBoundsRes{}, false
	}
	var end int
	if endLine+1 >= doc.LenLines() {
		end = doc.LenChars()
	} else {
		end, err = doc.LineToChar(endLine + 1)
		if err != nil {
			return resolveLineBoundsRes{}, false
		}
	}
	return resolveLineBoundsRes{startLine, endLine, start, end}, true
}

func selectLineImpl(e *view.Editor, above bool) {
	v, ok := e.FocusedView()
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	text := doc.Text()
	nLines := text.LenLines()
	sat := func(line int) int { return min(line, nLines) }
	lineChar := func(line int) int {
		pos, _ := text.LineToChar(line)
		return pos
	}
	count := countOrOne(e)
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	for i, r := range ranges {
		lr, err := r.LineRange(text)
		if err != nil {
			continue
		}
		startLine, endLine := lr.From, lr.To
		start := lineChar(startLine)
		end := lineChar(sat(endLine + 1))

		// Snapping to line bounds counts as one step
		cnt := count
		if r.From() != start || r.To() != end {
			cnt = max(cnt-1, 0)
		}

		var anchorLine, headLine int
		dir := r.Direction()
		if above {
			switch dir {
			case core.DirectionForward:
				anchorLine = startLine
				headLine = max(endLine-cnt, 0)
			default:
				anchorLine = endLine
				headLine = max(startLine-cnt, 0)
			}
		} else {
			switch dir {
			case core.DirectionForward:
				anchorLine = startLine
				headLine = sat(endLine + cnt)
			default:
				anchorLine = endLine
				headLine = sat(startLine + cnt)
			}
		}

		var anchor, head int
		switch {
		case anchorLine < headLine:
			anchor = lineChar(anchorLine)
			head = lineChar(sat(headLine + 1))
		case anchorLine == headLine:
			if above {
				anchor = lineChar(sat(anchorLine + 1))
				head = lineChar(headLine)
			} else {
				anchor = lineChar(headLine)
				head = lineChar(sat(anchorLine + 1))
			}
		default:
			anchor = lineChar(sat(anchorLine + 1))
			head = lineChar(headLine)
		}
		ranges[i] = core.NewRange(anchor, head)
	}
	e.ResetCount()
	newSel, err := core.NewSelection(ranges, sel.PrimaryIndex())
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), newSel)
}

// leadingWhitespace returns the leading whitespace of the line containing pos
func leadingWhitespace(text core.Rope, pos int) string {
	line, err := text.CharToLine(pos)
	if err != nil {
		return ""
	}
	lineStart, err := text.LineToChar(line)
	if err != nil {
		return ""
	}
	lineEnd, err := text.LineEndCharIndex(line)
	if err != nil {
		return ""
	}
	var b strings.Builder
	for i := lineStart; i < lineEnd; i++ {
		ch, err := text.CharAt(i)
		if err != nil {
			break
		}
		if ch != ' ' && ch != '\t' {
			break
		}
		b.WriteRune(ch)
	}
	return b.String()
}

func continuedIndent(
	e *view.Editor, doc *view.Document, line, pos int,
) (string, bool) {
	text := doc.Text()
	indent := leadingWhitespace(text, pos)
	if !e.Options().ContinueComments {
		return indent, false
	}
	lang := language.LoadLanguage(doc.Lang())
	token, ok := core.GetCommentToken(text, lang.CommentTokens, line)
	if !ok {
		return indent, false
	}
	return indent + token + " ", true
}

type newlineInsertArgs struct {
	editor    *view.Editor
	doc       *view.Document
	r         core.Range
	indent    string
	continued bool
}

func newlineInsertForCursor(args newlineInsertArgs) (string, int) {
	text := args.doc.Text()
	pairs, ok := autoPairsForDocument(args.editor, args.doc)
	if args.continued || !ok || !betweenAutoPair(text, args.r, pairs) {
		insert := "\n" + args.indent
		return insert, len([]rune(insert))
	}
	inner := args.indent + args.doc.IndentStyle().AsStr()
	insert := "\n" + inner + "\n" + args.indent
	return insert, 1 + len([]rune(inner))
}

func betweenAutoPair(text core.Rope, r core.Range, pairs core.AutoPairs) bool {
	pos := r.Cursor(text)
	if pos == 0 {
		return false
	}
	prev, err := text.CharAt(pos - 1)
	if err != nil {
		return false
	}
	curr, err := text.CharAt(pos)
	if err != nil {
		return false
	}
	pair, ok := pairs.Get(prev)
	return ok && pair.Open == prev && pair.Close == curr
}

func yankFragments(text core.Rope, sel core.Selection) []string {
	parts := make([]string, 0, len(sel.Ranges()))
	for _, r := range sel.Ranges() {
		frag, err := r.Fragment(text)
		if err != nil {
			continue
		}
		parts = append(parts, frag)
	}
	return parts
}

type applyDeletionsArgs struct {
	text   core.Rope
	sel    core.Selection
	ranges []core.Range
}

func applyDeletions(e *view.Editor, args applyDeletionsArgs) bool {
	changes := make([]core.Change, 0, len(args.ranges))
	for _, r := range args.ranges {
		eff := r.MinWidth1(args.text)
		changes = append(changes, core.DeleteChange(eff.From(), eff.To()))
	}
	cs, err := core.NewChangeSetFromChanges(args.text, changes)
	if err != nil {
		return false
	}
	newRanges := make([]core.Range, len(args.ranges))
	for i, r := range args.ranges {
		eff := r.MinWidth1(args.text)
		mapped, err := cs.MapRange(eff)
		if err != nil {
			return false
		}
		newRanges[i] = core.PointRange(mapped.From())
	}
	newSel, err := core.NewSelection(newRanges, args.sel.PrimaryIndex())
	if err != nil {
		return false
	}
	tx := core.NewTransaction(args.text).WithChanges(cs).WithSelection(newSel)
	_ = e.Apply(tx)
	return true
}

func autoPairsForDocument(
	e *view.Editor, doc *view.Document,
) (core.AutoPairs, bool) {
	global, ok := e.Options().AutoPairs()
	if !ok {
		return core.AutoPairs{}, false
	}
	lang := language.LoadLanguage(doc.Lang())
	if pairs, ok := lang.AutoPairs.AutoPairs(); ok {
		return pairs, true
	}
	return global, true
}
