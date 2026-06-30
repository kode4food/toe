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

	// CodeAction is a normalized language-server action or command
	CodeAction struct {
		ID        string
		Title     string
		Kind      string
		Server    string
		Preferred bool
	}

	// DocumentHighlight is a normalized same-document symbol highlight
	DocumentHighlight struct {
		From int
		To   int
	}

	// DocumentLink is a normalized link range in a document
	DocumentLink struct {
		ID     string
		From   int
		To     int
		Target string
		Server string
	}

	// DocumentColor is a normalized color range in a document
	DocumentColor struct {
		From  int
		To    int
		Red   uint8
		Green uint8
		Blue  uint8
	}

	// InlayHint is a normalized language-server hint at a document position
	InlayHint struct {
		Pos          int
		Label        string
		Kind         string
		PaddingLeft  bool
		PaddingRight bool
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
		RenameSymbolPrefill(*Document, Id) (string, error)
		RenameSymbol(*Document, Id, string) error
		CodeActions(*Document, Id) ([]CodeAction, error)
		ApplyCodeAction(*Document, Id, CodeAction) error
		DocumentHighlights(*Document, Id) ([]DocumentHighlight, error)
		DocumentLinks(*Document) ([]DocumentLink, error)
		ResolveDocumentLink(*Document, DocumentLink) (DocumentLink, error)
		FormatDocument(*Document, Id) error
		FormatSelection(*Document, Id) error
		DocumentSymbols(*Document) ([]Symbol, error)
		WorkspaceSymbols(*Document, string) ([]Symbol, error)
	}

	// FileOperationController handles user-initiated filesystem operations
	// for language-server clients interested in workspace file operations
	FileOperationController interface {
		WillCreateFile(path string, dir bool) error
		DidCreateFile(path string, dir bool) error
		WillRenameFile(oldPath, newPath string, dir bool) error
		DidRenameFile(oldPath, newPath string, dir bool) error
		WillDeleteFile(path string, dir bool) error
		DidDeleteFile(path string, dir bool) error
	}
)

// SetLanguageServerController installs the language-server request handler
func (e *Editor) SetLanguageServerController(c LanguageServerController) {
	e.langServers = c
}

// LanguageServerController returns the installed language-server controller
func (e *Editor) LanguageServerController() LanguageServerController {
	return e.langServers
}
