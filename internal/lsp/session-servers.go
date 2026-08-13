package lsp

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/language"
)

type (
	// serverState owns the attached language-server clients, the language
	// config they were started from, and each server's workspace root
	serverState struct {
		sync.RWMutex
		starting  sync.Mutex
		registry  *Registry
		languages language.Languages
		clients   map[string]*Client
		roots     map[string]string
	}

	initOptionsFunc func(string) (protocol.LSPAny, error)
)

const typescriptServerName = "typescript-language-server"

var langInitOptions = map[string]initOptionsFunc{
	typescriptServerName: tsInitOptions,
}

// RestartLanguageServers stops and restarts the named servers for the document
func (s *Session) RestartLanguageServers(
	doc *view.Document, names []string,
) ([]string, error) {
	lang := s.languageForDocument(doc)
	if lang == nil {
		return nil, view.ErrNoLanguageServer
	}
	selected, err := selectLanguageServers(lang, names)
	if err != nil {
		return nil, err
	}
	s.clearDocumentHighlightsForServers(selected)
	s.stopClients(selected)
	for _, name := range selected {
		if _, err := s.ensureClient(name, doc, lang); err != nil {
			return nil, err
		}
	}
	s.notify(doc, (*Client).DidOpen)
	return selected, nil
}

// StopLanguageServers shuts down the named language servers for the document
func (s *Session) StopLanguageServers(
	doc *view.Document, names []string,
) ([]string, error) {
	lang := s.languageForDocument(doc)
	if lang == nil {
		return nil, view.ErrNoLanguageServer
	}
	selected, err := selectLanguageServers(lang, names)
	if err != nil {
		return nil, err
	}
	s.clearDocumentHighlightsForServers(selected)
	s.stopClients(selected)
	return selected, nil
}

// ExecuteWorkspaceCommand runs a named workspace command on the matching server
func (s *Session) ExecuteWorkspaceCommand(
	doc *view.Document, name string, args []string,
) error {
	clients := s.clientsForDocument(doc)
	var matches []*Client
	for _, client := range clients {
		if clientSupportsCommand(client, name) {
			matches = append(matches, client)
		}
	}
	switch len(matches) {
	case 0:
		return fmt.Errorf("%w: %s", view.ErrWorkspaceCommand, name)
	case 1:
		values, err := stringsToLSPAny(args)
		if err != nil {
			return err
		}
		params := &protocol.ExecuteCommandParams{
			Command:   name,
			Arguments: values,
		}
		return matches[0].ExecuteCommand(s.ctx, params)
	default:
		return fmt.Errorf("%w: %s", view.ErrWorkspaceCommand, name)
	}
}

// LanguageServerNames returns the servers configured for the document
func (s *Session) LanguageServerNames(doc *view.Document) []string {
	lang := s.languageForDocument(doc)
	if lang == nil {
		return nil
	}
	return serverNames(lang.LanguageServers)
}

// WorkspaceCommands returns commands advertised by attached servers
func (s *Session) WorkspaceCommands(doc *view.Document) []string {
	clients := s.clientsForDocument(doc)
	var out []string
	for _, client := range clients {
		capabilities, ok := client.Capabilities()
		if !ok || len(capabilities.ExecuteCommandProvider.Commands) == 0 {
			continue
		}
		out = append(out, capabilities.ExecuteCommandProvider.Commands...)
	}
	return out
}

func (s *Session) languageForDocument(
	doc *view.Document,
) *language.Language {
	if doc == nil {
		return nil
	}
	lang := s.servers.language(doc.Lang())
	if lang == nil || len(lang.LanguageServers) == 0 {
		return nil
	}
	return lang
}

func (s *Session) ensureClient(
	name string, doc *view.Document, lang *language.Language,
) (*Client, error) {
	target := &workspaceServer{name: name, root: s.workspaceRoot(doc, lang)}
	client, _, err := s.ensureServer(target)
	return client, err
}

func (s *Session) ensureServer(target *workspaceServer) (*Client, bool, error) {
	s.servers.starting.Lock()
	defer s.servers.starting.Unlock()
	if client := s.servers.client(target.name); client != nil {
		return client, false, nil
	}
	client, err := s.startClient(target)
	return client, err == nil, err
}

func (s *Session) startClient(target *workspaceServer) (*Client, error) {
	options, err := constructInitOptions(target)
	if err != nil {
		return nil, err
	}
	handler := &clientHandler{session: s, name: target.name}
	client, err := s.servers.startRegistry(s.ctx, target, handler)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", target.name, err)
	}
	s.servers.setRoot(target)
	params := NewInitializeParams(InitializeConfig{
		WorkspaceRoot:         target.root,
		InitializationOptions: options,
	})
	if _, err := client.Initialize(s.ctx, params); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("%s: %w", target.name, err)
	}
	s.servers.setClient(target.name, client)
	return client, nil
}

func (s *Session) workspaceFolders() []protocol.WorkspaceFolder {
	roots := s.servers.allRoots()
	out := make([]protocol.WorkspaceFolder, 0, len(roots))
	for _, root := range roots {
		out = append(out, protocol.WorkspaceFolder{
			URI:  uri.File(root),
			Name: filepath.Base(root),
		})
	}
	slices.SortFunc(out, func(a, b protocol.WorkspaceFolder) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return out
}

func (s *Session) offsetForProvider(
	provider string,
) protocol.PositionEncodingKind {
	if client := s.servers.client(provider); client != nil {
		return client.OffsetEncoding()
	}
	return protocol.PositionEncodingKindUTF16
}

func (s *Session) workspaceRoot(
	doc *view.Document, lang *language.Language,
) string {
	if root, ok := s.resolveWorkspaceRoot(doc.Path(), lang); ok {
		return root
	}
	if dir := filepath.Dir(doc.Path()); dir != "." {
		return dir
	}
	return s.cwd
}

func (s *Session) resolveWorkspaceRoot(
	path string, lang *language.Language,
) (string, bool) {
	return ResolveWorkspace(WorkspaceRequest{
		FilePath:       path,
		Workspace:      s.cwd,
		WorkspaceIsCWD: true,
		RootMarkers:    lang.Roots,
	})
}

func (s *Session) stopClients(names []string) {
	for _, client := range s.servers.removeNamed(names) {
		_ = client.Close()
	}
}

func (s *Session) dropClient(name string, client *Client) {
	s.clearDocumentHighlightsForServers([]string{name})
	s.servers.removeIfCurrent(name, client)
	_ = client.Close()
}

func (s *serverState) client(name string) *Client {
	s.RLock()
	defer s.RUnlock()
	return s.clients[name]
}

func (s *serverState) allClients() []*Client {
	s.RLock()
	defer s.RUnlock()
	out := make([]*Client, 0, len(s.clients))
	for _, client := range s.clients {
		out = append(out, client)
	}
	return out
}

func (s *serverState) allRoots() map[string]string {
	s.RLock()
	defer s.RUnlock()
	return maps.Clone(s.roots)
}

func (s *serverState) setClient(name string, client *Client) {
	s.Lock()
	defer s.Unlock()
	s.clients[name] = client
}

func (s *serverState) setRoot(target *workspaceServer) {
	s.Lock()
	defer s.Unlock()
	s.roots[target.name] = target.root
}

func (s *serverState) startRegistry(
	ctx context.Context, target *workspaceServer, handler *clientHandler,
) (*Client, error) {
	s.RLock()
	defer s.RUnlock()
	return s.registry.Start(ctx, RegistryStartArgs{
		Name:    target.name,
		Dir:     target.root,
		Handler: handler,
	})
}

func (s *serverState) language(name string) *language.Language {
	s.RLock()
	defer s.RUnlock()
	for i := range s.languages.Languages {
		lang := &s.languages.Languages[i]
		if lang.Name == name {
			return lang
		}
	}
	return nil
}

func (s *serverState) languageConfig() language.Languages {
	s.RLock()
	defer s.RUnlock()
	return s.languages
}

func (s *serverState) languageID(name string) string {
	if lang := s.language(name); lang != nil {
		return lang.LanguageID
	}
	return ""
}

func (s *serverState) removeNamed(names []string) []*Client {
	s.Lock()
	defer s.Unlock()
	out := make([]*Client, 0, len(names))
	for _, name := range names {
		client := s.clients[name]
		if client == nil {
			continue
		}
		out = append(out, client)
		delete(s.clients, name)
	}
	return out
}

func (s *serverState) removeIfCurrent(name string, client *Client) {
	s.Lock()
	defer s.Unlock()
	if s.clients[name] == client {
		delete(s.clients, name)
	}
}

// reset replaces the server fleet for a config reload and returns the clients
// that were running, so the caller can close them outside the lock
func (s *serverState) reset(langs language.Languages) []*Client {
	s.Lock()
	defer s.Unlock()
	clients := make([]*Client, 0, len(s.clients))
	for _, client := range s.clients {
		clients = append(clients, client)
	}
	s.registry = NewRegistry(langs.LanguageServers)
	s.languages = langs
	s.clients = map[string]*Client{}
	s.roots = map[string]string{}
	return clients
}

func loadLanguages(cwd string) language.Languages {
	global := ""
	if path, ok := loader.LanguagesFile(); ok {
		global = path
	}
	workspace := loader.WorkspaceLanguagesFile(cwd)
	langs, ok := language.LoadLanguagesForWorkspace(loader.WorkspaceFiles{
		Global:    global,
		Workspace: workspace,
		Dir:       cwd,
	})
	if !ok {
		return language.Languages{}
	}
	return langs
}

func serverNames(names []string) []string {
	out := make([]string, 0, len(names))
	seen := map[string]bool{}
	for _, name := range names {
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

func selectLanguageServers(
	lang *language.Language, requested []string,
) ([]string, error) {
	names := serverNames(lang.LanguageServers)
	if len(requested) == 0 {
		return names, nil
	}
	valid := make(map[string]bool, len(names))
	for _, name := range names {
		valid[name] = true
	}
	out := make([]string, 0, len(requested))
	for _, name := range requested {
		if !valid[name] {
			return nil, fmt.Errorf("%w: %s",
				view.ErrUnknownLanguageServer, name,
			)
		}
		out = append(out, name)
	}
	return out, nil
}

func clientSupportsCommand(client *Client, name string) bool {
	if capabilities, ok := client.Capabilities(); ok {
		return slices.Contains(
			capabilities.ExecuteCommandProvider.Commands, name,
		)
	}
	return false
}

func stringsToLSPAny(args []string) ([]protocol.LSPAny, error) {
	if len(args) == 0 {
		return nil, nil
	}
	out := make([]protocol.LSPAny, len(args))
	for i, arg := range args {
		value, err := protocol.Marshal(arg)
		if err != nil {
			return nil, err
		}
		out[i] = value
	}
	return out, nil
}

func constructInitOptions(target *workspaceServer) (protocol.LSPAny, error) {
	if init, ok := langInitOptions[target.name]; ok {
		return init(target.root)
	}
	return nil, nil
}

func tsInitOptions(root string) (protocol.LSPAny, error) {
	for _, rel := range []string{
		"node_modules/typescript/lib/tsserver.js",
		".vscode/pnpify/typescript/lib/tsserver.js",
		".yarn/sdks/typescript/lib/tsserver.js",
	} {
		path := filepath.Join(root, rel)
		if _, err := os.Stat(path); err == nil {
			return protocol.Marshal(map[string]any{
				"tsserver": map[string]string{"path": path},
			})
		}
	}
	return nil, nil
}
