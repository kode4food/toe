//go:build !windows

package main_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/stretchr/testify/assert"
)

// tui drives a toe process running inside a pseudo-terminal. All output
// from the process is accumulated so tests can wait for screen content
type tui struct {
	t    *testing.T
	cmd  *exec.Cmd
	ptmx *os.File
	done chan error

	mu  sync.Mutex
	buf strings.Builder
}

const (
	binEnv      = "TOE_INTEGRATION_BIN"
	waitTimeout = 10 * time.Second

	// escPause must comfortably exceed the terminal reader's 50ms escape
	// timeout, or keys sent after a bare escape are read as part of an
	// alt-modified sequence
	escPause = 200 * time.Millisecond
)

var ansiPattern = regexp.MustCompile(
	`\x1b\[[0-9;?]*[ -/]*[@-~]|\x1b\][^\a]*(\a|\x1b\\)|\x1b[@-_]`,
)

func TestMain(m *testing.M) {
	path, err := buildBinary()
	if err != nil {
		fmt.Fprintf(os.Stderr, "build toe: %v\n", err)
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

		tt.reset()
		tt.send("i")
		tt.waitFor("INS")

		tt.reset()
		tt.escape()
		tt.waitFor("NOR")

		tt.reset()
		tt.send("v")
		tt.waitFor("SEL")

		tt.reset()
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

// send writes raw keystrokes to the terminal, pausing briefly so the
// process observes distinct key events
func (tt *tui) send(keys string) {
	tt.t.Helper()
	if _, err := tt.ptmx.WriteString(keys); err != nil {
		tt.t.Fatalf("send %q: %v", keys, err)
	}
	time.Sleep(50 * time.Millisecond)
}

// escape sends a bare escape key and waits out the escape timeout
func (tt *tui) escape() {
	tt.t.Helper()
	tt.send("\x1b")
	time.Sleep(escPause)
}

// waitFor polls the accumulated output, stripped of ANSI sequences, until
// it contains the wanted substring or the timeout expires
func (tt *tui) waitFor(want string) {
	tt.t.Helper()
	deadline := time.Now().Add(waitTimeout)
	for time.Now().Before(deadline) {
		if strings.Contains(tt.screen(), want) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	tt.t.Fatalf("timed out waiting for %q; output:\n%s", want, tt.screen())
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
		time.Sleep(20 * time.Millisecond)
	}
	tt.t.Fatalf(
		"file %s = %q, want %q; output:\n%s",
		path, last, want, tt.screen(),
	)
}

// reset discards accumulated output so waitFor only matches content the
// process renders after this point
func (tt *tui) reset() {
	tt.mu.Lock()
	defer tt.mu.Unlock()
	tt.buf.Reset()
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
			"timed out waiting for process exit; output:\n%s",
			tt.screen(),
		)
	}
}

func (tt *tui) screen() string {
	tt.mu.Lock()
	defer tt.mu.Unlock()
	return ansiPattern.ReplaceAllString(tt.buf.String(), "")
}

func (tt *tui) pump() {
	buf := make([]byte, 4096)
	for {
		n, err := tt.ptmx.Read(buf)
		if n > 0 {
			tt.mu.Lock()
			tt.buf.Write(buf[:n])
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
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})
	if err != nil {
		t.Fatalf("start pty: %v", err)
	}
	tt := &tui{t: t, cmd: cmd, ptmx: ptmx, done: make(chan error, 1)}
	go tt.pump()
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
