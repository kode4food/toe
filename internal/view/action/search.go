package action

import (
	"regexp"
	"strings"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/view"
)

const (
	StatusNoMoreMatches i18n.Key = "status.noMoreMatches"
	StatusSearchWrapped i18n.Key = "status.searchWrapped"
	StatusRegisterSet   i18n.Key = "status.registerSet"
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
	pat, ok := e.FirstRegister(view.RegisterSearch)
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
	e.WriteRegister(view.RegisterSearch, []string{out})
	setRegisterStatus(e, view.RegisterSearch, out)
}

// SearchForward executes a forward search with the given pattern, storing it
// in the '/' register, and moves each cursor to the first match
func SearchForward(e *view.Editor, pattern string) error {
	return searchImpl(searchArgs{
		editor:  e,
		pattern: pattern,
		forward: true,
		wrap:    e.Options().SearchWrapAround,
	})
}

// SearchBackward executes a backward search with the given pattern, storing
// it in the '/' register, and moves each cursor to the previous match
func SearchBackward(e *view.Editor, pattern string) error {
	return searchImpl(searchArgs{
		editor:  e,
		pattern: pattern,
		wrap:    e.Options().SearchWrapAround,
	})
}

// SearchNext repeats the last search forward, moving the selection
func SearchNext(e *view.Editor) {
	pat, ok := e.FirstRegister(view.RegisterSearch)
	if !ok {
		return
	}
	_ = searchImpl(searchArgs{
		editor:  e,
		pattern: pat,
		count:   e.CountOr(1),
		forward: true,
		wrap:    e.Options().SearchWrapAround,
	})
}

// SearchPrev repeats the last search backward, moving the selection
func SearchPrev(e *view.Editor) {
	pat, ok := e.FirstRegister(view.RegisterSearch)
	if !ok {
		return
	}
	_ = searchImpl(searchArgs{
		editor:  e,
		pattern: pat,
		count:   e.CountOr(1),
		wrap:    e.Options().SearchWrapAround,
	})
}

// ExtendSearchNext repeats the last search forward, extending the selection
func ExtendSearchNext(e *view.Editor) {
	pat, ok := e.FirstRegister(view.RegisterSearch)
	if !ok {
		return
	}
	_ = searchImpl(searchArgs{
		editor:  e,
		pattern: pat,
		count:   e.CountOr(1),
		forward: true,
		wrap:    e.Options().SearchWrapAround,
		extend:  true,
	})
}

// ExtendSearchPrev repeats the last search backward, extending the selection
func ExtendSearchPrev(e *view.Editor) {
	pat, ok := e.FirstRegister(view.RegisterSearch)
	if !ok {
		return
	}
	_ = searchImpl(searchArgs{
		editor:  e,
		pattern: pat,
		count:   e.CountOr(1),
		wrap:    e.Options().SearchWrapAround,
		extend:  true,
	})
}

func searchSelectionImpl(e *view.Editor, wordBoundaries bool) {
	doc := e.FocusedDocument()
	if doc == nil {
		return
	}
	v := e.FocusedView()
	if v == nil {
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
		slice, err := text.Slice(core.Span{From: from, To: to})
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
	e.WriteRegister(view.RegisterSearch, []string{pat})
	setRegisterStatus(e, view.RegisterSearch, pat)
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
	wrap := args.wrap
	re, err := compileSearchRegexp(pattern, e.Options().SearchSmartCase)
	if err != nil {
		return err
	}
	e.WriteRegister(view.RegisterSearch, []string{pattern})

	v := e.FocusedView()
	if v == nil {
		return nil
	}
	doc := e.FocusedDocument()
	if doc == nil {
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
			if args.forward {
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
			newRanges[i] = r.PutCursor(text, m.pos, args.extend)
		}
		setSearchStatus(e, setSearchStatusArgs{
			matched: matched,
			wrapped: wrapped,
		})
		newSel, err := core.NewSelection(newRanges, sel.PrimaryIndex())
		if err != nil {
			return nil
		}
		doc.SetSelectionFor(v.ID(), newSel)
	}
	doc.ShowSearchHighlights(v.ID())
	return nil
}

type setSearchStatusArgs struct {
	matched bool
	wrapped bool
}

func setSearchStatus(e *view.Editor, status setSearchStatusArgs) {
	if !status.matched {
		e.SetStatusMsg(i18n.Text(StatusNoMoreMatches))
		return
	}
	if status.wrapped {
		e.SetStatusMsg(i18n.Text(StatusSearchWrapped))
	}
}

func setRegisterStatus(e *view.Editor, reg rune, value string) {
	e.SetStatusMsg(i18n.Text(StatusRegisterSet, i18n.Vars{
		"register": string(reg),
		"value":    value,
	}))
}
