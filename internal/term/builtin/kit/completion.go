package kit

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

// FileCompleter completes a positional argument against filesystem entries
// under the editor's working directory
func FileCompleter(e *view.Editor, input string) []command.Completion {
	dir, pfx := filepath.Split(input)
	base := dir
	if base == "" {
		base = "."
	}
	if !filepath.IsAbs(base) {
		base = filepath.Join(e.Cwd(), base)
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil
	}
	out := make([]command.Completion, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, pfx) {
			continue
		}
		text := dir + name
		if entry.IsDir() {
			text += string(os.PathSeparator)
		}
		out = append(out, command.Completion{Text: text})
	}
	return out
}

// FileSig attaches file-path completion to a signature's positionals
func FileSig(sig command.Signature) command.Signature {
	sig.Completer = command.PositionalCompleter(FileCompleter)
	return sig
}

// StaticSig attaches fixed-choice completion to a signature's positionals
func StaticSig(sig command.Signature, items ...string) command.Signature {
	sig.Completer = command.PositionalCompleter(
		command.StaticCompleter(items...),
	)
	return sig
}
