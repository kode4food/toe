//go:build !windows

package main_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/x/vt"
	"github.com/creack/pty"
	"github.com/stretchr/testify/assert"
)

// tui drives a toe process running inside a pseudo-terminal. Output is fed into
// a virtual terminal emulator so tests assert against the rendered screen
// rather than the raw escape stream, which only carries cell deltas
type tui struct {
	t    *testing.T
	cmd  *exec.Cmd
	ptmx *os.File
	done chan error

	mu  sync.Mutex
	raw bytes.Buffer
	vt  *vt.SafeEmulator
}

const (
	binEnv      = "TOE_INTEGRATION_BIN"
	waitTimeout = 10 * time.Second
	pollPause   = 20 * time.Millisecond

	termCols = 80
	termRows = 24

	// escPause must comfortably exceed the terminal reader's 50ms escape
	// timeout, or keys sent after a bare escape are read as part of an
	// alt-modified sequence
	escPause = 200 * time.Millisecond
)

func TestMain(m *testing.M) {
	path, err := buildBinary()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "build toe: %v\n", err)
		os.Exit(1)
	}
	_ = os.Setenv(binEnv, path)
	os.Exit(m.Run())
}

func TestIntegration(t *testing.T) {
	t.Run("launches and quits", func(t *testing.T) {
		dir := t.TempDir()
		path := writeFile(t, dir, "note.txt", "hello toe\n")

		tt := startTUI(t, dir, path)
		tt.waitFor("hello toe")
		tt.waitFor("NOR")
		tt.quit()
	})

	t.Run("opens a terminal and runs a command", func(t *testing.T) {
		dir := t.TempDir()
		path := writeFile(t, dir, "note.txt", "hello toe\n")

		tt := startTUI(t, dir, path)
		tt.waitFor("NOR")

		tt.send(":terminal\r")
		tt.send("echo TERMINAL_PANE_WORKS\r")
		tt.waitFor("TERMINAL_PANE_WORKS")

		tt.send("\x17q") // Ctrl-w q closes the terminal
		tt.waitFor("NOR")
		tt.quit()
	})

	t.Run("OSC title updates the status label", func(t *testing.T) {
		dir := t.TempDir()
		path := writeFile(t, dir, "note.txt", "hello toe\n")

		tt := startTUI(t, dir, path)
		tt.waitFor("NOR")

		tt.send(":terminal\r")
		tt.waitFor(" terminal")
		tt.send(`printf '\033]0;MYTITLE\007'` + "\r")
		tt.waitFor("MYTITLE")

		tt.send("\x17q") // Ctrl-w q closes the terminal
		tt.quit()
	})

	t.Run("output does not overlap the status row", func(t *testing.T) {
		dir := t.TempDir()
		path := writeFile(t, dir, "note.txt", "hello toe\n")

		tt := startTUI(t, dir, path)
		tt.waitFor("NOR")

		tt.send(":terminal\r")
		tt.waitFor(" terminal")
		// fill well past the pane height so the shell's own idea of its last
		// row is exercised, then print a marker as the final visible line
		tt.send("seq 1 100; echo LAST_LINE_MARKER\r")
		tt.waitFor("LAST_LINE_MARKER")

		lines := strings.Split(tt.screen(), "\n")
		markerRow, statusRow := -1, -1
		for i, line := range lines {
			if strings.Contains(line, "LAST_LINE_MARKER") {
				markerRow = i
			}
			if strings.Contains(line, " TRM ") {
				statusRow = i
			}
		}
		if markerRow < 0 || statusRow < 0 {
			t.Fatalf(
				"expected both marker and status rows visible; screen:\n%s",
				tt.screen(),
			)
		}
		// the shell prints a fresh prompt on its own row right after the
		// marker; if the PTY thinks it has one row more than we draw, that
		// prompt row goes missing instead of appearing between them
		if statusRow-markerRow < 2 {
			t.Fatalf(
				"expected a prompt row between the marker and the pane's "+
					"status row; screen:\n%s",
				tt.screen(),
			)
		}

		tt.send("\x17q") // Ctrl-w q closes the terminal
		tt.quit()
	})

	t.Run("closes when the shell exits", func(t *testing.T) {
		dir := t.TempDir()
		path := writeFile(t, dir, "note.txt", "hello toe\n")

		tt := startTUI(t, dir, path)
		tt.waitFor("NOR")

		tt.send(":terminal\r")
		tt.waitFor(" terminal")
		tt.send("exit\r")

		deadline := time.Now().Add(waitTimeout)
		for time.Now().Before(deadline) &&
			strings.Contains(tt.screen(), " terminal") {
			time.Sleep(20 * time.Millisecond)
		}
		if strings.Contains(tt.screen(), " terminal") {
			t.Fatalf(
				"terminal pane still open after shell exit; screen:\n%s",
				tt.screen(),
			)
		}
		tt.quit()
	})

	t.Run("ctrl-w still navigates away", func(t *testing.T) {
		dir := t.TempDir()
		path := writeFile(t, dir, "note.txt", "hello toe\n")

		tt := startTUI(t, dir, path)
		tt.waitFor("NOR")

		tt.send(":vsplit\r")
		tt.send(":terminal\r")
		tt.waitFor(" terminal")

		tt.send("\x17h") // Ctrl-w h: focus the split to the left (the document)
		tt.send("ihi ")
		tt.waitFor("hi hello toe")

		tt.escape()
		tt.send(":write\r")
		tt.waitFileContent(path, "hi hello toe\n")
		tt.quit()
	})

	t.Run("terminal commands hide outside TRM mode", func(t *testing.T) {
		dir := t.TempDir()
		path := writeFile(t, dir, "note.txt", "hello toe\n")

		tt := startTUI(t, dir, path)
		tt.waitFor("NOR")

		tt.send("\x17") // Ctrl-w: open the window which-key menu on a doc pane
		tt.waitFor("Window")
		assert.NotContains(t, tt.screen(), "scrollback")
		tt.escape()

		tt.send(":terminal\r")
		tt.waitFor(" terminal")

		tt.send("\x17") // Ctrl-w: open the same menu with a terminal focused
		tt.waitFor("scrollback")

		tt.escape()
		tt.send("\x17q") // Ctrl-w q closes the terminal
		tt.quit()
	})

	t.Run("edits and saves", func(t *testing.T) {
		dir := t.TempDir()
		path := writeFile(t, dir, "note.txt", "world\n")

		tt := startTUI(t, dir, path)
		tt.waitFor("world")

		tt.send("ihello ")
		tt.waitFor("hello world")
		tt.escape()
		tt.send(":write\r")
		tt.waitFileContent(path, "hello world\n")
		tt.quit()
	})

	t.Run("transitions between modes", func(t *testing.T) {
		dir := t.TempDir()
		path := writeFile(t, dir, "note.txt", "abc\n")

		tt := startTUI(t, dir, path)
		tt.waitFor("NOR")

		tt.send("i")
		tt.waitFor("INS")

		tt.escape()
		tt.waitFor("NOR")

		tt.send("v")
		tt.waitFor("SEL")

		tt.escape()
		tt.waitFor("NOR")
		tt.quit()
	})

	t.Run("edits at multiple cursors", func(t *testing.T) {
		dir := t.TempDir()
		path := writeFile(t, dir, "note.txt", "aaa\nbbb\n")

		tt := startTUI(t, dir, path)
		tt.waitFor("bbb")

		tt.send("C")
		tt.send("iX")
		tt.waitFor("Xaaa")
		tt.waitFor("Xbbb")
		tt.escape()
		tt.send(":write\r")
		tt.waitFileContent(path, "Xaaa\nXbbb\n")
		tt.quit()
	})

	t.Run("searches and jumps between matches", func(t *testing.T) {
		dir := t.TempDir()
		path := writeFile(t, dir, "note.txt", "foo\nbar\nfoo\nbar\nfoo\n")

		tt := startTUI(t, dir, path)
		tt.waitFor("foo")

		tt.send("/foo\r")
		tt.waitFor("3:1") // first match strictly after the cursor: line 3

		tt.send("n")
		tt.waitFor("5:1") // next match wraps forward to the last line

		tt.send("N")
		tt.waitFor("3:1") // prev match steps back
		tt.quit()
	})

	t.Run("replaces all regex matches", func(t *testing.T) {
		dir := t.TempDir()
		path := writeFile(t, dir, "note.txt", "foo\nbar\nfoo\n")

		tt := startTUI(t, dir, path)
		tt.waitFor("foo")

		tt.send("%sfoo\r")
		tt.send("cX")
		tt.waitFor("X")
		tt.escape()
		tt.send(":write\r")
		tt.waitFileContent(path, "X\nbar\nX\n")
		tt.quit()
	})

	t.Run("splits show the same document", func(t *testing.T) {
		dir := t.TempDir()
		path := writeFile(t, dir, "note.txt", "hello\n")

		tt := startTUI(t, dir, path)
		tt.waitFor("hello")

		tt.send("\x17v") // ctrl-w v: vertical right split
		tt.waitFor("│")
		assert.Equal(t, 2, strings.Count(tt.screen(), "hello"))

		tt.send("A world")
		tt.waitFor("hello world")
		assert.Equal(t, 2, strings.Count(tt.screen(), "hello world"))

		tt.escape()
		tt.send(":write\r") // closing a modified doc needs it saved first
		tt.send("\x17q")    // ctrl-w q: close the focused split
		tt.waitFor("hello world")
		assert.Equal(t, 1, strings.Count(tt.screen(), "hello world"))
		tt.quit()
	})

	t.Run("cycles buffers across splits", func(t *testing.T) {
		dir := t.TempDir()
		pathA := writeFile(t, dir, "a.txt", "buffer a\n")
		writeFile(t, dir, "b.txt", "buffer b\n")

		tt := startTUI(t, dir, pathA)
		tt.waitFor("buffer a")

		tt.send("\x17v") // ctrl-w v: vertical right split (same doc for now)
		tt.waitFor("│")
		tt.send(":open b.txt\r") // swap the focused split's buffer to b.txt
		tt.waitFor("buffer b")
		assert.Contains(t, tt.screen(), "buffer a") // left split untouched

		tt.send("\x17w") // ctrl-w w: rotate focus back to the left split
		tt.send("A - edited")
		tt.waitFor("buffer a - edited")
		assert.NotContains(t, tt.screen(), "buffer b - edited")

		tt.escape()
		tt.send("\x17w") // ctrl-w w: rotate to the right split
		tt.send("A - edited")
		tt.waitFor("buffer b - edited")
		tt.escape()
		tt.quit()
	})

	t.Run("reloads external changes", func(t *testing.T) {
		dir := t.TempDir()
		path := writeFile(t, dir, "note.txt", "before\n")

		tt := startTUI(t, dir, path)
		tt.waitFor("before")

		err := os.WriteFile(path, []byte("after\n"), 0o644)
		assert.NoError(t, err)
		tt.send(":reload\r")
		tt.waitFor("after")
		tt.quit()
	})
}

// send types keystrokes one byte at a time so the process sees distinct key
// events with per-keystroke redraws, matching real typing
func (tt *tui) send(keys string) {
	tt.t.Helper()
	for i := 0; i < len(keys); i++ {
		if _, err := tt.ptmx.Write([]byte{keys[i]}); err != nil {
			tt.t.Fatalf("send %q: %v", keys, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)
}

// escape sends a bare escape key and waits out the escape timeout
func (tt *tui) escape() {
	tt.t.Helper()
	tt.send("\x1b")
	time.Sleep(escPause)
}

// waitFor polls the emulated screen until it contains the wanted substring or
// the timeout expires
func (tt *tui) waitFor(want string) {
	tt.t.Helper()
	deadline := time.Now().Add(waitTimeout)
	for time.Now().Before(deadline) {
		if strings.Contains(tt.screen(), want) {
			return
		}
		time.Sleep(pollPause)
	}
	tt.t.Fatalf("timed out waiting for %q; screen:\n%s", want, tt.screen())
}

// waitFileContent polls a file on disk until it holds the wanted content
// or the timeout expires
func (tt *tui) waitFileContent(path, want string) {
	tt.t.Helper()
	deadline := time.Now().Add(waitTimeout)
	var last string
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(path); err == nil {
			last = string(b)
			if last == want {
				return
			}
		}
		time.Sleep(pollPause)
	}
	tt.t.Fatalf(
		"file %s = %q, want %q; screen:\n%s",
		path, last, want, tt.screen(),
	)
}

// quit exits the editor from normal mode and waits for a clean process
// exit
func (tt *tui) quit() {
	tt.t.Helper()
	tt.send(":quit!\r")
	select {
	case err := <-tt.done:
		assert.NoError(tt.t, err)
	case <-time.After(waitTimeout):
		tt.t.Fatalf(
			"timed out waiting for process exit; screen:\n%s",
			tt.screen(),
		)
	}
}

func (tt *tui) screen() string {
	tt.mu.Lock()
	defer tt.mu.Unlock()
	return tt.vt.String()
}

func (tt *tui) transmittedImage() bool {
	tt.mu.Lock()
	defer tt.mu.Unlock()
	return bytes.Contains(tt.raw.Bytes(), []byte("\x1b_G"))
}

func (tt *tui) pump() {
	buf := make([]byte, 4096)
	for {
		n, err := tt.ptmx.Read(buf)
		if n > 0 {
			tt.mu.Lock()
			_, _ = tt.raw.Write(buf[:n])
			_, _ = tt.vt.Write(buf[:n])
			tt.mu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

func (tt *tui) stop() {
	if tt.cmd.Process != nil {
		_ = tt.cmd.Process.Kill()
	}
	_ = tt.ptmx.Close()
}

func startTUI(t *testing.T, dir string, args ...string) *tui {
	t.Helper()
	cmd := exec.Command(os.Getenv(binEnv), args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"HOME="+dir,
		"XDG_CONFIG_HOME="+filepath.Join(dir, ".config"),
	)
	// an empty .zshrc suppresses zsh's interactive new-user setup wizard,
	// which some Linux distros trigger on a HOME with no rc file
	writeFile(t, dir, ".zshrc", "")

	ptmx, err := pty.StartWithSize(
		cmd, &pty.Winsize{Rows: termRows, Cols: termCols},
	)
	if err != nil {
		t.Fatalf("start pty: %v", err)
	}
	tt := &tui{
		t:    t,
		cmd:  cmd,
		ptmx: ptmx,
		done: make(chan error, 1),
		vt:   vt.NewSafeEmulator(termCols, termRows),
	}
	go tt.pump()
	// drain the emulator's replies (mode queries, DA responses) back to the
	// child, or an unread reply blocks the next Write forever
	go func() { _, _ = io.Copy(tt.ptmx, tt.vt) }()
	go func() { tt.done <- cmd.Wait() }()
	t.Cleanup(tt.stop)
	return tt
}

func buildBinary() (string, error) {
	dir, err := os.MkdirTemp("", "toe-integration-")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "toe")
	out, err := exec.Command(
		"go", "build", "-o", path, ".",
	).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, out)
	}
	return path, nil
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}
