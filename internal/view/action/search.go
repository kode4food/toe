package action

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

const searchRegister = '/'

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
	return utf8.RuneCountInString(s[:byteOff])
}
