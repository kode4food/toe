package lsp

import (
	"context"
	"errors"
	"strings"
	"unicode/utf16"

	"github.com/kode4food/toe/internal/view"
	"go.lsp.dev/protocol"
)

func (c *Client) SignatureHelp(
	ctx context.Context, doc DocumentSnapshot, pos int,
	shCtx protocol.SignatureHelpContext,
) (*protocol.SignatureHelp, bool, error) {
	if !c.SupportsFeature(FeatureSignatureHelp) {
		return nil, false, nil
	}
	return clientPosRequest(c, ctx, doc, pos, func(
		ctx context.Context, tdp protocol.TextDocumentPositionParams,
	) (*protocol.SignatureHelp, bool, error) {
		help, err := c.server.SignatureHelp(ctx, &protocol.SignatureHelpParams{
			TextDocumentPositionParams: tdp, Context: shCtx,
		})
		if err != nil {
			return nil, true, err
		}
		return help, true, nil
	})
}

// SignatureHelp returns the signature help at the cursor for an invoked trigger
func (s *Session) SignatureHelp(
	doc *view.Document, viewID view.Id,
) (view.SignatureHelp, error) {
	context := protocol.SignatureHelpContext{
		TriggerKind: protocol.SignatureHelpTriggerKindInvoked,
	}
	return s.signatureHelp(doc, viewID, context)
}

// TriggerSignatureHelp returns signature help triggered by the character before the cursor
func (s *Session) TriggerSignatureHelp(
	doc *view.Document, viewID view.Id,
) (view.SignatureHelp, error) {
	trigger, ok := s.signatureTrigger(doc, viewID)
	if !ok {
		return view.SignatureHelp{}, nil
	}
	context := protocol.SignatureHelpContext{
		TriggerKind:      protocol.SignatureHelpTriggerKindTriggerCharacter,
		TriggerCharacter: &trigger,
	}
	return s.signatureHelp(doc, viewID, context)
}

func (s *Session) signatureHelp(
	doc *view.Document, viewID view.Id,
	context protocol.SignatureHelpContext,
) (view.SignatureHelp, error) {
	snap, ok := SnapshotDocument(doc)
	if !ok {
		return view.SignatureHelp{}, nil
	}
	sel := doc.SelectionFor(viewID)
	pos := sel.Primary().Cursor(doc.Text())
	clients := s.clientsForDocument(doc)
	var err error
	for _, client := range clients {
		help, sent, e := client.SignatureHelp(s.ctx, snap, pos, context)
		if e != nil {
			err = errors.Join(err, s.completionError(client, e))
			continue
		}
		if sent && help != nil && len(help.Signatures) > 0 {
			return normalizeSignatureHelp(help), err
		}
	}
	return view.SignatureHelp{}, err
}

func (s *Session) signatureTrigger(
	doc *view.Document, viewID view.Id,
) (string, bool) {
	sel := doc.SelectionFor(viewID)
	pos := sel.Primary().Cursor(doc.Text())
	before, err := doc.Text().SliceString(0, pos)
	if err != nil {
		return "", false
	}
	for _, client := range s.clientsForDocument(doc) {
		capabilities, ok := client.Capabilities()
		if !ok || capabilities.SignatureHelpProvider == nil {
			continue
		}
		for _, trigger := range signatureTriggerCharacters(capabilities) {
			if trigger != "" && strings.HasSuffix(before, trigger) {
				return trigger, true
			}
		}
	}
	return "", false
}

func normalizeSignatureHelp(help *protocol.SignatureHelp) view.SignatureHelp {
	active := activeSignature(help)
	out := view.SignatureHelp{
		Signatures: make([]view.SignatureInformation, 0, len(help.Signatures)),
		Active:     active,
	}
	for _, sig := range help.Signatures {
		param := activeParameter(help, sig)
		info := view.SignatureInformation{
			Label: sig.Label,
			Docs:  markupText(sig.Documentation),
		}
		if param >= 0 && param < len(sig.Parameters) {
			p := sig.Parameters[param]
			info.ParamDocs = markupText(p.Documentation)
			info.ActiveStart, info.ActiveEnd = activeParameterRange(sig, p)
		}
		out.Signatures = append(out.Signatures, info)
	}
	return out
}

func activeSignature(help *protocol.SignatureHelp) int {
	if help.ActiveSignature == nil {
		return 0
	}
	active := int(*help.ActiveSignature)
	if active >= len(help.Signatures) {
		return 0
	}
	return active
}

func activeParameter(
	help *protocol.SignatureHelp, sig protocol.SignatureInformation,
) int {
	if value, ok := sig.ActiveParameter.Get(); ok {
		return int(value)
	}
	if value, ok := help.ActiveParameter.Get(); ok {
		return int(value)
	}
	if len(sig.Parameters) > 0 {
		return 0
	}
	return -1
}

func activeParameterRange(
	sig protocol.SignatureInformation, param protocol.ParameterInformation,
) (int, int) {
	switch label := param.Label.(type) {
	case protocol.String:
		start := strings.Index(sig.Label, string(label))
		if start < 0 {
			return 0, 0
		}
		end := start + len(label)
		return runeIndex(sig.Label, start), runeIndex(sig.Label, end)
	case protocol.ParameterInformationLabelTuple:
		start := utf16Index(sig.Label, int(label[0]))
		end := utf16Index(sig.Label, int(label[1]))
		return start, end
	default:
		return 0, 0
	}
}

func signatureTriggerCharacters(
	capabilities protocol.ServerCapabilities,
) []string {
	opts := capabilities.SignatureHelpProvider
	triggers := append([]string(nil), opts.TriggerCharacters...)
	triggers = append(triggers, opts.RetriggerCharacters...)
	return triggers
}

func runeIndex(s string, byteOffset int) int {
	return len([]rune(s[:byteOffset]))
}

func utf16Index(s string, target int) int {
	units := 0
	for i, r := range []rune(s) {
		if units >= target {
			return i
		}
		units += len(utf16.Encode([]rune{r}))
	}
	return len([]rune(s))
}
