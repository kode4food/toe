package lsp

import (
	"context"
	"strconv"

	"go.lsp.dev/protocol"

	"github.com/kode4food/toe/internal/core"
)

type (
	clientCallFn[R any] func(context.Context) (R, bool, error)

	docCallFn[R any] func(
		context.Context, *Client, protocol.TextDocumentIdentifier,
	) (R, bool, error)

	posCallFn[R any] func(
		context.Context, protocol.TextDocumentPositionParams,
	) (R, bool, error)
)

// clientRunRequest wraps a server call with the client's request context.
func clientRunRequest[R any](
	c *Client, ctx context.Context, call clientCallFn[R],
) (R, bool, error) {
	ctx, cancel := c.requestContext(ctx)
	defer cancel()
	return call(ctx)
}

// clientDocRequest sends a document-scoped request with no cursor position.
func clientDocRequest[R any](
	c *Client, ctx context.Context, doc DocumentSnapshot,
	call docCallFn[R],
) (R, bool, error) {
	tdid := protocol.TextDocumentIdentifier{URI: doc.URI}
	return clientRunRequest(c, ctx, func(ctx context.Context) (R, bool, error) {
		return call(ctx, c, tdid)
	})
}

type posRequestArgs[R any] struct {
	ctx  context.Context
	doc  DocumentSnapshot
	pos  int
	call posCallFn[R]
}

// clientPosRequest sends a cursor-position request, computing the LSP position
// from doc.Text and pos, then passing a TextDocumentPositionParams to call.
func clientPosRequest[R any](
	c *Client, args posRequestArgs[R],
) (R, bool, error) {
	lspPos, err := lspPosition(
		core.NewRope(args.doc.Text), args.pos, c.OffsetEncoding(),
	)
	if err != nil {
		var zero R
		return zero, false, err
	}
	tdp := protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: args.doc.URI},
		Position:     lspPos,
	}
	return clientRunRequest(c, args.ctx,
		func(ctx context.Context) (R, bool, error) {
			return args.call(ctx, tdp)
		},
	)
}

func candidateID(server string, idx int) string {
	return server + ":" + strconv.Itoa(idx)
}

func markupText(v any) string {
	switch s := v.(type) {
	case protocol.String:
		return string(s)
	case *protocol.MarkupContent:
		if s == nil {
			return ""
		}
		return s.Value
	default:
		return ""
	}
}
