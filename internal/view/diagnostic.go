package view

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

func (d *Document) ReplaceDiagnostics(provider string, diags []Diagnostic) {
	out := d.diagnostics[:0]
	for _, diag := range d.diagnostics {
		if diag.Provider != provider {
			out = append(out, diag)
		}
	}
	d.diagnostics = append(out, diags...)
}

func (d *Document) Diagnostics() []Diagnostic {
	out := make([]Diagnostic, len(d.diagnostics))
	copy(out, d.diagnostics)
	return out
}

func (d *Document) DiagnosticCounts() DiagnosticCounts {
	var counts DiagnosticCounts
	for _, diag := range d.diagnostics {
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
