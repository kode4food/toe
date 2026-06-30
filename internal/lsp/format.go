package lsp

import (
	"context"
	"errors"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
	"go.lsp.dev/protocol"
)

// FormatDocument requests whole-document formatting edits
func (c *Client) FormatDocument(
	ctx context.Context, doc DocumentSnapshot, opts protocol.FormattingOptions,
) ([]protocol.TextEdit, bool, error) {
	if !c.SupportsFeature(FeatureFormat) {
		return nil, false, nil
	}
	params := &protocol.DocumentFormattingParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: doc.URI},
		Options:      opts,
	}
	ctx, cancel := c.requestContext(ctx)
	defer cancel()
	edits, err := c.server.Formatting(ctx, params)
	if err != nil {
		return nil, true, err
	}
	return edits, true, nil
}

// FormatRange requests range-formatting edits
func (c *Client) FormatRange(
	ctx context.Context, doc DocumentSnapshot, r core.Range,
	opts protocol.FormattingOptions,
) ([]protocol.TextEdit, bool, error) {
	if !c.SupportsFeature(FeatureRangeFormat) {
		return nil, false, nil
	}
	lspRange, err := lspRange(
		core.NewRope(doc.Text), r, c.OffsetEncoding(),
	)
	if err != nil {
		return nil, false, err
	}
	params := &protocol.DocumentRangeFormattingParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: doc.URI},
		Range:        lspRange,
		Options:      opts,
	}
	ctx, cancel := c.requestContext(ctx)
	defer cancel()
	edits, err := c.server.RangeFormatting(ctx, params)
	if err != nil {
		return nil, true, err
	}
	return edits, true, nil
}

// FormatDocument applies whole-document formatting from a language server
func (s *Session) FormatDocument(doc *view.Document, _ view.Id) error {
	return s.formatDocument(doc, nil)
}

// FormatSelection applies range formatting for the current primary selection
func (s *Session) FormatSelection(doc *view.Document, viewID view.Id) error {
	sel := doc.SelectionFor(viewID)
	if len(sel.Ranges()) != 1 {
		return ErrFormatSelection
	}
	r := sel.Primary()
	return s.formatDocument(doc, &r)
}

func (s *Session) formatDocument(
	doc *view.Document, r *core.Range,
) error {
	if s.editor == nil {
		return ErrNoLanguageServer
	}
	snap, ok := SnapshotDocument(doc)
	if !ok {
		return nil
	}
	clients := s.clientsForDocument(doc)
	if len(clients) == 0 {
		return ErrNoLanguageServer
	}
	opts := formattingOptions(doc)
	var err error
	for _, client := range clients {
		edits, sent, e := formatRequest(client, s.ctx, snap, r, opts)
		if e != nil {
			err = errors.Join(err, s.completionError(client, e))
			continue
		}
		if !sent {
			continue
		}
		return errors.Join(
			err,
			s.applyFormattingEdits(doc, edits, client.OffsetEncoding()),
		)
	}
	if err != nil {
		return err
	}
	return ErrNoLanguageServer
}

func formatRequest(
	client *Client, ctx context.Context, snap DocumentSnapshot,
	r *core.Range, opts protocol.FormattingOptions,
) ([]protocol.TextEdit, bool, error) {
	if r == nil {
		return client.FormatDocument(ctx, snap, opts)
	}
	return client.FormatRange(ctx, snap, *r, opts)
}

func (s *Session) applyFormattingEdits(
	doc *view.Document, edits []protocol.TextEdit,
	encoding protocol.PositionEncodingKind,
) error {
	if len(edits) == 0 {
		return nil
	}
	changes, err := textEditsToChanges(doc, edits, encoding)
	if err != nil {
		return err
	}
	edit := workspaceDocumentEdit{doc: doc, changes: changes}
	return s.applyWorkspaceDocumentEdit(edit)
}

func formattingOptions(doc *view.Document) protocol.FormattingOptions {
	return protocol.FormattingOptions{
		TabSize:      uint32(doc.TabWidth()),
		InsertSpaces: !doc.IndentStyle().IsTabs(),
	}
}
