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

const (
	statusPickerHintsKey    i18n.Key = "status.pickerHints"
	statusPickerInfoKey     i18n.Key = "status.pickerInfo"
	statusPickerWarningsKey i18n.Key = "status.pickerWarnings"
	statusPickerErrorsKey   i18n.Key = "status.pickerErrors"
)

var (
	diagnosticSeverityIcons = [...]string{
		view.DiagnosticSeverityHint:    "\uea61", // '' - cod-lightbulb
		view.DiagnosticSeverityInfo:    "\uea74", // '' - cod-info
		view.DiagnosticSeverityWarning: "\uea6c", // '' - cod-warning
		view.DiagnosticSeverityError:   "\uea87", // '' - cod-error
	}

	diagnosticSeverityAscii = [...]string{
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
		view.DiagnosticSeverityHint:    statusPickerHintsKey,
		view.DiagnosticSeverityInfo:    statusPickerInfoKey,
		view.DiagnosticSeverityWarning: statusPickerWarningsKey,
		view.DiagnosticSeverityError:   statusPickerErrorsKey,
	}
)

// NewDiagnosticPicker lists diagnostics for the focused document
func NewDiagnosticPicker(e *view.Editor) *ui.Picker {
	return ui.NewPicker(e, &diagnosticPickerSource{
		PickerBase: ui.PickerBase{
			Ident:       "diagnostics",
			Label:       "Diagnostics",
			Cols:        []string{"", ""},
			MatchCol:    1,
			Proportions: []int{0, 1},
		},
	})
}

// NewWorkspaceDiagnosticPicker lists diagnostics for all open documents
func NewWorkspaceDiagnosticPicker(e *view.Editor) *ui.Picker {
	return ui.NewPicker(e, &diagnosticPickerSource{
		PickerBase: ui.PickerBase{
			Ident:       "workspace-diagnostics",
			Label:       "Workspace Diagnostics",
			Cols:        []string{"", ""},
			MatchCol:    1,
			Proportions: []int{0, 1},
		},
		workspace: true,
	})
}

// Load lists every diagnostic across open documents
func (d *diagnosticPickerSource) Load(e *view.Editor) ui.PickerLoad {
	docs := diagnosticPickerDocuments(e, d.workspace)
	items := make([]*ui.PickerItem, 0)
	var slab ui.PickerItemSlab
	for _, doc := range docs {
		for _, diag := range doc.Diagnostics() {
			items = append(items, d.item(&slab, e, doc, diag))
		}
	}
	ui.SortPickerItems(items)
	return ui.PickerLoad{
		Items: append(diagnosticSections(), items...),
		Stop:  func() {},
	}
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
	msg := ui.DiagnosticMessageText(diag.Message)
	lbl := msg
	sec := 0
	if d.workspace {
		at := fmt.Sprintf("%s:%d", name, line+1)
		lbl, sec = ui.PickerTrailingPath(msg, at)
	}
	return slab.Add(ui.PickerItem{
		Group: diagnosticSeverityGroup(diag.Severity),
		Columns: []string{
			diagnosticSeverityIcon(diag.Severity, e.Options().NerdFonts), lbl,
		},
		StyleScopes: []string{diagnosticSeverityScope(diag.Severity), ""},
		SecFrom:     sec,
		SortKey:     fmt.Sprintf("%s:%06d", name, line+1),
		Location: ui.PickerLocation{
			Target: ui.PickerTarget{ID: doc.ID()},
			Lines:  lines,
		},
		Payload: diagnosticPickerPayload{id: doc.ID(), diag: diag},
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
) (int, *core.Span) {
	from, err := text.CharToLine(diag.Range.From)
	if err != nil {
		return 0, nil
	}
	to, err := text.CharToLine(diag.Range.To)
	if err != nil {
		to = from
	}
	return from, &core.Span{From: from, To: to}
}

func diagnosticSelection(diag view.Diagnostic) (core.Selection, error) {
	return core.NewSelection(
		[]core.Range{{
			Anchor: diag.Range.To,
			Head:   diag.Range.From,
		}}, 0,
	)
}

func diagnosticSeverityIcon(sev view.DiagnosticSeverity, nerd bool) string {
	icons := diagnosticSeverityIcons
	if !nerd {
		icons = diagnosticSeverityAscii
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
