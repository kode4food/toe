package view

import (
	"strings"

	"github.com/pmezard/go-difflib/difflib"

	"github.com/kode4food/toe/internal/core"
)

// DiffSides is the base text and the working text being compared against it
type DiffSides struct {
	Base core.Rope
	Doc  core.Rope
}

// MaxDiffLines and MaxDiffBytes bound the inputs Diff will process, so a
// pathological file cannot stall the diff. Larger documents simply render
// without diff gutter marks
const (
	MaxDiffLines = 65536
	MaxDiffBytes = 16 << 20
)

// Diff returns the line-level hunks that turn Base into Doc, sorted ascending
// and non-overlapping. It returns nil when either side exceeds the size caps
func Diff(sides DiffSides) []DiffHunk {
	base := sides.Base
	doc := sides.Doc
	if base.LenLines() > MaxDiffLines || doc.LenLines() > MaxDiffLines ||
		base.LenChars() > MaxDiffBytes || doc.LenChars() > MaxDiffBytes {
		return nil
	}
	return diffLines(diffLinesArgs{
		base: splitLines(base.String()),
		doc:  splitLines(doc.String()),
	})
}

type diffLinesArgs struct {
	base []string
	doc  []string
}

func diffLines(lines diffLinesArgs) []DiffHunk {
	var hunks []DiffHunk
	matcher := difflib.NewMatcher(lines.base, lines.doc)
	for _, op := range matcher.GetOpCodes() {
		if op.Tag == 'e' {
			continue
		}
		hunks = append(hunks, DiffHunk{
			BaseFrom: op.I1, BaseTo: op.I2, From: op.J1, To: op.J2,
		})
	}
	return hunks
}

func splitLines(s string) []string {
	return strings.SplitAfter(s, "\n")
}
