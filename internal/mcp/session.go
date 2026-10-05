// Package mcp hosts an in-process Model Context Protocol server that gives
// Claude Code the same IDE loop it has in VS Code or GoLand: diagnostics flow
// to Claude, and Claude's edits open as a diff the user accepts or rejects.
// Claude auto-connects over SSE via a lockfile under ~/.claude/ide
package mcp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/kode4food/toe/internal/view"
)

type (
	// Session owns the MCP server bound to an editor
	Session struct {
		editor     *view.Editor
		srv        *mcp.Server
		http       *http.Server
		listenAddr string
		boundAddr  string
		lockfile   string
		enabled    bool
		dispatch   func(func())
		showDiff   func(context.Context, DiffRequest) bool
		closeDiff  func(tabName string)

		mu      sync.Mutex
		pending map[string]chan struct{}
	}

	// DiffRequest describes a proposed file change for editor review
	DiffRequest struct {
		TabName     string
		Path        string
		NewContents string
	}
)

const (
	serverName = "toe"
	// HTTPPort is the fixed loopback port the MCP server binds
	// ponytail: fixed port. A second toe instance can't bind it and runs
	// without MCP, so use a per-workspace port if multi-instance matters
	HTTPPort = 7420
)

var _ view.IDEServer = (*Session)(nil)

// Attach creates the MCP server for the editor over SSE on addr and starts it.
// Disable or re-enable it later with SetEnabled
func Attach(_ context.Context, e *view.Editor, addr string) *Session {
	s := &Session{editor: e, listenAddr: addr}
	s.srv = mcp.NewServer(
		&mcp.Implementation{Name: serverName, Version: "0.1.0"}, nil,
	)
	registerTools(s, s.srv)
	s.start()
	return s
}

// Enabled reports whether the MCP server is currently serving
func (s *Session) Enabled() bool { return s.enabled }

// SetEnabled starts or stops the MCP server
func (s *Session) SetEnabled(enabled bool) {
	if enabled == s.enabled {
		return
	}
	if enabled {
		s.start()
	} else {
		s.stop()
	}
}

// Port is the loopback port the server listens on
func (s *Session) Port() int { return s.port() }

// URL is the MCP endpoint clients connect to, empty while the server is stopped
func (s *Session) URL() string {
	if s.boundAddr == "" {
		return ""
	}
	return "http://" + s.boundAddr + "/sse"
}

// SetDispatcher installs the function that runs editor access on the Bubble Tea
// loop. Until it is set, access runs inline on the caller's goroutine
func (s *Session) SetDispatcher(d func(func())) { s.dispatch = d }

// SetDiffHandlers installs the hooks that show and close a read-only diff
// review in the editor UI, wired by the app so mcp stays decoupled from the UI
func (s *Session) SetDiffHandlers(
	show func(context.Context, DiffRequest) bool, close func(tabName string),
) {
	s.showDiff = show
	s.closeDiff = close
}

// Close shuts the MCP server down and removes the lockfile
func (s *Session) Close() error {
	s.stop()
	return nil
}

// start begins serving and writes the lockfile; a no-op if already running
func (s *Session) start() {
	if s.http != nil {
		return
	}
	ln, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		slog.Error("mcp server not started", "err", err)
		return
	}
	s.boundAddr = ln.Addr().String()
	srv := &http.Server{Handler: newMux(s.srv)}
	s.http = srv
	go func() {
		if err := srv.Serve(ln); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			slog.Error("mcp server stopped", "err", err)
		}
	}()
	path, err := writeLockfile(s.port(), s.editor.Cwd(), newToken())
	if err != nil {
		slog.Warn("mcp lockfile not written", "err", err)
	} else {
		s.lockfile = path
	}
	s.enabled = true
}

// stop closes the server and removes the lockfile; a no-op if already stopped
func (s *Session) stop() {
	removeLockfile(s.lockfile)
	s.lockfile = ""
	if s.http != nil {
		_ = s.http.Close()
		s.http = nil
	}
	s.boundAddr = ""
	s.enabled = false
}

// onEditor runs fn against the editor on the UI loop and returns its result,
// keeping all editor access on the single Bubble Tea goroutine
func (s *Session) onEditor(fn func(*view.Editor) any) any {
	if s.dispatch == nil {
		return fn(s.editor)
	}
	done := make(chan any, 1)
	s.dispatch(func() { done <- fn(s.editor) })
	return <-done
}

func (s *Session) port() int {
	addr := s.boundAddr
	if addr == "" {
		addr = s.listenAddr
	}
	if _, p, err := net.SplitHostPort(addr); err == nil {
		if n, err := net.LookupPort("tcp", p); err == nil {
			return n
		}
	}
	return HTTPPort
}

// awaitDiffClose shows the diff and blocks until the tab closes or ctx is
// cancelled, so Claude gates on its own prompt rather than auto-applying
func (s *Session) awaitDiffClose(ctx context.Context, diff DiffRequest) bool {
	done := make(chan struct{})
	s.mu.Lock()
	if s.pending == nil {
		s.pending = map[string]chan struct{}{}
	}
	s.pending[diff.TabName] = done
	s.mu.Unlock()
	if s.showDiff == nil || !s.showDiff(ctx, diff) {
		s.mu.Lock()
		delete(s.pending, diff.TabName)
		s.mu.Unlock()
		return false
	}

	select {
	case <-done:
	case <-ctx.Done():
	}

	s.mu.Lock()
	delete(s.pending, diff.TabName)
	s.mu.Unlock()
	if s.closeDiff != nil {
		s.closeDiff(diff.TabName)
	}
	return true
}

// signalDiffClosed unblocks an awaitDiffClose waiting on tabName
func (s *Session) signalDiffClosed(tabName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tabName == "" {
		for name, ch := range s.pending {
			close(ch)
			delete(s.pending, name)
		}
		return
	}
	if ch, ok := s.pending[tabName]; ok {
		close(ch)
		delete(s.pending, tabName)
	}
}

func newMux(srv *mcp.Server) *http.ServeMux {
	get := func(*http.Request) *mcp.Server { return srv }
	mux := http.NewServeMux()
	mux.Handle("/sse", mcp.NewSSEHandler(get, nil))
	return mux
}

func newToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
