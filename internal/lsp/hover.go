package lsp

import (
	"context"
	"errors"
	"strings"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
	"go.lsp.dev/protocol"
)

func (c *Client) Hover(
	ctx context.Context, doc DocumentSnapshot, pos int,
) (*protocol.Hover, bool, error) {
	if !c.SupportsFeature(FeatureHover) {
		return nil, false, nil
	}
	lspPos, err := lspPosition(
		core.NewRope(doc.Text), pos, c.OffsetEncoding(),
	)
	if err != nil {
		return nil, false, err
	}
	params := &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: doc.URI},
			Position:     lspPos,
		},
	}
	ctx, cancel := c.requestContext(ctx)
	defer cancel()
	hover, err := c.server.Hover(ctx, params)
	if err != nil {
		return nil, true, err
	}
	return hover, true, nil
}

// Hover returns the hover text for the symbol at the cursor position
func (s *Session) Hover(
	doc *view.Document, viewID view.Id,
) (string, error) {
	snap, ok := SnapshotDocument(doc)
	if !ok {
		return "", nil
	}
	sel := doc.SelectionFor(viewID)
	pos := sel.Primary().Cursor(doc.Text())
	clients := s.clientsForDocument(doc)
	out := []string{}
	var err error
	for _, client := range clients {
		hover, sent, e := client.Hover(s.ctx, snap, pos)
		if e != nil {
			err = errors.Join(err, s.completionError(client, e))
			continue
		}
		if sent && hover != nil {
			text := hoverContentsText(hover.Contents)
			if text != "" {
				out = append(out, text)
			}
		}
	}
	return strings.Join(out, "\n\n"), err
}

func hoverContentsText(contents protocol.HoverContents) string {
	switch v := contents.(type) {
	case protocol.String:
		return string(v)
	case *protocol.MarkupContent:
		if v == nil {
			return ""
		}
		return v.Value
	default:
		return ""
	}
}
