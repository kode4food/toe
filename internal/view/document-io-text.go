package view

import (
	"strings"

	"github.com/alecthomas/chroma/v2/lexers"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view/config"
	"github.com/kode4food/toe/internal/view/language"
)

func diffChangeSet(oldText core.Rope, newText string) (core.ChangeSet, error) {
	oldRunes := []rune(oldText.String())
	newRunes := []rune(newText)
	pfx := commonPrefix(oldRunes, newRunes)
	sfx := commonSuffix(oldRunes[pfx:], newRunes[pfx:])
	from := pfx
	to := len(oldRunes) - sfx
	repl := string(newRunes[pfx : len(newRunes)-sfx])
	return core.NewChangeSetFromChanges(oldText, []core.Change{
		core.TextChange(from, to, repl),
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
		ranges[i] = core.NewRange(clipPos(r.Anchor, n), clipPos(r.Head, n))
	}
	out, err = core.NewSelection(ranges, sel.PrimaryIndex())
	if err != nil {
		return core.PointSelection(clipPos(sel.Primary().Head, n))
	}
	return out
}

func commonPrefix(a, b []rune) int {
	n := min(len(a), len(b))
	for i := range n {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

func commonSuffix(a, b []rune) int {
	n := min(len(a), len(b))
	for i := range n {
		if a[len(a)-1-i] != b[len(b)-1-i] {
			return i
		}
	}
	return n
}

func clipPos(pos, n int) int {
	return min(max(pos, 0), n)
}

// detectLang returns a Chroma-compatible language name for the given file path
// and content. Falls back to "text" if no match is found
func detectLang(path, content string) string {
	if lang, ok := language.DetectLanguage(path, content); ok {
		return lang
	}
	if lex := lexers.Match(path); lex != nil {
		return strings.ToLower(lex.Config().Name)
	}
	if lex := lexers.Analyse(content); lex != nil {
		return strings.ToLower(lex.Config().Name)
	}
	return "text"
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
	trim := opts.TrimTrailingWS
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
