package action

import (
	"errors"
	"fmt"
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

// ExtendToColumn extends each selection to the Nth character column
func ExtendToColumn(e *view.Editor) {
	gotoColumn(e, true)
}

// Increment increments the integer in each selection range by count
func Increment(e *view.Editor) {
	incrementImpl(e, 1)
}

// Decrement decrements the integer in each selection range by count
func Decrement(e *view.Editor) {
	incrementImpl(e, -1)
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
