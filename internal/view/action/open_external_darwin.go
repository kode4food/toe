//go:build darwin

package action

import "os/exec"

// OpenExternalURL opens a URL with the platform default handler
func OpenExternalURL(raw string) error {
	return exec.Command("open", raw).Start()
}
