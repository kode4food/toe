package ui

import (
	"cmp"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
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
	p.compDone = true
	p.compSel = nil
	if p.kind != promptCmd {
		p.comps = nil
		return
	}
	p.comps = completeCommandLine(cx, p.buf)
}

func (p *PromptComponent) changeCompletion(dir int) {
	if len(p.comps) == 0 {
		return
	}
	idx := 0
	if p.compSel != nil {
		idx = *p.compSel + dir
	}
	idx = ((idx % len(p.comps)) + len(p.comps)) % len(p.comps)
	p.compSel = &idx

	c := p.comps[idx]
	start := min(max(c.Start, 0), len(p.buf))
	p.buf = p.buf[:start] + c.Text
	p.caret = len([]rune(p.buf))
}

func (p *PromptComponent) completionMenuHeight(screen geom.Size) int {
	if len(p.comps) == 0 || screen.Width <= 4 || screen.Height <= 3 {
		p.comp = geom.Size{}
		return 0
	}
	innerW := screen.Width - 2 - 2*compPadX
	maxRows := min(compMaxRows, screen.Height-3)
	widths := make([]int, len(p.comps))
	for i, c := range p.comps {
		widths[i] = runewidth.StringWidth(c.completionText())
	}
	slices.Sort(widths)
	colW := max(widths[len(widths)-1], 1)
	fullCols := max(1, (innerW+compGap)/(colW+compGap))
	if len(p.comps) > fullCols*maxRows {
		idx := (len(widths) - 1) * compWidthPct / 100
		colW = max(widths[idx], 1)
	}
	cols := min(len(p.comps), max(1,
		(innerW+compGap)/(colW+compGap),
	))
	rowCount := (len(p.comps) + cols - 1) / cols
	rowCount = min(rowCount, maxRows)
	if rowCount <= 0 {
		p.comp = geom.Size{}
		return 0
	}
	cols = min(cols, (len(p.comps)+rowCount-1)/rowCount)
	p.comp = geom.Size{Width: cols, Height: rowCount}
	return rowCount + 2
}

func (p *PromptComponent) paintCompletions(
	cx *Context, buf *tui.Buffer, bounds geom.Area,
) {
	menuStyle, selected := promptCompletionStyles(cx)
	pop := popup{
		border: lipgloss.RoundedBorder(),
		borderStyle: lipglossToTUIStyle(
			menuStyle.Foreground(pickerFrameStyle(cx).GetForeground()),
		),
		contentStyle: lipglossToTUIStyle(menuStyle),
		padX:         compPadX,
	}
	innerW := bounds.Width - 2 - 2*compPadX
	colW := max(
		(innerW-compGap*(p.comp.Width-1))/p.comp.Width, 1,
	)
	area := pop.drawInto(buf, bounds)
	menuTUI := lipglossToTUIStyle(menuStyle)
	selectedTUI := lipglossToTUIStyle(selected)

	for row := range p.comp.Height {
		for col := range p.comp.Width {
			i := col*p.comp.Height + row
			if i >= len(p.comps) {
				continue
			}
			text := clipPad(p.comps[i].completionText(), colW)
			style := menuTUI
			if p.compSel != nil && *p.compSel == i {
				style = selectedTUI
			}
			buf.SetString(area.Point.Add(geom.Point{
				X: col * (colW + compGap),
				Y: row,
			}), text, style)
		}
	}
}

func promptCompletionStyles(cx *Context) (lipgloss.Style, lipgloss.Style) {
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
