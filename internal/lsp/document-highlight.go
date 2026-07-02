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

// DocumentHighlights requests same-document symbol highlights at pos
func (c *Client) DocumentHighlights(
	ctx context.Context, doc DocumentSnapshot, pos int,
) ([]protocol.DocumentHighlight, bool, error) {
	if !c.SupportsFeature(FeatureDocumentHighlight) {
		return nil, false, nil
	}
	return clientPosRequest(c, posRequestArgs[[]protocol.DocumentHighlight]{
		ctx: ctx,
		doc: doc,
		pos: pos,
		call: func(
			ctx context.Context, tdp protocol.TextDocumentPositionParams,
		) ([]protocol.DocumentHighlight, bool, error) {
			highlights, err := c.server.DocumentHighlight(ctx,
				&protocol.DocumentHighlightParams{
					TextDocumentPositionParams: tdp,
				},
			)
			if err != nil {
				return nil, true, err
			}
			return highlights, true, nil
		},
	})
}

// DocumentHighlights returns same-document symbol highlights at the cursor
func (s *Session) DocumentHighlights(
	doc *view.Document, viewID view.Id,
) ([]view.DocumentHighlight, error) {
	snap, ok := SnapshotDocument(doc)
	if !ok {
		doc.ClearDocumentHighlights(viewID)
		return nil, nil
	}
	pos := doc.SelectionFor(viewID).Primary().Cursor(doc.Text())
	clients := s.clientsForDocument(doc)
	var out []core.Range
	var sent bool
	var err error
	for _, client := range clients {
		highlights, ok, e := client.DocumentHighlights(s.ctx, snap, pos)
		if e != nil {
			err = errors.Join(err, s.completionError(client, e))
			continue
		}
		if !ok {
			continue
		}
		sent = true
		out = append(out, documentHighlightRanges(client, doc, highlights)...)
	}
	if !sent {
		doc.ClearDocumentHighlights(viewID)
		return nil, ErrNoLanguageServer
	}
	merged := mergeDocumentHighlightRanges(out)
	highlights := viewDocumentHighlights(merged)
	doc.SetDocumentHighlights(viewID, highlights)
	return highlights, err
}

func documentHighlightRanges(
	client *Client, doc *view.Document,
	highlights []protocol.DocumentHighlight,
) []core.Range {
	out := make([]core.Range, 0, len(highlights))
	for _, highlight := range highlights {
		from, to, ok := lspRangeToChars(doc, highlight.Range, client.OffsetEncoding())
		if !ok {
			continue
		}
		r := core.NewRange(from, to).MinWidth1(doc.Text())
		if r.From() < r.To() {
			out = append(out, r)
		}
	}
	return out
}

func mergeDocumentHighlightRanges(ranges []core.Range) []core.Range {
	slices.SortFunc(ranges, func(a, b core.Range) int {
		if n := cmp.Compare(a.From(), b.From()); n != 0 {
			return n
		}
		return cmp.Compare(a.To(), b.To())
	})
	out := ranges[:0]
	for _, r := range ranges {
		if len(out) == 0 {
			out = append(out, r)
			continue
		}
		last := out[len(out)-1]
		if r.From() > last.To() {
			out = append(out, r)
			continue
		}
		if r.To() > last.To() {
			out[len(out)-1] = last.Merge(r)
		}
	}
	return out
}

func viewDocumentHighlights(ranges []core.Range) []view.DocumentHighlight {
	out := make([]view.DocumentHighlight, 0, len(ranges))
	for _, r := range ranges {
		out = append(out, view.DocumentHighlight{
			From: r.From(),
			To:   r.To(),
		})
	}
	return out
}
