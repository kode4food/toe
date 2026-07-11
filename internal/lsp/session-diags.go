package lsp

import (
	"context"
	"errors"
	"fmt"

	"go.lsp.dev/protocol"

	"github.com/kode4food/toe/internal/view"
)

// PullDiagnostics requests fresh diagnostics from document servers
func (s *Session) PullDiagnostics(doc *view.Document) error {
	snap, ok := SnapshotDocument(doc)
	if !ok {
		return nil
	}
	clients := s.clientsForDocument(doc)
	if len(clients) == 0 {
		return ErrNoLanguageServer
	}
	if id := s.servers.languageID(doc.Lang()); id != "" {
		snap.LanguageID = id
	}
	var err error
	for _, client := range clients {
		if !client.SupportsFeature(FeaturePullDiagnostics) {
			continue
		}
		provider := client.Name()
		report, sent, e := client.DocumentDiagnostics(
			s.ctx, snap, s.docs.previousDiagnosticID(doc.ID(), provider),
		)
		if e != nil {
			err = errors.Join(err, fmt.Errorf("%w: %s", e, provider))
			continue
		}
		if sent {
			s.applyDiagnosticReport(doc, provider, client, report)
		}
	}
	return err
}

func (h *clientHandler) PublishDiagnostics(
	_ context.Context, params *protocol.PublishDiagnosticsParams,
) error {
	return h.session.publishDiagnostics(h.name, params)
}

func (h *clientHandler) DiagnosticRefresh(context.Context) error {
	h.session.pullAllDiagnosticsAsync()
	return nil
}

func (s *Session) publishDiagnostics(
	provider string, params *protocol.PublishDiagnosticsParams,
) error {
	if s.editor == nil {
		return nil
	}
	path := params.URI.FsPath()
	var target *view.Document
	for _, doc := range s.editor.AllDocuments() {
		if doc.Path() == path {
			target = doc
			break
		}
	}
	if target == nil {
		return nil
	}
	if version, ok := params.Version.Get(); ok {
		if int(version) != target.Revision() {
			return nil
		}
	}
	target.ReplaceDiagnostics(
		provider,
		s.convertDiagnostics(
			provider, target, params.Diagnostics, s.offsetForProvider(provider),
		),
	)
	return nil
}

func (s *Session) applyDiagnosticReport(
	doc *view.Document, provider string, client *Client,
	report protocol.DocumentDiagnosticReport,
) {
	switch r := report.(type) {
	case *protocol.RelatedFullDocumentDiagnosticReport:
		doc.ReplaceDiagnostics(
			provider,
			s.convertDiagnostics(
				provider, doc, r.Items, client.OffsetEncoding(),
			),
		)
		s.docs.setPreviousDiagnosticID(doc.ID(), provider, r.ResultID)
	case *protocol.RelatedUnchangedDocumentDiagnosticReport:
		s.docs.setPreviousDiagnosticID(doc.ID(), provider, &r.ResultID)
	}
}

func (s *Session) pullDiagnosticsAsync(doc *view.Document) {
	go func() {
		_ = s.PullDiagnostics(doc)
	}()
}

func (s *Session) pullAllDiagnosticsAsync() {
	if s.editor == nil {
		return
	}
	for _, doc := range s.editor.AllDocuments() {
		if !doc.Loaded() {
			continue
		}
		s.pullDiagnosticsAsync(doc)
	}
}

func (s *Session) convertDiagnostics(
	provider string, doc *view.Document, diags []protocol.Diagnostic,
	encoding protocol.PositionEncodingKind,
) []view.Diagnostic {
	out := make([]view.Diagnostic, 0, len(diags))
	for _, diag := range diags {
		dr, ok := lspRangeToChars(doc, diag.Range, encoding)
		if !ok {
			continue
		}
		source, _ := diag.Source.Get()
		out = append(out, view.Diagnostic{
			Range: view.DiagnosticRange{
				From: dr.From(),
				To:   dr.To(),
			},
			Severity: diagnosticSeverity(diag.Severity),
			Message:  markupText(diag.Message),
			Source:   source,
			Provider: provider,
		})
	}
	return out
}

func diagnosticSeverity(
	severity protocol.DiagnosticSeverity,
) view.DiagnosticSeverity {
	switch severity {
	case protocol.DiagnosticSeverityError:
		return view.DiagnosticSeverityError
	case protocol.DiagnosticSeverityWarning:
		return view.DiagnosticSeverityWarning
	case protocol.DiagnosticSeverityInformation:
		return view.DiagnosticSeverityInfo
	default:
		return view.DiagnosticSeverityHint
	}
}
