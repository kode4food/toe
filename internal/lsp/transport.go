package lsp

import (
	"context"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"go.lsp.dev/protocol"

	"github.com/kode4food/toe/internal/view/language"
)

type (
	// TransportConfig describes a language server process to start
	TransportConfig struct {
		Ctx     context.Context
		Name    string
		Server  language.Server
		Dir     string
		Handler protocol.Client
	}

	pipeConn struct {
		r io.ReadCloser
		w io.WriteCloser
	}
)

func (c *TransportConfig) context() context.Context {
	if c.Ctx == nil {
		return context.Background()
	}
	return c.Ctx
}

func (p pipeConn) Read(b []byte) (int, error) {
	return p.r.Read(b)
}

func (p pipeConn) Write(b []byte) (int, error) {
	return p.w.Write(b)
}

func (p pipeConn) Close() error {
	err := p.r.Close()
	if werr := p.w.Close(); err == nil {
		err = werr
	}
	return err
}

// Start launches the language server process and returns a connected Client
func Start(cfg *TransportConfig) (context.Context, *Client, error) {
	ctx := cfg.context()
	if cfg.Server.Command == "" {
		return ctx, nil, ErrCommandRequired
	}
	root := cfg.Dir
	if root == "" {
		root = "."
	}
	ok, err := RequiredRootFound(root, cfg.Server.RequiredRootPatterns)
	if err != nil {
		return ctx, nil, err
	}
	if !ok {
		return ctx, nil, ErrRequiredRoot
	}
	cmd := exec.Command(cfg.Server.Command, cfg.Server.Args...)
	cmd.Env = commandEnv(cfg.Server.Environment)
	if cfg.Dir != "" {
		cmd.Dir = cfg.Dir
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return ctx, nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return ctx, nil, err
	}
	tail := &stderrTail{}
	cmd.Stderr = tail
	if err := cmd.Start(); err != nil {
		return ctx, nil, err
	}
	ctx, client := NewClient(ctx, pipeConn{r: stdout, w: stdin}, cfg.Handler)
	client.name = cfg.Name
	client.cmd = cmd
	client.timeout = time.Duration(cfg.Server.Timeout) * time.Second
	client.processDone = make(chan struct{})
	client.stderr = tail
	go func() {
		client.markProcessDone(cmd.Wait())
	}()
	return ctx, client, nil
}

func commandEnv(env map[string]string) []string {
	if len(env) == 0 {
		return os.Environ()
	}
	out := make([]string, 0, len(os.Environ())+len(env))
	for _, e := range os.Environ() {
		k, _, _ := strings.Cut(e, "=")
		if _, override := env[k]; !override {
			out = append(out, e)
		}
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		out = append(out, k+"="+env[k])
	}
	return out
}
