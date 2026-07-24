package ui

import (
	"cmp"
	"slices"
	"strings"

	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/tui"
)

const (
	compWidthPct = 90
	compMaxRows  = 10
	compPadX     = 1
	compGap      = 2
)

func (p *PromptComponent) recalculateCompletion(cx *Context) {
	p.completion.done = true
	p.completion.selected = nil
	if p.kind != promptCmd {
		p.completion.items = nil
		return
	}
	p.completion.items = completeCommandLine(cx, p.buf)
}

func (p *PromptComponent) changeCompletion(dir int) {
	if len(p.completion.items) == 0 {
		return
	}
	idx := 0
	if p.completion.selected != nil {
		idx = *p.completion.selected + dir
	}
	n := len(p.completion.items)
	idx = ((idx % n) + n) % n
	p.completion.selected = &idx

	c := p.completion.items[idx]
	start := min(max(c.Start, 0), len(p.buf))
	p.buf = p.buf[:start] + c.Text
	p.caret = len([]rune(p.buf))
}

func (p *PromptComponent) completionMenuHeight(screen geom.Size) int {
	items := p.completion.items
	if len(items) == 0 || screen.Width <= 4 || screen.Height <= 3 {
		p.completion.size = geom.Size{}
		return 0
	}
	innerW := screen.Width - 2 - 2*compPadX
	maxRows := min(compMaxRows, screen.Height-3)
	widths := make([]int, len(items))
	for i, c := range items {
		widths[i] = runewidth.StringWidth(c.completionText())
	}
	slices.Sort(widths)
	colW := max(widths[len(widths)-1], 1)
	fullCols := max(1, (innerW+compGap)/(colW+compGap))
	if len(items) > fullCols*maxRows {
		idx := (len(widths) - 1) * compWidthPct / 100
		colW = max(widths[idx], 1)
	}
	cols := min(len(items), max(1,
		(innerW+compGap)/(colW+compGap),
	))
	rowCount := (len(items) + cols - 1) / cols
	rowCount = min(rowCount, maxRows)
	if rowCount <= 0 {
		p.completion.size = geom.Size{}
		return 0
	}
	cols = min(cols, (len(items)+rowCount-1)/rowCount)
	p.completion.size = geom.Size{Width: cols, Height: rowCount}
	return rowCount + 2
}

func (p *PromptComponent) paintCompletions(
	cx *Context, buf *tui.Buffer, bounds geom.Area,
) {
	menuStyle, selected := promptCompletionStyles(cx)
	pop := popup{
		borderStyle:  menuStyle.Fg(pickerFrameStyle(cx).FgColor()),
		contentStyle: menuStyle,
		padX:         compPadX,
	}
	innerW := bounds.Width - 2 - 2*compPadX
	size := p.completion.size
	colW := max(
		(innerW-compGap*(size.Width-1))/size.Width, 1,
	)
	area := pop.drawInto(buf, bounds)
	for row := range size.Height {
		for col := range size.Width {
			i := col*size.Height + row
			if i >= len(p.completion.items) {
				continue
			}
			text := clipPad(p.completion.items[i].completionText(), colW)
			style := menuStyle
			if p.completion.selected != nil &&
				*p.completion.selected == i {
				style = selected
			}
			buf.SetString(area.Point.Add(geom.Point{
				X: col * (colW + compGap),
				Y: row,
			}), text, style)
		}
	}
}

func promptCompletionStyles(cx *Context) (tui.Style, tui.Style) {
	return pickerItemStyle(cx), pickerSelStyle(cx)
}

func (p promptCompletion) completionText() string {
	if p.Display != "" {
		return p.Display
	}
	return p.Text
}

func completeCommandLine(cx *Context, input string) []promptCompletion {
	name, rest, complete := command.SplitCommandLine(input)
	if complete {
		return completeCommandNames(cx, name)
	}
	mode := cx.Editor.Mode().String()
	cmd, ok := cx.Keymaps.ResolveCommandIn(mode, name)
	if !ok {
		return nil
	}
	items := cmd.Signature.Completer.Complete(cx.Editor, cmd.Signature, rest)
	out := make([]promptCompletion, 0, len(items))
	offset := len(name) + 1
	for _, item := range items {
		item.Start += offset
		out = append(out, promptCompletion{Completion: item})
	}
	return out
}

func completeCommandNames(cx *Context, input string) []promptCompletion {
	out := make([]promptCompletion, 0)
	seen := map[string]bool{}
	input = strings.ToLower(input)
	for _, cmd := range cx.Keymaps.CommandsIn(cx.Editor.Mode().String()) {
		for _, name := range cmd.Aliases {
			if seen[name] {
				continue
			}
			seen[name] = true
			score, _ := fuzzyMatch(input, name)
			if score < 0 {
				continue
			}
			out = append(out, promptCompletion{
				Completion: command.Completion{Text: name},
				score:      score,
			})
		}
	}
	slices.SortStableFunc(out, func(a, b promptCompletion) int {
		return cmp.Compare(b.score, a.score)
	})
	return out
}
