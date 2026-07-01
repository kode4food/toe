package lsp

import (
	"context"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

type (
	// Client manages a connection to a single language server process
	Client struct {
		name          string
		cmd           *exec.Cmd
		conn          jsonrpc2.Conn
		server        protocol.Server
		capabilities  protocol.ServerCapabilities
		state         ClientState
		offset        protocol.PositionEncodingKind
		timeout       time.Duration
		processDone   chan struct{}
		processErr    error
		processExited bool
		stderr        *stderrTail
		initialized   bool
		mu            sync.RWMutex
	}

	stderrTail struct {
		sync.RWMutex
		text string
	}

	// ClientState is the lifecycle state of a Client
	ClientState int
)

const maxStderrTail = 4096

// ClientNew is the initial state; ClientInitialized means handshake complete;
// ClientShutdown means closed
const (
	ClientNew ClientState = iota
	ClientInitialized
	ClientShutdown
)

// NewClient dials the server and returns a ready Client
func NewClient(
	ctx context.Context, rwc io.ReadWriteCloser, handler protocol.Client,
) (context.Context, *Client) {
	if handler == nil {
		handler = protocol.UnimplementedClient{}
	}
	ctx, conn, server := protocol.NewClient(
		ctx, handler, jsonrpc2.NewHeaderStream(rwc),
	)
	return ctx, &Client{
		conn:   conn,
		server: server,
		state:  ClientNew,
		offset: protocol.PositionEncodingKindUTF16,
	}
}

func (c *Client) requestContext(
	ctx context.Context,
) (context.Context, context.CancelFunc) {
	c.mu.RLock()
	timeout := c.timeout
	c.mu.RUnlock()
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

// State returns the current lifecycle state of the client
func (c *Client) State() ClientState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// OffsetEncoding returns the position encoding kind negotiated with the server
func (c *Client) OffsetEncoding() protocol.PositionEncodingKind {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.offset
}

// Server returns the underlying protocol server handle
func (c *Client) Server() protocol.Server {
	return c.server
}

// Capabilities returns initialized server capabilities
func (c *Client) Capabilities() (protocol.ServerCapabilities, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.capabilities, c.initialized
}

// Name returns the registered name of the language server
func (c *Client) Name() string {
	return c.name
}

// SupportsFeature reports whether the server supports the given LSP feature
func (c *Client) SupportsFeature(feature Feature) bool {
	capabilities, ok := c.Capabilities()
	if !ok {
		return false
	}
	return SupportsFeature(capabilities, feature)
}

// ExecuteCommand executes a workspace command on the server
func (c *Client) ExecuteCommand(
	ctx context.Context, params *protocol.ExecuteCommandParams,
) error {
	ctx, cancel := c.requestContext(ctx)
	defer cancel()
	_, err := c.server.ExecuteCommand(ctx, params)
	return err
}

func (c *Client) processExitStatus() (bool, string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.processExited, c.stderrText(), c.processErr
}

func (c *Client) processExitedAfter(
	timeout time.Duration,
) (bool, string, error) {
	done := c.processDone
	if done == nil {
		return c.processExitStatus()
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
	}
	return c.processExitStatus()
}

// Initialize performs the LSP initialization handshake with the server
func (c *Client) Initialize(
	ctx context.Context, params *protocol.InitializeParams,
) (*protocol.InitializeResult, error) {
	result, err := c.server.Initialize(ctx, params)
	if err != nil {
		return nil, err
	}
	initialized := &protocol.InitializedParams{}
	if err := c.server.Initialized(ctx, initialized); err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.state = ClientInitialized
	c.capabilities = result.Capabilities
	c.offset = OffsetEncoding(result.Capabilities)
	c.initialized = true
	c.mu.Unlock()
	return result, nil
}

// Shutdown gracefully terminates the server session
func (c *Client) Shutdown(ctx context.Context) error {
	if err := c.server.Shutdown(ctx); err != nil {
		return err
	}
	if err := c.server.Exit(ctx); err != nil {
		return err
	}
	c.mu.Lock()
	c.state = ClientShutdown
	c.mu.Unlock()
	return nil
}

// Close closes the JSON-RPC connection and stops the server process
func (c *Client) Close() error {
	err := c.conn.Close()
	if c.cmd == nil || c.cmd.Process == nil {
		return err
	}
	done := c.processDone
	if done == nil {
		return err
	}
	select {
	case <-done:
		if werr := c.processErrValue(); err == nil {
			err = werr
		}
	default:
		_ = c.cmd.Process.Kill()
		<-done
	}
	return err
}

func (c *Client) markProcessDone(err error) {
	c.mu.Lock()
	c.processErr = err
	c.processExited = true
	c.mu.Unlock()
	if c.processDone != nil {
		close(c.processDone)
	}
}

func (c *Client) processErrValue() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.processErr
}

func (c *Client) stderrText() string {
	if c.stderr == nil {
		return ""
	}
	return c.stderr.String()
}

func (s *stderrTail) Write(b []byte) (int, error) {
	s.Lock()
	defer s.Unlock()
	s.text += string(b)
	if len(s.text) > maxStderrTail {
		s.text = s.text[len(s.text)-maxStderrTail:]
	}
	return len(b), nil
}

func (s *stderrTail) String() string {
	s.RLock()
	defer s.RUnlock()
	return strings.TrimSpace(s.text)
}
