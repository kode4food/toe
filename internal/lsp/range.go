package lsp

import (
	"github.com/kode4food/toe/internal/core"
	"go.lsp.dev/protocol"
)

func lspRange(
	doc core.Rope, r core.Range, encoding protocol.PositionEncodingKind,
) (protocol.Range, error) {
	start, err := lspPosition(doc, r.From(), encoding)
	if err != nil {
		return protocol.Range{}, err
	}
	end, err := lspPosition(doc, r.To(), encoding)
	if err != nil {
		return protocol.Range{}, err
	}
	return protocol.Range{Start: start, End: end}, nil
}
