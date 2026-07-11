package vcs

import (
	"strings"

	"github.com/pmezard/go-difflib/difflib"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// MaxDiffLines and MaxDiffBytes bound the inputs Diff will process, so a
// pathological file cannot stall the diff worker. Larger documents simply
// render without diff gutter marks
const (
	MaxDiffLines = 65536
	MaxDiffBytes = 16 << 20
)

// Diff returns the line-level hunks that turn base into doc, sorted ascending
// and non-overlapping. It returns nil when either side exceeds the size caps
func Diff(base, doc core.Rope) []view.DiffHunk {
	if base.LenLines() > MaxDiffLines || doc.LenLines() > MaxDiffLines ||
		base.LenChars() > MaxDiffBytes || doc.LenChars() > MaxDiffBytes {
		return nil
	}
	return diffLines(splitLines(base.String()), splitLines(doc.String()))
}

func diffLines(a, b []string) []view.DiffHunk {
	var hunks []view.DiffHunk
	for _, op := range difflib.NewMatcher(a, b).GetOpCodes() {
		if op.Tag == 'e' {
			continue
		}
		hunks = append(hunks, view.DiffHunk{
			BaseFrom: op.I1, BaseTo: op.I2, From: op.J1, To: op.J2,
		})
	}
	return hunks
}

// splitLines splits text into lines that keep their terminators, so a missing
// final newline is a real difference
func splitLines(s string) []string {
	return strings.SplitAfter(s, "\n")
}
