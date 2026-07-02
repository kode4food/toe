package action

import (
	"slices"
	"unicode/utf8"

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
)

// JoinSelections joins the lines within each selected line range by replacing
// each line ending (and surrounding whitespace) with nothing
func JoinSelections(e *view.Editor) {
	joinSelectionsImpl(e, false)
}

// JoinSelectionsSpace joins the lines within each selected line range by
// replacing each line ending (and surrounding whitespace) with a single space
func JoinSelectionsSpace(e *view.Editor) {
	joinSelectionsImpl(e, true)
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
	if doc.ReadOnly() {
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
			skip += utf8.RuneCountInString(token)
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
		end := pos + utf8.RuneCountInString(token)
		s, err := text.Slice(pos, end)
		if err == nil && s.String() == token {
			return token
		}
	}
	return ""
}
