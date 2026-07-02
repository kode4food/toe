package lsp

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"strings"

	"go.lsp.dev/protocol"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// InlayHints requests inlay hints for a document range
func (c *Client) InlayHints(
	ctx context.Context, doc DocumentSnapshot, r core.Range,
) ([]protocol.InlayHint, bool, error) {
	if !c.SupportsFeature(FeatureInlayHints) {
		return nil, false, nil
	}
	lspRange, err := lspRange(core.NewRope(doc.Text), r, c.OffsetEncoding())
	if err != nil {
		return nil, false, err
	}
	params := &protocol.InlayHintParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: doc.URI},
		Range:        lspRange,
	}
	ctx, cancel := c.requestContext(ctx)
	defer cancel()
	hints, err := c.server.InlayHint(ctx, params)
	if err != nil {
		return nil, true, err
	}
	return hints, true, nil
}

// InlayHints returns language-server hints and stores them for the view
func (s *Session) InlayHints(
	doc *view.Document, viewID view.Id,
) ([]view.InlayHint, error) {
	snap, ok := SnapshotDocument(doc)
	if !ok {
		doc.ClearInlayHints(viewID)
		return nil, nil
	}
	r := core.NewRange(0, doc.Text().LenChars())
	var out []view.InlayHint
	var sent bool
	var err error
	for _, client := range s.clientsForDocument(doc) {
		hints, ok, e := client.InlayHints(s.ctx, snap, r)
		if e != nil {
			err = errors.Join(err, s.completionError(client, e))
			continue
		}
		if !ok {
			continue
		}
		sent = true
		out = append(out, viewInlayHints(client, doc, hints)...)
	}
	if !sent {
		doc.ClearInlayHints(viewID)
		return nil, ErrNoLanguageServer
	}
	slices.SortStableFunc(out, func(a, b view.InlayHint) int {
		if n := cmp.Compare(a.Pos, b.Pos); n != 0 {
			return n
		}
		return cmp.Compare(a.Label, b.Label)
	})
	doc.SetInlayHints(viewID, out)
	return out, err
}

func viewInlayHints(
	client *Client, doc *view.Document, hints []protocol.InlayHint,
) []view.InlayHint {
	out := make([]view.InlayHint, 0, len(hints))
	for _, hint := range hints {
		pos, ok := lspPositionToChar(
			doc, hint.Position, client.OffsetEncoding(),
		)
		if !ok {
			continue
		}
		label := inlayHintLabel(hint.Label)
		if label == "" {
			continue
		}
		out = append(out, view.InlayHint{
			Pos:          pos,
			Label:        label,
			Kind:         inlayHintKind(hint.Kind),
			PaddingLeft:  boolValue(hint.PaddingLeft),
			PaddingRight: boolValue(hint.PaddingRight),
		})
	}
	return out
}

func inlayHintLabel(label protocol.InlayHintLabel) string {
	switch l := label.(type) {
	case protocol.String:
		return string(l)
	case protocol.InlayHintLabelPartSlice:
		var b strings.Builder
		for _, part := range l {
			b.WriteString(part.Value)
		}
		return b.String()
	default:
		return ""
	}
}

func inlayHintKind(kind protocol.InlayHintKind) string {
	switch kind {
	case protocol.InlayHintKindType:
		return "type"
	case protocol.InlayHintKindParameter:
		return "parameter"
	default:
		return ""
	}
}

func (s *Session) inlayHintsAsync(doc *view.Document) {
	if s.editor == nil {
		return
	}
	for _, v := range s.editor.AllViews() {
		if v.DocID() != doc.ID() {
			continue
		}
		id := v.ID()
		go func() {
			_, _ = s.InlayHints(doc, id)
		}()
	}
}
