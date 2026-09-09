package ui_test

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

type sizingPickerSource struct {
	ui.PickerBase
	items []*ui.PickerItem
}

func TestPickerColumnSpacing(t *testing.T) {
	for _, hidden := range []int{0, 1, 3} {
		t.Run(fmt.Sprintf("hidden=%d", hidden), func(t *testing.T) {
			cols := make([]string, hidden+1)
			cols[hidden] = strings.Repeat("x", 100)
			m := sizingPickerModel(t, []*ui.PickerItem{{
				Display: cols[hidden],
				Columns: cols,
			}})
			row := stripANSI(rawLineContaining(t, m.View().Content, " > "))
			// The name fills the row up to its single trailing margin.
			assert.Contains(t, row, strings.Repeat("x", 39)+" │")
		})
	}
}

// BenchmarkPickerColumns repaints a multi-column list while moving selection
func BenchmarkPickerColumns(b *testing.B) {
	items := make([]*ui.PickerItem, 1000)
	for i := range items {
		name := fmt.Sprintf("item-%04d", i)
		items[i] = &ui.PickerItem{
			Display: name,
			Columns: []string{"", name, "internal/term/ui"},
		}
	}
	m := sizingPickerModel(b, items)
	b.ReportAllocs()
	for b.Loop() {
		m = sendSpecial(m, tea.KeyDown)
		_ = m.View().Content
		m = sendSpecial(m, tea.KeyUp)
		_ = m.View().Content
	}
}

func (s *sizingPickerSource) Load() ui.PickerLoad {
	return ui.PickerLoad{Items: s.items, Stop: func() {}}
}

func (*sizingPickerSource) Accept(*ui.PickerItem, ui.PickerAcceptAction) {}

func (*sizingPickerSource) SkipPreview() {}

func sizingPickerModel(t testing.TB, items []*ui.PickerItem) ui.Model {
	t.Helper()
	e := view.NewEditor(t.TempDir())
	e.Options().NerdFonts = false
	km := command.NewKeymaps()
	m := ui.New(e, km)
	bindNormalTestAction(km, "sizing_picker", m.PickerAction(
		func(e *view.Editor) *ui.Picker {
			return ui.NewPicker(e, &sizingPickerSource{
				Ident: "sizing",
				Cols:  make([]string, len(items[0].Columns)),
				items: items,
			})
		},
	), []command.KeyEvent{char('p')})
	return sendKey(resize(m, 50, 20), 'p')
}
