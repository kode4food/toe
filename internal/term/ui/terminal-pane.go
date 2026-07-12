package ui

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
	"github.com/creack/pty"

	"github.com/kode4food/toe/internal/view"
)

// TerminalPane is a [view.Pane] backed by a real PTY and a VT100/xterm
// emulator, so full-screen programs (editors, pagers, TUIs) render correctly
type TerminalPane struct {
	id          view.Id
	area        view.Area
	dirty       bool
	emu         *vt.SafeEmulator
	pty         *os.File
	cmd         *exec.Cmd
	updates     chan struct{}
	closed      chan struct{}
	restore     view.Pane
	titleMu     sync.Mutex
	title       string
	scrollN     int
	mouseMu     sync.Mutex
	mouseOn     bool
	resizeMu    sync.Mutex
	resizeTimer *time.Timer
	pendingW    int
	pendingH    int
}

var ErrScrollbackNoMatch = errors.New("pattern not found in scrollback")

// resizeDebounce avoids flooding a full-screen program with SIGWINCH while
// a pane separator is being dragged
const resizeDebounce = 50 * time.Millisecond

var (
	_ view.Pane  = (*TerminalPane)(nil)
	_ PaneCursor = (*TerminalPane)(nil)
)

// NewTerminalPane spawns shell as a child process attached to a PTY sized
// w by h, and starts pumping its output into a VT emulator
func NewTerminalPane(shell string, w, h int) (*TerminalPane, error) {
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
		updates: make(chan struct{}, 1),
		closed:  make(chan struct{}),
	}
	tp.emu.SetCallbacks(vt.Callbacks{
		Title:       tp.setTitle,
		EnableMode:  func(m ansi.Mode) { tp.setMouseMode(m, true) },
		DisableMode: func(m ansi.Mode) { tp.setMouseMode(m, false) },
	})
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

// SetArea updates the pane's screen rectangle and schedules a debounced
// PTY/emulator resize, so a rapid run of calls only reflows the shell once
func (t *TerminalPane) SetArea(a view.Area) {
	if a == t.area {
		return
	}
	t.area = a
	t.dirty = true
	// reserve the bottom row for the status line, matching renderTerminalPane
	w, h := max(a.Width, 1), max(a.Height-1, 1)
	t.scheduleResize(w, h)
}

func (t *TerminalPane) scheduleResize(w, h int) {
	t.resizeMu.Lock()
	defer t.resizeMu.Unlock()
	t.pendingW, t.pendingH = w, h
	if t.resizeTimer != nil {
		t.resizeTimer.Stop()
	}
	t.resizeTimer = time.AfterFunc(resizeDebounce, t.applyResize)
}

func (t *TerminalPane) applyResize() {
	t.resizeMu.Lock()
	w, h := t.pendingW, t.pendingH
	t.resizeMu.Unlock()
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
	t.titleMu.Lock()
	defer t.titleMu.Unlock()
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

// MouseEnabled reports whether the program running in the shell has
// requested mouse tracking (e.g. vim, htop, tmux)
func (t *TerminalPane) MouseEnabled() bool {
	t.mouseMu.Lock()
	defer t.mouseMu.Unlock()
	return t.mouseOn
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

// SearchScrollback jumps to the nearest line above the current view
// containing pattern (case-insensitive), reporting whether one was found
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
	t.resizeMu.Lock()
	if t.resizeTimer != nil {
		t.resizeTimer.Stop()
	}
	t.resizeMu.Unlock()
	_ = t.cmd.Process.Kill()
	return t.pty.Close()
}

func (t *TerminalPane) setTitle(s string) {
	t.titleMu.Lock()
	t.title = s
	t.titleMu.Unlock()
	t.dirty = true
}

// setMouseMode tracks whether any mouse-tracking DEC mode is enabled, since
// vt exposes no query for it directly, only enable/disable callbacks
func (t *TerminalPane) setMouseMode(m ansi.Mode, on bool) {
	switch m {
	case ansi.ModeMouseNormal, ansi.ModeMouseHighlight,
		ansi.ModeMouseButtonEvent, ansi.ModeMouseAnyEvent:
		t.mouseMu.Lock()
		t.mouseOn = on
		t.mouseMu.Unlock()
	}
}

func (t *TerminalPane) pump() {
	buf := make([]byte, 4096)
	for {
		n, err := t.pty.Read(buf)
		if n > 0 {
			_, _ = t.emu.Write(buf[:n])
			t.dirty = true
			select {
			case t.updates <- struct{}{}:
			default:
			}
		}
		if err != nil {
			close(t.closed)
			return
		}
	}
}

// interactiveShell returns the user's shell for an interactive pane,
// distinct from [view.Options.Shell]'s non-login `sh -c` filter runner
func interactiveShell() string {
	if sh := os.Getenv("SHELL"); sh != "" {
		return sh
	}
	return view.DefaultShell()[0]
}
