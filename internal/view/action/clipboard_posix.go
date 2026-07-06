//go:build !windows

package action

import (
	"os/exec"
	"strings"
)

const ttyDevice = "/dev/tty"

func tryReadCmds(cmds [][]string) (string, bool) {
	for _, args := range cmds {
		if out, err := exec.Command(args[0], args[1:]...).Output(); err == nil {
			return string(out), true
		}
	}
	return "", false
}

func tryWriteCmds(cmds [][]string, text string) bool {
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdin = strings.NewReader(text)
		if cmd.Run() == nil {
			return true
		}
	}
	return false
}
