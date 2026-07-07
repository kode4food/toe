package lsp

import (
	"go.lsp.dev/protocol"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
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

func lspRangeToChars(
	doc *view.Document, r protocol.Range,
	encoding protocol.PositionEncodingKind,
) (core.Range, bool) {
	from, ok := lspPositionToChar(doc, r.Start, encoding)
	if !ok {
		return core.Range{}, false
	}
	to, ok := lspPositionToChar(doc, r.End, encoding)
	if !ok {
		return core.Range{}, false
	}
	return core.NewRange(from, to), true
}

func lspPositionToChar(
	doc *view.Document, pos protocol.Position,
	encoding protocol.PositionEncodingKind,
) (int, bool) {
	lineStart, err := doc.Text().LineToChar(int(pos.Line))
	if err != nil {
		return 0, false
	}
	lineEnd, err := doc.Text().LineEndCharIndex(int(pos.Line))
	if err != nil {
		return 0, false
	}
	line, err := doc.Text().SliceString(lineStart, lineEnd)
	if err != nil {
		return 0, false
	}
	chars, ok := encodedPositionToChar(line, int(pos.Character), encoding)
	if !ok {
		return 0, false
	}
	return lineStart + chars, true
}

func encodedPositionToChar(
	line string, target int, encoding protocol.PositionEncodingKind,
) (int, bool) {
	units := 0
	chars := 0
	for _, ch := range line {
		if units == target {
			return chars, true
		}
		units += encodedRuneLen(ch, encoding)
		chars++
		if units > target {
			return 0, false
		}
	}
	if units == target {
		return chars, true
	}
	return 0, false
}
