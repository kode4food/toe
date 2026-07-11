package ui

import (
	"slices"
	"strings"
	"unicode"

	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/view"
)

type (
	completionMatch struct {
		item  view.CompletionItem
		score int
		order int
	}

	// completionItemKey identifies a completion item for restoring the cursor
	// across a refresh, without allocating a concatenated string to compare
	completionItemKey struct {
		id                  string
		label, insert, kind string
	}
)

func filterCompletionItems(
	items []view.CompletionItem, query string,
) []view.CompletionItem {
	if query == "" {
		return append([]view.CompletionItem(nil), items...)
	}
	matches := make([]completionMatch, 0, len(items))
	for i, item := range items {
		if score, ok := completionMatchScore(item, query); ok {
			matches = append(matches, completionMatch{
				item:  item,
				score: score,
				order: i,
			})
		}
	}
	slices.SortStableFunc(matches, func(a, b completionMatch) int {
		if a.score > b.score {
			return -1
		}
		if a.score < b.score {
			return 1
		}
		return a.order - b.order
	})
	out := make([]view.CompletionItem, 0, len(matches))
	for _, m := range matches {
		out = append(out, m.item)
	}
	return out
}

func completionMatchScore(item view.CompletionItem, query string) (int, bool) {
	text := item.Filter
	if text == "" {
		text = item.Label
	}
	return fuzzyCompletionScore(text, query)
}

func keyOfCompletionItem(item view.CompletionItem) completionItemKey {
	if item.ID != "" {
		return completionItemKey{id: item.ID}
	}
	return completionItemKey{
		label: item.Label, insert: item.Insert, kind: item.Kind,
	}
}

func fuzzyCompletionScore(text, query string) (int, bool) {
	text = strings.TrimLeftFunc(text, unicode.IsSpace)
	if query == "" {
		return 0, true
	}
	if strings.HasPrefix(strings.ToLower(text), strings.ToLower(query)) {
		return 100000 - runewidth.StringWidth(text), true
	}
	rs := []rune(text)
	score := 0
	gaps := 0
	last := -1
	from := 0
	for _, q := range query {
		found := -1
		q = unicode.ToLower(q)
		for i := from; i < len(rs); i++ {
			if unicode.ToLower(rs[i]) == q {
				found = i
				break
			}
		}
		if found < 0 {
			return 0, false
		}
		if last >= 0 {
			gaps += found - last - 1
			if found == last+1 {
				score += 3
			}
		}
		if completionBoundary(rs, found) {
			score += 5
		}
		score += 10
		last = found
		from = found + 1
	}
	score -= gaps * 2
	score -= runewidth.StringWidth(text)
	return score, true
}

func completionBoundary(rs []rune, idx int) bool {
	if idx == 0 {
		return true
	}
	prev := rs[idx-1]
	return !unicode.IsLetter(prev) && !unicode.IsNumber(prev)
}
