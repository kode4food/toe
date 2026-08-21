package motion

import (
	"embed"
	"fmt"
	"slices"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/builtin/kit"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

type jumplistPickerSource struct {
	ui.PickerBase
}

//go:embed i18n/jumplist.*.json
var jumplistFS embed.FS

// JumplistModule returns the jumplist picker command. It is registered
// separately from CursorModule so its position in the space-leader menu can
// be controlled independently of the cursor-motion commands
func JumplistModule(model ui.Model) command.Module {
	return command.Module{
		Translations: i18n.LoadTranslations(jumplistFS),
		Commands: []command.Command{
			{
				Name:      actJumplistPicker,
				DocString: "Open jumplist picker",
				Run:       kit.Runner(model.PickerAction(JumplistPicker)),
				Modes:     command.DocNormalModes,
				Keys:      kit.Leader('j'),
			},
		},
	}
}

// JumplistPicker opens a picker listing the jump history for the focused view
func JumplistPicker(e *view.Editor) *ui.Picker {
	return ui.NewPicker(e, &jumplistPickerSource{
		Ident: "jumplist",
		Label: "Jump List",
		Cols:  []string{""},
	})
}

// Load lists the focused pane's jump history
func (j *jumplistPickerSource) Load(e *view.Editor) ui.PickerLoad {
	v := e.FocusedView()
	if v == nil {
		return ui.PickerLoad{Stop: func() {}}
	}
	jumps := v.Jumps()
	items := make([]*ui.PickerItem, 0, len(jumps))
	var slab ui.PickerItemSlab
	for i, entry := range slices.Backward(jumps) {

		doc := e.Document(entry.DocID)
		if doc == nil {
			continue
		}
		rel := doc.RelativeName(e.Cwd())
		text := doc.Text()
		line, lines := jumpLineRange(text, entry.Selection)
		lbl, sec := ui.PickerNamePath(fmt.Sprintf("%s:%d", rel, line+1))
		items = append(items, slab.Add(ui.PickerItem{
			Display: lbl,
			Columns: []string{lbl},
			SecFrom: sec,
			Location: ui.PickerLocation{
				Target: ui.PickerTarget{ID: entry.DocID},
				Lines:  lines,
			},
			Payload: i,
		}))
	}
	return ui.PickerLoad{Items: items, Stop: func() {}}
}

// Accept jumps to the chosen entry, moving the jump list head onto it
func (j *jumplistPickerSource) Accept(
	e *view.Editor, item *ui.PickerItem, action ui.PickerAcceptAction,
) {
	if index, ok := item.Payload.(int); ok {
		ui.GotoJump(e, index, action)
	}
}

func jumpLineRange(text core.Rope, sel core.Selection) (int, *core.Span) {
	if line, err := sel.Primary().CursorLine(text); err == nil {
		return line, &core.Span{From: line, To: line}
	}
	return 0, nil
}
