package view

type (
	// CompletionItem is a normalized language-server completion candidate
	CompletionItem struct {
		ID        string
		Label     string
		Detail    string
		Filter    string
		Sort      string
		Insert    string
		Kind      string
		Docs      string
		Server    string
		Preselect bool
	}

	// SignatureHelp is a normalized callable signature response
	SignatureHelp struct {
		Signatures []SignatureInformation
		Active     int
	}

	// SignatureInformation describes one callable signature
	SignatureInformation struct {
		Label       string
		Docs        string
		ParamDocs   string
		ActiveStart int
		ActiveEnd   int
	}

	// Location is a normalized language-server target location
	Location struct {
		Path string
		From int
		To   int
	}

	// Symbol is a normalized language-server document or workspace symbol
	Symbol struct {
		Name      string
		Kind      string
		Container string
		Location  Location
	}

	// LanguageServerController controls language-server sessions for commands
	LanguageServerController interface {
		RestartLanguageServers(*Document, []string) ([]string, error)
		StopLanguageServers(*Document, []string) ([]string, error)
		ExecuteWorkspaceCommand(*Document, string, []string) error
		WorkspaceCommands(*Document) []string
		Completions(*Document, Id) ([]CompletionItem, error)
		TriggerCompletions(*Document, Id) ([]CompletionItem, error)
		ResolveCompletion(*Document, Id, CompletionItem) (CompletionItem, error)
		ApplyCompletion(*Document, Id, CompletionItem) error
		Hover(*Document, Id) (string, error)
		SignatureHelp(*Document, Id) (SignatureHelp, error)
		TriggerSignatureHelp(*Document, Id) (SignatureHelp, error)
		GotoDeclaration(*Document, Id) ([]Location, error)
		GotoDefinition(*Document, Id) ([]Location, error)
		GotoTypeDefinition(*Document, Id) ([]Location, error)
		GotoImplementation(*Document, Id) ([]Location, error)
		GotoReference(*Document, Id) ([]Location, error)
		DocumentSymbols(*Document) ([]Symbol, error)
	}
)

// SetLanguageServerController installs the controller that handles language-server requests
func (e *Editor) SetLanguageServerController(c LanguageServerController) {
	e.langServers = c
}

// LanguageServerController returns the installed language-server controller
func (e *Editor) LanguageServerController() LanguageServerController {
	return e.langServers
}
