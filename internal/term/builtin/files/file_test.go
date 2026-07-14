package files_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/builtin/test"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/testutil"
)

func TestFileWrite(t *testing.T) {
	t.Run("write saves to the path from args", func(t *testing.T) {
		e, km := test.Env(t, "hello")
		out := filepath.Join(e.Cwd(), "out.txt")
		res := test.RunCmdArgs(t, km, e, "write", out)
		assert.Contains(t, res.Message, "written")
		data, err := os.ReadFile(out)
		assert.NoError(t, err)
		// the editor ensures a trailing newline on save
		assert.Equal(t, "hello\n", string(data))
	})

	t.Run("no diff reports nothing to write", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmd(t, km, e, "update")
		assert.Contains(t, res.Message, "no changes")
	})

	t.Run("modified file-backed buffer is written", func(t *testing.T) {
		e, km := test.Env(t, "")
		path := filepath.Join(e.Cwd(), "u.txt")
		assert.NoError(t, os.WriteFile(path, []byte("orig"), 0o644))
		test.RunCmdArgs(t, km, e, "open", path)
		testutil.SetEditorText(t, e, "X")
		res := test.RunCmd(t, km, e, "update")
		assert.Contains(t, res.Message, "written")
	})
}

func TestFileOpenNew(t *testing.T) {
	t.Run("open switches to a file from disk", func(t *testing.T) {
		e, km := test.Env(t, "")
		path := filepath.Join(e.Cwd(), "f.txt")
		assert.NoError(t, os.WriteFile(path, []byte("DISK"), 0o644))
		res := test.RunCmdArgs(t, km, e, "open", path)
		assert.Contains(t, res.Message, "opened")
		assert.Equal(t, "DISK", test.DocText(t, e))
	})

	t.Run("open without a filename errors", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmd(t, km, e, "open")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("new makes an empty scratch buffer", func(t *testing.T) {
		e, km := test.Env(t, "hello")
		res := test.RunCmd(t, km, e, "new")
		assert.Contains(t, res.Message, "scratch")
		assert.Equal(t, "", test.DocText(t, e))
	})
}

func TestFileMoveReadReload(t *testing.T) {
	t.Run("move relocates the file", func(t *testing.T) {
		e, km := test.Env(t, "")
		src := filepath.Join(e.Cwd(), "src.txt")
		assert.NoError(t, os.WriteFile(src, []byte("M"), 0o644))
		test.RunCmdArgs(t, km, e, "open", src)
		dst := filepath.Join(e.Cwd(), "dst.txt")
		res := test.RunCmdArgs(t, km, e, "move", dst)
		assert.Contains(t, res.Message, "moved")
		_, err := os.Stat(dst)
		assert.NoError(t, err)
		_, err = os.Stat(src)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("read inserts a file into the buffer", func(t *testing.T) {
		e, km := test.Env(t, "")
		path := filepath.Join(e.Cwd(), "r.txt")
		assert.NoError(t, os.WriteFile(path, []byte("READTEXT"), 0o644))
		res := test.RunCmdArgs(t, km, e, "read", path)
		assert.Contains(t, res.Message, "inserted")
		assert.Contains(t, test.DocText(t, e), "READTEXT")
	})

	t.Run("reload discards buffer changes", func(t *testing.T) {
		e, km := test.Env(t, "")
		path := filepath.Join(e.Cwd(), "rl.txt")
		assert.NoError(t, os.WriteFile(path, []byte("DISK"), 0o644))
		test.RunCmdArgs(t, km, e, "open", path)
		testutil.SetEditorText(t, e, "changed")
		res := test.RunCmd(t, km, e, "reload")
		assert.Contains(t, res.Message, "reloaded")
		assert.Equal(t, "DISK", test.DocText(t, e))
	})
}

func TestFileWriteVariants(t *testing.T) {
	t.Run("write! force-writes to path", func(t *testing.T) {
		e, km := test.Env(t, "hello")
		out := filepath.Join(e.Cwd(), "fw.txt")
		res := test.RunCmdArgs(t, km, e, "write!", out)
		assert.Contains(t, res.Message, "written")
	})

	t.Run("write_all writes all buffers", func(t *testing.T) {
		e, km := test.TwoBufferEnv(t)
		res := test.RunCmd(t, km, e, "write_all")
		assert.Contains(t, res.Message, "written")
	})

	t.Run("write_all reports save error", func(t *testing.T) {
		e, km := test.Env(t, "dirty")
		res := test.RunCmd(t, km, e, "write_all")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("write-all! force-writes all buffers", func(t *testing.T) {
		e, km := test.TwoBufferEnv(t)
		res := test.RunCmd(t, km, e, "write-all!")
		assert.Contains(t, res.Message, "written")
	})

	t.Run("write_quit saves and signals quit", func(t *testing.T) {
		e, km := test.Env(t, "hello")
		out := filepath.Join(e.Cwd(), "wq.txt")
		res := test.RunCmdArgs(t, km, e, "write_quit", out)
		assert.Equal(t, command.SignalQuit, res.Signal)
	})

	t.Run("write-quit! force-saves and signals quit", func(t *testing.T) {
		e, km := test.Env(t, "hello")
		out := filepath.Join(e.Cwd(), "wq2.txt")
		res := test.RunCmdArgs(t, km, e, "write-quit!", out)
		assert.Equal(t, command.SignalQuit, res.Signal)
	})

	t.Run("saves all and signals quit", func(t *testing.T) {
		e, km := test.TwoBufferEnv(t)
		res := test.RunCmd(t, km, e, "write_quit_all")
		assert.Equal(t, command.SignalQuit, res.Signal)
	})

	t.Run("write_quit_all reports save error", func(t *testing.T) {
		e, km := test.Env(t, "dirty")
		res := test.RunCmd(t, km, e, "write_quit_all")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("force-saves all and signals quit", func(t *testing.T) {
		e, km := test.TwoBufferEnv(t)
		res := test.RunCmd(t, km, e, "write-quit-all!")
		assert.Equal(t, command.SignalQuit, res.Signal)
	})

	t.Run("write_buffer_close saves and closes", func(t *testing.T) {
		e, km := test.Env(t, "hello")
		out := filepath.Join(e.Cwd(), "wbc.txt")
		res := test.RunCmdArgs(t, km, e, "write_buffer_close", out)
		assert.Contains(t, res.Message, "written")
	})

	t.Run("force-saves and closes", func(t *testing.T) {
		e, km := test.Env(t, "hello")
		out := filepath.Join(e.Cwd(), "wbc2.txt")
		res := test.RunCmdArgs(t, km, e, "write-buffer-close!", out)
		assert.Contains(t, res.Message, "written")
	})
}

func TestFileNoPath(t *testing.T) {
	t.Run("write nil args hits no-path branch", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmd(t, km, e, "write")
		// scratch buffer with no path will error - we just want no panic
		_ = res
	})

	t.Run("read nil args errors", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmd(t, km, e, "read")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("move nil args errors", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmd(t, km, e, "move")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("move! nil args errors", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmd(t, km, e, "move!")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("move on modified doc warns", func(t *testing.T) {
		e, km := test.Env(t, "dirty")
		res := test.RunCmdArgs(t, km, e, "move", "/tmp/dest.txt")
		assert.Contains(t, res.Message, "error")
	})
}

func TestFileReloadAll(t *testing.T) {
	t.Run("reloads all file-backed buffers", func(t *testing.T) {
		e, km := test.TwoBufferEnv(t)
		res := test.RunCmd(t, km, e, "reload_all")
		assert.NotContains(t, res.Message, "error")
	})
}

func TestFileReloadError(t *testing.T) {
	t.Run("reload scratch buffer errors", func(t *testing.T) {
		e, km := test.Env(t, "hello")
		res := test.RunCmd(t, km, e, "reload")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("read bad path errors", func(t *testing.T) {
		e, km := test.Env(t, "")
		res := test.RunCmdArgs(t, km, e, "read", "/no/such/dir/file_xyz.txt")
		assert.Contains(t, res.Message, "error")
	})
}

func TestFileMoveErrors(t *testing.T) {
	t.Run("move no document returns error", func(t *testing.T) {
		e, km := test.Env(t, "")
		v, ok := e.FocusedView()
		assert.True(t, ok)
		e.CloseView(v.ID())
		res := test.RunCmdArgs(t, km, e, "move", "/tmp/dest.txt")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("move MoveFocusedFile error returns error", func(t *testing.T) {
		e, km := test.Env(t, "")
		src := filepath.Join(e.Cwd(), "src.txt")
		assert.NoError(t, os.WriteFile(src, []byte("M"), 0o644))
		test.RunCmdArgs(t, km, e, "open", src)
		res := test.RunCmdArgs(t, km, e, "move", "/dev/null/cannot/create.txt")
		assert.Contains(t, res.Message, "error")
	})

	t.Run("move! no document returns error", func(t *testing.T) {
		e, km := test.Env(t, "")
		v, ok := e.FocusedView()
		assert.True(t, ok)
		e.CloseView(v.ID())
		res := test.RunCmdArgs(t, km, e, "move!", "/tmp/dest.txt")
		assert.Contains(t, res.Message, "error")
	})
}

func TestFileMoveForce(t *testing.T) {
	t.Run("move! relocates the file", func(t *testing.T) {
		e, km := test.Env(t, "")
		src := filepath.Join(e.Cwd(), "src2.txt")
		assert.NoError(t, os.WriteFile(src, []byte("M"), 0o644))
		test.RunCmdArgs(t, km, e, "open", src)
		dst := filepath.Join(e.Cwd(), "dst2.txt")
		res := test.RunCmdArgs(t, km, e, "move!", dst)
		assert.Contains(t, res.Message, "moved")
	})
}

func TestFileOptions(t *testing.T) {
	for _, tc := range []struct{ key, val string }{
		{"insert-final-newline", "true"},
		{"trim-final-newlines", "true"},
		{"trim-trailing-whitespace", "true"},
	} {
		t.Run("toggle "+tc.key, func(t *testing.T) {
			e, km := test.Env(t, "")
			res := test.RunCmdArgs(t, km, e, "toggle_option", tc.key)
			assert.Contains(t, res.Message, "is now set to")
		})
	}
}
