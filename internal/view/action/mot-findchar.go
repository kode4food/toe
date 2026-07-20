package action

import (
	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// FindCharArgs holds the parameters for a FindChar operation
type FindCharArgs struct {
	Editor    *view.Editor
	Ch        rune
	Forward   bool
	Inclusive bool
	Extend    bool
}

// FindChar moves (or extends) each cursor to the nth occurrence of ch in the
// given direction. inclusive=true lands on the char (f/F), false stops before/
// after it (t/T). extend=true keeps the anchor (select mode)
func FindChar(args FindCharArgs) {
	n := countOrOne(args.Editor)
	applyMove(args.Editor, func(doc core.Rope, r core.Range) core.Range {
		cursor := r.Cursor(doc)
		cursorHead := core.NextGraphemeBoundary(doc, cursor)

		// Compute first search start, preserving original skip semantics
		var start int
		search := findCharForward
		if args.Forward {
			start = cursorHead
			if !args.Inclusive {
				start = cursorHead + 1
			}
		} else {
			search = findCharBackward
			switch {
			case args.Inclusive:
				start = cursor - 1
			case cursor > 0:
				start = cursor - 2
			default:
				return r
			}
		}
		found := -1
		for range n {
			found, start = search(doc, start, args.Ch)
			if found == -1 {
				return r
			}
		}

		target := found
		if !args.Inclusive {
			if args.Forward {
				target--
			} else {
				target++
			}
		}

		if args.Extend {
			return r.PutCursor(doc, target, true)
		}
		return core.PointRange(cursor).PutCursor(doc, target, true)
	})
}

func findCharForward(doc core.Rope, start int, ch rune) (int, int) {
	for j := start; j < doc.LenChars(); j++ {
		c, err := doc.CharAt(j)
		if err != nil {
			break
		}
		if c == ch {
			return j, j + 1
		}
	}
	return -1, start
}

func findCharBackward(doc core.Rope, start int, ch rune) (int, int) {
	for j := start; j >= 0; j-- {
		c, err := doc.CharAt(j)
		if err != nil {
			break
		}
		if c == ch {
			return j, j - 1
		}
	}
	return -1, start
}
