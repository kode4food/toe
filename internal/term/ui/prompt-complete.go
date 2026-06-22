package ui

import (
	"cmp"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/kode4food/toe/internal/term/command"
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
	p.compSel = &idx
}

func (p *PromptComponent) renderCompletions(
	w, h int, cx *Context,
) string {
	if len(p.comps) == 0 || w <= 4 || h <= 3 {
		return ""
	}
	menuStyle, selected := promptCompletionStyles(cx)
	pop := popup{
		border: lipgloss.RoundedBorder(),
		borderStyle: lipglossToTUIStyle(
			menuStyle.Foreground(pickerFrameStyle(cx).GetForeground()),
		),
		contentStyle: lipglossToTUIStyle(menuStyle),
		padX:         1,
	}

	innerW := w - 2 - 2*pop.padX
	maxLen := promptCompletionBaseWidth
	for _, c := range p.comps {
		maxLen = max(maxLen, ansi.StringWidth(c.completionText()))
	}
	cols := max(1, innerW/maxLen)
	colW := max((innerW-cols)/cols, 1)
	rowCount := (len(p.comps) + cols - 1) / cols
	rowCount = min(rowCount, promptCompletionMaxRows)
	rowCount = min(rowCount, h-3)
	if rowCount <= 0 {
		return ""
	}

	buf, area := pop.draw(w, rowCount+2)
	menuTUI := lipglossToTUIStyle(menuStyle)
	selectedTUI := lipglossToTUIStyle(selected)

	for row := range rowCount {
		for col := range cols {
			i := col*rowCount + row
			if i >= len(p.comps) {
				continue
			}
			text := clipPad(p.comps[i].completionText(), colW)
			style := menuTUI
			if p.compSel != nil && *p.compSel == i {
				style = selectedTUI
			}
			buf.SetString(area.x+col*(colW+1), area.y+row, text, style)
		}
	}
	return buf.RenderToANSI()
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
	cmd, ok := cx.Keymaps.ResolveCommand(name)
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
	for _, cmd := range cx.Keymaps.Commands() {
		if len(cmd.Aliases) == 0 {
			continue
		}
		name := cmd.Aliases[0]
		if seen[name] {
			continue
		}
		score, _ := fuzzyMatch(strings.ToLower(input), name)
		if score < 0 {
			continue
		}
		seen[name] = true
		out = append(out, promptCompletion{
			Completion: command.Completion{Text: name},
			score:      score,
		})
	}
	slices.SortStableFunc(out, func(a, b promptCompletion) int {
		return cmp.Compare(b.score, a.score)
	})
	return out
}
