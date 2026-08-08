package ui

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"net/url"
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

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/view"
)

type (
	// TerminalPane is a [view.Pane] backed by a real PTY and a VT100/xterm
	// emulator, so full-screen programs (editors, pagers, TUIs) render
	// correctly
	TerminalPane struct {
		id     view.Id
		editor *view.Editor

		metadata  metadataState
		selection selectionState

		shell string
		cmd   *exec.Cmd
		pty   *os.File
		emu   *vt.SafeEmulator

		area      geom.Area
		dirty     bool
		scrollOff int

		clip   view.Clipboard
		notify func()

		closed  chan struct{}
		mouseOn atomic.Bool
		output  atomic.Bool
	}

	metadataState struct {
		sync.Mutex
		path     string
		title    string
		bellRung bool
	}

	selectionState struct {
		active bool
		span   selSpan
		drag   axisTicker
	}

	// selSpan is a terminal selection's start and end positions
	selSpan struct {
		start uv.Position
		end   uv.Position
	}
)

var ErrScrollbackNoMatch = errors.New("pattern not found in scrollback")

var (
	_ view.Pane          = (*TerminalPane)(nil)
	_ view.AsyncRenderer = (*TerminalPane)(nil)
	_ PaneInput          = (*TerminalPane)(nil)
	_ PaneCursor         = (*TerminalPane)(nil)
	_ Pasteable          = (*TerminalPane)(nil)
	_ Draggable          = (*TerminalPane)(nil)
)

// NewTerminalPane spawns shell in a PTY and pumps its output into a VT emulator
// sized w by h
func NewTerminalPane(
	e *view.Editor, shell string, size geom.Size,
) (*TerminalPane, error) {
	return NewTerminalPaneInDir(e, TerminalPaneArgs{
		Shell: shell,
		Dir:   e.Cwd(),
		Size:  size,
	})
}

// TerminalPaneArgs names the shell to spawn and the directory to run it in
type TerminalPaneArgs struct {
	Shell string
	Dir   string
	Size  geom.Size
}

// NewTerminalPaneInDir spawns the shell in the given directory
func NewTerminalPaneInDir(
	e *view.Editor, args TerminalPaneArgs,
) (*TerminalPane, error) {
	args.Size.Width = max(args.Size.Width, 1)
	args.Size.Height = max(args.Size.Height, 1)
	cmd := exec.Command(args.Shell)
	path := terminalPath(e, args.Dir)
	cmd.Dir = path
	f, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: uint16(args.Size.Height),
		Cols: uint16(args.Size.Width),
	})
	if err != nil {
		return nil, err
	}
	tp := &TerminalPane{
		editor: e,
		shell:  args.Shell,
		emu:    vt.NewSafeEmulator(args.Size.Width, args.Size.Height),
		pty:    f,
		cmd:    cmd,
		clip:   e.Clipboard(),
		notify: func() {},
		metadata: metadataState{
			path: path,
		},
		closed: make(chan struct{}),
	}
	tp.emu.SetCallbacks(vt.Callbacks{
		Title:       tp.setTitle,
		Bell:        tp.onBell,
		EnableMode:  func(m ansi.Mode) { tp.setMouseMode(m, true) },
		DisableMode: func(m ansi.Mode) { tp.setMouseMode(m, false) },
	})
	tp.emu.RegisterOscHandler(52, tp.handleOSC52)
	tp.emu.RegisterOscHandler(7, tp.handleOSC7)
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

// Split starts another terminal using the same shell
func (t *TerminalPane) Split() (view.Pane, error) {
	return NewTerminalPane(
		t.editor, t.shell,
		geom.Size{Width: t.area.Width, Height: max(t.area.Height-1, 1)},
	)
}

// Close terminates this terminal and closes its pane. The tree discards any
// panes stashed behind it when the node is removed
func (t *TerminalPane) Close() {
	_ = t.Stop()
	if t.editor != nil {
		t.editor.RemovePane(t.id)
	}
}

// Discard terminates this terminal when its slot is vacated without reverting
func (t *TerminalPane) Discard() {
	_ = t.Stop()
}

// Shutdown terminates this terminal, releasing its PTY
func (t *TerminalPane) Shutdown() {
	_ = t.Stop()
}

// Area returns the screen rectangle assigned by the layout engine
func (t *TerminalPane) Area() geom.Area {
	return t.area
}

// MarkDirty flags the pane as needing a repaint on the next frame
func (t *TerminalPane) MarkDirty() {
	t.dirty = true
}

// SetRedraw installs the hook the shell calls to wake the render loop when it
// mutates the pane off the event loop; the tree wires it on insertion
func (t *TerminalPane) SetRedraw(fn func()) {
	t.notify = fn
}

// Mode reports [view.ModeTerminal], since a terminal pane has no
// insert/select/normal distinction
func (t *TerminalPane) Mode() view.Mode {
	return view.ModeTerminal
}

// Path returns the shell working directory most recently reported by OSC 7
func (t *TerminalPane) Path() string {
	t.metadata.Lock()
	defer t.metadata.Unlock()
	return t.metadata.path
}

// SaveSession stores a terminal slot so a fresh shell can be reopened
func (t *TerminalPane) SaveSession(w *view.SessionWriter) {
	w.SaveSlot(view.SessionKindTerminal, t.Path())
}

// SetArea updates the pane's screen rectangle and resizes the PTY and
// emulator to match, reflowing the shell
func (t *TerminalPane) SetArea(a geom.Area) {
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
	t.metadata.Lock()
	defer t.metadata.Unlock()
	return t.metadata.title
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
	return t.scrollOff
}

// ScrollLines moves the view n lines back into scrollback (n < 0 moves toward
// live output); a no-op while the alt screen is active
func (t *TerminalPane) ScrollLines(n int) {
	if t.emu.IsAltScreen() {
		return
	}
	limit := t.emu.ScrollbackLen()
	t.scrollOff = min(limit, max(0, t.scrollOff+n))
	t.dirty = true
}

// ScrollToBottom returns the view to live output
func (t *TerminalPane) ScrollToBottom() {
	if t.scrollOff != 0 {
		t.scrollOff = 0
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
	top := sbLen - 1 - t.scrollOff
	for i := top - 1; i >= 0; i-- {
		if strings.Contains(strings.ToLower(sb.Line(i).String()), pattern) {
			t.scrollOff = sbLen - 1 - i
			t.dirty = true
			return true
		}
	}
	return false
}

// Stop terminates the shell process and releases the PTY
func (t *TerminalPane) Stop() error {
	_ = t.cmd.Process.Kill()
	return t.pty.Close()
}

// IngestOutput applies a chunk of output as if it had just been read from the
// PTY, letting tests simulate shell output without a real child process
func (t *TerminalPane) IngestOutput(data []byte) {
	_, _ = t.emu.Write(data)
	if len(data) > 0 {
		t.output.Store(true)
	}
	t.dirty = true
	t.notify()
}

// ConsumeBell reports whether the bell has rung since it was last consumed. A
// rung bell only clears when read while focused, so it stays visible in the
// status line until the pane is actually looked at
func (t *TerminalPane) ConsumeBell(focused bool) bool {
	t.metadata.Lock()
	defer t.metadata.Unlock()
	rung := t.metadata.bellRung
	if focused {
		t.metadata.bellRung = false
	}
	return rung
}

// Paste sends text to the shell, bracketing it with paste-mode escapes if the
// running program requested bracketed paste
func (t *TerminalPane) Paste(text string) {
	t.ScrollToBottom()
	t.emu.Paste(text)
}

func (t *TerminalPane) hasOutput() bool {
	return t.output.Load()
}

func (t *TerminalPane) setTitle(s string) {
	t.metadata.Lock()
	t.metadata.title = s
	t.metadata.Unlock()
	t.dirty = true
}

func (t *TerminalPane) onBell() {
	t.metadata.Lock()
	t.metadata.bellRung = true
	t.metadata.Unlock()
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

func (t *TerminalPane) handleOSC7(data []byte) bool {
	_, raw, ok := strings.Cut(string(data), ";")
	if !ok {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "file" || u.Path == "" {
		return false
	}
	t.metadata.Lock()
	t.metadata.path = u.Path
	t.metadata.Unlock()
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
			t.notify()
			return
		}
	}
}

// viewStart returns the top visible absolute row using drawViewport's window
// calculation
func (t *TerminalPane) viewStart(h int) int {
	total := t.emu.ScrollbackLen() + t.emu.Height()
	return max(total-h-t.scrollOff, 0)
}

func (t *TerminalPane) contentHeight() int {
	return max(t.area.Height-1, 0)
}

func (t *TerminalPane) toAbsolute(pos uv.Position) uv.Position {
	return uv.Position{X: pos.X, Y: t.viewStart(t.contentHeight()) + pos.Y}
}

func (t *TerminalPane) beginSelection(pos uv.Position) {
	t.selection.active = true
	abs := t.toAbsolute(pos)
	t.selection.span = selSpan{start: abs, end: abs}
	t.dirty = true
}

func (t *TerminalPane) extendSelection(pos uv.Position) {
	if !t.selection.active {
		return
	}
	t.selection.span.end = t.toAbsolute(pos)
	t.dirty = true
}

func (t *TerminalPane) endSelection(pos uv.Position) string {
	if !t.selection.active {
		return ""
	}
	t.selection.span.end = t.toAbsolute(pos)
	t.selection.active = false
	t.dirty = true
	return t.selectionText()
}

func (t *TerminalPane) selectedSpan() (selSpan, bool) {
	if !t.selection.active {
		return selSpan{}, false
	}
	return normalizeSelection(t.selection.span), true
}

func (t *TerminalPane) selectionText() string {
	sp := normalizeSelection(t.selection.span)
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
			c := t.cellAtAbsolute(geom.Point{X: x, Y: y})
			if c != nil && c.Content != "" {
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

func (t *TerminalPane) cellAtAbsolute(at geom.Point) *uv.Cell {
	sbLen := t.emu.ScrollbackLen()
	if at.Y < sbLen {
		return t.emu.Scrollback().CellAt(at.X, at.Y)
	}
	return t.emu.CellAt(at.X, at.Y-sbLen)
}

func normalizeSelection(s selSpan) selSpan {
	if s.start.Y > s.end.Y ||
		(s.start.Y == s.end.Y && s.start.X > s.end.X) {
		return selSpan{start: s.end, end: s.start}
	}
	return s
}

func terminalPath(e *view.Editor, dir string) string {
	if dir != "" && dirOK(dir) {
		return dir
	}
	return e.Cwd()
}

func dirOK(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func interactiveShell() string {
	// distinct from view.Options.Shell, which is a non-login `sh -c` filter
	// runner rather than the user's real interactive shell
	if sh := os.Getenv("SHELL"); sh != "" {
		return sh
	}
	return view.DefaultShell()[0]
}

func registerTerminalPane(e *view.Editor) {
	e.RegisterPaneRestorer(view.SessionKindTerminal,
		func(e *view.Editor, session *view.PaneSession) (view.Pane, error) {
			return NewTerminalPaneInDir(e, TerminalPaneArgs{
				Shell: interactiveShell(),
				Dir:   session.Path(),
			})
		})
}
