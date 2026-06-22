package action

import (
	"strings"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// AddNewlineAbove inserts blank lines above each selection's first line.
// Repeats count times using the document line ending
func AddNewlineAbove(e *view.Editor) {
	addNewlineImpl(e, true)
}

// AddNewlineBelow inserts blank lines below each selection's last line. Repeats
// count times using the document line ending
func AddNewlineBelow(e *view.Editor) {
	addNewlineImpl(e, false)
}

// OpenAbove inserts a new line above each cursor's current line, places
// the cursor at the start of the new line, and enters insert mode
func OpenAbove(e *view.Editor) {
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
	count := max(e.Count(), 1)

	changes := make([]core.Change, 0, len(ranges))
	targets := make([]newlineTarget, 0, len(ranges)*count)
	seen := map[int]bool{}
	for _, r := range ranges {
		cursor := r.Cursor(text)
		line, err := text.CharToLine(cursor)
		if err != nil {
			continue
		}
		var insertPos int
		if line == 0 {
			insertPos = 0
		} else {
			insertPos, err = text.LineEndCharIndex(line - 1)
			if err != nil {
				continue
			}
		}
		if seen[insertPos] {
			continue
		}
		seen[insertPos] = true
		indent, _ := continuedIndent(e, doc, line, cursor)
		var unit string
		var firstOff int
		if line == 0 {
			unit = indent + "\n"
			firstOff = len([]rune(indent))
		} else {
			unit = "\n" + indent
			firstOff = len([]rune(unit))
		}
		changes = append(changes,
			core.TextChange(insertPos, insertPos, strings.Repeat(unit, count)),
		)
		unitLen := len([]rune(unit))
		for i := range count {
			targets = append(targets, newlineTarget{
				pos: insertPos,
				off: i*unitLen + firstOff,
			})
		}
	}
	applyNewlines(e, applyNewlinesArgs{
		text: text, sel: sel, changes: changes, targets: targets,
	})
}

type newlineTarget struct {
	pos int
	off int
}

type applyNewlinesArgs struct {
	text    core.Rope
	sel     core.Selection
	changes []core.Change
	targets []newlineTarget
}

func applyNewlines(e *view.Editor, args applyNewlinesArgs) {
	if len(args.changes) == 0 {
		e.SetMode(view.ModeInsert)
		return
	}
	cs, err := core.NewChangeSetFromChanges(args.text, args.changes)
	if err != nil {
		return
	}
	newRanges := make([]core.Range, len(args.targets))
	for i, target := range args.targets {
		pos, err := cs.MapPos(target.pos, core.AssocBefore)
		if err != nil {
			return
		}
		newRanges[i] = core.PointRange(pos + target.off)
	}
	primary := min(args.sel.PrimaryIndex(), len(newRanges)-1)
	newSel, err := core.NewSelection(newRanges, primary)
	if err != nil {
		return
	}
	tx := core.NewTransaction(args.text).WithChanges(cs).WithSelection(newSel)
	_ = e.Apply(tx)
	e.SetMode(view.ModeInsert)
}

func addNewlineImpl(e *view.Editor, above bool) {
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
	count := max(1, e.Count())
	nl := strings.Repeat(string(doc.LineEnding()), count)
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	seen := map[int]bool{}
	changes := make([]core.Change, 0, len(sel.Ranges()))
	for _, r := range sel.Ranges() {
		lr, err := r.LineRange(text)
		if err != nil {
			continue
		}
		var targetLine int
		if above {
			targetLine = lr.From
		} else {
			targetLine = lr.To + 1
		}
		pos, err := text.LineToChar(targetLine)
		if err != nil {
			continue
		}
		if seen[pos] {
			continue
		}
		seen[pos] = true
		changes = append(changes, core.TextChange(pos, pos, nl))
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
