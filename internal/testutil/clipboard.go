package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	pbcopyScript = `#!/bin/sh
/bin/cat > "$CLIPFILE"
`
	pbpasteScript = `#!/bin/sh
/bin/cat "$CLIPFILE"
`
	xclipScript = `#!/bin/sh
case " $* " in
*" -o "*) /bin/cat "$CLIPFILE";;
*) /bin/cat > "$CLIPFILE";;
esac
`
	xselScript = `#!/bin/sh
case " $* " in
*" --output "*) /bin/cat "$CLIPFILE";;
*) /bin/cat > "$CLIPFILE";;
esac
`
	wlcopyScript = `#!/bin/sh
/bin/cat > "$CLIPFILE"
`
	wlpasteScript = `#!/bin/sh
/bin/cat "$CLIPFILE"
`
)

func WriteFakeClipboardTools(t testing.TB, clipFile string) {
	t.Helper()
	dir := t.TempDir()
	writeTool(t, dir, "pbcopy", pbcopyScript)
	writeTool(t, dir, "pbpaste", pbpasteScript)
	writeTool(t, dir, "xclip", xclipScript)
	writeTool(t, dir, "xsel", xselScript)
	writeTool(t, dir, "wl-copy", wlcopyScript)
	writeTool(t, dir, "wl-paste", wlpasteScript)
	t.Setenv("PATH", dir)
	t.Setenv("CLIPFILE", clipFile)
}

func writeTool(t testing.TB, dir, name, script string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755)
	if err != nil {
		t.Fatalf("write fake clipboard tool %s: %v", name, err)
	}
}
