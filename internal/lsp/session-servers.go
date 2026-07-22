package lsp

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"sync"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/kode4food/toe/internal/loader"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/language"
)

// serverState owns the attached language-server clients, the language config
// they were started from, and each server's workspace root
type serverState struct {
	sync.RWMutex
	starting  sync.Mutex
	registry  *Registry
	languages map[string]language.Language
	clients   map[string]*Client
	roots     map[string]string
}

// RestartLanguageServers stops and restarts the named servers for the document
func (s *Session) RestartLanguageServers(
	doc *view.Document, names []string,
) ([]string, error) {
	lang, ok := s.languageForDocument(doc)
	if !ok {
		return nil, view.ErrNoLanguageServer
	}
	selected, err := selectLanguageServers(lang, names)
	if err != nil {
		return nil, err
	}
	s.clearDocumentHighlightsForServers(selected)
	s.stopClients(selected)
	for _, name := range selected {
		_, _ = s.ensureClient(name, doc, lang)
	}
	s.notify(doc, (*Client).DidOpen)
	return selected, nil
}

// StopLanguageServers shuts down the named language servers for the document
func (s *Session) StopLanguageServers(
	doc *view.Document, names []string,
) ([]string, error) {
	lang, ok := s.languageForDocument(doc)
	if !ok {
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
		params := &protocol.ExecuteCommandParams{
			Command:   name,
			Arguments: stringsToLSPAny(args),
		}
		return matches[0].ExecuteCommand(s.ctx, params)
	default:
		return fmt.Errorf("%w: %s", view.ErrWorkspaceCommand, name)
	}
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
) (language.Language, bool) {
	if doc == nil {
		return language.Language{}, false
	}
	lang, ok := s.servers.language(doc.Lang())
	if !ok || len(lang.LanguageServers) == 0 {
		return language.Language{}, false
	}
	return lang, true
}

func (s *Session) ensureClient(
	name string, doc *view.Document, lang language.Language,
) (*Client, bool) {
	s.servers.starting.Lock()
	defer s.servers.starting.Unlock()
	if client, ok := s.servers.client(name); ok {
		return client, true
	}
	return s.startClient(name, doc, lang)
}

func (s *Session) startClient(
	name string, doc *view.Document, lang language.Language,
) (*Client, bool) {
	root := s.workspaceRoot(doc, lang)
	handler := &clientHandler{session: s, name: name}
	client, err := s.servers.startRegistry(s.ctx, name, root, handler)
	if err != nil {
		return nil, false
	}
	s.servers.setRoot(name, root)
	params := NewInitializeParams(InitializeConfig{WorkspaceRoot: root})
	if _, err := client.Initialize(s.ctx, params); err != nil {
		_ = client.Close()
		return nil, false
	}
	s.servers.setClient(name, client)
	return client, true
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
	client, ok := s.servers.client(provider)
	if !ok {
		return protocol.PositionEncodingKindUTF16
	}
	return client.OffsetEncoding()
}

func (s *Session) workspaceRoot(
	doc *view.Document, lang language.Language,
) string {
	workspace, ok := ResolveWorkspace(WorkspaceRequest{
		FilePath:       doc.Path(),
		Workspace:      s.cwd,
		WorkspaceIsCWD: true,
		RootMarkers:    lang.Roots,
	})
	if ok {
		return workspace
	}
	if dir := filepath.Dir(doc.Path()); dir != "." {
		return dir
	}
	return s.cwd
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

func (s *serverState) client(name string) (*Client, bool) {
	s.RLock()
	defer s.RUnlock()
	client, ok := s.clients[name]
	return client, ok
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

func (s *serverState) setRoot(name, root string) {
	s.Lock()
	defer s.Unlock()
	s.roots[name] = root
}

func (s *serverState) startRegistry(
	ctx context.Context, name, root string, handler *clientHandler,
) (*Client, error) {
	s.RLock()
	defer s.RUnlock()
	return s.registry.Start(ctx, name, root, handler)
}

func (s *serverState) language(name string) (language.Language, bool) {
	s.RLock()
	defer s.RUnlock()
	lang, ok := s.languages[name]
	return lang, ok
}

func (s *serverState) languageID(name string) string {
	lang, ok := s.language(name)
	if !ok {
		return ""
	}
	return lang.LanguageID
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
	s.languages = languagesByName(langs)
	s.clients = map[string]*Client{}
	s.roots = map[string]string{}
	return clients
}

func loadLanguages(cwd string) language.Languages {
	global, ok := loader.LanguagesFile()
	if !ok {
		global = ""
	}
	workspace := loader.WorkspaceLanguagesFile(cwd)
	langs, ok := language.LoadLanguagesForWorkspace(global, workspace, cwd)
	if !ok {
		return language.Languages{}
	}
	return langs
}

func languagesByName(langs language.Languages) map[string]language.Language {
	out := make(map[string]language.Language, len(langs.Languages))
	for _, lang := range langs.Languages {
		out[lang.Name] = lang
	}
	return out
}

func serverNames(features []language.ServerFeatures) []string {
	out := make([]string, 0, len(features))
	seen := map[string]bool{}
	for _, feature := range features {
		if feature.Name == "" || seen[feature.Name] {
			continue
		}
		seen[feature.Name] = true
		out = append(out, feature.Name)
	}
	return out
}

func selectLanguageServers(
	lang language.Language, requested []string,
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
	capabilities, ok := client.Capabilities()
	if !ok {
		return false
	}
	return slices.Contains(capabilities.ExecuteCommandProvider.Commands, name)
}

func stringsToLSPAny(args []string) []protocol.LSPAny {
	if len(args) == 0 {
		return nil
	}
	out := make([]protocol.LSPAny, len(args))
	for i, arg := range args {
		b, _ := json.Marshal(arg)
		out[i] = b
	}
	return out
}
