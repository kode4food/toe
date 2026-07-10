package action

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

const (
	searchRegister = '/'

	searchNoMoreMsg  = "No more matches"
	searchWrappedMsg = "Wrapped around document"
)

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
	pat, ok := e.FirstRegister(searchRegister)
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
	e.WriteRegister(searchRegister, []string{out})
	setRegisterStatus(e, searchRegister, out)
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
	pat, ok := e.FirstRegister(searchRegister)
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
	pat, ok := e.FirstRegister(searchRegister)
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
	pat, ok := e.FirstRegister(searchRegister)
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
	pat, ok := e.FirstRegister(searchRegister)
	if !ok {
		return
	}
	_ = searchImpl(searchArgs{
		editor: e, pattern: pat, count: countOrOne(e),
		wrap: e.Options().SearchWrapAround, extend: true,
	})
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
	e.WriteRegister(searchRegister, []string{pat})
	setRegisterStatus(e, searchRegister, pat)
}

type searchArgs struct {
	editor  *view.Editor
	pattern string
	count   int
	forward bool
	wrap    bool
	extend  bool
}

type searchMatch struct {
	pos     int
	wrapped bool
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
	e.WriteRegister(searchRegister, []string{pattern})

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
		matched := false
		wrapped := false
		for i, r := range ranges {
			cursor := r.Cursor(text)
			var m searchMatch
			if forward {
				m = findNextMatch(re, fullStr, cursor+1, wrap)
			} else {
				m = findPrevMatch(re, fullStr, cursor, wrap)
			}
			if m.pos < 0 {
				newRanges[i] = r
				continue
			}
			matched = true
			wrapped = wrapped || m.wrapped
			newRanges[i] = r.PutCursor(text, m.pos, extend)
		}
		setSearchStatus(e, matched, wrapped)
		newSel, err2 := core.NewSelection(newRanges, sel.PrimaryIndex())
		if err2 != nil {
			return nil
		}
		doc.SetSelectionFor(v.ID(), newSel)
	}
	doc.ShowSearchHighlights(v.ID())
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

func setSearchStatus(e *view.Editor, matched, wrapped bool) {
	if !matched {
		e.SetStatusMsg(searchNoMoreMsg)
		return
	}
	if wrapped {
		e.SetStatusMsg(searchWrappedMsg)
	}
}

func setRegisterStatus(e *view.Editor, reg rune, value string) {
	e.SetStatusMsg("register '" + string(reg) + "' set to '" + value + "'")
}

func hasUppercase(pattern string) bool {
	for _, ch := range pattern {
		if unicode.IsUpper(ch) {
			return true
		}
	}
	return false
}

func findNextMatch(
	re *regexp.Regexp, text string, from int, wrap bool,
) searchMatch {
	runes := []rune(text)
	wrapped := false
	if from >= len(runes) {
		if !wrap {
			return searchMatch{pos: -1}
		}
		from = 0
		wrapped = true
	}
	byteFrom := runeOffsetToByteOffset(text, from)
	for _, idx := range re.FindAllStringIndex(text[byteFrom:], -1) {
		if idx[0] == idx[1] {
			continue
		}
		pos := from + byteOffsetToRuneOffset(text[byteFrom:], idx[0])
		return searchMatch{pos: pos, wrapped: wrapped}
	}
	if wrap {
		for _, idx := range re.FindAllStringIndex(text[:byteFrom], -1) {
			if idx[0] == idx[1] {
				continue
			}
			pos := byteOffsetToRuneOffset(text, idx[0])
			return searchMatch{pos: pos, wrapped: true}
		}
	}
	return searchMatch{pos: -1}
}

func findPrevMatch(
	re *regexp.Regexp, text string, before int, wrap bool,
) searchMatch {
	runes := []rune(text)
	wrapped := false
	if before <= 0 {
		if !wrap {
			return searchMatch{pos: -1}
		}
		before = len(runes)
		wrapped = true
	}
	byteEnd := runeOffsetToByteOffset(text, before)
	all := re.FindAllStringIndex(text[:byteEnd], -1)
	if last, ok := lastNonEmptyMatch(all); ok {
		pos := byteOffsetToRuneOffset(text, last[0])
		return searchMatch{pos: pos, wrapped: wrapped}
	}
	if wrap {
		all2 := re.FindAllStringIndex(text[byteEnd:], -1)
		if last, ok := lastNonEmptyMatch(all2); ok {
			pos := before + byteOffsetToRuneOffset(text[byteEnd:], last[0])
			return searchMatch{pos: pos, wrapped: true}
		}
	}
	return searchMatch{pos: -1}
}

func lastNonEmptyMatch(matches [][]int) ([]int, bool) {
	for i := len(matches) - 1; i >= 0; i-- {
		m := matches[i]
		if m[0] != m[1] {
			return m, true
		}
	}
	return nil, false
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
	return utf8.RuneCountInString(s[:byteOff])
}
