package files_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/lsp"
	"github.com/kode4food/toe/internal/term/builtin/test"
	"github.com/kode4food/toe/internal/view"
)

// stubController is a minimal LanguageServerController for format tests
type stubController struct {
	formatDocErr error
	formatSelErr error
}

func (s *stubController) FormatDocument(_ *view.Document, _ view.Id) error {
	return s.formatDocErr
}

func (s *stubController) FormatSelection(_ *view.Document, _ view.Id) error {
	return s.formatSelErr
}

func (s *stubController) RestartLanguageServers(
	_ *view.Document, _ []string,
) ([]string, error) {
	return nil, nil
}

func (s *stubController) StopLanguageServers(
	_ *view.Document, _ []string,
) ([]string, error) {
	return nil, nil
}

func (s *stubController) ExecuteWorkspaceCommand(
	_ *view.Document, _ string, _ []string,
) error {
	return nil
}

func (s *stubController) WorkspaceCommands(_ *view.Document) []string {
	return nil
}

func (s *stubController) Completions(
	_ *view.Document, _ view.Id,
) (view.CompletionResult, error) {
	return view.CompletionResult{}, nil
}

func (s *stubController) TriggerCompletions(
	_ *view.Document, _ view.Id,
) (view.CompletionResult, error) {
	return view.CompletionResult{}, nil
}

func (s *stubController) ResolveCompletion(
	_ *view.Document, _ view.Id, item view.CompletionItem,
) (view.CompletionItem, error) {
	return item, nil
}

func (s *stubController) ApplyCompletion(
	_ *view.Document, _ view.Id, _ view.CompletionItem,
) error {
	return nil
}

func (s *stubController) Hover(_ *view.Document, _ view.Id) (string, error) {
	return "", nil
}

func (s *stubController) SignatureHelp(
	_ *view.Document, _ view.Id,
) (view.SignatureHelp, error) {
	return view.SignatureHelp{}, nil
}

func (s *stubController) TriggerSignatureHelp(
	_ *view.Document, _ view.Id,
) (view.SignatureHelp, error) {
	return view.SignatureHelp{}, nil
}

func (s *stubController) GotoDeclaration(
	_ *view.Document, _ view.Id,
) ([]view.Location, error) {
	return nil, nil
}

func (s *stubController) GotoDefinition(
	_ *view.Document, _ view.Id,
) ([]view.Location, error) {
	return nil, nil
}

func (s *stubController) GotoTypeDefinition(
	_ *view.Document, _ view.Id,
) ([]view.Location, error) {
	return nil, nil
}

func (s *stubController) GotoImplementation(
	_ *view.Document, _ view.Id,
) ([]view.Location, error) {
	return nil, nil
}

func (s *stubController) GotoReference(
	_ *view.Document, _ view.Id,
) ([]view.Location, error) {
	return nil, nil
}

func (s *stubController) RenameSymbolPrefill(
	_ *view.Document, _ view.Id,
) (string, error) {
	return "", nil
}

func (s *stubController) RenameSymbol(
	_ *view.Document, _ view.Id, _ string,
) error {
	return nil
}

func (s *stubController) CodeActions(
	_ *view.Document, _ view.Id,
) ([]view.CodeAction, error) {
	return nil, nil
}

func (s *stubController) ApplyCodeAction(
	_ *view.Document, _ view.Id, _ view.CodeAction,
) error {
	return nil
}

func (s *stubController) DocumentHighlights(
	_ *view.Document, _ view.Id,
) ([]view.DocumentHighlight, error) {
	return nil, nil
}

func (s *stubController) DocumentLinks(
	_ *view.Document,
) ([]view.DocumentLink, error) {
	return nil, nil
}

func (s *stubController) ResolveDocumentLink(
	_ *view.Document, link view.DocumentLink,
) (view.DocumentLink, error) {
	return link, nil
}

func (s *stubController) DocumentSymbols(
	_ *view.Document,
) ([]view.Symbol, error) {
	return nil, nil
}

func (s *stubController) WorkspaceSymbols(
	_ *view.Document, _ string,
) ([]view.Symbol, error) {
	return nil, nil
}

func (s *stubController) Busy() bool {
	return false
}

func TestFormatCommands(t *testing.T) {
	t.Run("format on plain text reports no formatter", func(t *testing.T) {
		e, km := test.Env(t, "hello\n")
		res := test.RunCmd(t, km, e, "format")
		assert.Contains(t, res.Message, "no formatter")
	})

	t.Run("reflow runs without panic", func(t *testing.T) {
		e, km := test.Env(t, "hello world\n")
		test.RunCmd(t, km, e, "reflow")
	})

	t.Run("sort runs without panic", func(t *testing.T) {
		e, km := test.Env(t, "b\na\nc\n")
		test.RunCmd(t, km, e, "sort")
	})

	t.Run("format_selections runs", func(t *testing.T) {
		e, km := test.Env(t, "  hello\n")
		test.RunCmd(t, km, e, "format_selections")
	})
}

func TestFormatWithFormatter(t *testing.T) {
	t.Run("formatter noop returns no message", func(t *testing.T) {
		writeFormatterConfig(t, `command = "cat"`)
		e, km := test.Env(t, "hello\n")
		res := test.RunCmd(t, km, e, "format")
		assert.Equal(t, "", res.Message)
	})

	t.Run("formatter changes text applies diff", func(t *testing.T) {
		writeFormatterConfig(t, `command = "tr"
args = ["a-z", "A-Z"]`)
		e, km := test.Env(t, "hello\n")
		res := test.RunCmd(t, km, e, "format")
		assert.Equal(t, "", res.Message)
		assert.Equal(t, "HELLO\n", test.DocText(t, e))
	})

	t.Run("failing formatter reports error", func(t *testing.T) {
		writeFormatterConfig(t, `command = "false"`)
		e, km := test.Env(t, "hello\n")
		res := test.RunCmd(t, km, e, "format")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("failing formatter reports stderr", func(t *testing.T) {
		writeFormatterConfig(t, `command = "sh"
args = ["-c", "echo nope >&2; exit 1"]`)
		e, km := test.Env(t, "hello\n")
		res := test.RunCmd(t, km, e, "format")
		assert.Contains(t, res.Message, "nope")
	})
}

func TestLSPFormatCommands(t *testing.T) {
	t.Run("lsp format succeeds", func(t *testing.T) {
		e, km := test.Env(t, "hello\n")
		e.SetLanguageServerController(&stubController{})
		res := test.RunCmd(t, km, e, "format")
		assert.Equal(t, "", res.Message)
	})

	t.Run("ErrNoLanguageServer no formatter", func(t *testing.T) {
		e, km := test.Env(t, "hello\n")
		e.SetLanguageServerController(&stubController{
			formatDocErr: lsp.ErrNoLanguageServer,
		})
		res := test.RunCmd(t, km, e, "format")
		assert.Contains(t, res.Message, "no formatter")
	})

	t.Run("lsp format other error reports error", func(t *testing.T) {
		e, km := test.Env(t, "hello\n")
		e.SetLanguageServerController(&stubController{
			formatDocErr: errors.New("boom"),
		})
		res := test.RunCmd(t, km, e, "format")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("lsp format_selection succeeds", func(t *testing.T) {
		e, km := test.Env(t, "hello\n")
		e.SetLanguageServerController(&stubController{})
		res := test.RunCmd(t, km, e, "format_selections")
		assert.Equal(t, "", res.Message)
	})

	t.Run("ErrNoLanguageServer no range formatter", func(t *testing.T) {
		e, km := test.Env(t, "hello\n")
		e.SetLanguageServerController(&stubController{
			formatSelErr: lsp.ErrNoLanguageServer,
		})
		res := test.RunCmd(t, km, e, "format_selections")
		assert.Contains(t, res.Message, "language server")
	})

	t.Run("format_selection selection count error", func(t *testing.T) {
		e, km := test.Env(t, "hello\n")
		e.SetLanguageServerController(&stubController{
			formatSelErr: lsp.ErrFormatSelection,
		})
		res := test.RunCmd(t, km, e, "format_selections")
		assert.Contains(t, res.Message, "single selection")
	})

	t.Run("other error reports error", func(t *testing.T) {
		e, km := test.Env(t, "hello\n")
		e.SetLanguageServerController(&stubController{
			formatSelErr: errors.New("bang"),
		})
		res := test.RunCmd(t, km, e, "format_selections")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("no controller no range formatter", func(t *testing.T) {
		e, km := test.Env(t, "hello\n")
		res := test.RunCmd(t, km, e, "format_selections")
		assert.Contains(t, res.Message, "language server")
	})
}

func writeFormatterConfig(t *testing.T, fmtToml string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	content := fmt.Sprintf(
		"[[language]]\nname = \"text\"\n[language.formatter]\n%s\n", fmtToml,
	)
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(content), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}

func writeAutoFormatConfig(t *testing.T, fmtToml string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	content := fmt.Sprintf(
		"[[language]]\nname = \"text\"\nauto-format = true\n"+
			"[language.formatter]\n%s\n",
		fmtToml,
	)
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(content), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}

func writeAutoFormatLSPConfig(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	content := "[[language]]\nname = \"text\"\nauto-format = true\n"
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(content), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}

func TestAutoFormat(t *testing.T) {
	t.Run("auto-format runs formatter on write", func(t *testing.T) {
		writeAutoFormatConfig(t, `command = "tr"
args = ["a-z", "A-Z"]`)
		e, km := test.Env(t, "hello\n")
		out := filepath.Join(e.Cwd(), "out.txt")
		res := test.RunCmdArgs(t, km, e, "write", out)
		assert.Contains(t, res.Message, "written")
		assert.Equal(t, "HELLO\n", test.DocText(t, e))
	})

	t.Run("auto-format falls through to lsp", func(t *testing.T) {
		writeAutoFormatLSPConfig(t)
		e, km := test.Env(t, "hello\n")
		e.SetLanguageServerController(&stubController{})
		out := filepath.Join(e.Cwd(), "out.txt")
		res := test.RunCmdArgs(t, km, e, "write", out)
		assert.Contains(t, res.Message, "written")
	})
}
