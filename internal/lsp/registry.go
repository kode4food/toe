package lsp

import (
	"context"
	"errors"
	"fmt"
	"maps"

	"go.lsp.dev/protocol"

	"github.com/kode4food/toe/internal/view/language"
)

// Registry maps named server configurations to running client connections
type Registry struct {
	servers map[string]language.Server
}

var (
	ErrCommandRequired = errors.New("language server command required")
	ErrRequiredRoot    = errors.New("required language server root not found")
	ErrServerNotFound  = errors.New("language server not found")
)

// NewRegistry creates a Registry seeded with the given server configurations
func NewRegistry(servers map[string]language.Server) *Registry {
	r := &Registry{servers: map[string]language.Server{}}
	maps.Copy(r.servers, servers)
	return r
}

// Server returns the configuration for the named language server
func (r *Registry) Server(name string) (language.Server, bool) {
	cfg, ok := r.servers[name]
	return cfg, ok
}

// RegistryStartArgs names the server to launch and the directory to run it in
type RegistryStartArgs struct {
	Name    string
	Dir     string
	Handler protocol.Client
}

// Start launches a named server and returns the resulting client
func (r *Registry) Start(
	ctx context.Context, args RegistryStartArgs,
) (*Client, error) {
	cfg, ok := r.Server(args.Name)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrServerNotFound, args.Name)
	}
	_, client, err := Start(&TransportConfig{
		Context: ctx,
		Name:    args.Name,
		Server:  cfg,
		Dir:     args.Dir,
		Handler: args.Handler,
	})
	if err != nil {
		return nil, err
	}
	return client, nil
}
