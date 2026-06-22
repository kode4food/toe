package action

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

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
