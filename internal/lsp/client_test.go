package lsp_test

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/kode4food/toe/internal/lsp"
	"github.com/stretchr/testify/assert"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

type lifecycleServer struct {
	protocol.UnimplementedServer
	capabilities protocol.ServerCapabilities
	fileOps      chan string
	initialized  chan struct{}
	exited       chan struct{}
}

const waitTimeout = time.Second

func TestClient(t *testing.T) {
	t.Run("returns server handle", func(t *testing.T) {
		ctx := t.Context()
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()

		server := &lifecycleServer{
			initialized: make(chan struct{}),
			exited:      make(chan struct{}),
		}
		_, serverRPC, _ := protocol.NewServer(
			ctx, server, jsonrpc2.NewHeaderStream(serverConn),
		)
		defer serverRPC.Close()

		_, client := lsp.NewClient(ctx, clientConn, nil)
		defer client.Close()

		srv := client.Server()

		assert.NotNil(t, srv)
	})

	t.Run("initializes and exits", func(t *testing.T) {
		ctx := t.Context()

		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()

		server := &lifecycleServer{
			initialized: make(chan struct{}),
			exited:      make(chan struct{}),
		}
		_, serverRPC, _ := protocol.NewServer(
			ctx, server, jsonrpc2.NewHeaderStream(serverConn),
		)
		defer serverRPC.Close()

		_, client := lsp.NewClient(ctx, clientConn, nil)
		defer client.Close()

		result, err := client.Initialize(
			ctx, lsp.NewInitializeParams(lsp.InitializeConfig{}),
		)
		assert.NoError(t, err)
		assert.Equal(t,
			protocol.PositionEncodingKindUTF8,
			result.Capabilities.PositionEncoding,
		)
		assert.Equal(t, lsp.ClientInitialized, client.State())
		assert.Equal(t,
			protocol.PositionEncodingKindUTF8,
			client.OffsetEncoding(),
		)
		capabilities, ok := client.Capabilities()
		assert.True(t, ok)
		assert.Equal(t,
			protocol.PositionEncodingKindUTF8,
			capabilities.PositionEncoding,
		)
		assert.True(t, client.SupportsFeature(lsp.FeatureHover))
		assert.False(t, client.SupportsFeature(lsp.FeatureCompletion))
		assert.True(t, waitFor(server.initialized))

		assert.NoError(t, client.Shutdown(ctx))
		assert.Equal(t, lsp.ClientShutdown, client.State())
		assert.True(t, waitFor(server.exited))
	})

	t.Run("sends file operations by server interest", func(t *testing.T) {
		ctx := t.Context()
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()

		server := &lifecycleServer{
			capabilities: fileOperationCapabilities("**/*.go"),
			fileOps:      make(chan string, 2),
			initialized:  make(chan struct{}),
			exited:       make(chan struct{}),
		}
		_, serverRPC, _ := protocol.NewServer(
			ctx, server, jsonrpc2.NewHeaderStream(serverConn),
		)
		defer serverRPC.Close()

		_, client := lsp.NewClient(ctx, clientConn, nil)
		defer client.Close()
		_, err := client.Initialize(
			ctx, lsp.NewInitializeParams(lsp.InitializeConfig{}),
		)
		assert.NoError(t, err)

		path := filepath.Join(t.TempDir(), "main.go")
		_, sent, err := client.WillCreateFile(ctx, path, false)
		assert.NoError(t, err)
		assert.True(t, sent)
		sent, err = client.DidCreateFile(ctx, path, false)
		assert.NoError(t, err)
		assert.True(t, sent)
		_, sent, err = client.WillCreateFile(ctx, "main.txt", false)
		assert.NoError(t, err)
		assert.False(t, sent)

		assert.Equal(t, "willCreate", <-server.fileOps)
		assert.Equal(t, "didCreate", <-server.fileOps)
	})
}

func (s *lifecycleServer) Initialize(
	context.Context, *protocol.InitializeParams,
) (*protocol.InitializeResult, error) {
	hasCaps := s.capabilities.Workspace != nil ||
		s.capabilities.PositionEncoding != ""
	if hasCaps {
		return &protocol.InitializeResult{Capabilities: s.capabilities}, nil
	}
	return &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			PositionEncoding: protocol.PositionEncodingKindUTF8,
			HoverProvider:    protocol.Boolean(true),
		},
	}, nil
}

func (s *lifecycleServer) Initialized(
	context.Context, *protocol.InitializedParams,
) error {
	close(s.initialized)
	return nil
}

func (s *lifecycleServer) Shutdown(context.Context) error {
	return nil
}

func (s *lifecycleServer) Exit(context.Context) error {
	close(s.exited)
	return nil
}

func (s *lifecycleServer) WillCreateFiles(
	context.Context, *protocol.CreateFilesParams,
) (*protocol.WorkspaceEdit, error) {
	s.fileOps <- "willCreate"
	return nil, nil
}

func (s *lifecycleServer) DidCreateFiles(
	context.Context, *protocol.CreateFilesParams,
) error {
	s.fileOps <- "didCreate"
	return nil
}

func fileOperationCapabilities(pattern string) protocol.ServerCapabilities {
	opts := protocol.FileOperationRegistrationOptions{
		Filters: []protocol.FileOperationFilter{{
			Pattern: protocol.FileOperationPattern{Glob: pattern},
		}},
	}
	return protocol.ServerCapabilities{
		PositionEncoding: protocol.PositionEncodingKindUTF8,
		Workspace: &protocol.WorkspaceOptions{
			FileOperations: &protocol.FileOperationOptions{
				WillCreate: opts,
				DidCreate:  opts,
			},
		},
	}
}

func waitFor(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	case <-time.After(waitTimeout):
		return false
	}
}
