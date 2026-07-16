package lsp_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"

	"github.com/kode4food/toe/internal/lsp"
	"github.com/kode4food/toe/internal/view"
)

type watchServer struct {
	protocol.UnimplementedServer
	changes chan *protocol.DidChangeWatchedFilesParams
}

func TestDidChangeWatchedFile(t *testing.T) {
	t.Run("ignores empty path", func(t *testing.T) {
		ctx := t.Context()
		clientConn, serverConn := net.Pipe()
		defer func() { _ = clientConn.Close() }()
		defer func() { _ = serverConn.Close() }()

		server := &watchServer{
			changes: make(chan *protocol.DidChangeWatchedFilesParams, 1),
		}
		_, serverRPC, _ := protocol.NewServer(
			ctx, server, jsonrpc2.NewHeaderStream(serverConn),
		)
		defer func() { _ = serverRPC.Close() }()

		_, client := lsp.NewClient(ctx, clientConn, nil)
		defer func() { _ = client.Close() }()

		assert.NoError(t, client.DidChangeWatchedFile(ctx, ""))
		select {
		case <-server.changes:
			t.Fatal("unexpected notification for empty path")
		case <-time.After(50 * time.Millisecond):
		}
	})

	t.Run("forwards changed event", func(t *testing.T) {
		ctx := t.Context()
		clientConn, serverConn := net.Pipe()
		defer func() { _ = clientConn.Close() }()
		defer func() { _ = serverConn.Close() }()

		server := &watchServer{
			changes: make(chan *protocol.DidChangeWatchedFilesParams, 1),
		}
		_, serverRPC, _ := protocol.NewServer(
			ctx, server, jsonrpc2.NewHeaderStream(serverConn),
		)
		defer func() { _ = serverRPC.Close() }()

		_, client := lsp.NewClient(ctx, clientConn, nil)
		defer func() { _ = client.Close() }()

		assert.NoError(t, client.DidChangeWatchedFile(ctx, "/tmp/main.go"))
		params := <-server.changes
		assert.Len(t, params.Changes, 1)
		assert.Equal(t,
			protocol.FileChangeTypeChanged, params.Changes[0].Type,
		)
	})
}

func TestWatchRegistrationEdgeCases(t *testing.T) {
	t.Run("ignores nil and mismatched registrations", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		notifyFile := filepath.Join(t.TempDir(), "watched")
		writeWatchRegEdgeLanguages(t, exe, notifyFile)
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)

		// Trigger initialization; the fake server registers, then fully
		// unregisters, a "*.watched" watcher via a mix of nil, mismatched,
		// and malformed registrations before the real one is removed
		_, _ = session.Completions(doc, v.ID())
		time.Sleep(100 * time.Millisecond)

		created := filepath.Join(dir, "created.watched")
		assert.NoError(t, os.WriteFile(created, []byte("new\n"), 0o644))

		assert.Never(t, func() bool {
			_, err := os.Stat(notifyFile)
			return err == nil
		}, time.Second, 25*time.Millisecond)
	})
}

func writeWatchRegEdgeLanguages(t *testing.T, exe, notifyFile string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := fmt.Sprintf(`[language-server.session-test]
command = %q
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { %s = "1", %s = "1", %s = "1", %s = %q }

[[language]]
name = "session"
language-id = "session"
file-types = ["session"]
language-servers = ["session-test"]
`, exe, testServerEnv, testServerCompletionEnv, testServerWatchRegEdgeEnv,
		testServerFileWatchNotifyEnv, notifyFile)
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}

func (s *watchServer) Initialize(
	context.Context, *protocol.InitializeParams,
) (*protocol.InitializeResult, error) {
	return &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			PositionEncoding: protocol.PositionEncodingKindUTF8,
		},
	}, nil
}

func (s *watchServer) DidChangeWatchedFiles(
	_ context.Context, params *protocol.DidChangeWatchedFilesParams,
) error {
	s.changes <- params
	return nil
}
