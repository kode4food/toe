package action

import (
	"os/exec"
	"strings"
)

// clipboardProvider names the external commands used to read and write the
// system clipboard and, where supported, the X11 PRIMARY selection. A nil
// command means that operation is unavailable
type clipboardProvider struct {
	name      string
	read      []string
	write     []string
	readPrim  []string
	writePrim []string
}

func lookPath(name string) (string, bool) {
	path, err := exec.LookPath(name)
	return path, err == nil
}

func runRead(cmd []string) (string, error) {
	if cmd == nil {
		return "", ErrNoClipboardProvider
	}
	out, err := exec.Command(cmd[0], cmd[1:]...).Output()
	if err != nil {
		return "", ErrNoClipboardProvider
	}
	return string(out), nil
}

func runWrite(cmd []string, text string) error {
	if cmd == nil {
		return ErrNoClipboardProvider
	}
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Stdin = strings.NewReader(text)
	if c.Run() != nil {
		return ErrNoClipboardProvider
	}
	return nil
}
