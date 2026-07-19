//go:build !windows

package main_test

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImageSession(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	t.Setenv("SSH_CONNECTION", "")
	t.Setenv("SSH_TTY", "")
	dir := t.TempDir()
	path := writeSessionImage(t, dir)
	if path == "" {
		return
	}

	first := startTUI(t, dir)
	first.waitFor("NOR")
	first.send(":e pic.png\r")
	if !assert.Eventually(t,
		first.transmittedImage, waitTimeout, pollPause,
	) {
		return
	}
	first.send(":save_session\r")
	first.quit()

	next := startTUI(t, dir)
	next.waitFor("NOR")
	next.send(":restore_session\r")
	assert.Eventually(t,
		next.transmittedImage, waitTimeout, pollPause,
	)
	next.quit()
}

func writeSessionImage(t *testing.T, dir string) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	var buf bytes.Buffer
	if !assert.NoError(t, png.Encode(&buf, img)) {
		return ""
	}
	return writeFile(t, dir, "pic.png", buf.String())
}
