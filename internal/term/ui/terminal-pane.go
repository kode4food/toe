package ui

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
	"github.com/creack/pty"

	"github.com/kode4food/toe/internal/view"
)

type (
	// TerminalPane is a [view.Pane] backed by a real PTY and a VT100/xterm
	// emulator, so full-screen programs (editors, pagers, TUIs) render
	// correctly
	TerminalPane struct {
		id         view.Id
		area       view.Area
		dirty      bool
		emu        *vt.SafeEmulator
		pty        *os.File
		cmd        *exec.Cmd
		clip       view.Clipboard
		updates    chan struct{}
		closed     chan struct{}
		restore    view.Pane
		stateMu    sync.Mutex
		title      string
		bellRung   bool
		scrollN    int
		mouseOn    atomic.Bool
		selActive  bool
		selA, selB uv.Position
		drag       axisTicker
	}

	selSpan struct {
		start, end uv.Position
	}
)

var ErrScrollbackNoMatch = errors.New("pattern not found in scrollback")

var (
	_ view.Pane = (*TerminalPane)(nil)
	_ RawPane   = (*TerminalPane)(nil)
)

// NewTerminalPane spawns shell as a child process attached to a PTY sized
// w by h, and starts pumping its output into a VT emulator. clip may be nil
// to ignore OSC 52 clipboard writes from programs running inside the shell
func NewTerminalPane(
	shell string, w, h int, clip view.Clipboard,
) (*TerminalPane, error) {
	w, h = max(w, 1), max(h, 1)
	cmd := exec.Command(shell)
	f, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: uint16(h),
		Cols: uint16(w),
	})
	if err != nil {
		return nil, err
	}
	tp := &TerminalPane{
		emu:     vt.NewSafeEmulator(w, h),
		pty:     f,
		cmd:     cmd,
		clip:    clip,
		updates: make(chan struct{}, 1),
		closed:  make(chan struct{}),
	}
	tp.emu.SetCallbacks(vt.Callbacks{
		Title:       tp.setTitle,
		Bell:        tp.onBell,
		EnableMode:  func(m ansi.Mode) { tp.setMouseMode(m, true) },
		DisableMode: func(m ansi.Mode) { tp.setMouseMode(m, false) },
	})
	tp.emu.RegisterOscHandler(52, tp.handleOSC52)
	go tp.pump()
	go func() { _, _ = io.Copy(tp.pty, tp.emu) }()
	return tp, nil
}

// ID returns the pane identifier
func (t *TerminalPane) ID() view.Id {
	return t.id
}

// SetID sets the pane identifier (called by the tree on insertion)
func (t *TerminalPane) SetID(id view.Id) {
	t.id = id
}

// Area returns the screen rectangle assigned by the layout engine
func (t *TerminalPane) Area() view.Area {
	return t.area
}

// MarkDirty flags the pane as needing a repaint on the next frame
func (t *TerminalPane) MarkDirty() {
	t.dirty = true
}

// Mode reports [view.ModeTerminal], since a terminal pane has no
// insert/select/normal distinction
func (t *TerminalPane) Mode() view.Mode {
	return view.ModeTerminal
}

// SetArea updates the pane's screen rectangle and resizes the PTY and
// emulator to match, reflowing the shell
func (t *TerminalPane) SetArea(a view.Area) {
	if a == t.area {
		return
	}
	t.area = a
	t.dirty = true
	// reserve the bottom row for the status line, matching renderTerminalPane
	w, h := max(a.Width, 1), max(a.Height-1, 1)
	t.emu.Resize(w, h)
	_ = pty.Setsize(t.pty, &pty.Winsize{Rows: uint16(h), Cols: uint16(w)})
}

// ConsumeDirty reports whether the pane has changed since the last call,
// clearing the flag
func (t *TerminalPane) ConsumeDirty() bool {
	d := t.dirty
	t.dirty = false
	return d
}

// Cursor reports the shell's cursor position, translated to screen space
func (t *TerminalPane) Cursor(cx *Context) (tea.Cursor, bool) {
	a := t.Area()
	yOff := a.Y
	if bufferlineVisible(cx) {
		yOff++
	}
	pos := t.emu.CursorPosition()
	return tea.Cursor{
		Position: tea.Position{X: a.X + pos.X, Y: yOff + pos.Y},
		Shape:    tea.CursorBlock,
	}, true
}

// Updates delivers a signal each time new output arrives from the shell
func (t *TerminalPane) Updates() <-chan struct{} {
	return t.updates
}

// Closed delivers a signal once the shell process has exited
func (t *TerminalPane) Closed() <-chan struct{} {
	return t.closed
}

// Emulator returns the underlying VT emulator for rendering and input
func (t *TerminalPane) Emulator() *vt.SafeEmulator {
	return t.emu
}

// Title returns the terminal title most recently set by the shell or the
// program running in it (OSC 0/2), or "" if none has been set yet
func (t *TerminalPane) Title() string {
	t.stateMu.Lock()
	defer t.stateMu.Unlock()
	return t.title
}

// SendKey forwards a key event to the shell. Printable text bypasses vt's
// encoder, which silently drops runes whose Mod is non-zero (e.g. shifted).
// Any keypress returns the view to live output, like a real terminal
func (t *TerminalPane) SendKey(k uv.KeyEvent) {
	t.ScrollToBottom()
	if kp, ok := k.(uv.KeyPressEvent); ok && kp.Text != "" {
		_, _ = t.pty.Write([]byte(kp.Text))
		return
	}
	t.emu.SendKey(k)
}

// MouseEnabled reports whether the program running in the shell has requested
// mouse tracking (e.g. vim, htop, tmux)
func (t *TerminalPane) MouseEnabled() bool {
	return t.mouseOn.Load()
}

// SendMouse forwards a mouse event to the shell
func (t *TerminalPane) SendMouse(m uv.MouseEvent) {
	t.emu.SendMouse(m)
}

// ScrollOffset returns the number of lines scrolled back from live output
func (t *TerminalPane) ScrollOffset() int {
	return t.scrollN
}

// ScrollLines moves the view n lines back into scrollback (n < 0 moves toward
// live output); a no-op while the alt screen is active
func (t *TerminalPane) ScrollLines(n int) {
	if t.emu.IsAltScreen() {
		return
	}
	limit := t.emu.ScrollbackLen()
	t.scrollN = min(limit, max(0, t.scrollN+n))
	t.dirty = true
}

// ScrollToBottom returns the view to live output
func (t *TerminalPane) ScrollToBottom() {
	if t.scrollN != 0 {
		t.scrollN = 0
		t.dirty = true
	}
}

// SearchScrollback jumps to the nearest line above the current view containing
// pattern (case-insensitive), reporting whether one was found
func (t *TerminalPane) SearchScrollback(pattern string) bool {
	if pattern == "" {
		return false
	}
	sb := t.emu.Scrollback()
	sbLen := sb.Len()
	pattern = strings.ToLower(pattern)
	top := sbLen - 1 - t.scrollN
	for i := top - 1; i >= 0; i-- {
		if strings.Contains(strings.ToLower(sb.Line(i).String()), pattern) {
			t.scrollN = sbLen - 1 - i
			t.dirty = true
			return true
		}
	}
	return false
}

// Close terminates the shell process and releases the PTY
func (t *TerminalPane) Close() error {
	_ = t.cmd.Process.Kill()
	return t.pty.Close()
}

// IngestOutput applies a chunk of output as if it had just been read from the
// PTY, letting tests simulate shell output without a real child process
func (t *TerminalPane) IngestOutput(data []byte) {
	_, _ = t.emu.Write(data)
	t.dirty = true
	select {
	case t.updates <- struct{}{}:
	default:
	}
}

// ConsumeBell reports whether the bell has rung since it was last consumed. A
// rung bell only clears when read while focused, so it stays visible in the
// status line until the pane is actually looked at
func (t *TerminalPane) ConsumeBell(focused bool) bool {
	t.stateMu.Lock()
	defer t.stateMu.Unlock()
	rung := t.bellRung
	if focused {
		t.bellRung = false
	}
	return rung
}

// Paste sends text to the shell, bracketing it with paste-mode escapes if the
// running program requested bracketed paste
func (t *TerminalPane) Paste(text string) {
	t.ScrollToBottom()
	t.emu.Paste(text)
}

func (t *TerminalPane) setTitle(s string) {
	t.stateMu.Lock()
	t.title = s
	t.stateMu.Unlock()
	t.dirty = true
}

func (t *TerminalPane) onBell() {
	t.stateMu.Lock()
	t.bellRung = true
	t.stateMu.Unlock()
	t.dirty = true
}

func (t *TerminalPane) setMouseMode(m ansi.Mode, on bool) {
	switch m {
	case ansi.ModeMouseNormal, ansi.ModeMouseHighlight,
		ansi.ModeMouseButtonEvent, ansi.ModeMouseAnyEvent:
		// vt exposes no query for tracking mode, only these enable/disable
		// callbacks, so track it ourselves
		t.mouseOn.Store(on)
	default:
		// no-op
	}
}

func (t *TerminalPane) handleOSC52(data []byte) bool {
	parts := bytes.SplitN(data, []byte{';'}, 3)
	if len(parts) != 3 || t.clip == nil {
		return false
	}
	payload := string(parts[2])
	if payload == "?" {
		// ignore clipboard queries, matching most terminals' default
		// disallow-read stance
		return true
	}
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return false
	}
	if bytes.ContainsRune(parts[1], 'p') {
		_ = t.clip.WritePrimary(string(decoded))
	} else {
		_ = t.clip.Write(string(decoded))
	}
	return true
}

func (t *TerminalPane) pump() {
	buf := make([]byte, 4096)
	for {
		n, err := t.pty.Read(buf)
		if n > 0 {
			t.IngestOutput(buf[:n])
		}
		if err != nil {
			close(t.closed)
			return
		}
	}
}

// viewStart returns the top visible absolute row using drawViewport's window
// calculation
func (t *TerminalPane) viewStart(h int) int {
	total := t.emu.ScrollbackLen() + t.emu.Height()
	return max(total-h-t.scrollN, 0)
}

func (t *TerminalPane) contentHeight() int {
	return max(t.area.Height-1, 0)
}

func (t *TerminalPane) toAbsolute(pos uv.Position) uv.Position {
	return uv.Position{X: pos.X, Y: t.viewStart(t.contentHeight()) + pos.Y}
}

func (t *TerminalPane) beginSelection(pos uv.Position) {
	t.selActive = true
	abs := t.toAbsolute(pos)
	t.selA, t.selB = abs, abs
	t.dirty = true
}

func (t *TerminalPane) extendSelection(pos uv.Position) {
	if !t.selActive {
		return
	}
	t.selB = t.toAbsolute(pos)
	t.dirty = true
}

func (t *TerminalPane) endSelection(pos uv.Position) string {
	if !t.selActive {
		return ""
	}
	t.selB = t.toAbsolute(pos)
	t.selActive = false
	t.dirty = true
	return t.selectionText()
}

type selectionRes struct {
	span   selSpan
	active bool
}

func (t *TerminalPane) selection() selectionRes {
	if !t.selActive {
		return selectionRes{}
	}
	return selectionRes{span: normalizeSelection(t.selA, t.selB), active: true}
}

func (t *TerminalPane) selectionText() string {
	sp := normalizeSelection(t.selA, t.selB)
	w := t.emu.Width()
	lines := make([]string, 0, sp.end.Y-sp.start.Y+1)
	for y := sp.start.Y; y <= sp.end.Y; y++ {
		startX, endX := 0, w-1
		if y == sp.start.Y {
			startX = sp.start.X
		}
		if y == sp.end.Y {
			endX = sp.end.X
		}
		var b strings.Builder
		for x := startX; x <= endX && x < w; x++ {
			if c := t.cellAtAbsolute(x, y); c != nil && c.Content != "" {
				b.WriteString(c.Content)
			} else {
				b.WriteByte(' ')
			}
		}
		// terminal selection is line-oriented, not a rectangular block
		lines = append(lines, strings.TrimRight(b.String(), " "))
	}
	return strings.Join(lines, "\n")
}

func (t *TerminalPane) cellAtAbsolute(x, y int) *uv.Cell {
	sbLen := t.emu.ScrollbackLen()
	if y < sbLen {
		return t.emu.Scrollback().CellAt(x, y)
	}
	return t.emu.CellAt(x, y-sbLen)
}

func normalizeSelection(a, b uv.Position) selSpan {
	if a.Y > b.Y || (a.Y == b.Y && a.X > b.X) {
		a, b = b, a
	}
	return selSpan{start: a, end: b}
}

func interactiveShell() string {
	// distinct from view.Options.Shell, which is a non-login `sh -c` filter
	// runner rather than the user's real interactive shell
	if sh := os.Getenv("SHELL"); sh != "" {
		return sh
	}
	return view.DefaultShell()[0]
}
