package lsp

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sync"

	"github.com/kode4food/toe/internal/view/language"
	"go.lsp.dev/protocol"
)

type (
	// ClientID identifies a registered language server client in a Registry
	ClientID int

	// Registry maps named server configurations to running client connections
	Registry struct {
		servers map[string]language.Server
		clients map[ClientID]*Client
		byName  map[string][]ClientID
		nextID  ClientID
		mu      sync.Mutex
	}
)

var (
	ErrCommandRequired = errors.New("language server command required")
	ErrRequiredRoot    = errors.New("required language server root not found")
	ErrServerExists    = errors.New("language server already registered")
	ErrServerNotFound  = errors.New("language server not found")
)

// NewRegistry creates a Registry seeded with the given server configurations
func NewRegistry(servers map[string]language.Server) *Registry {
	r := &Registry{
		servers: map[string]language.Server{},
		clients: map[ClientID]*Client{},
		byName:  map[string][]ClientID{},
	}
	maps.Copy(r.servers, servers)
	return r
}

// Register adds a named server configuration to the registry
func (r *Registry) Register(name string, cfg language.Server) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.servers[name]; ok {
		return fmt.Errorf("%w: %s", ErrServerExists, name)
	}
	r.servers[name] = cfg
	return nil
}

// Server returns the configuration for the named language server
func (r *Registry) Server(name string) (language.Server, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cfg, ok := r.servers[name]
	return cfg, ok
}

// Start launches a named server and registers the resulting client
func (r *Registry) Start(
	ctx context.Context, name, dir string, handler protocol.Client,
) (ClientID, *Client, error) {
	cfg, ok := r.Server(name)
	if !ok {
		return 0, nil, fmt.Errorf("%w: %s", ErrServerNotFound, name)
	}
	_, client, err := Start(ctx, name, cfg, dir, handler)
	if err != nil {
		return 0, nil, err
	}
	id := r.addClient(name, client)
	return id, client, nil
}

// Client returns the registered client with the given ID
func (r *Registry) Client(id ClientID) (*Client, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	client, ok := r.clients[id]
	return client, ok
}

// Clients returns all registered clients for the named server
func (r *Registry) Clients(name string) []*Client {
	r.mu.Lock()
	defer r.mu.Unlock()
	ids := r.byName[name]
	out := make([]*Client, 0, len(ids))
	for _, id := range ids {
		if client, ok := r.clients[id]; ok {
			out = append(out, client)
		}
	}
	return out
}

func (r *Registry) addClient(name string, client *Client) ClientID {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	id := r.nextID
	r.clients[id] = client
	r.byName[name] = append(r.byName[name], id)
	return id
}
