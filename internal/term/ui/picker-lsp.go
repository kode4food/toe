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

	lspWorkspaceSymbolSource struct {
		pickerMeta
		query string
	}
)

func newLSPLocationPicker(e *view.Editor, locations []view.Location) *Picker {
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
			title:       "LSP symbols",
			columns:     []string{"name"},
			matchColumn: 0,
		},
		symbols: symbols,
	})
}

func newLSPWorkspaceSymbolPicker(e *view.Editor) *Picker {
	return NewPicker(e, &lspWorkspaceSymbolSource{
		pickerMeta: pickerMeta{
			title:       "LSP workspace symbols",
			columns:     []string{"name", "path"},
			matchColumn: 0,
			proportions: []int{0, 1},
		},
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
		doc, err := e.PeekDoc(loc.Path)
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
		doc, err := e.PeekDoc(loc.Path)
		if err != nil {
			continue
		}
		_, lines := locationLineRange(doc.Text(), loc)
		display := symbolDisplayName(sym.Kind, symbolName(sym))
		items = append(items, PickerItem{
			Display: display,
			Columns: []string{display},
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

func (l *lspSymbolSource) Accept(
	e *view.Editor, item PickerItem, action PickerAcceptAction,
) {
	acceptLocation(e, item, action)
}

func (l *lspWorkspaceSymbolSource) Search(query string) {
	l.query = query
}

func (l *lspWorkspaceSymbolSource) Load(
	e *view.Editor,
) ([]PickerItem, <-chan PickerItem, StopFunc) {
	if l.query == "" {
		return nil, nil, func() {}
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return nil, nil, func() {}
	}
	ctl := e.LanguageServerController()
	if ctl == nil {
		return nil, nil, func() {}
	}
	symbols, err := ctl.WorkspaceSymbols(doc, l.query)
	if err != nil {
		e.SetStatusMsg(err.Error())
		return nil, nil, func() {}
	}
	items := make([]PickerItem, 0, len(symbols))
	for _, sym := range symbols {
		item, ok := l.item(e, sym)
		if ok {
			items = append(items, item)
		}
	}
	return items, nil, func() {}
}

func (l *lspWorkspaceSymbolSource) item(
	e *view.Editor, sym view.Symbol,
) (PickerItem, bool) {
	loc := sym.Location
	doc, err := e.PeekDoc(loc.Path)
	if err != nil {
		return PickerItem{}, false
	}
	line, lines := locationLineRange(doc.Text(), loc)
	name := doc.RelativeName(e.Cwd())
	display := fmt.Sprintf("%s:%d %s", name, line+1, sym.Name)
	return PickerItem{
		Display: display,
		Columns: []string{symbolDisplayName(sym.Kind, sym.Name), name},
		SortKey: sym.Name,
		Location: PickerLocation{
			Target: PickerTarget{Path: loc.Path},
			Lines:  lines,
		},
		Payload: loc,
	}, true
}

func (l *lspWorkspaceSymbolSource) Accept(
	e *view.Editor, item PickerItem, action PickerAcceptAction,
) {
	acceptLocation(e, item, action)
}

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

func symbolName(sym view.Symbol) string {
	if sym.Container == "" {
		return sym.Name
	}
	return sym.Container + "." + sym.Name
}

func symbolDisplayName(kind, name string) string {
	if icon := symbolKindIcon(kind); icon != "" {
		return icon + " " + name
	}
	return name
}

func symbolKindIcon(kind string) string {
	switch kind {
	case "construct":
		kind = "constructor"
	case "enummem":
		kind = "enum_member"
	case "typeparam":
		kind = "type_param"
	}
	return completionKindCodicon(kind)
}
