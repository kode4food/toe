//go:build windows

package action

import "os/exec"

// OpenExternalURL opens a URL with the platform default handler
func OpenExternalURL(raw string) error {
	return exec.Command(
		"rundll32", "url.dll,FileProtocolHandler", raw,
	).Start()
}
