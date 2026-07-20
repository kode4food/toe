package action

import (
	"strings"
	"unicode"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// SwitchCase toggles the case of every character in each selection
func SwitchCase(e *view.Editor) {
	switchCaseImpl(e, func(s string) string {
		var b strings.Builder
		for _, ch := range s {
			if unicode.IsLower(ch) {
				b.WriteRune(unicode.ToUpper(ch))
			} else if unicode.IsUpper(ch) {
				b.WriteRune(unicode.ToLower(ch))
			} else {
				b.WriteRune(ch)
			}
		}
		return b.String()
	})
}

// SwitchToUppercase converts every character in each selection to uppercase
func SwitchToUppercase(e *view.Editor) {
	switchCaseImpl(e, strings.ToUpper)
}

// SwitchToLowercase converts every character in each selection to lowercase
func SwitchToLowercase(e *view.Editor) {
	switchCaseImpl(e, strings.ToLower)
}

func switchCaseImpl(e *view.Editor, transform func(string) string) {
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
	ranges := sel.Ranges()

	changes := make([]core.Change, 0, len(ranges))
	for _, r := range ranges {
		frag, err := r.Fragment(text)
		if err != nil || frag == "" {
			continue
		}
		changes = append(changes,
			core.TextChange(r.From(), r.To(), transform(frag)),
		)
	}
	applyChangesFrom(e, applyChangesFromArgs{
		text:    text,
		sel:     sel,
		ranges:  ranges,
		changes: changes,
	})
}
