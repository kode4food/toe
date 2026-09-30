package ui_test

import (
	"image/color"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"

	"github.com/stretchr/testify/assert"
)

func TestImageDisplay(t *testing.T) {
	t.Run("retries starved resize", func(t *testing.T) {
		m := imagePaneModel(t)
		m2, inFlight := m.Update(tea.WindowSizeMsg{Width: 20, Height: 10})
		m = m2.(ui.Model)

		m2, held := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		m = m2.(ui.Model)
		m, raws := collectModelRawMsgs(m, held)
		assert.Empty(t, imagePlacementSizes(raws))

		_, raws = collectModelRawMsgs(m, inFlight)
		sizes := imagePlacementSizes(raws)
		assert.NotEmpty(t, sizes)
		assert.NotEqual(t, sizes[0], sizes[len(sizes)-1])
	})

	t.Run("steady frame sends nothing", func(t *testing.T) {
		m := shownImageModel(t)
		m2, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		_, raws := collectModelRawMsgs(m2.(ui.Model), cmd)
		assert.NotContains(t, strings.Join(raws, ""), "a=T")
	})

	t.Run("resume retransmits", func(t *testing.T) {
		m := shownImageModel(t)
		m2, cmd := m.Update(tea.ResumeMsg{})
		_, raws := collectModelRawMsgs(m2.(ui.Model), cmd)
		raw := strings.Join(raws, "")
		assert.Contains(t, raw, "a=T")
		assert.Contains(t, raw, "\x1b[16t")
	})
}

func imagePaneModel(t *testing.T) ui.Model {
	t.Helper()
	t.Setenv("KITTY_WINDOW_ID", "1")
	root := t.TempDir()
	path := writeRenderImage(t, root, 40, 20, color.RGBA{G: 255, A: 255})
	e := view.NewEditor(root)
	openRenderImagePane(t, e, path)
	return ui.New(e, command.NewKeymaps())
}

func shownImageModel(t *testing.T) ui.Model {
	t.Helper()
	m := imagePaneModel(t)
	m2, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, raws := collectModelRawMsgs(m2.(ui.Model), cmd)
	assert.Contains(t, strings.Join(raws, ""), "a=T")
	return m
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
