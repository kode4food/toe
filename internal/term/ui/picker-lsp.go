package ui

import (
	"fmt"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

type (
	lspWorkspaceCommandSource struct {
		pickerMeta
		commands []string
	}

	lspLocationSource struct {
		pickerMeta
		locations []view.Location
	}

	lspSymbolSource struct {
		pickerMeta
		symbols []view.Symbol
	}
)

// LSPWorkspaceCommandPicker opens commands exposed by language servers
func LSPWorkspaceCommandPicker(e *view.Editor) *Picker {
	ctl := e.LanguageServerController()
	var commands []string
	if doc, ok := e.FocusedDocument(); ok && ctl != nil {
		commands = ctl.WorkspaceCommands(doc)
	}
	return NewPicker(e, &lspWorkspaceCommandSource{
		pickerMeta: pickerMeta{
			title:   "LSP workspace command",
			columns: []string{"command"},
		},
		commands: commands,
	})
}

func newLSPLocationPicker(
	e *view.Editor, locations []view.Location,
) *Picker {
	return NewPicker(e, &lspLocationSource{
		pickerMeta: pickerMeta{
			title:   "LSP locations",
			columns: []string{"location"},
		},
		locations: locations,
	})
}

func newLSPSymbolPicker(e *view.Editor, symbols []view.Symbol) *Picker {
	return NewPicker(e, &lspSymbolSource{
		pickerMeta: pickerMeta{
			title:   "LSP symbols",
			columns: []string{"kind", "name", "container"},
			primary: 1,
		},
		symbols: symbols,
	})
}

func (l *lspWorkspaceCommandSource) Load(
	_ *view.Editor,
) ([]PickerItem, <-chan PickerItem, StopFunc) {
	items := make([]PickerItem, 0, len(l.commands))
	for _, command := range l.commands {
		items = append(items, PickerItem{
			Display: command,
			Columns: []string{command},
			SortKey: command,
			Payload: command,
		})
	}
	return items, nil, func() {}
}

func (l *lspWorkspaceCommandSource) Match(
	query string, item PickerItem,
) (int, []int, bool) {
	return fuzzyMatchItem(query, item, l.Columns(), l.Primary())
}

func (l *lspWorkspaceCommandSource) Accept(
	e *view.Editor, item PickerItem, _ PickerAcceptAction,
) {
	name, ok := item.Payload.(string)
	if !ok {
		return
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	if ctl := e.LanguageServerController(); ctl != nil {
		_ = ctl.ExecuteWorkspaceCommand(doc, name, nil)
	}
}

func (l *lspLocationSource) Load(
	e *view.Editor,
) ([]PickerItem, <-chan PickerItem, StopFunc) {
	items := make([]PickerItem, 0, len(l.locations))
	for _, loc := range l.locations {
		doc, err := e.SwitchOrOpenDoc(loc.Path)
		if err != nil {
			continue
		}
		line, lines := locationLineRange(doc.Text(), loc)
		name := doc.RelativeName(e.Cwd())
		display := fmt.Sprintf("%s:%d", name, line+1)
		items = append(items, PickerItem{
			Display: display,
			Columns: []string{display},
			SortKey: display,
			Location: PickerLocation{
				Target: PickerTarget{Path: loc.Path},
				Lines:  lines,
			},
			Payload: loc,
		})
	}
	return items, nil, func() {}
}

func (l *lspLocationSource) Match(
	query string, item PickerItem,
) (int, []int, bool) {
	return fuzzyMatchItem(query, item, l.Columns(), l.Primary())
}

func (l *lspLocationSource) Accept(
	e *view.Editor, item PickerItem, action PickerAcceptAction,
) {
	acceptLocation(e, item, action)
}

func (l *lspSymbolSource) Load(
	e *view.Editor,
) ([]PickerItem, <-chan PickerItem, StopFunc) {
	items := make([]PickerItem, 0, len(l.symbols))
	for _, sym := range l.symbols {
		loc := sym.Location
		doc, err := e.SwitchOrOpenDoc(loc.Path)
		if err != nil {
			continue
		}
		line, lines := locationLineRange(doc.Text(), loc)
		name := doc.RelativeName(e.Cwd())
		display := fmt.Sprintf("%s:%d %s", name, line+1, sym.Name)
		items = append(items, PickerItem{
			Display: display,
			Columns: []string{sym.Kind, sym.Name, sym.Container},
			SortKey: sym.Name,
			Location: PickerLocation{
				Target: PickerTarget{Path: loc.Path},
				Lines:  lines,
			},
			Payload: loc,
		})
	}
	return items, nil, func() {}
}

func (l *lspSymbolSource) Match(
	query string, item PickerItem,
) (int, []int, bool) {
	return fuzzyMatchItem(query, item, l.Columns(), l.Primary())
}

func (l *lspSymbolSource) Accept(
	e *view.Editor, item PickerItem, action PickerAcceptAction,
) {
	acceptLocation(e, item, action)
}

func acceptLocation(
	e *view.Editor, item PickerItem, action PickerAcceptAction,
) {
	loc, ok := item.Payload.(view.Location)
	if !ok {
		return
	}
	v, ok := acceptPath(e, loc.Path, action)
	if !ok {
		return
	}
	doc, ok := e.Document(v.DocID())
	if !ok {
		return
	}
	sel, err := locationSelection(loc)
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), sel)
	alignAcceptedView(e, v, doc)
}

func locationLineRange(
	text core.Rope, loc view.Location,
) (int, *PickerLineRange) {
	from, err := text.CharToLine(loc.From)
	if err != nil {
		return 0, nil
	}
	to, err := text.CharToLine(loc.To)
	if err != nil {
		to = from
	}
	return from, &PickerLineRange{From: from, To: to}
}

func locationSelection(loc view.Location) (core.Selection, error) {
	return core.NewSelection(
		[]core.Range{core.NewRange(loc.To, loc.From)}, 0,
	)
}
