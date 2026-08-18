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
		items   []toast
		log     []string
		bounds  geom.Area
		shown   time.Time
		leaving time.Time
		slide   int
		gen     int
		rev     int
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
	toastCommand
	toastTerminal
)

const (
	toastTickEvery  = 250 * time.Millisecond
	toastFrameEvery = 40 * time.Millisecond
	toastRise       = 200 * time.Millisecond
	toastFade       = 500 * time.Millisecond
	toastMaxRows    = 5
	// width tracks the screen rather than the message, so a new entry never
	// resizes the popup
	toastWidthPct = 40
	toastMinWidth = 40
	toastMaxWidth = 80
	toastGapX     = 2 // columns kept clear to the right of the popup
	toastGapY     = 2 // rows kept clear below it, statusline included
	toastChrome   = 2 // top and bottom border
)

// an error is worth reading, so it outlasts the messages that merely report
const (
	toastInfoTimeout    = 4 * time.Second
	toastWarningTimeout = 8 * time.Second
	toastErrorTimeout   = 12 * time.Second
)

func (t *toastState) push(text string, level toastLevel) {
	now := time.Now()
	// a new message cancels an exit, the spent entries going with it. The
	// rise resumes from where the exit got to, so the popup never jerks
	if !t.leaving.IsZero() {
		t.items = nil
		t.leaving = time.Time{}
		t.shown = now.Add(-toastRise * time.Duration(100-t.slide) / 100)
	} else if len(t.items) == 0 {
		t.shown = now
	}
	t.items = append(t.items, toast{
		text:    text,
		level:   level,
		expires: now.Add(toastTimeout(level)),
	})
	if over := len(t.items) - toastMaxRows; over > 0 {
		t.items = t.items[over:]
	}
	t.log = append(t.log, logTag(level)+highlight.LogSeparator+text)
	t.rev++
}

// the messages not yet written to the log document
func (t *toastState) takeLog() []string {
	out := t.log
	t.log = nil
	return out
}

// drops the timed-out entries, reporting whether any were dropped
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

// advances expiry, the slide, and any fade, reporting whether the popup changed
func (t *toastState) step(now time.Time, animate bool) bool {
	if !t.leaving.IsZero() && now.Sub(t.leaving) >= toastRise {
		return t.dismiss()
	}
	changed := t.slideTo(t.slidePct(now, animate))
	if !t.leaving.IsZero() {
		return changed
	}
	if animate && t.spent(now) {
		return t.leave(now) || changed
	}
	if animate && t.fading(now) {
		t.rev++ // the colors move on even when nothing else does
		changed = true
	}
	return t.expire(now) || changed
}

// distance from the resting place, 100 being fully off screen. Frames read
// this rather than the clock, so a repaint cannot move the popup uncached
func (t *toastState) slidePct(now time.Time, animate bool) int {
	if !animate {
		return 0
	}
	if !t.leaving.IsZero() {
		return min(int(now.Sub(t.leaving)*100/toastRise), 100)
	}
	// a clock that has not reached the push yet means the rise has not
	// started, never that it has finished
	elapsed := max(now.Sub(t.shown), 0)
	if elapsed >= toastRise {
		return 0
	}
	return int((toastRise - elapsed) * 100 / toastRise)
}

// ends any slide in progress, so turning animation off never strands the
// popup part way off screen
func (t *toastState) snap() {
	if !t.leaving.IsZero() {
		t.dismiss()
		return
	}
	t.slideTo(0)
}

func (t *toastState) slideTo(pct int) bool {
	if pct == t.slide {
		return false
	}
	t.slide = pct
	t.rev++
	return true
}

// closes the popup, sinking it off screen first when animating
func (t *toastState) close(now time.Time, animate bool) bool {
	if animate {
		return t.leave(now)
	}
	return t.dismiss()
}

// starts the slide back off screen, the entries staying drawn until it lands
func (t *toastState) leave(now time.Time) bool {
	if len(t.items) == 0 || !t.leaving.IsZero() {
		return false
	}
	t.leaving = now
	t.rev++
	return true
}

// true once nothing is left worth reading, so the popup can slide away whole
func (t *toastState) spent(now time.Time) bool {
	for _, item := range t.items {
		if item.expires.After(now) {
			return false
		}
	}
	return len(t.items) > 0
}

// true while the popup is sliding either way or an entry is fading out, so
// frames are due at frame rate rather than at the idle tick
func (t *toastState) animating(now time.Time) bool {
	if len(t.items) == 0 {
		return false
	}
	return t.slide != 0 || !t.leaving.IsZero() || t.fading(now)
}

// true while an entry other than the last is inside its fade-out window
func (t *toastState) fading(now time.Time) bool {
	for _, item := range t.items[:max(len(t.items)-1, 0)] {
		// a frame early, so the slow tick can't sleep through the fade-out
		if item.expires.Sub(now) <= toastFade+toastTickEvery {
			return true
		}
	}
	return false
}

func (t *toastState) dismiss() bool {
	if len(t.items) == 0 {
		return false
	}
	t.items = nil
	t.bounds = geom.Area{}
	t.leaving = time.Time{}
	t.slide = 0
	t.rev++
	return true
}

// drops the entry drawn at a point inside the popup
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
	e.toasts.slideTo(e.toasts.slidePct(time.Now(), e.animation))
	e.requestRedraw()
}

// started when a toast is queued, so a tick is only pending while something is
// waiting to expire
func (e *EditorComponent) toastTickCmd() tea.Cmd {
	if !e.toasts.pending() {
		return nil
	}
	e.toasts.gen++
	return e.toastNextCmd(e.toasts.gen)
}

// frame rate while something is moving or fading, slow the rest of the time
func (e *EditorComponent) toastNextCmd(gen int) tea.Cmd {
	every := toastTickEvery
	if e.animation && e.toasts.animating(time.Now()) {
		every = toastFrameEvery
	}
	return tea.Tick(every, func(time.Time) tea.Msg {
		return toastTickMsg{gen: gen}
	})
}

func (e *EditorComponent) handleToastTick(
	msg toastTickMsg,
) (EventResult, tea.Cmd) {
	if msg.gen != e.toasts.gen {
		return consumed(), nil
	}
	if e.toasts.step(time.Now(), e.animation) {
		e.requestRedraw()
	}
	if !e.toasts.pending() {
		return consumed(), nil
	}
	return consumed(), e.toastNextCmd(msg.gen)
}

// draws the queued messages as one popup in the bottom-right corner, its last
// row above bottom so the statusline stays clear
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
		padX:         overlayPadX,
	}
	title := i18n.Text(i18n.ToastTitle)
	boxW := toastBoxWidth(r.size.Width)
	boxH := len(items) + toastChrome
	restY := max(bottom-toastGapY-boxH+1, 0)
	now := time.Now()
	box := geom.Area{
		Point: geom.Point{
			X: max(r.size.Width-boxW-toastGapX, 0),
			Y: restY + r.toastSlideOffset(restY),
		},
		Size: geom.Size{Width: boxW, Height: boxH},
	}
	r.editor.toasts.bounds = box
	area := pop.drawInto(buf, box)
	drawPopupTitle(buf, box, title, pop.borderStyle)
	for i, item := range items {
		st := pop.contentStyle.Fg(toastColor(th, item.level))
		// the last entry takes the popup with it, so it never fades
		if r.editor.animation && i < len(items)-1 {
			st = fadedStyle(st, item.expires.Sub(now))
		}
		buf.SetString(
			area.Add(geom.Point{Y: i}),
			runewidth.Truncate(item.text, area.Width, promptEllipsis),
			st,
		)
	}
}

// the rows the popup sits below its resting place, rising from off the bottom
// of the screen when it appears and sinking back down as it leaves
func (r *renderPass) toastSlideOffset(restY int) int {
	travel := max(r.size.Height-restY, 0)
	return travel * r.editor.toasts.slide / 100
}

// a share of the screen, held between the two bounds and whatever width the
// screen can spare
func toastBoxWidth(screenW int) int {
	want := min(max(screenW*toastWidthPct/100, toastMinWidth), toastMaxWidth)
	return min(want, max(screenW-toastGapX, 1))
}

// sinks the text toward the popup background over the entry's last moments, or
// dims it when the popup has no background of its own
func fadedStyle(style tui.Style, left time.Duration) tui.Style {
	if left >= toastFade {
		return style
	}
	pct := int((toastFade - max(left, 0)) * 100 / toastFade)
	text := style.FgColor()
	back := style.BgColor()
	if text.IsReset() {
		return style
	}
	if back.IsReset() {
		return style.Fg(text.Darkened(100 - pct))
	}
	tr, tg, tb, _ := text.RGBA()
	br, bg, bb, _ := back.RGBA()
	mix := func(from, to uint32) uint8 {
		return uint8((from>>8*uint32(100-pct) + to>>8*uint32(pct)) / 100)
	}
	return style.Fg(tui.ColorRGB(mix(tr, br), mix(tg, bg), mix(tb, bb)))
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
	case toastCommand:
		return highlight.LogCommand
	case toastTerminal:
		return highlight.LogTerminal
	default:
		return highlight.LogInfo
	}
}

func toastColor(th *theme.Theme, level toastLevel) tui.Color {
	switch level {
	case toastError:
		return th.Get("ui.log.error").FgColor()
	case toastWarning:
		return th.Get("ui.log.warning").FgColor()
	case toastCommand:
		return th.Get("ui.log.command").FgColor()
	case toastTerminal:
		return th.Get("ui.log.terminal").FgColor()
	default:
		return th.Get("ui.log.info").FgColor()
	}
}
