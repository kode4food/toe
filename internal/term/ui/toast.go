package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/highlight"
	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/tui"
)

type (
	toastState struct {
		items  []toast
		log    []string
		bounds geom.Area
		gen    int
		rev    int
	}

	toast struct {
		text    string
		expires time.Time
		level   toastLevel
	}

	toastLevel uint8
)

const (
	toastInfo toastLevel = iota
	toastWarning
	toastError
)

const (
	toastTickEvery = 250 * time.Millisecond
	toastMaxRows   = 5
	toastPadX      = 1
	toastGapX      = 2 // columns kept clear to the right of the popup
	toastGapY      = 2 // rows kept clear below it, statusline included
	toastChrome    = 2 // top and bottom border
)

// an error is worth reading, so it outlasts the messages that merely report
const (
	toastInfoTimeout    = 4 * time.Second
	toastWarningTimeout = 8 * time.Second
	toastErrorTimeout   = 12 * time.Second
)

func (t *toastState) push(text string, level toastLevel) {
	t.items = append(t.items, toast{
		text:    text,
		level:   level,
		expires: time.Now().Add(toastTimeout(level)),
	})
	if over := len(t.items) - toastMaxRows; over > 0 {
		t.items = t.items[over:]
	}
	t.log = append(t.log, logTag(level)+highlight.LogSeparator+text)
	t.rev++
}

// takeLog returns the messages not yet written to the log document
func (t *toastState) takeLog() []string {
	out := t.log
	t.log = nil
	return out
}

// expire drops the timed-out entries, reporting whether any were dropped
func (t *toastState) expire(now time.Time) bool {
	kept := t.items[:0]
	for _, item := range t.items {
		if item.expires.After(now) {
			kept = append(kept, item)
		}
	}
	dropped := len(kept) != len(t.items)
	t.items = kept
	if dropped {
		t.rev++
	}
	return dropped
}

func (t *toastState) pending() bool {
	return len(t.items) > 0
}

func (t *toastState) dismiss() bool {
	if len(t.items) == 0 {
		return false
	}
	t.items = nil
	t.bounds = geom.Area{}
	t.rev++
	return true
}

// dismissAt drops the entry drawn at a point inside the popup
func (t *toastState) dismissAt(at geom.Point) bool {
	if !t.bounds.Contains(at) {
		return false
	}
	idx := at.Y - t.bounds.Y - 1
	if idx < 0 || idx >= len(t.items) {
		return true
	}
	t.items = append(t.items[:idx], t.items[idx+1:]...)
	t.rev++
	return true
}

func (e *EditorComponent) pushToast(text string, level toastLevel) {
	e.toasts.push(text, level)
	e.requestRedraw()
}

// toastTickCmd is started when a toast is queued, so a tick is only pending
// while something is waiting to expire
func (e *EditorComponent) toastTickCmd() tea.Cmd {
	if !e.toasts.pending() {
		return nil
	}
	e.toasts.gen++
	return toastTickCmd(e.toasts.gen)
}

func (e *EditorComponent) handleToastTick(
	msg toastTickMsg,
) (EventResult, tea.Cmd) {
	if msg.gen != e.toasts.gen {
		return consumed(), nil
	}
	if e.toasts.expire(time.Now()) {
		e.requestRedraw()
	}
	if !e.toasts.pending() {
		return consumed(), nil
	}
	return consumed(), toastTickCmd(msg.gen)
}

// renderToasts draws the queued messages as one popup in the bottom-right
// corner, its last row above bottom so the statusline stays clear
func (r *renderPass) renderToasts(buf *tui.Buffer, bottom int) {
	items := r.editor.toasts.items
	if len(items) == 0 {
		r.editor.toasts.bounds = geom.Area{}
		return
	}
	th := r.context.Theme()
	pop := popup{
		borderStyle:  th.Get("ui.popup"),
		contentStyle: th.Get("ui.popup"),
		padX:         toastPadX,
	}
	title := i18n.Text(i18n.ToastTitle)
	textW := runewidth.StringWidth(title) + 2
	for _, item := range items {
		textW = max(textW, runewidth.StringWidth(item.text))
	}
	boxW := min(textW+2*toastPadX+2, max(r.size.Width-toastGapX, 1))
	boxH := len(items) + toastChrome
	box := geom.Area{
		Point: geom.Point{
			X: max(r.size.Width-boxW-toastGapX, 0),
			Y: max(bottom-toastGapY-boxH+1, 0),
		},
		Size: geom.Size{Width: boxW, Height: boxH},
	}
	r.editor.toasts.bounds = box
	area := pop.drawInto(buf, box)
	drawPopupTitle(buf, box, title, pop.borderStyle)
	for i, item := range items {
		st := pop.contentStyle.Fg(toastColor(th, item.level))
		buf.SetString(
			area.Add(geom.Point{Y: i}),
			runewidth.Truncate(item.text, area.Width, promptEllipsis),
			st,
		)
	}
}

func toastTickCmd(gen int) tea.Cmd {
	return tea.Tick(toastTickEvery, func(time.Time) tea.Msg {
		return toastTickMsg{gen: gen}
	})
}

func toastTimeout(level toastLevel) time.Duration {
	switch level {
	case toastError:
		return toastErrorTimeout
	case toastWarning:
		return toastWarningTimeout
	default:
		return toastInfoTimeout
	}
}

func logTag(level toastLevel) string {
	switch level {
	case toastError:
		return highlight.LogError
	case toastWarning:
		return highlight.LogWarning
	default:
		return highlight.LogInfo
	}
}

func toastColor(th *theme.Theme, level toastLevel) tui.Color {
	switch level {
	case toastError:
		return th.Get("error").FgColor()
	case toastWarning:
		return th.Get("warning").FgColor()
	default:
		return th.Get("info").FgColor()
	}
}
