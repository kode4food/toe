package lsp_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/lsp"
	"github.com/kode4food/toe/internal/view"
)

func TestSession(t *testing.T) {
	t.Run("opens configured server", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		file := filepath.Join(dir, "main.session")
		marker := filepath.Join(dir, "did-open")
		writeSessionLanguages(t, exe, marker)
		assert.NoError(t, os.WriteFile(file, []byte("hello\n"), 0o644))

		e := view.NewEditor(dir)
		_, err = e.OpenFile(file)
		assert.NoError(t, err)

		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()

		assert.Eventually(t, func() bool {
			got, err := os.ReadFile(marker)
			return err == nil && string(got) == string(uri.File(file))
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("serves workspace folders", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)

		dir := t.TempDir()
		file := filepath.Join(dir, "main.session")
		marker := filepath.Join(dir, "did-open")
		writeWorkspaceFolderLanguages(t, exe, marker)
		assert.NoError(t, os.WriteFile(file, []byte("hello\n"), 0o644))

		e := view.NewEditor(dir)
		_, err = e.OpenFile(file)
		assert.NoError(t, err)

		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()

		assert.Eventually(t, func() bool {
			got, err := os.ReadFile(marker)
			want := string(uri.File(file)) + "\n" + string(uri.File(dir))
			return err == nil && string(got) == want
		}, time.Second, 10*time.Millisecond)
	})

}

func TestSessionMethods(t *testing.T) {
	t.Run("saves document", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeCompletionLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("test\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		session.DocumentSaved(doc)
	})

	t.Run("closes document", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeCompletionLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("test\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		session.DocumentClosed(doc)
	})

	t.Run("lists workspace commands", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeCompletionLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("test\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		commands := session.WorkspaceCommands(doc)

		assert.Contains(t, commands, "session.afterCompletion")
	})

	t.Run("executes workspace command", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeCompletionLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("test\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		err = session.ExecuteWorkspaceCommand(
			doc, "session.afterCompletion", nil,
		)

		assert.False(t, errors.Is(err, lsp.ErrWorkspaceCommand))
	})

	t.Run("stops language servers", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeCompletionLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("test\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		names, err := session.StopLanguageServers(doc, nil)

		assert.NoError(t, err)
		assert.NotEmpty(t, names)
	})

	t.Run("restarts language servers", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeCompletionLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("test\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		names, err := session.RestartLanguageServers(doc, nil)

		assert.NoError(t, err)
		assert.NotEmpty(t, names)
	})

	t.Run("stops named language server", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeCompletionLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("test\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		names, err := session.StopLanguageServers(doc, []string{"session-test"})

		assert.NoError(t, err)
		assert.Equal(t, []string{"session-test"}, names)
	})

	t.Run("rejects unknown language server", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeCompletionLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("test\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		_, err = session.StopLanguageServers(doc, []string{"unknown-server"})

		assert.ErrorIs(t, err, lsp.ErrUnknownLanguageServer)
	})

	t.Run("stop scratch doc no language", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		_, err := session.StopLanguageServers(doc, nil)

		assert.ErrorIs(t, err, lsp.ErrNoLanguageServer)
	})

	t.Run("restart rejects unknown server", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeCompletionLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("test\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		_, err = session.RestartLanguageServers(
			doc, []string{"unknown-server"},
		)

		assert.ErrorIs(t, err, lsp.ErrUnknownLanguageServer)
	})

	t.Run("restart scratch doc no language", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		_, err := session.RestartLanguageServers(doc, nil)

		assert.ErrorIs(t, err, lsp.ErrNoLanguageServer)
	})

	t.Run("unknown workspace command errors", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeCompletionLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("test\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		err = session.ExecuteWorkspaceCommand(doc, "session.unknown", nil)

		assert.ErrorIs(t, err, lsp.ErrWorkspaceCommand)
	})

	t.Run("executes workspace command with args", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeCompletionLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("test\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		err = session.ExecuteWorkspaceCommand(
			doc, "session.afterCompletion", []string{"arg1"},
		)

		assert.False(t, errors.Is(err, lsp.ErrWorkspaceCommand))
	})
}

func TestWorkspaceEdit(t *testing.T) {
	t.Run("applies changes without focusing target", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		source := filepath.Join(dir, "source.session")
		target := filepath.Join(dir, "target.session")
		writeCompletionLanguages(t, exe)
		assert.NoError(t, os.WriteFile(source, []byte("source\n"), 0o644))
		assert.NoError(t, os.WriteFile(target, []byte("old\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(source)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		waitForWorkspaceServer(t, session)

		err = session.ApplyWorkspaceEdit("session-test", protocol.WorkspaceEdit{
			Changes: map[uri.URI][]protocol.TextEdit{
				uri.File(target): {
					{
						Range: protocol.Range{
							Start: protocol.Position{},
							End: protocol.Position{
								Line: 0, Character: 3,
							},
						},
						NewText: "new",
					},
				},
			},
		})

		assert.NoError(t, err)
		doc, err := e.SwitchOrOpenDoc(target)
		assert.NoError(t, err)
		assert.Equal(t, "new\n", doc.Text().String())
		focused, ok := e.FocusedDocument()
		assert.True(t, ok)
		assert.Equal(t, source, focused.Path())
	})

	t.Run("applies document changes", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeCompletionLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("abc\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		waitForWorkspaceServer(t, session)
		textDoc := protocol.OptionalVersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{
				URI: uri.File(path),
			},
		}

		err = session.ApplyWorkspaceEdit("session-test", protocol.WorkspaceEdit{
			DocumentChanges: []protocol.DocumentChange{
				&protocol.TextDocumentEdit{
					TextDocument: textDoc,
					Edits: []protocol.TextDocumentEditElement{
						&protocol.TextEdit{
							Range: protocol.Range{
								Start: protocol.Position{
									Line: 0, Character: 1,
								},
								End: protocol.Position{
									Line: 0, Character: 2,
								},
							},
							NewText: "B",
						},
					},
				},
			},
		})

		assert.NoError(t, err)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		assert.Equal(t, "aBc\n", doc.Text().String())
	})

	t.Run("applies resource operations", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		oldPath := filepath.Join(dir, "old.session")
		newPath := filepath.Join(dir, "new.session")
		createPath := filepath.Join(dir, "created.session")
		deletePath := filepath.Join(dir, "delete.session")
		writeCompletionLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("abc\n"), 0o644))
		assert.NoError(t, os.WriteFile(oldPath, []byte("old\n"), 0o644))
		assert.NoError(t, os.WriteFile(deletePath, []byte("delete\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		oldDoc, err := e.SwitchOrOpenDoc(oldPath)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		waitForWorkspaceServer(t, session)

		err = session.ApplyWorkspaceEdit("session-test", protocol.WorkspaceEdit{
			DocumentChanges: []protocol.DocumentChange{
				&protocol.CreateFile{URI: uri.File(createPath)},
				&protocol.RenameFile{
					OldURI: uri.File(oldPath),
					NewURI: uri.File(newPath),
				},
				&protocol.DeleteFile{URI: uri.File(deletePath)},
			},
		})

		assert.NoError(t, err)
		_, err = os.Stat(createPath)
		assert.NoError(t, err)
		_, err = os.Stat(oldPath)
		assert.True(t, errors.Is(err, os.ErrNotExist))
		_, err = os.Stat(newPath)
		assert.NoError(t, err)
		_, err = os.Stat(deletePath)
		assert.True(t, errors.Is(err, os.ErrNotExist))
		assert.Equal(t, newPath, oldDoc.Path())
	})

	t.Run("unknown server returns error", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		e := view.NewEditor(t.TempDir())
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		err := session.ApplyWorkspaceEdit(
			"nonexistent", protocol.WorkspaceEdit{},
		)
		assert.ErrorIs(t, err, lsp.ErrUnknownLanguageServer)
	})

	t.Run("file operation options", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeCompletionLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("abc\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		waitForWorkspaceServer(t, session)

		trueVal := true

		t.Run("create ignores existing", func(t *testing.T) {
			p := filepath.Join(dir, "existing.txt")
			assert.NoError(t, os.WriteFile(p, []byte("keep"), 0o644))
			opts := &protocol.CreateFileOptions{IgnoreIfExists: &trueVal}
			err := session.ApplyWorkspaceEdit(
				"session-test", protocol.WorkspaceEdit{
					DocumentChanges: []protocol.DocumentChange{
						&protocol.CreateFile{
							URI:     uri.File(p),
							Options: opts,
						},
					},
				})
			assert.NoError(t, err)
			data, _ := os.ReadFile(p)
			assert.Equal(t, "keep", string(data))
		})

		t.Run("create rejects existing", func(t *testing.T) {
			p := filepath.Join(dir, "existing2.txt")
			assert.NoError(t, os.WriteFile(p, []byte("x"), 0o644))
			err := session.ApplyWorkspaceEdit(
				"session-test", protocol.WorkspaceEdit{
					DocumentChanges: []protocol.DocumentChange{
						&protocol.CreateFile{URI: uri.File(p)},
					},
				})
			assert.ErrorIs(t, err, lsp.ErrWorkspaceEditFile)
		})

		t.Run("rename ignores existing", func(t *testing.T) {
			oldP := filepath.Join(dir, "rename-old.txt")
			newP := filepath.Join(dir, "rename-exists.txt")
			assert.NoError(t, os.WriteFile(oldP, []byte("old"), 0o644))
			assert.NoError(t, os.WriteFile(newP, []byte("new"), 0o644))
			opts := &protocol.RenameFileOptions{IgnoreIfExists: &trueVal}
			err := session.ApplyWorkspaceEdit(
				"session-test", protocol.WorkspaceEdit{
					DocumentChanges: []protocol.DocumentChange{
						&protocol.RenameFile{
							OldURI:  uri.File(oldP),
							NewURI:  uri.File(newP),
							Options: opts,
						},
					},
				})
			assert.NoError(t, err)
		})

		t.Run("rename overwrites", func(t *testing.T) {
			oldP := filepath.Join(dir, "overwrite-old.txt")
			newP := filepath.Join(dir, "overwrite-new.txt")
			assert.NoError(t, os.WriteFile(oldP, []byte("old"), 0o644))
			assert.NoError(t, os.WriteFile(newP, []byte("existing"), 0o644))
			opts := &protocol.RenameFileOptions{Overwrite: &trueVal}
			err := session.ApplyWorkspaceEdit(
				"session-test", protocol.WorkspaceEdit{
					DocumentChanges: []protocol.DocumentChange{
						&protocol.RenameFile{
							OldURI:  uri.File(oldP),
							NewURI:  uri.File(newP),
							Options: opts,
						},
					},
				})
			assert.NoError(t, err)
			data, _ := os.ReadFile(newP)
			assert.Equal(t, "old", string(data))
		})

		t.Run("rename rejects existing", func(t *testing.T) {
			oldP := filepath.Join(dir, "no-overwrite-old.txt")
			newP := filepath.Join(dir, "no-overwrite-new.txt")
			assert.NoError(t, os.WriteFile(oldP, []byte("old"), 0o644))
			assert.NoError(t, os.WriteFile(newP, []byte("new"), 0o644))
			edit := protocol.WorkspaceEdit{
				DocumentChanges: []protocol.DocumentChange{
					&protocol.RenameFile{
						OldURI: uri.File(oldP),
						NewURI: uri.File(newP),
					},
				},
			}
			err := session.ApplyWorkspaceEdit("session-test", edit)
			assert.ErrorIs(t, err, lsp.ErrWorkspaceEditFile)
		})

		t.Run("delete ignores missing", func(t *testing.T) {
			p := filepath.Join(dir, "missing.txt")
			opts := &protocol.DeleteFileOptions{IgnoreIfNotExists: &trueVal}
			err := session.ApplyWorkspaceEdit(
				"session-test", protocol.WorkspaceEdit{
					DocumentChanges: []protocol.DocumentChange{
						&protocol.DeleteFile{
							URI:     uri.File(p),
							Options: opts,
						},
					},
				})
			assert.NoError(t, err)
		})

		t.Run("delete rejects missing", func(t *testing.T) {
			p := filepath.Join(dir, "missing2.txt")
			err := session.ApplyWorkspaceEdit(
				"session-test", protocol.WorkspaceEdit{
					DocumentChanges: []protocol.DocumentChange{
						&protocol.DeleteFile{URI: uri.File(p)},
					},
				})
			assert.ErrorIs(t, err, lsp.ErrWorkspaceEditFile)
		})

		t.Run("delete recursive", func(t *testing.T) {
			p := filepath.Join(dir, "subdir")
			assert.NoError(t, os.MkdirAll(filepath.Join(p, "inner"), 0o755))
			path := filepath.Join(p, "inner", "f.txt")
			assert.NoError(t, os.WriteFile(path, nil, 0o644))
			opts := &protocol.DeleteFileOptions{Recursive: &trueVal}
			err := session.ApplyWorkspaceEdit(
				"session-test", protocol.WorkspaceEdit{
					DocumentChanges: []protocol.DocumentChange{
						&protocol.DeleteFile{
							URI:     uri.File(p),
							Options: opts,
						},
					},
				})
			assert.NoError(t, err)
			_, statErr := os.Stat(p)
			assert.True(t, errors.Is(statErr, os.ErrNotExist))
		})
	})
}

func TestSessionCallbacks(t *testing.T) {
	t.Run("processes server-side callbacks", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeCallbacksLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("Pr\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)

		// triggers initialization; server fires all callbacks during init
		_, err = session.Completions(doc, v.ID())
		assert.NoError(t, err)

		// brief wait so async callbacks (DiagnosticRefresh) can be dispatched
		time.Sleep(50 * time.Millisecond)
	})
}

func TestSessionPullDiagnosticsWithServer(t *testing.T) {
	t.Run("applies diagnostic report", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writePullDiagnosticsLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("old\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		err = session.PullDiagnostics(doc)

		assert.NoError(t, err)
		assert.NotEmpty(t, doc.Diagnostics())
	})

	t.Run("caches result ID across calls", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writePullDiagnosticsLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("old\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		// First call: stores the resultID from the Full report
		err = session.PullDiagnostics(doc)
		assert.NoError(t, err)
		// Second call: previousDiagnosticID now finds the stored resultID
		err = session.PullDiagnostics(doc)
		assert.NoError(t, err)
		assert.NotEmpty(t, doc.Diagnostics())
	})
}

func TestSessionPullDiagnostics(t *testing.T) {
	t.Run("error without server", func(t *testing.T) {
		// Isolate from any system-level toe language config
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		dir := t.TempDir()
		path := filepath.Join(dir, "main.nolsp")
		assert.NoError(t, os.WriteFile(
			path, []byte("hello\n"), 0o644,
		))
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.NewSession(t.Context(), dir)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		err = session.PullDiagnostics(doc)

		assert.ErrorIs(t, err, lsp.ErrNoLanguageServer)
	})
}

func TestSessionScratchDocument(t *testing.T) {
	// Scratch documents have no URI, so requests return no server results
	setup := func(t *testing.T) (*lsp.Session, *view.Document, view.Id) {
		t.Helper()
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		e := view.NewEditor(t.TempDir())
		session := lsp.NewSession(t.Context(), t.TempDir())
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		return session, doc, v.ID()
	}

	t.Run("code actions empty", func(t *testing.T) {
		session, doc, viewID := setup(t)
		actions, err := session.CodeActions(doc, viewID)
		assert.NoError(t, err)
		assert.Nil(t, actions)
	})

	t.Run("hover empty", func(t *testing.T) {
		session, doc, viewID := setup(t)
		text, err := session.Hover(doc, viewID)
		assert.NoError(t, err)
		assert.Empty(t, text)
	})

	t.Run("highlights empty", func(t *testing.T) {
		session, doc, viewID := setup(t)
		highlights, err := session.DocumentHighlights(doc, viewID)
		assert.NoError(t, err)
		assert.Nil(t, highlights)
	})

	t.Run("format selection no-op", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		e := view.NewEditor(t.TempDir())
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		err := session.FormatSelection(doc, v.ID())
		assert.NoError(t, err)
	})

	t.Run("signature help empty", func(t *testing.T) {
		session, doc, viewID := setup(t)
		help, err := session.SignatureHelp(doc, viewID)
		assert.NoError(t, err)
		assert.Empty(t, help.Signatures)
	})

	t.Run("rename prefill empty", func(t *testing.T) {
		session, doc, viewID := setup(t)
		name, err := session.RenameSymbolPrefill(doc, viewID)
		assert.NoError(t, err)
		assert.Empty(t, name)
	})

	t.Run("RenameSymbol returns nil for scratch doc", func(t *testing.T) {
		session, doc, viewID := setup(t)
		err := session.RenameSymbol(doc, viewID, "newname")
		assert.NoError(t, err)
	})

	t.Run("scratch doc returns nil", func(t *testing.T) {
		session, doc, viewID := setup(t)
		locs, err := session.GotoDeclaration(doc, viewID)
		assert.NoError(t, err)
		assert.Nil(t, locs)
	})
}

func TestSessionPullDiagnosticsError(t *testing.T) {
	t.Run("returns error from server", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writePullDiagnosticsErrorLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("old\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		err = session.PullDiagnostics(doc)
		assert.Error(t, err)
	})
}

func TestSessionPullDiagnosticsRegOptions(t *testing.T) {
	t.Run("uses diagnostic registration options", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writePullDiagnosticsRegOptionsLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("old\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		err = session.PullDiagnostics(doc)
		assert.NoError(t, err)
		assert.NotEmpty(t, doc.Diagnostics())
	})
}

func TestSessionPullDiagnosticsUnchanged(t *testing.T) {
	t.Run("handles unchanged diagnostic report", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writePullDiagnosticsUnchangedLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("old\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		err = session.PullDiagnostics(doc)
		assert.NoError(t, err)
	})
}

func TestSessionProgress(t *testing.T) {
	t.Run("receives begin/report/end progress", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeProgressLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)
		// Trigger initialization and wait for progress callbacks to fire
		_, _ = session.Completions(doc, v.ID())
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("reports busy during progress", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeProgressLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)

		_, _ = session.Completions(doc, v.ID())

		assert.Eventually(t, session.Busy, time.Second, 5*time.Millisecond)
		assert.Eventually(t, func() bool { return !session.Busy() },
			time.Second, 5*time.Millisecond)
	})
}

func TestFileWatching(t *testing.T) {
	t.Run("notifies server on file save", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		notifyFile := filepath.Join(t.TempDir(), "watched")
		writeFileWatchLanguages(t, exe, notifyFile)
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)

		// Trigger initialization; server registers file watcher
		_, _ = session.Completions(doc, v.ID())

		// Saving resends didChangeWatchedFiles each attempt, so the poll
		// succeeds as soon as the async watch registration lands
		doc.SetSelectionFor(v.ID(), doc.SelectionFor(v.ID()))
		assert.Eventually(t, func() bool {
			session.DocumentSaved(doc)
			_, err := os.Stat(notifyFile)
			return err == nil
		}, 5*time.Second, 25*time.Millisecond)
	})

	t.Run("notifies server on external create", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		notifyFile := filepath.Join(t.TempDir(), "watched")
		writeFileWatchLanguages(t, exe, notifyFile)
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)

		_, _ = session.Completions(doc, v.ID())

		// Rewriting the file each attempt raises a fresh fsnotify event, so
		// the poll succeeds as soon as the directory watch is established
		created := filepath.Join(dir, "created.session")
		assert.Eventually(t, func() bool {
			assert.NoError(t, os.WriteFile(created, []byte("new\n"), 0o644))
			_, err := os.Stat(notifyFile)
			return err == nil
		}, 5*time.Second, 25*time.Millisecond)
	})
}

func TestInlayHints(t *testing.T) {
	t.Run("returns inlay hints from server", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeInlayHintsLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)

		hints, err := session.InlayHints(doc, v.ID())
		assert.NoError(t, err)
		assert.NotEmpty(t, hints)
	})

	t.Run("refreshes on document lifecycle", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeInlayHintsLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)

		assert.Eventually(t, func() bool {
			return len(doc.InlayHints(v.ID())) > 0
		}, time.Second, 10*time.Millisecond)

		doc.ClearInlayHints(v.ID())
		session.DocumentSaved(doc)

		assert.Eventually(t, func() bool {
			return len(doc.InlayHints(v.ID())) > 0
		}, time.Second, 10*time.Millisecond)
	})
}

func TestDocumentColors(t *testing.T) {
	t.Run("returns colors from server", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeDocumentColorLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("red\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		colors, err := session.DocumentColors(doc)
		assert.NoError(t, err)
		assert.NotEmpty(t, colors)
	})
}

func TestSessionAllOperationsError(t *testing.T) {
	exe, err := os.Executable()
	assert.NoError(t, err)
	dir := t.TempDir()
	path := filepath.Join(dir, "main.session")
	writeAllErrorLanguages(t, exe)
	assert.NoError(t, os.WriteFile(path, []byte("old\n"), 0o644))
	e := view.NewEditor(dir)
	_, err = e.OpenFile(path)
	assert.NoError(t, err)
	session := lsp.Attach(t.Context(), e)
	defer func() { _ = session.Close() }()
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	v, ok := e.FocusedView()
	assert.True(t, ok)

	t.Run("hover returns error", func(t *testing.T) {
		_, err := session.Hover(doc, v.ID())
		assert.Error(t, err)
	})
	t.Run("code actions returns error", func(t *testing.T) {
		_, err := session.CodeActions(doc, v.ID())
		assert.Error(t, err)
	})
	t.Run("document highlights returns error", func(t *testing.T) {
		_, err := session.DocumentHighlights(doc, v.ID())
		assert.Error(t, err)
	})
	t.Run("format document returns error", func(t *testing.T) {
		err := session.FormatDocument(doc, v.ID())
		assert.Error(t, err)
	})
	t.Run("document symbols returns error", func(t *testing.T) {
		_, err := session.DocumentSymbols(doc)
		assert.Error(t, err)
	})
	t.Run("workspace symbols returns error", func(t *testing.T) {
		_, err := session.WorkspaceSymbols(doc, "old")
		assert.Error(t, err)
	})
	t.Run("rename prefill returns error", func(t *testing.T) {
		_, err := session.RenameSymbolPrefill(doc, v.ID())
		assert.Error(t, err)
	})
	t.Run("rename symbol returns error", func(t *testing.T) {
		err := session.RenameSymbol(doc, v.ID(), "new")
		assert.Error(t, err)
	})
	t.Run("goto declaration returns error", func(t *testing.T) {
		_, err := session.GotoDeclaration(doc, v.ID())
		assert.Error(t, err)
	})
	t.Run("goto definition returns error", func(t *testing.T) {
		_, err := session.GotoDefinition(doc, v.ID())
		assert.Error(t, err)
	})
	t.Run("goto type definition returns error", func(t *testing.T) {
		_, err := session.GotoTypeDefinition(doc, v.ID())
		assert.Error(t, err)
	})
	t.Run("goto implementation returns error", func(t *testing.T) {
		_, err := session.GotoImplementation(doc, v.ID())
		assert.Error(t, err)
	})
	t.Run("goto reference returns error", func(t *testing.T) {
		_, err := session.GotoReference(doc, v.ID())
		assert.Error(t, err)
	})
	t.Run("document colors returns error", func(t *testing.T) {
		_, err := session.DocumentColors(doc)
		assert.Error(t, err)
	})
	t.Run("signature help returns error", func(t *testing.T) {
		_, err := session.SignatureHelp(doc, v.ID())
		assert.Error(t, err)
	})
	t.Run("format selection returns error", func(t *testing.T) {
		err := session.FormatSelection(doc, v.ID())
		assert.Error(t, err)
	})
	t.Run("trigger signature without trigger", func(t *testing.T) {
		help, err := session.TriggerSignatureHelp(doc, v.ID())
		assert.NoError(t, err)
		assert.Empty(t, help.Signatures)
	})
	t.Run("resolve document link returns error", func(t *testing.T) {
		links, err := session.DocumentLinks(doc)
		assert.NoError(t, err)
		assert.NotEmpty(t, links)
		_, err = session.ResolveDocumentLink(doc, links[0])
		assert.Error(t, err)
	})
}

func TestSessionWithoutEditor(t *testing.T) {
	exe, err := os.Executable()
	assert.NoError(t, err)
	dir := t.TempDir()
	path := filepath.Join(dir, "main.session")
	writeCompletionLanguages(t, exe)
	assert.NoError(t, os.WriteFile(path, []byte("test\n"), 0o644))
	session := lsp.NewSession(t.Context(), dir)
	defer func() { _ = session.Close() }()

	t.Run("reload config is a no-op", func(t *testing.T) {
		assert.NoError(t, session.ReloadConfig())
	})

	t.Run("publish diagnostics is a no-op", func(t *testing.T) {
		err := session.PublishDiagnostics(
			t.Context(), &protocol.PublishDiagnosticsParams{},
		)
		assert.NoError(t, err)
	})

	t.Run("rejects nil document", func(t *testing.T) {
		_, err := session.StopLanguageServers(nil, nil)
		assert.True(t, errors.Is(err, lsp.ErrNoLanguageServer))
	})

	t.Run("no attached editor", func(t *testing.T) {
		e := view.NewEditor(dir)
		_, err := e.OpenFile(path)
		assert.NoError(t, err)
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		names, err := session.StopLanguageServers(doc, nil)

		assert.NoError(t, err)
		assert.Equal(t, []string{"session-test"}, names)
	})
}

func TestSessionMultiServer(t *testing.T) {
	t.Run("ambiguous workspace command", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeMultiServerSessionLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("test\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		err = session.ExecuteWorkspaceCommand(
			doc, "session.afterCompletion", nil,
		)

		assert.True(t, errors.Is(err, lsp.ErrWorkspaceCommand))
	})

	t.Run("serves folders for every server", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeMultiServerSessionLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("test\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)
		v, ok := e.FocusedView()
		assert.True(t, ok)

		_, err = session.Completions(doc, v.ID())
		assert.NoError(t, err)
	})

	t.Run("deduplicates repeated server names", func(t *testing.T) {
		exe, err := os.Executable()
		assert.NoError(t, err)
		dir := t.TempDir()
		path := filepath.Join(dir, "main.session")
		writeDuplicateServerLanguages(t, exe)
		assert.NoError(t, os.WriteFile(path, []byte("test\n"), 0o644))
		e := view.NewEditor(dir)
		_, err = e.OpenFile(path)
		assert.NoError(t, err)
		session := lsp.Attach(t.Context(), e)
		defer func() { _ = session.Close() }()
		doc, ok := e.FocusedDocument()
		assert.True(t, ok)

		commands := session.WorkspaceCommands(doc)

		assert.Equal(t, []string{"session.afterCompletion"}, commands)
	})
}

func TestSessionBareServer(t *testing.T) {
	exe, err := os.Executable()
	assert.NoError(t, err)
	dir := t.TempDir()
	path := filepath.Join(dir, "main.session")
	writeBareLanguages(t, exe)
	assert.NoError(t, os.WriteFile(path, []byte("f(x)\n"), 0o644))
	e := view.NewEditor(dir)
	_, err = e.OpenFile(path)
	assert.NoError(t, err)
	session := lsp.Attach(t.Context(), e)
	defer func() { _ = session.Close() }()
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	v, ok := e.FocusedView()
	assert.True(t, ok)

	t.Run("code actions unsupported", func(t *testing.T) {
		actions, err := session.CodeActions(doc, v.ID())
		assert.NoError(t, err)
		assert.Empty(t, actions)
	})
	t.Run("rename prefill unsupported", func(t *testing.T) {
		_, err := session.RenameSymbolPrefill(doc, v.ID())
		assert.ErrorIs(t, err, lsp.ErrNoLanguageServer)
	})
	t.Run("rename unsupported", func(t *testing.T) {
		err := session.RenameSymbol(doc, v.ID(), "new")
		assert.ErrorIs(t, err, lsp.ErrNoLanguageServer)
	})
	t.Run("format unsupported", func(t *testing.T) {
		err := session.FormatDocument(doc, v.ID())
		assert.ErrorIs(t, err, lsp.ErrNoLanguageServer)
	})
	t.Run("format selection multi range", func(t *testing.T) {
		sel, err := core.NewSelection([]core.Range{
			core.NewRange(0, 1),
			core.NewRange(2, 3),
		}, 0)
		assert.NoError(t, err)
		doc.SetSelectionFor(v.ID(), sel)
		assert.ErrorIs(t,
			session.FormatSelection(doc, v.ID()), lsp.ErrFormatSelection,
		)
		doc.SetSelectionFor(v.ID(), core.PointSelection(2))
	})
	t.Run("document symbols unsupported", func(t *testing.T) {
		symbols, err := session.DocumentSymbols(doc)
		assert.NoError(t, err)
		assert.Empty(t, symbols)
	})
	t.Run("workspace symbols unsupported", func(t *testing.T) {
		symbols, err := session.WorkspaceSymbols(doc, "f")
		assert.NoError(t, err)
		assert.Empty(t, symbols)
	})
	t.Run("signature help unsupported", func(t *testing.T) {
		help, err := session.SignatureHelp(doc, v.ID())
		assert.NoError(t, err)
		assert.Empty(t, help.Signatures)
	})
	t.Run("trigger signature unsupported", func(t *testing.T) {
		help, err := session.TriggerSignatureHelp(doc, v.ID())
		assert.NoError(t, err)
		assert.Empty(t, help.Signatures)
	})
	t.Run("document links unsupported", func(t *testing.T) {
		_, err := session.DocumentLinks(doc)
		assert.ErrorIs(t, err, lsp.ErrNoLanguageServer)
	})
	t.Run("inlay hints unsupported", func(t *testing.T) {
		_, err := session.InlayHints(doc, v.ID())
		assert.ErrorIs(t, err, lsp.ErrNoLanguageServer)
	})
	t.Run("pull diagnostics unsupported", func(t *testing.T) {
		assert.NoError(t, session.PullDiagnostics(doc))
	})
	t.Run("resolve unknown document link", func(t *testing.T) {
		_, err := session.ResolveDocumentLink(doc, view.DocumentLink{
			ID: "missing",
		})
		assert.ErrorIs(t, err, lsp.ErrDocumentLinkUnavailable)
	})
}

func TestSessionUnconfiguredDocument(t *testing.T) {
	exe, err := os.Executable()
	assert.NoError(t, err)
	dir := t.TempDir()
	path := filepath.Join(dir, "plain.txt")
	writeBareLanguages(t, exe)
	assert.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o644))
	e := view.NewEditor(dir)
	_, err = e.OpenFile(path)
	assert.NoError(t, err)
	session := lsp.Attach(t.Context(), e)
	defer func() { _ = session.Close() }()
	doc, ok := e.FocusedDocument()
	assert.True(t, ok)
	v, ok := e.FocusedView()
	assert.True(t, ok)

	t.Run("code actions no server", func(t *testing.T) {
		_, err := session.CodeActions(doc, v.ID())
		assert.ErrorIs(t, err, lsp.ErrNoLanguageServer)
	})
	t.Run("rename prefill no server", func(t *testing.T) {
		_, err := session.RenameSymbolPrefill(doc, v.ID())
		assert.ErrorIs(t, err, lsp.ErrNoLanguageServer)
	})
	t.Run("rename no server", func(t *testing.T) {
		err := session.RenameSymbol(doc, v.ID(), "new")
		assert.ErrorIs(t, err, lsp.ErrNoLanguageServer)
	})
	t.Run("format no server", func(t *testing.T) {
		err := session.FormatDocument(doc, v.ID())
		assert.ErrorIs(t, err, lsp.ErrNoLanguageServer)
	})
}

func writeSessionLanguages(t *testing.T, exe, marker string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := fmt.Sprintf(`
[language-server.session-test]
command = %q
args = ["-test.run=TestLSPServerProcess"]
environment = { %s = "1", %s = %q }

[[language]]
name = "session"
language-id = "session"
file-types = ["session"]
language-servers = ["session-test"]
`, exe, testServerEnv, testServerDidOpenFileEnv, marker)
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}

func writeCallbacksLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerCompletionEnv + ` = "1", ` +
		testServerCallbacksEnv + ` = "1" }

[[language]]
name = "session"
language-id = "session"
file-types = ["session"]
language-servers = ["session-test"]
`
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}

func writePullDiagnosticsLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerPullDiagnosticsEnv + ` = "1" }

[[language]]
name = "session"
language-id = "session"
file-types = ["session"]
language-servers = ["session-test"]
`
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}

func writePullDiagnosticsErrorLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerPullDiagnosticsEnv + ` = "1", ` +
		testServerDiagnosticErrorEnv + ` = "1" }

[[language]]
name = "session"
language-id = "session"
file-types = ["session"]
language-servers = ["session-test"]
`
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}

func writePullDiagnosticsRegOptionsLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerPullDiagnosticsEnv + ` = "1", ` +
		testServerDiagRegOptionsEnv + ` = "1" }

[[language]]
name = "session"
language-id = "session"
file-types = ["session"]
language-servers = ["session-test"]
`
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}

func writePullDiagnosticsUnchangedLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerPullDiagnosticsEnv + ` = "1", ` +
		testServerDiagnosticUnchangedEnv + ` = "1" }

[[language]]
name = "session"
language-id = "session"
file-types = ["session"]
language-servers = ["session-test"]
`
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}

func writeProgressLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerCompletionEnv + ` = "1", ` +
		testServerProgressEnv + ` = "1" }

[[language]]
name = "session"
language-id = "session"
file-types = ["session"]
language-servers = ["session-test"]
`
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}

func writeFileWatchLanguages(t *testing.T, exe, notifyFile string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := fmt.Sprintf(`[language-server.session-test]
command = %q
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { %s = "1", %s = "1", %s = "1", %s = %q }

[[language]]
name = "session"
language-id = "session"
file-types = ["session"]
language-servers = ["session-test"]
`, exe, testServerEnv, testServerCompletionEnv, testServerFileWatchEnv,
		testServerFileWatchNotifyEnv, notifyFile)
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}

func writeInlayHintsLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerInlayHintsEnv + ` = "1" }

[[language]]
name = "session"
language-id = "session"
file-types = ["session"]
language-servers = ["session-test"]
`
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}

func writeDocumentColorLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerDocumentColorEnv + ` = "1" }

[[language]]
name = "session"
language-id = "session"
file-types = ["session"]
language-servers = ["session-test"]
`
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}

func writeAllErrorLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1", ` +
		testServerAllErrorEnv + ` = "1", ` +
		testServerCodeActionEnv + ` = "1", ` +
		testServerHighlightEnv + ` = "1", ` +
		testServerFormatEnv + ` = "1", ` +
		testServerSignatureEnv + ` = "1", ` +
		testServerSymbolsEnv + ` = "1", ` +
		testServerWorkspaceSymbolsEnv + ` = "1", ` +
		testServerRenameEnv + ` = "1", ` +
		testServerNavigationEnv + ` = "1", ` +
		testServerDocumentLinkEnv + ` = "1", ` +
		testServerDocumentLinkResolveEnv + ` = "1", ` +
		testServerDocumentColorEnv + ` = "1" }

[[language]]
name = "session"
language-id = "session"
file-types = ["session"]
language-servers = ["session-test"]
`
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}

func writeWorkspaceFolderLanguages(t *testing.T, exe, marker string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := fmt.Sprintf(`
[language-server.session-test]
command = %q
args = ["-test.run=TestLSPServerProcess"]
environment = { %s = "1", %s = %q, %s = "1" }

[[language]]
name = "session"
language-id = "session"
file-types = ["session"]
language-servers = ["session-test"]
`, exe, testServerEnv, testServerDidOpenFileEnv, marker,
		testServerWorkspaceFoldersEnv)
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}

func writeMultiServerSessionLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := fmt.Sprintf(`
[language-server.session-test-a]
command = %q
args = ["-test.run=TestLSPServerProcess"]
environment = { %s = "1", %s = "1", %s = "1" }

[language-server.session-test-b]
command = %q
args = ["-test.run=TestLSPServerProcess"]
environment = { %s = "1", %s = "1", %s = "1" }

[[language]]
name = "session"
language-id = "session"
file-types = ["session"]
language-servers = ["session-test-a", "session-test-b"]
`, exe, testServerEnv, testServerCompletionEnv, testServerWorkspaceFoldersEnv,
		exe, testServerEnv, testServerCompletionEnv,
		testServerWorkspaceFoldersEnv)
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}

func writeDuplicateServerLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := fmt.Sprintf(`
[language-server.session-test]
command = %q
args = ["-test.run=TestLSPServerProcess"]
environment = { %s = "1", %s = "1" }

[[language]]
name = "session"
language-id = "session"
file-types = ["session"]
language-servers = ["session-test", "session-test"]
`, exe, testServerEnv, testServerCompletionEnv)
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}

func writeBareLanguages(t *testing.T, exe string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "toe")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	text := `[language-server.session-test]
command = "` + exe + `"
args = ["-test.run=TestLSPServerProcess"]
timeout = 1
environment = { ` + testServerEnv + ` = "1" }

[[language]]
name = "session"
language-id = "session"
file-types = ["session"]
language-servers = ["session-test"]
`
	assert.NoError(t, os.WriteFile(
		filepath.Join(dir, "languages.toml"), []byte(text), 0o644,
	))
	t.Setenv("XDG_CONFIG_HOME", root)
}

func waitForWorkspaceServer(t *testing.T, session *lsp.Session) {
	t.Helper()
	assert.Eventually(t, func() bool {
		return !errors.Is(
			session.ApplyWorkspaceEdit("session-test", protocol.WorkspaceEdit{}),
			lsp.ErrUnknownLanguageServer,
		)
	}, 2*time.Second, 5*time.Millisecond)
}
