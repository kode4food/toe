//go:build !windows && !darwin

package action

import (
	"os"
	"os/exec"
)

func detectClipboardProvider() clipboardProvider {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if paste, ok := lookPath("wl-paste"); ok {
			if copyBin, ok := lookPath("wl-copy"); ok {
				return clipboardProvider{
					name:      "wayland",
					read:      []string{paste, "--no-newline"},
					write:     []string{copyBin, "--type", "text/plain"},
					readPrim:  []string{paste, "-p", "--no-newline"},
					writePrim: []string{copyBin, "-p", "--type", "text/plain"},
				}
			}
		}
	}
	if os.Getenv("DISPLAY") != "" {
		if xclip, ok := lookPath("xclip"); ok {
			return clipboardProvider{
				name:      "xclip",
				read:      []string{xclip, "-o", "-selection", "clipboard"},
				write:     []string{xclip, "-i", "-selection", "clipboard"},
				readPrim:  []string{xclip, "-o"},
				writePrim: []string{xclip, "-i"},
			}
		}
		if xsel, ok := lookPath("xsel"); ok && xselWorks(xsel) {
			return clipboardProvider{
				name:      "xsel",
				read:      []string{xsel, "-o", "-b"},
				write:     []string{xsel, "-i", "-b"},
				readPrim:  []string{xsel, "-o"},
				writePrim: []string{xsel, "-i"},
			}
		}
	}
	return clipboardProvider{name: "none"}
}

func xselWorks(xsel string) bool {
	return exec.Command(xsel, "-o", "-b").Run() == nil
}
