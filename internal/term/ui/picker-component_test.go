package ui_test

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

// fixedPickerSource is a deterministic, synchronous picker source: a fixed list
// of items in a known order, each previewing distinct content. The preview of
// the selected item reveals the selection, independent of the scroll position
type fixedPickerSource struct {
	items []ui.PickerItem
	title string
}

func TestPickerScroll(t *testing.T) {
	// At 120x20: areaW=108, areaH=16, left=6, top=1, preview layout active.
	// listBounds: x=7, y=4, w=53, h=12; valid x [7,59], y [4,15]. Preview pane
	// occupies x >= 60 inside the picker

	t.Run("wheel list keeps selection", func(t *testing.T) {
		m := fixedPicker(t, 30, 120, 20)

		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "CONTENT-00")
		assert.NotContains(t, out, "item29")

		for range 10 {
			m2, _ := m.Update(tea.MouseWheelMsg{
				X: 30, Y: 10, Button: tea.MouseWheelDown,
			})
			m = m2.(ui.Model)
		}
		out = stripANSI(m.View().Content)
		// list scrolled to reveal the far item; selection (and preview)
		// unchanged
		assert.Contains(t, out, "item29")
		assert.Contains(t, out, "CONTENT-00")
		assert.NotContains(t, out, "CONTENT-29")
	})

	t.Run("wheel preview keeps list", func(t *testing.T) {
		m := fixedPicker(t, 30, 120, 20)

		out := stripANSI(m.View().Content)
		assert.NotContains(t, out, "item29")

		for range 10 {
			// X=65 is inside the picker but in the preview pane, outside
			// listBounds
			m2, _ := m.Update(tea.MouseWheelMsg{
				X: 65, Y: 10, Button: tea.MouseWheelDown,
			})
			m = m2.(ui.Model)
		}
		out = stripANSI(m.View().Content)
		assert.NotContains(t, out, "item29")
	})

	t.Run("click selects the item under the cursor", func(t *testing.T) {
		m := fixedPicker(t, 30, 120, 20)

		// locate the rendered row of item05 and click it; the preview should
		// then show that item's body
		lines := strings.Split(m.View().Content, "\n")
		clickX, clickY := -1, -1
		for y, line := range lines {
			if col := strings.Index(stripANSI(line), "item05"); col >= 0 {
				clickX, clickY = col, y
				break
			}
		}
		assert.GreaterOrEqual(t, clickY, 0)

		m2, _ := m.Update(tea.MouseClickMsg{
			X: clickX, Y: clickY, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)

		out := stripANSI(m.View().Content)
		assert.Contains(t, out, "CONTENT-05")
		assert.NotContains(t, out, "CONTENT-00")
	})

	t.Run("click outside picker dismisses it", func(t *testing.T) {
		m := fixedPicker(t, 5, 120, 20)

		m2, _ := m.Update(tea.MouseClickMsg{
			X: 0, Y: 0, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)

		out := stripANSI(m.View().Content)
		assert.NotContains(t, out, "fixed")
	})

	t.Run("drag divider updates split ratio", func(t *testing.T) {
		m := fixedPicker(t, 30, 120, 20)
		_ = m.View()

		m2, _ := m.Update(tea.MouseClickMsg{
			X: 60, Y: 8, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		m2, _ = m.Update(tea.MouseMotionMsg{
			X: 75, Y: 8, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)
		m2, _ = m.Update(tea.MouseReleaseMsg{
			Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)

		ratios := m.PickerLayoutOptions().SplitRatios
		assert.InDelta(t, 0.648, ratios["fixed"], 0.001)
	})

	t.Run("drag stays monotonic", func(t *testing.T) {
		m := fixedPicker(t, 30, 120, 20)
		_ = m.View()

		m2, _ := m.Update(tea.MouseClickMsg{
			X: 60, Y: 8, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)

		prev := ui.MinPickerSplitRatio
		for x := 0; x <= 120; x++ {
			m2, _ = m.Update(tea.MouseMotionMsg{
				X: x, Y: 8, Button: tea.MouseLeft,
			})
			m = m2.(ui.Model)
			r := m.PickerLayoutOptions().SplitRatios["fixed"]
			assert.GreaterOrEqual(t, r, ui.MinPickerSplitRatio)
			assert.LessOrEqual(t, r, ui.MaxPickerSplitRatio)
			// dragging rightward must never step the ratio back; the
			// zero-width edge once snapped to the default and broke this
			assert.GreaterOrEqual(t, r, prev)
			prev = r
		}
	})

	t.Run("split panes clamp at edges", func(t *testing.T) {
		m := fixedPicker(t, 1, 120, 20)
		_ = m.View()

		m2, _ := m.Update(tea.MouseClickMsg{
			X: 60, Y: 8, Button: tea.MouseLeft,
		})
		m = m2.(ui.Model)

		for _, x := range []int{0, 120} {
			m2, _ = m.Update(tea.MouseMotionMsg{
				X: x, Y: 8, Button: tea.MouseLeft,
			})
			m = m2.(ui.Model)
			out := stripANSI(m.View().Content)
			assert.Contains(t, out, "item00")
			assert.Contains(t, out, "CONTENT-00")
		}
	})

	t.Run("drag split is per picker", func(t *testing.T) {
		m := fixedPickers(t, 120, 20)
		m = sendKey(m, 'a')
		m = dragPickerSplit(m, 60, 75)
		opts := m.PickerLayoutOptions()
		assert.InDelta(t, 0.648, opts.SplitRatios["alpha"], 0.001)
		_, ok := opts.SplitRatios["beta"]
		assert.False(t, ok)

		m = sendSpecial(m, tea.KeyEscape)
		m = sendKey(m, 'b')
		m = dragPickerSplit(m, 60, 45)
		opts = m.PickerLayoutOptions()
		assert.InDelta(t, 0.648, opts.SplitRatios["alpha"], 0.001)
		assert.InDelta(t, 0.362, opts.SplitRatios["beta"], 0.001)
	})

	t.Run("tiny picker skips overlay", func(t *testing.T) {
		m := fixedPicker(t, 1, 3, 3)
		out := stripANSI(m.View().Content)
		assert.NotContains(t, out, "fixed")
	})
}

func (s fixedPickerSource) ID() string {
	if s.title != "" {
		return s.title
	}
	return "fixed"
}

func (fixedPickerSource) Columns() []string {
	return []string{"name"}
}
func (fixedPickerSource) MatchColumn() int {
	return 0
}

func (fixedPickerSource) ColumnProportions() []int {
	return []int{1}
}

func (fixedPickerSource) Accept(
	*view.Editor, ui.PickerItem, ui.PickerAcceptAction,
) {
}

func (s fixedPickerSource) Load(
	*view.Editor,
) ([]ui.PickerItem, <-chan ui.PickerItem, ui.StopFunc) {
	return s.items, nil, func() {}
}

func fixedPicker(t *testing.T, n, w, h int) ui.Model {
	t.Helper()
	items := fixedPickerItems(n)
	e := view.NewEditor(t.TempDir())
	km := command.NewKeymaps()
	m := ui.New(e, km)
	src := fixedPickerSource{items: items}
	bindNormalTestAction(
		km, "fixed_picker",
		m.PickerAction(func(*view.Editor) *ui.Picker {
			return ui.NewPicker(e, src)
		}),
		[]command.KeyEvent{char('p')},
	)
	m = resize(m, w, h)
	m = sendKey(m, 'p')
	return m
}

func fixedPickers(t *testing.T, w, h int) ui.Model {
	t.Helper()
	items := fixedPickerItems(30)
	e := view.NewEditor(t.TempDir())
	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(
		km, "alpha_picker",
		m.PickerAction(func(*view.Editor) *ui.Picker {
			return ui.NewPicker(
				e, fixedPickerSource{items: items, title: "alpha"},
			)
		}),
		[]command.KeyEvent{char('a')},
	)
	bindNormalTestAction(
		km, "beta_picker",
		m.PickerAction(func(*view.Editor) *ui.Picker {
			return ui.NewPicker(
				e, fixedPickerSource{items: items, title: "beta"},
			)
		}),
		[]command.KeyEvent{char('b')},
	)
	return resize(m, w, h)
}

func fixedPickerItems(n int) []ui.PickerItem {
	items := make([]ui.PickerItem, n)
	for i := range n {
		body := fmt.Sprintf("CONTENT-%02d", i)
		items[i] = ui.PickerItem{
			Display: fmt.Sprintf("item%02d", i),
			Columns: []string{fmt.Sprintf("item%02d", i)},
			Preview: func(geom.Size) string { return body },
		}
	}
	return items
}

func dragPickerSplit(m ui.Model, from, to int) ui.Model {
	_ = m.View()
	m2, _ := m.Update(tea.MouseClickMsg{
		X: from, Y: 8, Button: tea.MouseLeft,
	})
	m = m2.(ui.Model)
	m2, _ = m.Update(tea.MouseMotionMsg{
		X: to, Y: 8, Button: tea.MouseLeft,
	})
	m = m2.(ui.Model)
	m2, _ = m.Update(tea.MouseReleaseMsg{
		Button: tea.MouseLeft,
	})
	return m2.(ui.Model)
}
