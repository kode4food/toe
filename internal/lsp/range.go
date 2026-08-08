package lsp

import (
	"go.lsp.dev/protocol"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

func lspRange(
	doc core.Rope, r core.Range, enc protocol.PositionEncodingKind,
) (protocol.Range, error) {
	start, err := lspPosition(doc, r.From(), enc)
	if err != nil {
		return protocol.Range{}, err
	}
	end, err := lspPosition(doc, r.To(), enc)
	if err != nil {
		return protocol.Range{}, err
	}
	return protocol.Range{Start: start, End: end}, nil
}

func lspRangeToChars(
	doc *view.Document, r protocol.Range, enc protocol.PositionEncodingKind,
) (core.Range, bool) {
	if from, ok := lspPositionToChar(doc, r.Start, enc); ok {
		if to, ok := lspPositionToChar(doc, r.End, enc); ok {
			return core.Range{Anchor: from, Head: to}, true
		}
	}
	return core.Range{}, false
}

func lspPositionToChar(
	doc *view.Document, pos protocol.Position,
	enc protocol.PositionEncodingKind,
) (int, bool) {
	return serverPosition(pos).Resolve(doc.Text(), viewEncoding(enc))
}

func serverPosition(pos protocol.Position) view.ServerPosition {
	return view.ServerPosition{
		Line:      int(pos.Line),
		Character: int(pos.Character),
	}
}

func viewEncoding(enc protocol.PositionEncodingKind) view.PositionEncoding {
	switch enc {
	case protocol.PositionEncodingKindUTF8:
		return view.PositionEncodingUTF8
	case protocol.PositionEncodingKindUTF32:
		return view.PositionEncodingUTF32
	default:
		return view.PositionEncodingUTF16
	}
}
