package lsp_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/lsp"
	"github.com/kode4food/toe/internal/view"
)

func TestWorkspaceEditChangesErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.session")
	session := attachWorkspaceEditSession(t, dir, path)

	t.Run("non-file URI", func(t *testing.T) {
		err := session.ApplyWorkspaceEdit(
			"session-test", protocol.WorkspaceEdit{
				Changes: map[uri.URI][]protocol.TextEdit{
					"untitled:Untitled-1": {{NewText: "x"}},
				},
			})
		assert.True(t, errors.Is(err, lsp.ErrWorkspaceEditURI))
	})

	t.Run("invalid range", func(t *testing.T) {
		err := session.ApplyWorkspaceEdit(
			"session-test", protocol.WorkspaceEdit{
				Changes: map[uri.URI][]protocol.TextEdit{
					uri.File(path): {hugeRangeEdit()},
				},
			})
		assert.True(t, errors.Is(err, lsp.ErrWorkspaceEditRange))
	})

	t.Run("overlapping edits", func(t *testing.T) {
		err := session.ApplyWorkspaceEdit(
			"session-test", protocol.WorkspaceEdit{
				Changes: map[uri.URI][]protocol.TextEdit{
					uri.File(path): {
						{
							Range: protocol.Range{
								Start: protocol.Position{Character: 0},
								End:   protocol.Position{Character: 2},
							},
							NewText: "x",
						},
						{
							Range: protocol.Range{
								Start: protocol.Position{Character: 1},
								End:   protocol.Position{Character: 3},
							},
							NewText: "y",
						},
					},
				},
			})
		assert.True(t, errors.Is(err, core.ErrChangeOrder))
	})
}

func TestWorkspaceEditDocumentChangesErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.session")
	session := attachWorkspaceEditSession(t, dir, path)

	t.Run("unsupported change kind", func(t *testing.T) {
		err := session.ApplyWorkspaceEdit(
			"session-test", protocol.WorkspaceEdit{
				DocumentChanges: []protocol.DocumentChange{nil},
			})
		assert.True(t, errors.Is(err, lsp.ErrWorkspaceEditUnsupported))
	})

	t.Run("text document edit unknown document", func(t *testing.T) {
		err := session.ApplyWorkspaceEdit(
			"session-test", protocol.WorkspaceEdit{
				DocumentChanges: []protocol.DocumentChange{
					&protocol.TextDocumentEdit{
						TextDocument: versionedTextDocumentID(
							"untitled:Untitled-1",
						),
						Edits: []protocol.TextDocumentEditElement{
							&protocol.TextEdit{NewText: "x"},
						},
					},
				},
			})
		assert.True(t, errors.Is(err, lsp.ErrWorkspaceEditURI))
	})

	t.Run("applies annotated text edit", func(t *testing.T) {
		err := session.ApplyWorkspaceEdit(
			"session-test", protocol.WorkspaceEdit{
				DocumentChanges: []protocol.DocumentChange{
					&protocol.TextDocumentEdit{
						TextDocument: versionedTextDocumentID(uri.File(path)),
						Edits: []protocol.TextDocumentEditElement{
							&protocol.AnnotatedTextEdit{
								TextEdit: protocol.TextEdit{
									Range: protocol.Range{
										Start: protocol.Position{Character: 0},
										End:   protocol.Position{Character: 1},
									},
									NewText: "z",
								},
							},
						},
					},
				},
			})
		assert.NoError(t, err)
	})

	t.Run("rejects snippet text edit", func(t *testing.T) {
		err := session.ApplyWorkspaceEdit(
			"session-test", protocol.WorkspaceEdit{
				DocumentChanges: []protocol.DocumentChange{
					&protocol.TextDocumentEdit{
						TextDocument: versionedTextDocumentID(uri.File(path)),
						Edits: []protocol.TextDocumentEditElement{
							&protocol.SnippetTextEdit{},
						},
					},
				},
			})
		assert.True(t, errors.Is(err, lsp.ErrWorkspaceEditUnsupported))
	})

	t.Run("rejects unknown edit element", func(t *testing.T) {
		err := session.ApplyWorkspaceEdit(
			"session-test", protocol.WorkspaceEdit{
				DocumentChanges: []protocol.DocumentChange{
					&protocol.TextDocumentEdit{
						TextDocument: versionedTextDocumentID(uri.File(path)),
						Edits:        []protocol.TextDocumentEditElement{nil},
					},
				},
			})
		assert.True(t, errors.Is(err, lsp.ErrWorkspaceEditUnsupported))
	})

	t.Run("text document edit invalid range", func(t *testing.T) {
		err := session.ApplyWorkspaceEdit(
			"session-test", protocol.WorkspaceEdit{
				DocumentChanges: []protocol.DocumentChange{
					&protocol.TextDocumentEdit{
						TextDocument: versionedTextDocumentID(uri.File(path)),
						Edits:        []protocol.TextDocumentEditElement{new(hugeRangeEdit())},
					},
				},
			})
		assert.True(t, errors.Is(err, lsp.ErrWorkspaceEditRange))
	})
}

func TestWorkspaceEditCreateFileErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.session")
	session := attachWorkspaceEditSession(t, dir, path)

	t.Run("non-file URI", func(t *testing.T) {
		err := session.ApplyWorkspaceEdit(
			"session-test", protocol.WorkspaceEdit{
				DocumentChanges: []protocol.DocumentChange{
					&protocol.CreateFile{URI: "untitled:Untitled-1"},
				},
			})
		assert.True(t, errors.Is(err, lsp.ErrWorkspaceEditURI))
	})

	t.Run("existing file without overwrite", func(t *testing.T) {
		existing := filepath.Join(dir, "existing.session")
		assert.NoError(t, os.WriteFile(existing, []byte("x"), 0o644))

		err := session.ApplyWorkspaceEdit(
			"session-test", protocol.WorkspaceEdit{
				DocumentChanges: []protocol.DocumentChange{
					&protocol.CreateFile{URI: uri.File(existing)},
				},
			})
		assert.True(t, errors.Is(err, lsp.ErrWorkspaceEditFile))
	})

	t.Run("existing file with ignoreIfExists is a no-op", func(t *testing.T) {
		existing := filepath.Join(dir, "existing-ignored.session")
		assert.NoError(t, os.WriteFile(existing, []byte("x"), 0o644))
		err := session.ApplyWorkspaceEdit(
			"session-test", protocol.WorkspaceEdit{
				DocumentChanges: []protocol.DocumentChange{
					&protocol.CreateFile{
						URI: uri.File(existing),
						Options: &protocol.CreateFileOptions{
							IgnoreIfExists: new(true),
						},
					},
				},
			})
		assert.NoError(t, err)
		content, err := os.ReadFile(existing)
		assert.NoError(t, err)
		assert.Equal(t, "x", string(content))
	})
}

func TestWorkspaceEditRenameFileErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.session")
	session := attachWorkspaceEditSession(t, dir, path)

	t.Run("non-file old URI", func(t *testing.T) {
		err := session.ApplyWorkspaceEdit(
			"session-test", protocol.WorkspaceEdit{
				DocumentChanges: []protocol.DocumentChange{
					&protocol.RenameFile{
						OldURI: "untitled:Untitled-1",
						NewURI: uri.File(filepath.Join(dir, "new.session")),
					},
				},
			})
		assert.True(t, errors.Is(err, lsp.ErrWorkspaceEditURI))
	})

	t.Run("non-file new URI", func(t *testing.T) {
		oldPath := filepath.Join(dir, "old.session")
		assert.NoError(t, os.WriteFile(oldPath, []byte("x"), 0o644))

		err := session.ApplyWorkspaceEdit(
			"session-test", protocol.WorkspaceEdit{
				DocumentChanges: []protocol.DocumentChange{
					&protocol.RenameFile{
						OldURI: uri.File(oldPath),
						NewURI: "untitled:Untitled-1",
					},
				},
			})
		assert.True(t, errors.Is(err, lsp.ErrWorkspaceEditURI))
	})

	t.Run("missing source file", func(t *testing.T) {
		err := session.ApplyWorkspaceEdit(
			"session-test", protocol.WorkspaceEdit{
				DocumentChanges: []protocol.DocumentChange{
					&protocol.RenameFile{
						OldURI: uri.File(filepath.Join(dir, "missing.session")),
						NewURI: uri.File(filepath.Join(dir, "renamed.session")),
					},
				},
			})
		assert.True(t, errors.Is(err, lsp.ErrWorkspaceEditFile))
	})

	t.Run("target directory blocked by file", func(t *testing.T) {
		oldPath := filepath.Join(dir, "blocked-old.session")
		assert.NoError(t, os.WriteFile(oldPath, []byte("x"), 0o644))
		blocker := filepath.Join(dir, "blocker")
		assert.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))
		newPath := filepath.Join(blocker, "sub", "new.session")

		err := session.ApplyWorkspaceEdit(
			"session-test", protocol.WorkspaceEdit{
				DocumentChanges: []protocol.DocumentChange{
					&protocol.RenameFile{
						OldURI: uri.File(oldPath),
						NewURI: uri.File(newPath),
					},
				},
			})
		assert.True(t, errors.Is(err, lsp.ErrWorkspaceEditFile))
	})
}

func TestWorkspaceEditDeleteFileErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.session")
	session := attachWorkspaceEditSession(t, dir, path)

	t.Run("non-file URI", func(t *testing.T) {
		err := session.ApplyWorkspaceEdit(
			"session-test", protocol.WorkspaceEdit{
				DocumentChanges: []protocol.DocumentChange{
					&protocol.DeleteFile{URI: "untitled:Untitled-1"},
				},
			})
		assert.True(t, errors.Is(err, lsp.ErrWorkspaceEditURI))
	})

	t.Run("missing file without ignore", func(t *testing.T) {
		err := session.ApplyWorkspaceEdit(
			"session-test", protocol.WorkspaceEdit{
				DocumentChanges: []protocol.DocumentChange{
					&protocol.DeleteFile{
						URI: uri.File(filepath.Join(dir, "missing.session")),
					},
				},
			})
		assert.True(t, errors.Is(err, lsp.ErrWorkspaceEditFile))
	})

	t.Run("missing file with ignoreIfNotExists is a no-op", func(t *testing.T) {
		err := session.ApplyWorkspaceEdit(
			"session-test", protocol.WorkspaceEdit{
				DocumentChanges: []protocol.DocumentChange{
					&protocol.DeleteFile{
						URI: uri.File(filepath.Join(dir, "missing.session")),
						Options: &protocol.DeleteFileOptions{
							IgnoreIfNotExists: new(true),
						},
					},
				},
			})
		assert.NoError(t, err)
	})

	t.Run("recursive delete removes directory", func(t *testing.T) {
		target := filepath.Join(dir, "sub")
		assert.NoError(t, os.MkdirAll(target, 0o755))
		assert.NoError(t, os.WriteFile(
			filepath.Join(target, "file.session"), []byte("x"), 0o644,
		))
		err := session.ApplyWorkspaceEdit(
			"session-test", protocol.WorkspaceEdit{
				DocumentChanges: []protocol.DocumentChange{
					&protocol.DeleteFile{
						URI: uri.File(target),
						Options: &protocol.DeleteFileOptions{
							Recursive: new(true),
						},
					},
				},
			})
		assert.NoError(t, err)
		_, err = os.Stat(target)
		assert.True(t, errors.Is(err, os.ErrNotExist))
	})
}

func attachWorkspaceEditSession(t *testing.T, dir, path string) *lsp.Session {
	t.Helper()
	exe, err := os.Executable()
	assert.NoError(t, err)
	writeCompletionLanguages(t, exe)
	assert.NoError(t, os.WriteFile(path, []byte("abc\n"), 0o644))
	e := view.NewEditor(dir)
	_, err = e.OpenFile(path)
	assert.NoError(t, err)
	session := lsp.Attach(t.Context(), e)
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func hugeRangeEdit() protocol.TextEdit {
	return protocol.TextEdit{
		Range: protocol.Range{
			Start: protocol.Position{Line: 999},
			End:   protocol.Position{Line: 999, Character: 1},
		},
		NewText: "x",
	}
}

func versionedTextDocumentID(
	u uri.URI,
) protocol.OptionalVersionedTextDocumentIdentifier {
	return protocol.OptionalVersionedTextDocumentIdentifier{
		TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: u},
	}
}
