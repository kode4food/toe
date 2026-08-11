package ui

import (
	"fmt"
	"sync"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/view"
)

type (
	lspWorkspaceCommandSource struct {
		PickerBase
		commands []string
	}

	lspLocationSource struct {
		PickerBase
		request locationRequest
	}

	// locationRequest fetches the locations a location picker lists
	locationRequest func() ([]view.Location, error)

	lspSymbolSource struct {
		PickerBase
		symbols []view.Symbol
	}

	lspWorkspaceSymbolSource struct {
		PickerBase
		query string
	}
)

func newLSPLocationPicker(e *view.Editor, request locationRequest) *Picker {
	return NewPicker(e, &lspLocationSource{
		PickerBase: PickerBase{
			Ident: "lsp-locations",
			Label: "Locations",
			Cols:  []string{""},
		},
		request: request,
	})
}

func newLSPSymbolPicker(e *view.Editor, symbols []view.Symbol) *Picker {
	return NewPicker(e, &lspSymbolSource{
		PickerBase: PickerBase{
			Ident:       "lsp-symbols",
			Label:       "Document Symbols",
			Cols:        []string{"", ""},
			MatchCol:    1,
			Proportions: []int{0, 1},
		},
		symbols: symbols,
	})
}

func newLSPWorkspaceSymbolPicker(e *view.Editor) *Picker {
	return NewPicker(e, &lspWorkspaceSymbolSource{
		PickerBase: PickerBase{
			Ident:       "lsp-workspace-symbols",
			Label:       "Workspace Symbols",
			Cols:        []string{"", "", ""},
			MatchCol:    1,
			Proportions: []int{0, 0, 1},
		},
	})
}

// Load lists the commands the language servers offer
func (l *lspWorkspaceCommandSource) Load(*view.Editor) PickerLoad {
	items := make([]*PickerItem, 0, len(l.commands))
	var slab PickerItemSlab
	for _, command := range l.commands {
		items = append(items, slab.Add(PickerItem{
			Display: command,
			Columns: []string{command},
			SortKey: command,
			Payload: command,
		}))
	}
	return PickerLoad{Items: items, Stop: func() {}}
}

// Accept executes the chosen server command
func (l *lspWorkspaceCommandSource) Accept(
	e *view.Editor, item *PickerItem, _ PickerAcceptAction,
) {
	name, ok := item.Payload.(string)
	if !ok {
		return
	}
	doc := e.FocusedDocument()
	if doc == nil {
		return
	}
	if ctl := e.LanguageServerController(); ctl != nil {
		_ = ctl.ExecuteWorkspaceCommand(doc, name, nil)
	}
}

// Load streams the request's locations in as they arrive
func (l *lspLocationSource) Load(
	e *view.Editor,
) PickerLoad {
	cwd := e.Cwd()
	ch := make(chan *PickerItem, pickerFeedBatchSize)
	done := make(chan struct{})
	var once sync.Once
	cancel := func() { once.Do(func() { close(done) }) }
	go func() {
		defer close(ch)
		locations, _ := l.request()
		var slab PickerItemSlab
		for _, loc := range locations {
			select {
			case ch <- locationItem(&slab, loc, cwd):
			case <-done:
				return
			}
		}
	}()
	return PickerLoad{Feed: ch, Stop: cancel}
}

// Accept jumps to the chosen location
func (l *lspLocationSource) Accept(
	e *view.Editor, item *PickerItem, action PickerAcceptAction,
) {
	acceptLocation(e, item, action)
}

// Load lists the symbols in the focused document
func (l *lspSymbolSource) Load(
	e *view.Editor,
) PickerLoad {
	nerd := e.Options().NerdFonts
	items := make([]*PickerItem, 0, len(l.symbols))
	var slab PickerItemSlab
	for _, sym := range l.symbols {
		loc := sym.Location
		_, lines := locationLineRange(loc)
		kind := symbolKind(sym.Kind)
		name := symbolName(sym)
		items = append(items, slab.Add(PickerItem{
			Display:     name,
			Columns:     []string{completionKindIcon(kind, nerd), name},
			StyleScopes: []string{completionKindStyleScope(kind), ""},
			SortKey:     sym.Name,
			Location: PickerLocation{
				Target: PickerTarget{Path: loc.Path},
				Lines:  lines,
			},
			Payload: loc,
		}))
	}
	return PickerLoad{Items: items, Stop: func() {}}
}

// Accept jumps to the chosen symbol
func (l *lspSymbolSource) Accept(
	e *view.Editor, item *PickerItem, action PickerAcceptAction,
) {
	acceptLocation(e, item, action)
}

// Search re-queries the servers for a new symbol name
func (l *lspWorkspaceSymbolSource) Search(query string) {
	l.query = query
}

// Load lists workspace symbols matching the current query
func (l *lspWorkspaceSymbolSource) Load(
	e *view.Editor,
) PickerLoad {
	if l.query == "" {
		return PickerLoad{Stop: func() {}}
	}
	doc := e.FocusedDocument()
	ctl := e.LanguageServerController()
	if ctl == nil {
		return PickerLoad{Stop: func() {}}
	}
	symbols, err := ctl.WorkspaceSymbols(doc, l.query)
	if err != nil {
		e.SetStatusMsg(i18n.ErrorText(err))
	}
	items := make([]*PickerItem, 0, len(symbols))
	var slab PickerItemSlab
	for _, sym := range symbols {
		if item, ok := l.item(&slab, e, sym); ok {
			items = append(items, item)
		}
	}
	return PickerLoad{Items: items, Stop: func() {}}
}

// Accept jumps to the chosen symbol
func (l *lspWorkspaceSymbolSource) Accept(
	e *view.Editor, item *PickerItem, action PickerAcceptAction,
) {
	acceptLocation(e, item, action)
}

func (l *lspWorkspaceSymbolSource) item(
	slab *PickerItemSlab, e *view.Editor, sym view.Symbol,
) (*PickerItem, bool) {
	loc := sym.Location
	line, lines := locationLineRange(loc)
	path := view.DocumentRelativeName(view.DocumentRelativeNameArgs{
		Path:    loc.Path,
		BaseDir: e.Cwd(),
	})
	kind := symbolKind(sym.Kind)
	icon := completionKindIcon(kind, e.Options().NerdFonts)
	return slab.Add(PickerItem{
		Display: fmt.Sprintf("%s:%d %s", path, line+1, sym.Name),
		Columns: []string{icon, sym.Name, path},
		StyleScopes: []string{
			completionKindStyleScope(kind), "", "ui.picker.secondary",
		},
		SortKey: sym.Name,
		Location: PickerLocation{
			Target: PickerTarget{Path: loc.Path},
			Lines:  lines,
		},
		Payload: loc,
	}), true
}

// LSPWorkspaceCommandPicker opens commands exposed by language servers
func LSPWorkspaceCommandPicker(e *view.Editor) *Picker {
	ctl := e.LanguageServerController()
	var commands []string
	if doc := e.FocusedDocument(); doc != nil && ctl != nil {
		commands = ctl.WorkspaceCommands(doc)
	}
	return NewPicker(e, &lspWorkspaceCommandSource{
		PickerBase: PickerBase{
			Ident: "lsp-workspace-command",
			Label: "Language Server Commands",
			Cols:  []string{"command"},
		},
		commands: commands,
	})
}

func locationItem(
	slab *PickerItemSlab, loc view.Location, cwd string,
) *PickerItem {
	line, lines := locationLineRange(loc)
	rel := view.DocumentRelativeName(view.DocumentRelativeNameArgs{
		Path:    loc.Path,
		BaseDir: cwd,
	})
	display := fmt.Sprintf("%s:%d", rel, line+1)
	lbl, sec := PickerNamePath(display)
	return slab.Add(PickerItem{
		Display:       lbl,
		Columns:       []string{lbl},
		SortKey:       display,
		SecondaryFrom: sec,
		Location: PickerLocation{
			Target: PickerTarget{Path: loc.Path},
			Lines:  lines,
		},
		Payload: loc,
	})
}

func acceptLocation(
	e *view.Editor, item *PickerItem, action PickerAcceptAction,
) {
	loc, ok := item.Payload.(view.Location)
	if !ok {
		return
	}
	v, ok := AcceptPath(e, loc.Path, action)
	if !ok {
		return
	}
	doc := e.Document(v.DocID())
	if doc == nil {
		return
	}
	sel, ok := locationSelection(doc.Text(), loc)
	if !ok {
		return
	}
	doc.SetSelectionFor(v.ID(), sel)
	AlignAcceptedView(e, v, doc)
}

func locationLineRange(loc view.Location) (int, *core.Span) {
	from := loc.From.Line
	to := max(loc.To.Line, from)
	return from, &core.Span{From: from, To: to}
}

func locationSelection(
	text core.Rope, loc view.Location,
) (core.Selection, bool) {
	r, ok := loc.ResolveRange(text)
	if !ok {
		return core.Selection{}, false
	}
	sel, err := core.NewSelection([]core.Range{r}, 0)
	return sel, err == nil
}

func symbolName(sym view.Symbol) string {
	if sym.Container == "" {
		return sym.Name
	}
	return sym.Container + "." + sym.Name
}

// symbolKind normalizes language-server symbol kind aliases to the kind
// keys shared with completion items
func symbolKind(kind string) string {
	switch kind {
	case "construct":
		return "constructor"
	case "enummem":
		return "enum_member"
	case "typeparam":
		return "type_param"
	}
	return kind
}
