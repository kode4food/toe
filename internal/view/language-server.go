package view

import (
	"errors"

	"github.com/kode4food/toe/internal/core"
)

type (
	// CompletionResult is a normalized language-server completion response
	CompletionResult struct {
		Items      []*CompletionItem
		Incomplete bool
	}

	// CompletionItem is a normalized language-server completion candidate
	CompletionItem struct {
		ID     string
		Server string
		Kind   string

		Label            string
		LabelDetail      string
		LabelDescription string
		Detail           string
		Docs             string

		Filter string
		Sort   string
		Insert string

		Preselect  bool
		Deprecated bool
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

	// Location is a normalized language-server target location, holding the
	// server's own positions so listing never reads the files; ResolveRange
	// converts them against a document at jump time
	Location struct {
		Path     string
		From     ServerPosition
		To       ServerPosition
		Encoding PositionEncoding
	}

	// ServerPosition is a zero-based line and in-line character offset, the
	// character counted in the server's PositionEncoding
	ServerPosition struct {
		Line      int
		Character int
	}

	// PositionEncoding names the units a language server counts a line's
	// characters in
	PositionEncoding int

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
		Completions(*Document, Id) (CompletionResult, error)
		TriggerCompletions(*Document, Id) (CompletionResult, error)
		ResolveCompletion(
			*Document, Id, *CompletionItem,
		) (*CompletionItem, error)
		ApplyCompletion(*Document, Id, *CompletionItem) error
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
		Busy() bool
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

// Position encodings a language server may count line characters in; UTF-16 is
// the protocol default
const (
	PositionEncodingUTF16 PositionEncoding = iota
	PositionEncodingUTF8
	PositionEncodingUTF32
)

var (
	// ErrNoLanguageServer reports that no language server is configured for a
	// document's language
	ErrNoLanguageServer = errors.New("LSP not defined for document")

	// ErrUnknownLanguageServer reports that a named language server was not
	// found among the document's configured servers
	ErrUnknownLanguageServer = errors.New("unknown language server")

	// ErrWorkspaceCommand reports that a requested workspace command is not
	// offered by any configured language server
	ErrWorkspaceCommand = errors.New("workspace command unavailable")

	// ErrFormatSelection reports that range formatting cannot be performed for
	// the current selection
	ErrFormatSelection = errors.New("format selection unsupported")
)

// ResolveRange converts the location's server positions into a character range
// in text, reversed so the cursor lands on the start of the target
func (l Location) ResolveRange(text core.Rope) (core.Range, bool) {
	from, ok := l.From.Resolve(text, l.Encoding)
	if !ok {
		return core.Range{}, false
	}
	to, ok := l.To.Resolve(text, l.Encoding)
	if !ok {
		to = from
	}
	return core.NewRange(to, from), true
}

// Resolve converts a server position into a character offset in text
func (p ServerPosition) Resolve(
	text core.Rope, encoding PositionEncoding,
) (int, bool) {
	lineStart, err := text.LineToChar(p.Line)
	if err != nil {
		return 0, false
	}
	lineEnd, err := text.LineEndCharIndex(p.Line)
	if err != nil {
		return 0, false
	}
	line, err := text.SliceString(lineStart, lineEnd)
	if err != nil {
		return 0, false
	}
	chars, ok := encoding.charsToOffset(line, p.Character)
	if !ok {
		return 0, false
	}
	return lineStart + chars, true
}

// RuneLen reports how many encoding units ch occupies
func (e PositionEncoding) RuneLen(ch rune) int {
	switch e {
	case PositionEncodingUTF8:
		return len(string(ch))
	case PositionEncodingUTF32:
		return 1
	default:
		if ch > 0xffff {
			return 2
		}
		return 1
	}
}

// SetLanguageServerController installs the language-server request handler
func (e *Editor) SetLanguageServerController(c LanguageServerController) {
	e.langServers = c
}

// LanguageServerController returns the installed language-server controller
func (e *Editor) LanguageServerController() LanguageServerController {
	return e.langServers
}

// charsToOffset converts an in-line encoded offset into a rune count
func (e PositionEncoding) charsToOffset(line string, target int) (int, bool) {
	units := 0
	chars := 0
	for _, ch := range line {
		if units == target {
			return chars, true
		}
		units += e.RuneLen(ch)
		chars++
		if units > target {
			return 0, false
		}
	}
	if units == target {
		return chars, true
	}
	return 0, false
}
