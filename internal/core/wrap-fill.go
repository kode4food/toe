package core

import "strings"

func fillHardWrap(uw *hardWrapUnfilled, width int) string {
	body := strings.TrimSuffix(uw.text, uw.ending)
	words := hardWrapWordItems(
		hardWrapWords(body), hardWrapCapacity(uw, width),
	)
	if len(words) == 0 {
		return ""
	}

	breaks := hardWrapBreaks(words, uw, width)
	var b strings.Builder
	start := 0
	for i, end := range breaks {
		if i > 0 {
			b.WriteString(uw.ending)
		}
		if i == 0 {
			b.WriteString(uw.initial)
		} else {
			b.WriteString(uw.subsequent)
		}
		writeHardWrapWords(&b, words[start:end])
		start = end
	}
	return b.String()
}

func hardWrapCapacity(uw *hardWrapUnfilled, width int) int {
	capacity := width - textWidth(uw.initial)
	if uw.subsequent != "" {
		capacity = min(capacity, width-textWidth(uw.subsequent))
	}
	return max(capacity, 1)
}

func hardWrapWordItems(words []string, capacity int) []hardWrapWord {
	res := make([]hardWrapWord, 0, len(words))
	for _, word := range words {
		for word != "" {
			split := hardWrapTake(word, capacity)
			if split.before == "" {
				split = hardWrapTake(word, 1)
			}
			res = append(res, hardWrapWord{
				text:  split.before,
				width: textWidth(split.before),
			})
			word = split.after
		}
	}
	return res
}

func hardWrapBreaks(
	words []hardWrapWord, uw *hardWrapUnfilled, width int,
) []int {
	n := len(words)
	cost := make([]int, n+1)
	next := make([]int, n)
	for i := n - 1; i >= 0; i-- {
		bestCost := int(^uint(0) >> 1)
		bestNext := i + 1
		limit := width - textWidth(uw.subsequent)
		if i == 0 {
			limit = width - textWidth(uw.initial)
		}
		limit = max(limit, 1)
		lineW := 0
		for j := i; j < n; j++ {
			if j > i {
				lineW++
			}
			lineW += words[j].width
			if lineW > limit {
				break
			}
			c := hardWrapLineCost(hardWrapLineCostArgs{
				words:     words,
				start:     i,
				end:       j + 1,
				lineWidth: lineW,
				limit:     limit,
				count:     n,
			})
			if j+1 < n {
				c += wrapLinePenalty + cost[j+1]
			}
			if c < bestCost {
				bestCost = c
				bestNext = j + 1
			}
		}
		cost[i] = bestCost
		next[i] = bestNext
	}

	var breaks []int
	for i := 0; i < n; i = next[i] {
		breaks = append(breaks, next[i])
	}
	return breaks
}

type hardWrapLineCostArgs struct {
	words     []hardWrapWord
	start     int
	end       int
	lineWidth int
	limit     int
	count     int
}

func hardWrapLineCost(args hardWrapLineCostArgs) int {
	if args.end == args.count {
		last := args.words[args.start]
		if args.end-args.start == 1 &&
			last.width*wrapShortTailDiv < args.limit {
			return wrapShortTailCost
		}
		return 0
	}
	gap := args.limit - args.lineWidth
	return gap * gap
}

func writeHardWrapWords(b *strings.Builder, words []hardWrapWord) {
	for i, word := range words {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(word.text)
	}
}

func hardWrapWords(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		return r == ' '
	})
}

func hardWrapTake(s string, width int) stringSplit {
	w := 0
	for i, ch := range s {
		next := w + graphemeWidth(string(ch))
		if next > width {
			return stringSplit{before: s[:i], after: s[i:]}
		}
		w = next
	}
	return stringSplit{before: s}
}

func textWidth(s string) int {
	w := 0
	for _, ch := range s {
		w += graphemeWidth(string(ch))
	}
	return w
}
