package action

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

type selectionBoundsKey struct {
	from int
	to   int
}

// Increment increments the integer in each selection range by count
func Increment(e *view.Editor) {
	incrementImpl(e, 1)
}

// Decrement decrements the integer in each selection range by count
func Decrement(e *view.Editor) {
	incrementImpl(e, -1)
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
	if doc.ReadOnly() {
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
