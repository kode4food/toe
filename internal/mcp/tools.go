package mcp

import (
	"context"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

func registerTools(s *Session, srv *mcp.Server) {
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "getDiagnostics",
		Description: "Errors and warnings across open files in the workspace",
	}, func(context.Context, *mcp.CallToolRequest, struct{}) (
		*mcp.CallToolResult, any, error,
	) {
		return nil, s.onEditor(collectDiagnostics), nil
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "openDiff",
		Description: "Show a proposed change as a read-only diff for review",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in struct {
		OldPath     string `json:"old_file_path"`
		NewPath     string `json:"new_file_path"`
		NewContents string `json:"new_file_contents"`
		TabName     string `json:"tab_name"`
	}) (*mcp.CallToolResult, any, error) {
		path := in.NewPath
		if path == "" {
			path = in.OldPath
		}
		// Block until the tab closes: this keeps Claude waiting on its own
		// prompt instead of treating the IDE as having approved the edit
		diff := DiffRequest{
			TabName:     in.TabName,
			Path:        path,
			NewContents: in.NewContents,
		}
		if !s.awaitDiffClose(ctx, diff) {
			return newErrorResult(
				"Diff unavailable in the current editor layout",
			), nil, nil
		}
		return textResult("TAB_CLOSED"), nil, nil
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "close_tab",
		Description: "Close a diff tab opened by openDiff",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in struct {
		TabName string `json:"tab_name"`
	}) (*mcp.CallToolResult, any, error) {
		s.signalDiffClosed(in.TabName)
		return textResult("TAB_CLOSED"), nil, nil
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "closeAllDiffTabs",
		Description: "Close all diff tabs opened by openDiff",
	}, func(context.Context, *mcp.CallToolRequest, struct{}) (
		*mcp.CallToolResult, any, error,
	) {
		s.signalDiffClosed("")
		return textResult("TABS_CLOSED"), nil, nil
	})
}

func textResult(texts ...string) *mcp.CallToolResult {
	content := make([]mcp.Content, len(texts))
	for i, t := range texts {
		content[i] = &mcp.TextContent{Text: t}
	}
	return &mcp.CallToolResult{Content: content}
}

func newErrorResult(text string) *mcp.CallToolResult {
	res := textResult(text)
	res.IsError = true
	return res
}

func collectDiagnostics(e *view.Editor) any {
	out := []any{}
	for _, doc := range e.AllDocuments() {
		diags := doc.Diagnostics()
		if len(diags) == 0 {
			continue
		}
		text := doc.Text()
		ds := make([]any, 0, len(diags))
		for _, d := range diags {
			ds = append(ds, map[string]any{
				"range":    rangeOf(text, d.Range),
				"severity": lspSeverity(d.Severity),
				"message":  d.Message,
				"source":   d.Source,
			})
		}
		out = append(out, map[string]any{
			"uri":         pathToURI(doc.Path()),
			"diagnostics": ds,
		})
	}
	return out
}

// lspSeverity maps toe's order (Hint=1..Error=4) to the LSP order
// (Error=1..Hint=4) that MCP clients expect
func lspSeverity(s view.DiagnosticSeverity) int { return 5 - int(s) }

func rangeOf(text core.Rope, span core.Span) map[string]any {
	return map[string]any{
		"start": posOf(text, span.From),
		"end":   posOf(text, span.To),
	}
}

// posOf converts a character offset to a 0-based LSP line/character position
func posOf(text core.Rope, off int) map[string]int {
	p, err := text.Position(off)
	if err != nil {
		return map[string]int{"line": 0, "character": 0}
	}
	return map[string]int{"line": p.Line - 1, "character": p.Column - 1}
}

func pathToURI(path string) string {
	if path == "" {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	return "file://" + abs
}
