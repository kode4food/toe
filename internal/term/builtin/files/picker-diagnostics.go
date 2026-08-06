package files

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

type (
	diagnosticPickerSource struct {
		ui.PickerBase
		workspace bool
	}

	diagnosticPickerPayload struct {
		id   view.DocumentId
		diag view.Diagnostic
	}
)

var (
	diagnosticSeverityIcons = [...]string{
		view.DiagnosticSeverityHint:    "\uea61", // '' - cod-lightbulb
		view.DiagnosticSeverityInfo:    "\uea74", // '' - cod-info
		view.DiagnosticSeverityWarning: "\uea6c", // '' - cod-warning
		view.DiagnosticSeverityError:   "\uea87", // '' - cod-error
	}

	diagnosticSeverityASCII = [...]string{
		view.DiagnosticSeverityHint:    "H",
		view.DiagnosticSeverityInfo:    "I",
		view.DiagnosticSeverityWarning: "W",
		view.DiagnosticSeverityError:   "E",
	}

	diagnosticSeverityScopes = [...]string{
		view.DiagnosticSeverityHint:    "hint",
		view.DiagnosticSeverityInfo:    "info",
		view.DiagnosticSeverityWarning: "warning",
		view.DiagnosticSeverityError:   "error",
	}

	diagnosticSeverityLabels = [...]i18n.Key{
		view.DiagnosticSeverityHint:    i18n.StatusPickerHints,
		view.DiagnosticSeverityInfo:    i18n.StatusPickerInfo,
		view.DiagnosticSeverityWarning: i18n.StatusPickerWarnings,
		view.DiagnosticSeverityError:   i18n.StatusPickerErrors,
	}
)

// NewDiagnosticPicker lists diagnostics for the focused document
func NewDiagnosticPicker(e *view.Editor) *ui.Picker {
	return newDiagnosticPicker(e, false)
}

// NewWorkspaceDiagnosticPicker lists diagnostics for all open documents
func NewWorkspaceDiagnosticPicker(e *view.Editor) *ui.Picker {
	return newDiagnosticPicker(e, true)
}

// Load lists every diagnostic across open documents
func (d *diagnosticPickerSource) Load(
	e *view.Editor,
) ([]*ui.PickerItem, <-chan *ui.PickerItem, ui.StopFunc) {
	docs := diagnosticPickerDocuments(e, d.workspace)
	items := make([]*ui.PickerItem, 0)
	var slab ui.PickerItemSlab
	for _, doc := range docs {
		for _, diag := range doc.Diagnostics() {
			items = append(items, d.item(&slab, e, doc, diag))
		}
	}
	ui.SortPickerItems(items)
	return append(diagnosticSections(), items...), nil, func() {}
}

// Accept jumps to the chosen diagnostic
func (d *diagnosticPickerSource) Accept(
	e *view.Editor, item *ui.PickerItem, action ui.PickerAcceptAction,
) {
	payload, ok := item.Payload.(diagnosticPickerPayload)
	if !ok {
		return
	}
	v := ui.AcceptDocumentID(e, payload.id, action)
	if v == nil {
		return
	}
	doc := e.Document(v.DocID())
	if doc == nil {
		return
	}
	sel, err := diagnosticSelection(payload.diag)
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), sel)
	ui.AlignAcceptedView(e, v, doc)
}

func (d *diagnosticPickerSource) item(
	slab *ui.PickerItemSlab, e *view.Editor, doc *view.Document,
	diag view.Diagnostic,
) *ui.PickerItem {
	name := doc.RelativeName(e.Cwd())
	line, lines := diagnosticLineRange(doc.Text(), diag)
	display := fmt.Sprintf("%s:%d %s", name, line+1, diag.Message)
	columns := []string{
		diagnosticSeverityIcon(diag.Severity, e.Options().NerdFonts),
		diag.Message,
	}
	scopes := []string{diagnosticSeverityScope(diag.Severity), ""}
	if d.workspace {
		columns = append(columns, name)
		scopes = append(scopes, "ui.picker.secondary")
	}
	return slab.Add(ui.PickerItem{
		Display:     display,
		Group:       diagnosticSeverityGroup(diag.Severity),
		Columns:     columns,
		StyleScopes: scopes,
		SortKey:     display,
		Location: ui.PickerLocation{
			Target: ui.PickerTarget{ID: doc.ID()},
			Lines:  lines,
		},
		Payload: diagnosticPickerPayload{id: doc.ID(), diag: diag},
	})
}

func newDiagnosticPicker(e *view.Editor, workspace bool) *ui.Picker {
	id := "diagnostics"
	columns := []string{"", ""}
	matchColumn := 1
	proportions := []int{0, 1}
	if workspace {
		id = "workspace-diagnostics"
		columns = []string{"", "", ""}
		proportions = []int{0, 2, 1}
	}
	return ui.NewPicker(e, &diagnosticPickerSource{
		PickerBase: ui.NewPickerBase(
			id, columns, matchColumn, proportions,
		),
		workspace: workspace,
	})
}

func diagnosticPickerDocuments(
	e *view.Editor, workspace bool,
) []*view.Document {
	if workspace {
		docs := e.AllDocuments()
		slices.SortStableFunc(docs, func(a, b *view.Document) int {
			return cmp.Compare(a.RelativeName(e.Cwd()), b.RelativeName(e.Cwd()))
		})
		return docs
	}
	if doc := e.FocusedDocument(); doc != nil {
		return []*view.Document{doc}
	}
	return nil
}

func diagnosticLineRange(
	text core.Rope, diag view.Diagnostic,
) (int, *ui.PickerLineRange) {
	from, err := text.CharToLine(diag.Range.From)
	if err != nil {
		return 0, nil
	}
	to, err := text.CharToLine(diag.Range.To)
	if err != nil {
		to = from
	}
	return from, &ui.PickerLineRange{From: from, To: to}
}

func diagnosticSelection(diag view.Diagnostic) (core.Selection, error) {
	return core.NewSelection(
		[]core.Range{core.NewRange(diag.Range.To, diag.Range.From)}, 0,
	)
}

func diagnosticSeverityIcon(sev view.DiagnosticSeverity, nerd bool) string {
	icons := diagnosticSeverityIcons
	if !nerd {
		icons = diagnosticSeverityASCII
	}
	if sev <= 0 || int(sev) >= len(icons) {
		return icons[view.DiagnosticSeverityHint]
	}
	return icons[sev]
}

func diagnosticSections() []*ui.PickerItem {
	out := make([]*ui.PickerItem, 0, view.DiagnosticSeverityError)
	for sev := view.DiagnosticSeverityError; sev > 0; sev-- {
		out = append(out, &ui.PickerItem{
			Display: i18n.Text(diagnosticSeverityLabels[sev]),
			Group:   diagnosticSeverityGroup(sev),
			Section: true,
		})
	}
	return out
}

func diagnosticSeverityGroup(sev view.DiagnosticSeverity) int {
	if sev <= 0 || int(sev) >= len(diagnosticSeverityLabels) {
		sev = view.DiagnosticSeverityHint
	}
	return int(view.DiagnosticSeverityError - sev)
}

func diagnosticSeverityScope(sev view.DiagnosticSeverity) string {
	if sev <= 0 || int(sev) >= len(diagnosticSeverityScopes) {
		return "hint"
	}
	return diagnosticSeverityScopes[sev]
}
