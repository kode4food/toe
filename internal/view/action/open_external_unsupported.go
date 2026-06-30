//go:build !darwin && !linux && !windows

package action

import "fmt"

// OpenExternalURL opens a URL with the platform default handler
func OpenExternalURL(raw string) error {
	return fmt.Errorf("%w: %s", ErrExternalURLOpener, raw)
}
