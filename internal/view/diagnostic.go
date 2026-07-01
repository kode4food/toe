package view

import "slices"

type (
	// Diagnostic is a document diagnostic reported by an external provider
	Diagnostic struct {
		Range    DiagnosticRange
		Severity DiagnosticSeverity
		Message  string
		Source   string
		Provider string
	}

	// DiagnosticRange is a character range in a document
	DiagnosticRange struct {
		From int
		To   int
	}

	// DiagnosticCounts groups diagnostics by severity
	DiagnosticCounts struct {
		Errors   int
		Warnings int
		Info     int
		Hints    int
	}

	// DiagnosticSeverity orders diagnostics by user-facing severity
	DiagnosticSeverity int
)

const (
	DiagnosticSeverityHint DiagnosticSeverity = iota + 1
	DiagnosticSeverityInfo
	DiagnosticSeverityWarning
	DiagnosticSeverityError
)

// ReplaceDiagnostics replaces all diagnostics from provider with diags
func (d *Document) ReplaceDiagnostics(provider string, diags []Diagnostic) {
	d.ls.Lock()
	defer d.ls.Unlock()
	out := d.ls.diagnostics[:0]
	for _, diag := range d.ls.diagnostics {
		if diag.Provider != provider {
			out = append(out, diag)
		}
	}
	d.ls.diagnostics = append(out, diags...)
}

// ClearDiagnostics removes all diagnostics from the document
func (d *Document) ClearDiagnostics() {
	d.ls.Lock()
	defer d.ls.Unlock()
	d.ls.diagnostics = nil
}

// Diagnostics returns a snapshot of all current diagnostics
func (d *Document) Diagnostics() []Diagnostic {
	d.ls.RLock()
	defer d.ls.RUnlock()
	return slices.Clone(d.ls.diagnostics)
}

// DiagnosticCounts returns severity counts for all current diagnostics
func (d *Document) DiagnosticCounts() DiagnosticCounts {
	d.ls.RLock()
	defer d.ls.RUnlock()
	var counts DiagnosticCounts
	for _, diag := range d.ls.diagnostics {
		switch diag.Severity {
		case DiagnosticSeverityError:
			counts.Errors++
		case DiagnosticSeverityWarning:
			counts.Warnings++
		case DiagnosticSeverityInfo:
			counts.Info++
		default:
			counts.Hints++
		}
	}
	return counts
}
