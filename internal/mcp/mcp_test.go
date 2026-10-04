package mcp_test

import (
	"context"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/mcp"
	"github.com/kode4food/toe/internal/view"
)

func TestServer(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // lockfile goes to a temp ~/.claude/ide
	e := view.NewEditor(t.TempDir())
	e.FocusedDocument().ReplaceDiagnostics("gopls", []view.Diagnostic{
		{Severity: view.DiagnosticSeverityError, Message: "boom"},
	})
	s := mcp.Attach(context.Background(), e, "127.0.0.1:0")
	defer s.Close()
	s.SetDispatcher(func(fn func()) { fn() })

	shownc := make(chan string, 1)
	s.SetDiffHandlers(
		func(_, _, content string) { shownc <- content },
		func(string) {},
	)

	sess := connect(t, s.URL())
	if sess == nil {
		return
	}
	defer sess.Close()
	ctx := context.Background()

	t.Run("lists tools", func(t *testing.T) {
		res, err := sess.ListTools(ctx, nil)
		assert.NoError(t, err)
		names := map[string]bool{}
		for _, tl := range res.Tools {
			names[tl.Name] = true
		}
		for _, want := range []string{
			"getDiagnostics", "openDiff", "close_tab", "closeAllDiffTabs",
		} {
			assert.True(t, names[want])
		}
	})

	t.Run("reports diagnostics", func(t *testing.T) {
		res, err := sess.CallTool(ctx,
			&sdk.CallToolParams{Name: "getDiagnostics"})
		assert.NoError(t, err)
		text := res.Content[0].(*sdk.TextContent).Text
		assert.Contains(t, text, "boom")
		assert.Contains(t, text, `"severity":1`)
	})

	t.Run("openDiff blocks and does not approve", func(t *testing.T) {
		cctx, cancel := context.WithCancel(ctx)
		defer cancel()
		done := make(chan struct{})
		go func() {
			_, _ = sess.CallTool(cctx, &sdk.CallToolParams{
				Name: "openDiff",
				Arguments: map[string]any{
					"old_file_path": "/x.go", "new_file_path": "/x.go",
					"new_file_contents": "hello", "tab_name": "t1",
				},
			})
			close(done)
		}()
		assert.Equal(t, "hello", <-shownc) // the diff was shown
		select {
		case <-done:
			assert.Fail(t, "openDiff returned without a decision")
		case <-time.After(50 * time.Millisecond):
		}
		cancel() // Claude cancelling the request unblocks it
		<-done
	})

	t.Run("close_tab", func(t *testing.T) {
		res := call(t, sess, "close_tab", map[string]any{"tab_name": "none"})
		assert.Equal(t, "TAB_CLOSED", res.Content[0].(*sdk.TextContent).Text)
	})
}

func TestEnableDisable(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := view.NewEditor(t.TempDir())
	s := mcp.Attach(context.Background(), e, "127.0.0.1:0")
	defer s.Close()

	assert.True(t, s.Enabled())
	assert.NotEmpty(t, s.URL())

	s.SetEnabled(false)
	assert.False(t, s.Enabled())
	assert.Empty(t, s.URL())

	s.SetEnabled(true)
	assert.True(t, s.Enabled())
	assert.NotEmpty(t, s.URL())
}

func TestAttachBindFailure(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := view.NewEditor(t.TempDir())
	s := mcp.Attach(context.Background(), e, "127.0.0.1:99999")
	defer s.Close()
	assert.Equal(t, "", s.URL())
}

func connect(t *testing.T, url string) *sdk.ClientSession {
	t.Helper()
	client := sdk.NewClient(
		&sdk.Implementation{Name: "test", Version: "0"}, nil)
	sess, err := client.Connect(context.Background(),
		&sdk.SSEClientTransport{Endpoint: url}, nil)
	assert.NoError(t, err)
	return sess
}

func call(
	t *testing.T, sess *sdk.ClientSession, name string, args map[string]any,
) *sdk.CallToolResult {
	t.Helper()
	res, err := sess.CallTool(context.Background(),
		&sdk.CallToolParams{Name: name, Arguments: args})
	assert.NoError(t, err)
	return res
}
