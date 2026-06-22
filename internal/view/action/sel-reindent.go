package action

import (
	"strings"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// ReindentSelections normalizes the leading whitespace on each selected
// line to use the document's current indent style at the same depth
// Lines with mixed indentation (tabs and spaces) are converted
func ReindentSelections(e *view.Editor) {
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
	unit := doc.IndentStyle().AsStr()
	tabW := doc.TabWidth()

	lines := selectionLines(text, sel)
	changes := make([]core.Change, 0, len(lines))
	for _, line := range lines {
		lineStart, err := text.LineToChar(line)
		if err != nil {
			continue
		}
		lineEnd, err := text.LineEndCharIndex(line)
		if err != nil {
			continue
		}
		// Measure existing leading whitespace in columns
		cols := 0
		wsEnd := lineStart
	wsLoop:
		for i := lineStart; i < lineEnd; i++ {
			ch, err2 := text.CharAt(i)
			if err2 != nil {
				break
			}
			switch ch {
			case ' ':
				cols++
				wsEnd = i + 1
			case '\t':
				cols = (cols/tabW + 1) * tabW
				wsEnd = i + 1
			default:
				break wsLoop
			}
		}
		// Rebuild indentation using current style
		var depth int
		if unit == "\t" {
			depth = cols / tabW
		} else {
			depth = cols / max(len(unit), 1)
		}
		newWS := strings.Repeat(unit, depth)
		// Collect old whitespace for comparison
		var sb strings.Builder
		for i := lineStart; i < wsEnd; i++ {
			ch, _ := text.CharAt(i)
			sb.WriteRune(ch)
		}
		if sb.String() == newWS {
			continue
		}
		changes = append(changes, core.TextChange(lineStart, wsEnd, newWS))
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
	tx := core.NewTransaction(text).WithChanges(cs).WithSelection(newSel)
	_ = e.Apply(tx)
}
