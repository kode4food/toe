package action

import (
	"strings"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/language"
)

// InsertNewline inserts a newline at every cursor in insert mode, then
// replicates the previous line's leading whitespace (auto-indent)
func InsertNewline(e *view.Editor) {
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
	targets := make([]int, len(ranges))
	targetOffs := make([]int, len(ranges))
	seen := map[int]bool{}
	for i, r := range ranges {
		pos := r.Cursor(text)
		if seen[pos] {
			targets[i] = pos
			continue
		}
		seen[pos] = true
		line, err := text.CharToLine(pos)
		if err != nil {
			targets[i] = pos
			continue
		}
		lineStart, err := text.LineToChar(line)
		if err != nil {
			targets[i] = pos
			continue
		}
		// Find the last non-whitespace char on the current line up to pos
		firstTrailingWS := -1
		for i := pos - 1; i >= lineStart; i-- {
			ch, err := text.CharAt(i)
			if err != nil {
				break
			}
			if ch != ' ' && ch != '\t' {
				firstTrailingWS = i + 1
				break
			}
		}
		if firstTrailingWS < 0 {
			// Entire line up to pos is whitespace: insert bare newline at
			// line start, leaving old whitespace on the new line
			changes = append(changes,
				core.TextChange(lineStart, lineStart, "\n"),
			)
			targets[i] = lineStart
			targetOffs[i] = 1
		} else if firstTrailingWS < pos {
			// Trim trailing whitespace then insert newline with indent
			indent, continued := continuedIndent(e, doc, line, pos)
			insert, off := newlineInsertForCursor(newlineInsertArgs{
				editor: e, doc: doc, r: r,
				indent: indent, continued: continued,
			})
			changes = append(changes,
				core.TextChange(firstTrailingWS, pos, insert))
			targets[i] = firstTrailingWS
			targetOffs[i] = off
		} else {
			// No trailing whitespace: plain newline with indent
			indent, continued := continuedIndent(e, doc, line, pos)
			insert, off := newlineInsertForCursor(newlineInsertArgs{
				editor: e, doc: doc, r: r,
				indent: indent, continued: continued,
			})
			changes = append(changes, core.TextChange(pos, pos, insert))
			targets[i] = pos
			targetOffs[i] = off
		}
	}
	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return
	}
	newRanges := make([]core.Range, len(ranges))
	for i := range ranges {
		pos, err := cs.MapPos(targets[i], core.AssocBefore)
		if err != nil {
			return
		}
		newRanges[i] = core.PointRange(pos + targetOffs[i])
	}
	newSel, err := core.NewSelection(newRanges, sel.PrimaryIndex())
	if err != nil {
		return
	}
	_ = e.Apply(core.NewTransaction(text).WithChanges(cs).WithSelection(newSel))
}

func leadingWhitespace(text core.Rope, pos int) string {
	line, err := text.CharToLine(pos)
	if err != nil {
		return ""
	}
	lineStart, err := text.LineToChar(line)
	if err != nil {
		return ""
	}
	lineEnd, err := text.LineEndCharIndex(line)
	if err != nil {
		return ""
	}
	var b strings.Builder
	for i := lineStart; i < lineEnd; i++ {
		ch, err := text.CharAt(i)
		if err != nil {
			break
		}
		if ch != ' ' && ch != '\t' {
			break
		}
		b.WriteRune(ch)
	}
	return b.String()
}

func continuedIndent(
	e *view.Editor, doc *view.Document, line, pos int,
) (string, bool) {
	text := doc.Text()
	indent := leadingWhitespace(text, pos)
	if !e.Options().ContinueComments {
		return indent, false
	}
	lang := language.LoadLanguage(doc.Lang())
	token, ok := core.GetCommentToken(text, lang.CommentTokens, line)
	if !ok {
		return indent, false
	}
	return indent + token + " ", true
}

type newlineInsertArgs struct {
	editor    *view.Editor
	doc       *view.Document
	r         core.Range
	indent    string
	continued bool
}

func newlineInsertForCursor(args newlineInsertArgs) (string, int) {
	text := args.doc.Text()
	pairs, ok := autoPairsForDocument(args.editor, args.doc)
	if args.continued || !ok || !betweenAutoPair(text, args.r, pairs) {
		insert := "\n" + args.indent
		return insert, len([]rune(insert))
	}
	inner := args.indent + args.doc.IndentStyle().AsStr()
	insert := "\n" + inner + "\n" + args.indent
	return insert, 1 + len([]rune(inner))
}

func betweenAutoPair(text core.Rope, r core.Range, pairs core.AutoPairs) bool {
	pos := r.Cursor(text)
	if pos == 0 {
		return false
	}
	prev, err := text.CharAt(pos - 1)
	if err != nil {
		return false
	}
	curr, err := text.CharAt(pos)
	if err != nil {
		return false
	}
	pair, ok := pairs.Get(prev)
	return ok && pair.Open == prev && pair.Close == curr
}
