package motion

import (
	"fmt"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/term/builtin/kit"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

type jumplistPickerSource struct {
	ui.PickerBase
}

// JumplistModule returns the jumplist picker command. It is registered
// separately from CursorModule so its position in the space-leader menu can
// be controlled independently of the cursor-motion commands
func JumplistModule(model ui.Model) command.Module {
	return command.Module{
		Commands: []command.Command{
			{
				Name:      actJumplistPicker,
				DocString: "Open jumplist picker",
				Run:       kit.Continuation(model.PickerAction(JumplistPicker)),
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Leader('j'),
			},
		},
	}
}

// JumplistPicker opens a picker listing the jump history for the focused view
func JumplistPicker(e *view.Editor) *ui.Picker {
	return ui.NewPicker(e, &jumplistPickerSource{
		PickerBase: ui.NewPickerBase("jumplist", []string{"path"}, 0, []int{1}),
	})
}

func (j *jumplistPickerSource) Load(
	e *view.Editor,
) ([]ui.PickerItem, <-chan ui.PickerItem, ui.StopFunc) {
	v, ok := e.FocusedView()
	if !ok {
		return nil, nil, func() {}
	}
	jumps := v.Jumps()
	items := make([]ui.PickerItem, 0, len(jumps))
	for i := len(jumps) - 1; i >= 0; i-- {
		entry := jumps[i]
		doc, ok := e.Document(entry.DocID)
		if !ok {
			continue
		}
		name := doc.RelativeName(e.Cwd())
		text := doc.Text()
		line, lines := jumpLineRange(text, entry.Selection)
		display := fmt.Sprintf("%s:%d", name, line+1)
		items = append(items, ui.PickerItem{
			Display: display,
			Columns: []string{display},
			Location: ui.PickerLocation{
				Target: ui.PickerTarget{ID: entry.DocID},
				Lines:  lines,
			},
			Payload: entry,
		})
	}
	return items, nil, func() {}
}

func (j *jumplistPickerSource) Accept(
	e *view.Editor, item ui.PickerItem, action ui.PickerAcceptAction,
) {
	entry, ok := item.Payload.(view.JumpEntry)
	if !ok {
		return
	}
	v, ok := ui.AcceptDocumentID(e, entry.DocID, action)
	if !ok {
		return
	}
	if doc, ok := e.Document(v.DocID()); ok {
		doc.SetSelectionFor(v.ID(), jumpSelection(entry))
		ui.AlignAcceptedView(e, v, doc)
	}
}

func jumpSelection(j view.JumpEntry) core.Selection {
	return j.Selection
}

func jumpLineRange(
	text core.Rope, sel core.Selection,
) (int, *ui.PickerLineRange) {
	line, err := sel.Primary().CursorLine(text)
	if err != nil {
		return 0, nil
	}
	return line, &ui.PickerLineRange{From: line, To: line}
}
