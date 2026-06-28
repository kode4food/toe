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
	//lint:ignore SA1019 LSP hover contents still permit MarkedString variants
	case *protocol.MarkedStringWithLanguage:
		return markedStringWithLanguageText(v)
	case protocol.MarkedStringSlice:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			if text := markedStringText(item); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n\n")
	default:
		return ""
	}
}

//lint:ignore SA1019 LSP hover contents still permit MarkedString variants
func markedStringText(s protocol.MarkedString) string {
	switch v := s.(type) {
	case protocol.String:
		return string(v)
	//lint:ignore SA1019 LSP hover contents still permit MarkedString variants
	case *protocol.MarkedStringWithLanguage:
		return markedStringWithLanguageText(v)
	default:
		return ""
	}
}

//lint:ignore SA1019 LSP hover contents still permit MarkedString variants
func markedStringWithLanguageText(s *protocol.MarkedStringWithLanguage) string {
	if s == nil {
		return ""
	}
	if s.Language == "markdown" {
		return s.Value
	}
	return "```" + s.Language + "\n" + s.Value + "\n```"
}
