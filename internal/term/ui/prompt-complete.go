package ui

import (
	"cmp"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/tui"
)

const (
	compNamePct = 40
	compGap     = 2
)

func (p *PromptComponent) recalculateCompletion(cx *Context) {
	p.completion.done = true
	p.completion.list = listScroll{cursor: -1}
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
	idx := p.completion.list.cursor
	if idx < 0 && dir < 0 {
		idx = 0
	}
	idx += dir
	n := len(p.completion.items)
	idx = ((idx % n) + n) % n
	p.selectCompletion(idx)
}

func (p *PromptComponent) selectCompletion(idx int) {
	if idx < 0 || idx >= len(p.completion.items) {
		return
	}
	p.completion.list.cursor = idx

	c := p.completion.items[idx]
	start := min(max(c.Start, 0), len(p.buf))
	p.buf = p.buf[:start] + c.Text
	p.caret = len([]rune(p.buf))
	p.ensureCompletionVisible()
}

func (p *PromptComponent) syncCompletionRows(frame geom.Size) {
	items := p.completion.items
	innerW := frame.Width - 2 - 2*promptPadX
	fit := frame.Height - promptChrome - promptRule
	rows := min(len(items), max(fit, 0))
	if rows <= 0 || innerW < 1 {
		p.completion.list.resize(len(items), 0)
		return
	}
	nameW := 0
	detailed := false
	for _, c := range items {
		nameW = max(nameW, runewidth.StringWidth(c.completionText()))
		detailed = detailed || c.Detail != ""
	}
	limit := innerW
	if detailed {
		limit = max(innerW*compNamePct/100, 1)
	}
	oldRows := p.completion.list.rows
	p.completion.nameWidth = min(nameW, limit)
	p.completion.list.resize(len(p.completion.items), rows)
	if oldRows != rows {
		p.ensureCompletionVisible()
	}
}

func (p *PromptComponent) paintCompletions(
	cx *Context, buf *tui.Buffer, bounds geom.Area,
) {
	styles := promptCompletionStyles(cx)
	styles.item = p.rowStyle(cx)
	detail := cx.Theme().Get("ui.text.inactive").FgColor()
	for row := range bounds.Height {
		i := p.completion.list.scroll + row
		if i >= len(p.completion.items) {
			return
		}
		item := p.completion.items[i]
		at := bounds.Add(geom.Point{Y: row})
		style := styles.item
		match := pickerMatchStyle(cx).Bg(p.bg)
		if p.completion.list.cursor == i {
			style = styles.selected
			match = pickerSelMatchStyle(cx)
		}
		buf.FillRange(at, bounds.Width, style)
		writeMatchedItem(writeMatchedItemArgs{
			buf:      buf,
			at:       at,
			maxWidth: p.completion.nameWidth,
			text:     item.completionText(),
			indices:  item.Indices,
			base:     style,
			match:    match,
		})
		if item.Detail == "" {
			continue
		}
		x := p.completion.nameWidth + compGap
		buf.SetString(
			at.Add(geom.Point{X: x}),
			runewidth.Truncate(item.Detail, bounds.Width-x, promptEllipsis),
			style.Fg(detail),
		)
	}
}

func (p *PromptComponent) ensureCompletionVisible() {
	if p.completion.list.cursor < 0 {
		return
	}
	p.completion.list.rows = max(p.completion.list.rows, 1)
	p.completion.list.scroll = p.completion.list.ensureCursorVisible()
}

func (p *PromptComponent) handleMouseClick(msg tea.MouseClickMsg) EventResult {
	at := geom.Point{X: msg.X, Y: msg.Y}
	if idx, ok := p.completion.list.indexAt(p.completion.bounds, at); ok {
		p.selectCompletion(idx)
	}
	return consumed()
}

func (p *PromptComponent) handleMouseWheel(
	cx *Context, msg tea.MouseWheelMsg,
) EventResult {
	if !p.completion.bounds.Contains(geom.Point{X: msg.X, Y: msg.Y}) {
		return consumed()
	}
	p.completion.list.wheel(msg.Button, cx.Editor.Options().ScrollLines)
	return consumed()
}

func (p promptCompletion) completionText() string {
	if p.Display != "" {
		return p.Display
	}
	return p.Text
}

type promptCompletionStylesRes struct {
	item     tui.Style
	selected tui.Style
}

func promptCompletionStyles(cx *Context) promptCompletionStylesRes {
	return promptCompletionStylesRes{
		item:     pickerItemStyle(cx),
		selected: pickerSelStyle(cx),
	}
}

func completeCommandLine(cx *Context, input string) []promptCompletion {
	line, complete := command.SplitCommandLine(input)
	if complete {
		return completeCommandNames(cx, line.Name)
	}
	cmd := cx.Keymaps.ResolveCommandIn(cx.Editor.Mode(), line.Name)
	if cmd == nil {
		return nil
	}
	items := cmd.Signature.Completer.Complete(
		cx.Editor, cmd.Signature, line.Rest,
	)
	out := make([]promptCompletion, 0, len(items))
	offset := len(line.Name) + 1
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
	for _, cmd := range cx.Keymaps.CommandsIn(cx.Editor.Mode()) {
		for _, name := range cmd.Aliases {
			if seen[name] {
				continue
			}
			seen[name] = true
			res, ok := fuzzyMatch(fuzzyMatchArgs{pattern: input, text: name})
			if !ok || res.Score < 0 {
				continue
			}
			out = append(out, promptCompletion{
				Completion: command.Completion{
					Text:    name,
					Detail:  cmd.DocString,
					Indices: res.Indices,
				},
				score: res.Score,
			})
		}
	}
	slices.SortStableFunc(out, func(a, b promptCompletion) int {
		return cmp.Compare(b.score, a.score)
	})
	return out
}
