package lsp

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
	"go.lsp.dev/protocol"
)

type codeActionCandidate struct {
	client *Client
	item   protocol.CommandOrCodeAction
}

// Sort priority constants for code action ordering (lower = listed first).
const (
	codeActionQuickfix         = iota // 0
	codeActionRefactorExtract         // 1
	codeActionRefactorInline          // 2
	codeActionRefactorRewrite         // 3
	codeActionRefactorMove            // 4
	codeActionRefactorSurround        // 5
	codeActionSource                  // 6
	codeActionOther                   // 7
)

var codeActionTopCategory = map[string]int{
	"quickfix": codeActionQuickfix,
	"source":   codeActionSource,
}

var codeActionRefactorSubcategory = map[string]int{
	"extract":  codeActionRefactorExtract,
	"inline":   codeActionRefactorInline,
	"rewrite":  codeActionRefactorRewrite,
	"move":     codeActionRefactorMove,
	"surround": codeActionRefactorSurround,
}

// CodeActions requests code actions and commands for a selection
func (c *Client) CodeActions(
	ctx context.Context, doc DocumentSnapshot, r core.Range,
	diags []protocol.Diagnostic,
) ([]protocol.CommandOrCodeAction, bool, error) {
	if !c.SupportsFeature(FeatureCodeAction) {
		return nil, false, nil
	}
	lspRange, err := lspRange(
		core.NewRope(doc.Text), r, c.OffsetEncoding(),
	)
	if err != nil {
		return nil, false, err
	}
	params := &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: doc.URI},
		Range:        lspRange,
		Context: protocol.CodeActionContext{
			Diagnostics: diags,
			TriggerKind: protocol.CodeActionTriggerKindInvoked,
		},
	}
	ctx, cancel := c.requestContext(ctx)
	defer cancel()
	result, err := c.server.CodeAction(ctx, params)
	if err != nil {
		return nil, true, err
	}
	return enabledCodeActions(result), true, nil
}

// CodeActions returns code actions and commands for the current selection
func (s *Session) CodeActions(
	doc *view.Document, viewID view.Id,
) ([]view.CodeAction, error) {
	snap, ok := SnapshotDocument(doc)
	if !ok {
		return nil, nil
	}
	r := doc.SelectionFor(viewID).Primary()
	clients := s.clientsForDocument(doc)
	if len(clients) == 0 {
		return nil, ErrNoLanguageServer
	}
	out := []view.CodeAction{}
	raw := map[string]codeActionCandidate{}
	var err error
	for _, client := range clients {
		diags := s.codeActionDiagnostics(doc, r, client.OffsetEncoding())
		actions, sent, e := client.CodeActions(s.ctx, snap, r, diags)
		if e != nil {
			err = errors.Join(err, s.completionError(client, e))
			continue
		}
		if !sent {
			continue
		}
		for i, action := range actions {
			id := candidateID(client.Name(), i)
			item, ok := viewCodeAction(id, client.Name(), action)
			if !ok {
				continue
			}
			out = append(out, item)
			raw[id] = codeActionCandidate{client: client, item: action}
		}
	}
	s.storeCodeActions(raw)
	return out, err
}

// ApplyCodeAction applies a selected code action or command
func (s *Session) ApplyCodeAction(
	_ *view.Document, _ view.Id, item view.CodeAction,
) error {
	c, ok := s.codeAction(item.ID)
	if !ok {
		return ErrCodeActionUnavailable
	}
	switch action := c.item.(type) {
	case *protocol.Command:
		return c.client.ExecuteCommand(s.ctx, commandParams(*action))
	case *protocol.CodeAction:
		resolved, err := s.resolveCodeAction(c.client, action)
		if err != nil {
			return err
		}
		if resolved.Edit != nil {
			if err := s.applyWorkspaceEdit(
				*resolved.Edit, c.client.OffsetEncoding(),
			); err != nil {
				return err
			}
		}
		if resolved.Command.Command != "" {
			return c.client.ExecuteCommand(
				s.ctx, commandParams(resolved.Command),
			)
		}
	}
	return nil
}

func (s *Session) resolveCodeAction(
	client *Client, action *protocol.CodeAction,
) (*protocol.CodeAction, error) {
	if !clientResolvesCodeAction(client) {
		return action, nil
	}
	if action.Edit != nil && action.Command.Command != "" {
		return action, nil
	}
	ctx, cancel := client.requestContext(s.ctx)
	defer cancel()
	resolved, err := client.server.CodeActionResolve(ctx, action)
	if err != nil {
		return action, s.completionError(client, err)
	}
	if resolved == nil {
		return action, nil
	}
	return resolved, nil
}

func (s *Session) codeActionDiagnostics(
	doc *view.Document, r core.Range,
	encoding protocol.PositionEncodingKind,
) []protocol.Diagnostic {
	var out []protocol.Diagnostic
	for _, diag := range doc.Diagnostics() {
		dr := core.NewRange(diag.Range.From, diag.Range.To)
		if !r.Overlaps(dr) {
			continue
		}
		converted, ok := protocolDiagnostic(doc, diag, encoding)
		if ok {
			out = append(out, converted)
		}
	}
	return out
}

func (s *Session) storeCodeActions(items map[string]codeActionCandidate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.actions = items
}

func (s *Session) codeAction(id string) (codeActionCandidate, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.actions[id]
	return c, ok
}

func enabledCodeActions(
	items []protocol.CommandOrCodeAction,
) []protocol.CommandOrCodeAction {
	out := make([]protocol.CommandOrCodeAction, 0, len(items))
	for _, item := range items {
		action, ok := item.(*protocol.CodeAction)
		if ok && action.Disabled.Reason != "" {
			continue
		}
		out = append(out, item)
	}
	slices.SortStableFunc(out, compareCodeActionItems)
	return out
}

func compareCodeActionItems(
	a, b protocol.CommandOrCodeAction,
) int {
	if order := cmp.Compare(
		codeActionCategory(a), codeActionCategory(b),
	); order != 0 {
		return order
	}
	if order := compareBoolDesc(
		codeActionFixesDiagnostics(a), codeActionFixesDiagnostics(b),
	); order != 0 {
		return order
	}
	return compareBoolDesc(codeActionPreferred(a), codeActionPreferred(b))
}

func viewCodeAction(
	id, server string, item protocol.CommandOrCodeAction,
) (view.CodeAction, bool) {
	switch action := item.(type) {
	case *protocol.Command:
		if action.Title == "" {
			return view.CodeAction{}, false
		}
		return view.CodeAction{
			ID: id, Title: action.Title, Server: server,
		}, true
	case *protocol.CodeAction:
		if action.Title == "" {
			return view.CodeAction{}, false
		}
		kind := ""
		if action.Kind != nil {
			kind = string(*action.Kind)
		}
		return view.CodeAction{
			ID: id, Title: action.Title, Kind: kind,
			Server: server, Preferred: codeActionPreferred(item),
		}, true
	default:
		return view.CodeAction{}, false
	}
}

func codeActionCategory(item protocol.CommandOrCodeAction) int {
	action, ok := item.(*protocol.CodeAction)
	if !ok || action.Kind == nil {
		return codeActionOther
	}
	parts := strings.Split(string(*action.Kind), ".")
	if cat, ok := codeActionTopCategory[parts[0]]; ok {
		return cat
	}
	if parts[0] == "refactor" && len(parts) >= 2 {
		if cat, ok := codeActionRefactorSubcategory[parts[1]]; ok {
			return cat
		}
	}
	return codeActionOther
}

func codeActionPreferred(item protocol.CommandOrCodeAction) bool {
	action, ok := item.(*protocol.CodeAction)
	return ok && action.IsPreferred != nil && *action.IsPreferred
}

func codeActionFixesDiagnostics(item protocol.CommandOrCodeAction) bool {
	action, ok := item.(*protocol.CodeAction)
	return ok && len(action.Diagnostics) > 0
}

func compareBoolDesc(a, b bool) int {
	switch {
	case a == b:
		return 0
	case a:
		return -1
	default:
		return 1
	}
}

func clientResolvesCodeAction(client *Client) bool {
	capabilities, ok := client.Capabilities()
	if !ok {
		return false
	}
	opts, ok := capabilities.CodeActionProvider.(*protocol.CodeActionOptions)
	return ok && opts != nil &&
		opts.ResolveProvider != nil && *opts.ResolveProvider
}

func commandParams(command protocol.Command) *protocol.ExecuteCommandParams {
	return &protocol.ExecuteCommandParams{
		Command:   command.Command,
		Arguments: command.Arguments,
	}
}

func protocolDiagnostic(
	doc *view.Document, diag view.Diagnostic,
	encoding protocol.PositionEncodingKind,
) (protocol.Diagnostic, bool) {
	r, err := lspRange(
		doc.Text(), core.NewRange(diag.Range.From, diag.Range.To), encoding,
	)
	if err != nil {
		return protocol.Diagnostic{}, false
	}
	out := protocol.Diagnostic{
		Range:    r,
		Severity: protocolDiagnosticSeverity(diag.Severity),
		Message:  protocol.String(diag.Message),
	}
	if diag.Source != "" {
		out.Source = protocol.NewOptional(diag.Source)
	}
	return out, true
}

func protocolDiagnosticSeverity(
	severity view.DiagnosticSeverity,
) protocol.DiagnosticSeverity {
	switch severity {
	case view.DiagnosticSeverityError:
		return protocol.DiagnosticSeverityError
	case view.DiagnosticSeverityWarning:
		return protocol.DiagnosticSeverityWarning
	case view.DiagnosticSeverityInfo:
		return protocol.DiagnosticSeverityInformation
	default:
		return protocol.DiagnosticSeverityHint
	}
}
