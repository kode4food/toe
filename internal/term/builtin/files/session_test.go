package files_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/geom"
	"github.com/kode4food/toe/internal/term/builtin"
	"github.com/kode4food/toe/internal/term/builtin/test"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

func TestSession(t *testing.T) {
	t.Run("save reports success", func(t *testing.T) {
		dir := t.TempDir()
		e, km := sessionEnv(t, dir, "file.go")
		res := test.RunCmd(t, km, e, "save_session")
		assert.Equal(t, "session saved", res.Message)
	})

	t.Run("no session file returns not found", func(t *testing.T) {
		e, km := sessionEditorInDir(t, t.TempDir())
		res := test.RunCmd(t, km, e, "restore_session")
		assert.Equal(t, "no session found", res.Message)
	})

	t.Run("restore restores documents", func(t *testing.T) {
		dir := t.TempDir()
		e, km := sessionEnv(t, dir, "file.go")
		res := test.RunCmd(t, km, e, "save_session")
		assert.Equal(t, "session saved", res.Message)

		e2, km2 := sessionEditorInDir(t, dir)
		res = test.RunCmd(t, km2, e2, "restore_session")
		assert.Empty(t, res.Message)
		assert.Len(t, e2.AllViews(), 1)
	})

	t.Run("restore restores options", func(t *testing.T) {
		dir := t.TempDir()
		e, km := sessionEnv(t, dir, "file.go")
		test.RunCmdArgs(t, km, e, "set_option", "auto-size true")
		test.RunCmdArgs(t, km, e, "set_option", "file-watch false")
		res := test.RunCmd(t, km, e, "save_session")
		assert.Equal(t, "session saved", res.Message)

		e2, km2 := sessionEditorInDir(t, dir)
		res = test.RunCmd(t, km2, e2, "restore_session")
		assert.Empty(t, res.Message)
		res = test.RunCmdArgs(t, km2, e2, "get_option", "auto-size")
		assert.Equal(t, "true", res.Message)
		res = test.RunCmdArgs(t, km2, e2, "get_option", "file-watch")
		assert.Equal(t, "false", res.Message)
	})

	t.Run("save stores only options off the base", func(t *testing.T) {
		dir := t.TempDir()
		e, km := sessionEnv(t, dir, "file.go")
		base := baseOptions(t, e)
		e.SetBaseOptions(func() map[string]string { return base })
		test.RunCmdArgs(t, km, e, "set_option", "scrolloff 12")
		res := test.RunCmd(t, km, e, "save_session")
		assert.Equal(t, "session saved", res.Message)

		data, err := os.ReadFile(view.WorkspaceSessionFile(dir))
		assert.NoError(t, err)
		var sess struct {
			Options map[string]string `json:"options"`
		}
		assert.NoError(t, json.Unmarshal(data, &sess))
		assert.Equal(t, map[string]string{"scrolloff": "12"}, sess.Options)
	})

	t.Run("restore invalid session errors", func(t *testing.T) {
		dir := t.TempDir()
		sessionDir := filepath.Join(dir, ".toe")
		assert.NoError(t, os.MkdirAll(sessionDir, 0o755))
		assert.NoError(t, os.WriteFile(
			filepath.Join(sessionDir, view.SessionFile),
			[]byte("{{{not toml"), 0o644,
		))
		e, km := sessionEditorInDir(t, dir)
		res := test.RunCmd(t, km, e, "restore_session")
		assert.Contains(t, res.Message, "error")
	})
}

func sessionEnv(
	t *testing.T, dir, filename string,
) (*view.Editor, *command.Keymaps) {
	t.Helper()
	p := filepath.Join(dir, filename)
	assert.NoError(t, os.WriteFile(p, []byte("package main\n"), 0o644))
	e, km := sessionEditorInDir(t, dir)
	_, err := e.OpenFile(p)
	assert.NoError(t, err)
	return e, km
}

func baseOptions(t *testing.T, e *view.Editor) map[string]string {
	t.Helper()
	km := command.NewKeymaps()
	reg, err := builtin.Register(ui.New(e, km), km)
	assert.NoError(t, err)
	values, err := reg.OptionValues(e)
	assert.NoError(t, err)
	return values
}

func sessionEditorInDir(
	t *testing.T, dir string,
) (*view.Editor, *command.Keymaps) {
	t.Helper()
	// pin the workspace root so FindWorkspace can't escape the temp dir to
	// an ancestor with a stray .git/.toe marker (e.g. /tmp/.git)
	assert.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	km := command.NewKeymaps()
	e := view.NewEditor(dir)
	e.ResizeTree(geom.Size{Width: 80, Height: 24})
	_, _ = builtin.Register(ui.New(e, km), km)
	return e, km
}
