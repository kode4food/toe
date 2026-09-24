package ui_test

import (
	"image/color"
	"regexp"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"

	"github.com/stretchr/testify/assert"
)

// TestImageDisplayNoStarvation guards a resize held during an in-flight
// initial transmit against being dropped instead of retried once sent
func TestImageDisplayNoStarvation(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	root := t.TempDir()
	path := writeRenderImage(t, root, 40, 20, color.RGBA{G: 255, A: 255})
	e := view.NewEditor(root)
	openRenderImagePane(t, e, path)
	m := ui.New(e, command.NewKeymaps())

	m2, inFlight := m.Update(tea.WindowSizeMsg{Width: 20, Height: 10})
	m = m2.(ui.Model)

	// held: a put now would name an id the terminal lacks
	m2, held := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = m2.(ui.Model)
	m, raws := collectModelRawMsgs(m, held)
	assert.Empty(t, imagePlacementSizes(raws))

	_, raws = collectModelRawMsgs(m, inFlight)
	sizes := imagePlacementSizes(raws)
	assert.NotEmpty(t, sizes)
	assert.NotEqual(t, sizes[0], sizes[len(sizes)-1])
}

func imagePlacementSizes(raws []string) []string {
	re := regexp.MustCompile(`\x1b_G[^;]*\bc=(\d+),r=(\d+)`)
	var out []string
	for _, raw := range raws {
		for _, m := range re.FindAllStringSubmatch(raw, -1) {
			out = append(out, m[1]+"x"+m[2])
		}
	}
	return out
}
