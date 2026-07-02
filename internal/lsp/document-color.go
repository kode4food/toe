package lsp

import (
	"cmp"
	"context"
	"errors"
	"slices"

	"go.lsp.dev/protocol"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// DocumentColors requests document-wide colors
func (c *Client) DocumentColors(
	ctx context.Context, doc DocumentSnapshot,
) ([]protocol.ColorInformation, bool, error) {
	return clientDocRequest(c, ctx, doc, func(ctx context.Context, c *Client, tdid protocol.TextDocumentIdentifier) ([]protocol.ColorInformation, bool, error) {
		if !c.SupportsFeature(FeatureDocumentColors) {
			return nil, false, nil
		}
		colors, err := c.server.DocumentColor(ctx, &protocol.DocumentColorParams{TextDocument: tdid})
		if err != nil {
			return nil, true, err
		}
		return colors, true, nil
	})
}

// DocumentColors returns document-wide colors and stores them on the document
func (s *Session) DocumentColors(
	doc *view.Document,
) ([]view.DocumentColor, error) {
	snap, ok := SnapshotDocument(doc)
	if !ok {
		doc.ClearDocumentColors()
		return nil, nil
	}
	var out []view.DocumentColor
	var sent bool
	var err error
	for _, client := range s.clientsForDocument(doc) {
		colors, ok, e := client.DocumentColors(s.ctx, snap)
		if e != nil {
			err = errors.Join(err, s.completionError(client, e))
			continue
		}
		if !ok {
			continue
		}
		sent = true
		out = append(out, documentColors(client, doc, colors)...)
	}
	if !sent {
		doc.ClearDocumentColors()
		return nil, ErrNoLanguageServer
	}
	slices.SortFunc(out, func(a, b view.DocumentColor) int {
		if n := cmp.Compare(a.From, b.From); n != 0 {
			return n
		}
		return cmp.Compare(a.To, b.To)
	})
	doc.SetDocumentColors(out)
	return out, err
}

func (s *Session) documentColorsAsync(doc *view.Document) {
	go func() {
		_, _ = s.DocumentColors(doc)
	}()
}

func documentColors(
	client *Client, doc *view.Document, colors []protocol.ColorInformation,
) []view.DocumentColor {
	out := make([]view.DocumentColor, 0, len(colors))
	for _, color := range colors {
		from, to, ok := lspRangeToChars(
			doc, color.Range, client.OffsetEncoding(),
		)
		if !ok {
			continue
		}
		r := core.NewRange(from, to).MinWidth1(doc.Text())
		if r.From() >= r.To() {
			continue
		}
		out = append(out, view.DocumentColor{
			From:  r.From(),
			To:    r.To(),
			Red:   colorByte(color.Color.Red),
			Green: colorByte(color.Color.Green),
			Blue:  colorByte(color.Color.Blue),
		})
	}
	return out
}

func colorByte(v float64) uint8 {
	return uint8(min(max(v, 0), 1) * 255)
}
