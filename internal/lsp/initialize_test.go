package lsp_test

import (
	"path/filepath"
	"testing"

	"github.com/kode4food/toe/internal/lsp"
	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestInitialize(t *testing.T) {
	t.Run("uses defaults", func(t *testing.T) {
		params := lsp.NewInitializeParams(lsp.InitializeConfig{})

		assert.NotNil(t, params.ProcessID)
		assert.Equal(t, "toe", params.ClientInfo.Name)
		assert.NotNil(t, params.Capabilities.Workspace)
		assert.NotNil(t, params.Capabilities.TextDocument)
		assert.NotNil(t, params.Capabilities.Window)
		completion := params.Capabilities.TextDocument.Completion
		assert.NotNil(t, completion)
		assert.True(t, *completion.ContextSupport)
		assert.False(t, *completion.CompletionItem.SnippetSupport)
		symbols := params.Capabilities.TextDocument.DocumentSymbol
		assert.NotNil(t, symbols)
		assert.False(t, *symbols.HierarchicalDocumentSymbolSupport)
		assert.Equal(t,
			[]protocol.PositionEncodingKind{
				protocol.PositionEncodingKindUTF8,
				protocol.PositionEncodingKindUTF32,
				protocol.PositionEncodingKindUTF16,
			},
			params.Capabilities.General.PositionEncodings,
		)
	})

	t.Run("includes workspace folder", func(t *testing.T) {
		root := t.TempDir()
		params := lsp.NewInitializeParams(lsp.InitializeConfig{
			WorkspaceRoot: filepath.Join(root, "."),
		})

		folders, ok := params.WorkspaceFolders.Get()
		assert.True(t, ok)
		assert.Len(t, folders, 1)
		assert.Equal(t, uri.File(root), folders[0].URI)
		assert.Equal(t, filepath.Base(root), folders[0].Name)
	})

}

func TestOffsetEncoding(t *testing.T) {
	t.Run("uses server encoding", func(t *testing.T) {
		got := lsp.OffsetEncoding(protocol.ServerCapabilities{
			PositionEncoding: protocol.PositionEncodingKindUTF32,
		})

		assert.Equal(t, protocol.PositionEncodingKindUTF32, got)
	})

	t.Run("defaults to utf16", func(t *testing.T) {
		got := lsp.OffsetEncoding(protocol.ServerCapabilities{})

		assert.Equal(t, protocol.PositionEncodingKindUTF16, got)
	})
}
