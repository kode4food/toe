package action

import (
	"strings"
	"unicode/utf8"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

type newlineTarget struct {
	pos int
	off int
}

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

// OpenAbove opens a line above each cursor's line and enters insert mode
func OpenAbove(e *view.Editor) {
	openImpl(openArgs{
		editor:          e,
		count:           e.CountOr(1),
		continueComment: true,
	})
}

// OpenBelow opens a line below each cursor's line and enters insert mode
func OpenBelow(e *view.Editor) {
	openImpl(openArgs{
		editor:          e,
		count:           e.CountOr(1),
		below:           true,
		continueComment: true,
	})
}

type openArgs struct {
	editor          *view.Editor
	count           int
	below           bool
	continueComment bool
}

func openImpl(args openArgs) {
	e := args.editor
	v := e.FocusedView()
	if v == nil {
		return
	}
	doc := e.FocusedDocument()
	if doc == nil {
		return
	}
	if doc.ReadOnly() {
		return
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()

	changes := make([]core.Change, 0, len(ranges))
	targets := make([]newlineTarget, 0, len(ranges)*args.count)
	// a cursor maps before every line inserted at its point
	opened := map[int]int{}
	for _, r := range ranges {
		line, ok := openLine(text, r, args.below)
		if !ok {
			continue
		}
		insertPos, ok := openInsertPos(text, line, args.below)
		if !ok {
			continue
		}
		unit, firstOff := openUnit(openUnitArgs{
			indent:      openIndent(e, doc, line, args.continueComment),
			lineEnding:  string(doc.LineEnding()),
			atFileStart: !args.below && line == 0,
		})
		changes = append(changes,
			core.TextChange(core.Span{
				From: insertPos,
				To:   insertPos,
			}, strings.Repeat(unit, args.count)),
		)
		unitLen := utf8.RuneCountInString(unit)
		for i := range args.count {
			targets = append(targets, newlineTarget{
				pos: insertPos,
				off: opened[insertPos] + i*unitLen + firstOff,
			})
		}
		opened[insertPos] += unitLen * args.count
	}
	applyNewlines(e, applyNewlinesArgs{
		text:    text,
		sel:     sel,
		changes: changes,
		targets: targets,
	})
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
		if pos, err := cs.MapPos(target.pos, core.AssocBefore); err == nil {
			newRanges[i] = core.PointRange(pos + target.off)
			continue
		}
		return
	}
	primary := min(args.sel.PrimaryIndex(), len(newRanges)-1)
	if newSel, err := core.NewSelection(newRanges, primary); err == nil {
		tx := core.NewTransaction(args.text).WithChanges(cs).
			WithSelection(newSel)
		_ = e.Apply(tx)
		e.SetMode(view.ModeInsert)
	}
}

func addNewlineImpl(e *view.Editor, above bool) {
	v := e.FocusedView()
	if v == nil {
		return
	}
	doc := e.FocusedDocument()
	if doc == nil {
		return
	}
	if doc.ReadOnly() {
		return
	}
	count := e.CountOr(1)
	nl := strings.Repeat(string(doc.LineEnding()), count)
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	seen := map[int]bool{}
	changes := make([]core.Change, 0, len(sel.Ranges()))
	for _, r := range sel.Ranges() {
		lr, err := r.LineSpan(text)
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
		changes = append(changes, core.TextChange(core.Span{
			From: pos,
			To:   pos,
		}, nl))
	}
	if len(changes) == 0 {
		return
	}
	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return
	}
	if newSel, err := sel.Map(cs); err == nil {
		_ = e.Apply(
			core.NewTransaction(text).WithChanges(cs).WithSelection(newSel),
		)
	}
}

func openLine(text core.Rope, r core.Range, below bool) (int, bool) {
	span, err := r.LineSpan(text)
	if err != nil {
		return 0, false
	}
	if below {
		return span.To, true
	}
	return span.From, true
}

func openInsertPos(text core.Rope, line int, below bool) (int, bool) {
	if !below {
		if line == 0 {
			return 0, true
		}
		line--
	}
	pos, err := text.LineEndCharIndex(line)
	if err != nil {
		return 0, false
	}
	return pos, true
}

type openUnitArgs struct {
	indent      string
	lineEnding  string
	atFileStart bool
}

func openUnit(args openUnitArgs) (string, int) {
	if args.atFileStart {
		return args.indent + args.lineEnding,
			utf8.RuneCountInString(args.indent)
	}
	unit := args.lineEnding + args.indent
	return unit, utf8.RuneCountInString(unit)
}

func openIndent(
	e *view.Editor, doc *view.Document, line int, continueComment bool,
) string {
	text := doc.Text()
	pos, err := text.LineEndCharIndex(line)
	if err != nil {
		return ""
	}
	at := core.LinePos{Line: line, Pos: pos}
	if continueComment {
		indent, _ := continuedIndent(e, doc, at)
		return indent
	}
	return structuralIndent(structuralIndentArgs{
		editor: e,
		text:   text,
		line:   line,
		pos:    pos,
		indent: leadingWhitespace(text, pos),
		doc:    doc,
	})
}
