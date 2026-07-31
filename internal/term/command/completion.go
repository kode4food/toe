package command

import (
	"cmp"
	"slices"
	"strings"

	"github.com/kode4food/toe/internal/fuzzy"
	"github.com/kode4food/toe/internal/view"
)

type (
	// Completion replaces prompt input from Start to the end of the line
	Completion struct {
		Start   int
		Text    string
		Display string
		Indices []int
	}

	// Completer describes positional and raw argument completion
	Completer struct {
		Positionals []CompletionFunc
		Raw         CompletionFunc
	}

	// CompletionFunc returns completions for command-line input, given the
	// arguments already parsed for the current command
	CompletionFunc func(*view.Editor, *Args, string) []Completion
)

// PositionalCompleter completes positionals by argument index
func PositionalCompleter(c ...CompletionFunc) Completer {
	return Completer{Positionals: c}
}

// StaticCompleter completes from a fixed string set
func StaticCompleter[T ~string](items ...T) CompletionFunc {
	return func(_ *view.Editor, _ *Args, input string) []Completion {
		return matchFuzzy(items, input)
	}
}

// Complete returns argument completions for a command signature
func (c Completer) Complete(
	e *view.Editor, sig Signature, input string,
) []Completion {
	start, token := completionToken(input)
	args, err := ParseArgs(input, sig, false, nil)
	if err != nil {
		return nil
	}
	state := args.CompletionState()
	if state.Kind == CompletionStateFlagArgument && state.Flag != nil {
		return completeStaticAt(start, token, state.Flag.Completions)
	}
	if state.Kind == CompletionStateFlag {
		return completeFlagsAt(start, token, sig.Flags)
	}
	idx := args.Len()
	if token != "" && !strings.HasSuffix(input, " ") &&
		!strings.HasSuffix(input, "\t") {
		idx--
	}
	if idx < 0 {
		return nil
	}
	if sig.RawAfter > 0 && idx >= sig.RawAfter && c.Raw != nil {
		return offsetCompletions(c.Raw(e, args, input[start:]), start)
	}
	if idx >= len(c.Positionals) || c.Positionals[idx] == nil {
		return nil
	}
	return offsetCompletions(c.Positionals[idx](e, args, token), start)
}

func completionToken(input string) (int, string) {
	if strings.HasSuffix(input, " ") || strings.HasSuffix(input, "\t") {
		return len(input), ""
	}
	t := NewTokenizer(input, false)
	var last Token
	for {
		tok, ok, err := t.Next()
		if err != nil || !ok {
			break
		}
		last = tok
	}
	return last.ContentStart, last.Content
}

func completeFlagsAt(start int, token string, flags []Flag) []Completion {
	names := make([]string, 0, len(flags)*2)
	for _, f := range flags {
		names = append(names, "--"+f.Name)
		if f.Alias != 0 {
			names = append(names, "-"+string(f.Alias))
		}
	}
	out := matchFuzzy(names, token)
	for i := range out {
		out[i].Start = start
	}
	return out
}

func completeStaticAt(start int, token string, items []string) []Completion {
	out := matchFuzzy(items, token)
	for i := range out {
		out[i].Start = start
	}
	return out
}

func offsetCompletions(items []Completion, offset int) []Completion {
	out := make([]Completion, 0, len(items))
	for _, item := range items {
		item.Start += offset
		out = append(out, item)
	}
	return out
}

func matchFuzzy[T ~string](items []T, input string) []Completion {
	type scored struct {
		Completion
		score int
		order int
	}
	m := fuzzy.NewMatcher(input)
	matches := make([]scored, 0, len(items))
	for i, item := range items {
		text := string(item)
		res, ok := m.Match(text)
		if !ok {
			continue
		}
		matches = append(matches, scored{
			Completion: Completion{Text: text, Indices: res.Indices},
			score:      res.Score,
			order:      i,
		})
	}
	slices.SortStableFunc(matches, func(a, b scored) int {
		return cmp.Or(
			cmp.Compare(b.score, a.score), cmp.Compare(a.order, b.order),
		)
	})
	out := make([]Completion, len(matches))
	for i, m := range matches {
		out[i] = m.Completion
	}
	return out
}
