package ui

import (
	"github.com/charmbracelet/x/ansi"

	"github.com/kode4food/toe/internal/tui"
)

type writePickerMatchedArgs struct {
	x, y    int
	maxW    int
	text    string
	indices []int
	base    tui.Style
	match   tui.Style
}

func writePickerMatched(buf *tui.Buffer, args writePickerMatchedArgs) {
	if args.maxW <= 0 {
		return
	}
	runes := []rune(args.text)
	indices := args.indices
	ptr := 0
	col := args.x
	budget := args.maxW
	for i := 0; i < len(runes) && budget > 0; {
		matched := ptr < len(indices) && indices[ptr] == i
		j := i + 1
		if matched {
			ptr++
			for j < len(runes) && ptr < len(indices) && indices[ptr] == j {
				ptr++
				j++
			}
		} else {
			for j < len(runes) &&
				!(ptr < len(indices) && indices[ptr] == j) {
				j++
			}
		}
		run := string(runes[i:j])
		rw := ansi.StringWidth(run)
		if rw > budget {
			run = ansi.Truncate(run, budget, "")
			rw = ansi.StringWidth(run)
		}
		st := args.base
		if matched {
			st = args.match
		}
		buf.SetString(col, args.y, run, st)
		col += rw
		budget -= rw
		i = j
	}
}
