package ui_test

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"

	"github.com/stretchr/testify/assert"
)

func TestTransmit(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")

	t.Run("local png reads off disk", func(t *testing.T) {
		t.Setenv("SSH_CONNECTION", "")
		t.Setenv("SSH_TTY", "")
		green := color.RGBA{G: 255, A: 255}
		raw := transmitRaw(t, writeRenderImage(t, t.TempDir(), 40, 20, green))
		assert.Contains(t, raw, "t=f")
		assert.NotContains(t, raw, "t=t")
	})

	t.Run("oversized png still reads off disk", func(t *testing.T) {
		t.Setenv("SSH_CONNECTION", "")
		t.Setenv("SSH_TTY", "")
		blue := color.RGBA{B: 255, A: 255}
		path := writeRenderImage(t, t.TempDir(), 4000, 3000, blue)
		raw := transmitRaw(t, path)
		assert.Contains(t, raw, "t=f")
		assert.NotContains(t, raw, "t=t")
	})

	t.Run("non-png uses temp file", func(t *testing.T) {
		t.Setenv("SSH_CONNECTION", "")
		t.Setenv("SSH_TTY", "")
		raw := transmitRaw(t, writeRenderJPEG(t, t.TempDir(), 40, 20))
		assert.Contains(t, raw, "t=t")
		assert.NotContains(t, raw, "t=f")
	})

	t.Run("ssh streams inline", func(t *testing.T) {
		t.Setenv("SSH_CONNECTION", "1.2.3.4 5 6.7.8.9 22")
		red := color.RGBA{R: 255, A: 255}
		raw := transmitRaw(t, writeRenderImage(t, t.TempDir(), 40, 20, red))
		assert.NotContains(t, raw, "t=f")
		assert.NotContains(t, raw, "t=t")
	})
}

func transmitRaw(t testing.TB, path string) string {
	t.Helper()
	e := view.NewEditor(filepath.Dir(path))
	openRenderImagePane(t, e, path)
	m := ui.New(e, command.NewKeymaps())
	m2, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	_, raw := collectModelRawMsgs(m2.(ui.Model), cmd)
	return strings.Join(raw, "")
}

func writeRenderJPEG(t testing.TB, dir string, w, h int) string {
	t.Helper()
	var buf bytes.Buffer
	assert.NoError(t,
		jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h)), nil))
	path := filepath.Join(dir, "pic.jpg")
	assert.NoError(t, os.WriteFile(path, buf.Bytes(), 0o644))
	return path
}
