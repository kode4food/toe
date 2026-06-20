package defaults_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileWrite(t *testing.T) {
	t.Run("write saves to the path from args", func(t *testing.T) {
		e, km := defaultsEnv(t, "hello")
		out := filepath.Join(e.Cwd(), "out.txt")
		res := runCmdArgs(t, km, e, "write", out)
		assert.Contains(t, res.Message, "written")
		data, err := os.ReadFile(out)
		assert.NoError(t, err)
		// the editor ensures a trailing newline on save
		assert.Equal(t, "hello\n", string(data))
	})

	t.Run("update without diff reports nothing to write", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		res := runCmd(t, km, e, "update")
		assert.Contains(t, res.Message, "no changes")
	})

	t.Run("update writes a modified file-backed buffer", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		path := filepath.Join(e.Cwd(), "u.txt")
		assert.NoError(t, os.WriteFile(path, []byte("orig"), 0o644))
		runCmdArgs(t, km, e, "open", path)
		setText(t, e, "X")
		res := runCmd(t, km, e, "update")
		assert.Contains(t, res.Message, "written")
	})
}

func TestFileOpenNew(t *testing.T) {
	t.Run("open switches to a file from disk", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		path := filepath.Join(e.Cwd(), "f.txt")
		assert.NoError(t, os.WriteFile(path, []byte("DISK"), 0o644))
		res := runCmdArgs(t, km, e, "open", path)
		assert.Contains(t, res.Message, "opened")
		assert.Equal(t, "DISK", docText(t, e))
	})

	t.Run("open without a filename errors", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		res := runCmd(t, km, e, "open")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("new makes an empty scratch buffer", func(t *testing.T) {
		e, km := defaultsEnv(t, "hello")
		res := runCmd(t, km, e, "new")
		assert.Contains(t, res.Message, "scratch")
		assert.Equal(t, "", docText(t, e))
	})
}

func TestFileMoveReadReload(t *testing.T) {
	t.Run("move relocates the file", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		src := filepath.Join(e.Cwd(), "src.txt")
		assert.NoError(t, os.WriteFile(src, []byte("M"), 0o644))
		runCmdArgs(t, km, e, "open", src)
		dst := filepath.Join(e.Cwd(), "dst.txt")
		res := runCmdArgs(t, km, e, "move", dst)
		assert.Contains(t, res.Message, "moved")
		_, err := os.Stat(dst)
		assert.NoError(t, err)
	})

	t.Run("read inserts a file into the buffer", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		path := filepath.Join(e.Cwd(), "r.txt")
		assert.NoError(t, os.WriteFile(path, []byte("READTEXT"), 0o644))
		res := runCmdArgs(t, km, e, "read", path)
		assert.Contains(t, res.Message, "inserted")
		assert.Contains(t, docText(t, e), "READTEXT")
	})

	t.Run("reload discards buffer changes", func(t *testing.T) {
		e, km := defaultsEnv(t, "")
		path := filepath.Join(e.Cwd(), "rl.txt")
		assert.NoError(t, os.WriteFile(path, []byte("DISK"), 0o644))
		runCmdArgs(t, km, e, "open", path)
		setText(t, e, "changed")
		res := runCmd(t, km, e, "reload")
		assert.Contains(t, res.Message, "reloaded")
		assert.Equal(t, "DISK", docText(t, e))
	})
}
