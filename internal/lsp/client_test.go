package lsp_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/kode4food/toe/internal/lsp"
	"github.com/stretchr/testify/assert"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

type lifecycleServer struct {
	protocol.UnimplementedServer
	initialized chan struct{}
	exited      chan struct{}
}

const waitTimeout = time.Second

func TestClient(t *testing.T) {
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
}

func (s *lifecycleServer) Initialize(
	context.Context, *protocol.InitializeParams,
) (*protocol.InitializeResult, error) {
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

func waitFor(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	case <-time.After(waitTimeout):
		return false
	}
}
