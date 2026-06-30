//go:build linux

package action

import "os/exec"

// OpenExternalURL opens a URL with the platform default handler
func OpenExternalURL(raw string) error {
	return exec.Command("xdg-open", raw).Start()
}
