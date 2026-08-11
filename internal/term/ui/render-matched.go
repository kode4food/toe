package ui

import (
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/tui"
)

type writeMatchedItemArgs struct {
	buf           *tui.Buffer
	at            geom.Point
	maxWidth      int
	text          string
	indices       []int
	base          tui.Style
	match         tui.Style
	secondary     tui.Style
	secondaryFrom int
}

func writeMatchedItem(args writeMatchedItemArgs) {
	indices := args.indices
	if args.maxWidth <= 0 {
		return
	}
	runes := []rune(args.text)
	ptr := 0
	col := args.at.X
	for i := 0; i < len(runes) && args.maxWidth > 0; {
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
				!(ptr < len(indices) && indices[ptr] == j) &&
				j != args.secondaryFrom {
				j++
			}
		}
		run := string(runes[i:j])
		rw := runewidth.StringWidth(run)
		if rw > args.maxWidth {
			run = runewidth.Truncate(run, args.maxWidth, "")
			rw = runewidth.StringWidth(run)
		}
		st := args.base
		switch {
		case matched:
			st = args.match
		case args.secondaryFrom > 0 && i >= args.secondaryFrom:
			st = args.secondary
		}
		args.buf.SetString(geom.Point{X: col, Y: args.at.Y}, run, st)
		col += rw
		args.maxWidth -= rw
		i = j
	}
}
