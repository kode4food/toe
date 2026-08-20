package view

import (
	"strings"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view/config"
)

type revisedText struct {
	before []rune
	after  []rune
}

func diffChangeSet(oldText core.Rope, newText string) (core.ChangeSet, error) {
	oldRunes := []rune(oldText.String())
	newRunes := []rune(newText)
	pfx := commonPrefix(revisedText{before: oldRunes, after: newRunes})
	sfx := commonSuffix(revisedText{
		before: oldRunes[pfx:],
		after:  newRunes[pfx:],
	})
	from := pfx
	to := len(oldRunes) - sfx
	repl := string(newRunes[pfx : len(newRunes)-sfx])
	return core.NewChangeSetFromChanges(oldText, []core.Change{
		core.TextChange(core.Span{From: from, To: to}, repl),
	})
}

func mapSelections(
	selections map[Id]core.Selection, cs core.ChangeSet, n int,
) map[Id]core.Selection {
	out := make(map[Id]core.Selection, len(selections))
	for vid, sel := range selections {
		out[vid] = mapSelection(sel, cs, n)
	}
	return out
}

func mapSelection(sel core.Selection, cs core.ChangeSet, n int) core.Selection {
	out, err := sel.Map(cs)
	if err == nil {
		return out
	}
	ranges := sel.Ranges()
	for i, r := range ranges {
		ranges[i] = core.Range{
			Anchor: min(max(r.Anchor, 0), n),
			Head:   min(max(r.Head, 0), n),
		}
	}
	out, err = core.NewSelection(ranges, sel.PrimaryIndex())
	if err != nil {
		return core.PointSelection(min(max(sel.Primary().Head, 0), n))
	}
	return out
}

func commonPrefix(text revisedText) int {
	before := text.before
	after := text.after
	n := min(len(before), len(after))
	for i := range n {
		if before[i] != after[i] {
			return i
		}
	}
	return n
}

func commonSuffix(text revisedText) int {
	before := text.before
	after := text.after
	n := min(len(before), len(after))
	for i := range n {
		if before[len(before)-1-i] != after[len(after)-1-i] {
			return i
		}
	}
	return n
}

func defaultLineEnding(le core.LineEnding) core.LineEnding {
	if le == "" {
		return core.NativeLineEnding()
	}
	return le
}

func prepareSaveText(
	s string, le core.LineEnding, opts *Options, ec *config.EditorConfig,
) string {
	trim := opts.TrimTrailingWhitespace
	if ec != nil && ec.TrimTrailingWhitespace != nil {
		trim = *ec.TrimTrailingWhitespace
	}
	insert := opts.InsertFinalNewline
	if ec != nil && ec.InsertFinalNewline != nil {
		insert = *ec.InsertFinalNewline
	}
	if trim {
		s = trimTrailingWhitespace(s)
	}
	if opts.TrimFinalNewlines {
		s = trimFinalNewlines(s)
	}
	if insert && s != "" {
		if _, ok := core.GetLineEndingOfString(s); !ok {
			s += string(le)
		}
	}
	return s
}

func trimTrailingWhitespace(s string) string {
	lines := strings.SplitAfter(s, "\n")
	var b strings.Builder
	for _, line := range lines {
		ending := ""
		body := line
		if strings.HasSuffix(line, "\r\n") {
			ending = "\r\n"
			body = strings.TrimSuffix(line, ending)
		} else if strings.HasSuffix(line, "\n") {
			ending = "\n"
			body = strings.TrimSuffix(line, ending)
		}
		b.WriteString(strings.TrimRight(body, " \t"))
		b.WriteString(ending)
	}
	return b.String()
}

func hasBOMBytes(data []byte) bool {
	return len(data) >= 3 &&
		data[0] == 0xef && data[1] == 0xbb && data[2] == 0xbf
}

func trimFinalNewlines(s string) string {
	total := 0
	final := 0
	for {
		le, ok := core.GetLineEndingOfString(s[:len(s)-total])
		if !ok {
			break
		}
		n := len(le)
		total += n
		final = n
	}
	if total == final {
		return s
	}
	return s[:len(s)-total+final]
}
