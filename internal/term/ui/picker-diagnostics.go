package ui

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

type (
	diagnosticPickerSource struct {
		pickerMeta
		workspace bool
	}

	diagnosticPickerPayload struct {
		id   view.DocumentId
		diag view.Diagnostic
	}
)

// NewDiagnosticPicker lists diagnostics for the focused document
func NewDiagnosticPicker(e *view.Editor) *Picker {
	return newDiagnosticPicker(e, false)
}

// NewWorkspaceDiagnosticPicker lists diagnostics for all open documents
func NewWorkspaceDiagnosticPicker(e *view.Editor) *Picker {
	return newDiagnosticPicker(e, true)
}

func (d *diagnosticPickerSource) Load(
	e *view.Editor,
) ([]PickerItem, <-chan PickerItem, StopFunc) {
	docs := diagnosticPickerDocuments(e, d.workspace)
	items := make([]PickerItem, 0)
	for _, doc := range docs {
		for _, diag := range doc.Diagnostics() {
			items = append(items, d.item(e, doc, diag))
		}
	}
	sortDiagnosticPickerItems(items)
	return items, nil, func() {}
}

func (d *diagnosticPickerSource) Accept(
	e *view.Editor, item PickerItem, action PickerAcceptAction,
) {
	payload, ok := item.Payload.(diagnosticPickerPayload)
	if !ok {
		return
	}
	v, ok := acceptDocumentID(e, payload.id, action)
	if !ok {
		return
	}
	doc, ok := e.Document(v.DocID())
	if !ok {
		return
	}
	sel, err := diagnosticSelection(payload.diag)
	if err != nil {
		return
	}
	doc.SetSelectionFor(v.ID(), sel)
	alignAcceptedView(e, v, doc)
}

func newDiagnosticPicker(e *view.Editor, workspace bool) *Picker {
	title := "Diagnostics"
	matchColumn := 3
	proportions := []int{0, 0, 0, 1}
	if workspace {
		title = "Workspace diagnostics"
		matchColumn = 4
		proportions = []int{0, 0, 0, 1, 2}
	}
	return NewPicker(e, &diagnosticPickerSource{
		pickerMeta: pickerMeta{
			title:       title,
			columns:     diagnosticPickerColumns(workspace),
			matchColumn: matchColumn,
			proportions: proportions,
		},
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
	doc, ok := e.FocusedDocument()
	if !ok {
		return nil
	}
	return []*view.Document{doc}
}

func (d *diagnosticPickerSource) item(
	e *view.Editor, doc *view.Document, diag view.Diagnostic,
) PickerItem {
	name := doc.RelativeName(e.Cwd())
	line, lines := diagnosticLineRange(doc.Text(), diag)
	display := fmt.Sprintf(
		"%s:%d %s %s", name, line+1, diagnosticSeverityText(diag.Severity),
		diag.Message,
	)
	columns := []string{
		diagnosticSeverityText(diag.Severity), diag.Source, "", diag.Message,
	}
	scopes := []string{diagnosticSeverityScope(diag.Severity), "", "", ""}
	if d.workspace {
		columns = slices.Insert(columns, 3, name)
		scopes = slices.Insert(scopes, 3, "")
	}
	return PickerItem{
		Display:     display,
		Columns:     columns,
		StyleScopes: scopes,
		SortKey:     display,
		Location: PickerLocation{
			Target: PickerTarget{ID: doc.ID()},
			Lines:  lines,
		},
		Payload: diagnosticPickerPayload{id: doc.ID(), diag: diag},
	}
}

func diagnosticPickerColumns(workspace bool) []string {
	cols := []string{"severity", "source", "code", "message"}
	if workspace {
		return slices.Insert(cols, 3, "path")
	}
	return cols
}

func sortDiagnosticPickerItems(items []PickerItem) {
	slices.SortStableFunc(items, func(a, b PickerItem) int {
		aPayload, aOK := a.Payload.(diagnosticPickerPayload)
		bPayload, bOK := b.Payload.(diagnosticPickerPayload)
		if !aOK || !bOK {
			return cmp.Compare(a.SortKey, b.SortKey)
		}
		if c := cmp.Compare(
			bPayload.diag.Severity, aPayload.diag.Severity,
		); c != 0 {
			return c
		}
		return cmp.Compare(a.SortKey, b.SortKey)
	})
}

func diagnosticLineRange(
	text core.Rope, diag view.Diagnostic,
) (int, *PickerLineRange) {
	from, err := text.CharToLine(diag.Range.From)
	if err != nil {
		return 0, nil
	}
	to, err := text.CharToLine(diag.Range.To)
	if err != nil {
		to = from
	}
	return from, &PickerLineRange{From: from, To: to}
}

func diagnosticSelection(diag view.Diagnostic) (core.Selection, error) {
	return core.NewSelection(
		[]core.Range{core.NewRange(diag.Range.To, diag.Range.From)}, 0,
	)
}

func diagnosticSeverityText(sev view.DiagnosticSeverity) string {
	switch sev {
	case view.DiagnosticSeverityError:
		return "ERROR"
	case view.DiagnosticSeverityWarning:
		return "WARN"
	case view.DiagnosticSeverityInfo:
		return "INFO"
	default:
		return "HINT"
	}
}

func diagnosticSeverityScope(sev view.DiagnosticSeverity) string {
	switch sev {
	case view.DiagnosticSeverityError:
		return "error"
	case view.DiagnosticSeverityWarning:
		return "warning"
	case view.DiagnosticSeverityInfo:
		return "info"
	default:
		return "hint"
	}
}
