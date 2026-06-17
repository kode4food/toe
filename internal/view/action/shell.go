package action

import (
	"bytes"
	"os"
	"os/exec"
	"strings"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// ShellPipe pipes each selection through a shell command, replacing the
// selection with the command's stdout
func ShellPipe(e *view.Editor, cmdStr string) error {
	v, ok := e.FocusedView()
	if !ok {
		return nil
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return nil
	}
	if doc.Readonly() {
		return view.ErrReadonly
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	changes := make([]core.Change, 0, len(ranges))
	for _, r := range ranges {
		frag, err := r.Fragment(text)
		if err != nil {
			continue
		}
		out, err := runShell(e, cmdStr, frag)
		if err != nil {
			return err
		}
		changes = append(changes, core.TextChange(r.From(), r.To(), out))
	}
	if len(changes) == 0 {
		return nil
	}
	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return err
	}
	newSel, err := sel.Map(cs)
	if err != nil {
		return err
	}
	tx := core.NewTransaction(text).WithChanges(cs).WithSelection(newSel)
	return e.Apply(tx)
}

// ShellPipeTo pipes each selection to a shell command, discarding output
func ShellPipeTo(e *view.Editor, cmdStr string) error {
	v, ok := e.FocusedView()
	if !ok {
		return nil
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return nil
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	for _, r := range sel.Ranges() {
		frag, err := r.Fragment(text)
		if err != nil {
			continue
		}
		cmd := makeShellCmd(e, cmdStr)
		cmd.Stdin = strings.NewReader(frag)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

// ShellInsertOutput runs a shell command and inserts its output before each
// cursor position
func ShellInsertOutput(e *view.Editor, cmdStr string) error {
	out, err := runShell(e, cmdStr, "")
	if err != nil {
		return err
	}
	v, ok := e.FocusedView()
	if !ok {
		return nil
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return nil
	}
	if doc.Readonly() {
		return view.ErrReadonly
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	changes := make([]core.Change, 0, len(ranges))
	seen := map[int]bool{}
	for _, r := range ranges {
		pos := r.From()
		if seen[pos] {
			continue
		}
		seen[pos] = true
		changes = append(changes, core.TextChange(pos, pos, out))
	}
	if len(changes) == 0 {
		return nil
	}
	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return err
	}
	newSel, err := sel.Map(cs)
	if err != nil {
		return err
	}
	tx := core.NewTransaction(text).WithChanges(cs).WithSelection(newSel)
	return e.Apply(tx)
}

// ShellAppendOutput runs a shell command and appends its output after each
// selection
func ShellAppendOutput(e *view.Editor, cmdStr string) error {
	out, err := runShell(e, cmdStr, "")
	if err != nil {
		return err
	}
	v, ok := e.FocusedView()
	if !ok {
		return nil
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return nil
	}
	if doc.Readonly() {
		return view.ErrReadonly
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	changes := make([]core.Change, 0, len(ranges))
	seen := map[int]bool{}
	for _, r := range ranges {
		pos := r.To()
		if seen[pos] {
			continue
		}
		seen[pos] = true
		changes = append(changes, core.TextChange(pos, pos, out))
	}
	if len(changes) == 0 {
		return nil
	}
	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return err
	}
	newSel, err := sel.Map(cs)
	if err != nil {
		return err
	}
	tx := core.NewTransaction(text).WithChanges(cs).WithSelection(newSel)
	return e.Apply(tx)
}

// ShellKeepPipe pipes each selection through a shell command and keeps only
// selections for which the command exits with status 0
func ShellKeepPipe(e *view.Editor, cmdStr string) error {
	v, ok := e.FocusedView()
	if !ok {
		return nil
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return nil
	}
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	var kept []core.Range
	primary := 0
	for i, r := range ranges {
		frag, err := r.Fragment(text)
		if err != nil {
			continue
		}
		cmd := makeShellCmd(e, cmdStr)
		cmd.Stdin = strings.NewReader(frag)
		if err := cmd.Run(); err == nil {
			if i == sel.PrimaryIndex() {
				primary = len(kept)
			}
			kept = append(kept, r)
		}
	}
	if len(kept) == 0 {
		return nil
	}
	newSel, err := core.NewSelection(kept, primary)
	if err != nil {
		return err
	}
	doc.SetSelectionFor(v.ID(), newSel)
	return nil
}

// ReadFile inserts the contents of the given file before each cursor
func ReadFile(e *view.Editor, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	v, ok := e.FocusedView()
	if !ok {
		return nil
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return nil
	}
	if doc.Readonly() {
		return view.ErrReadonly
	}
	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	text := doc.Text()
	sel := doc.SelectionFor(v.ID())
	ranges := sel.Ranges()
	changes := make([]core.Change, 0, len(ranges))
	seen := map[int]bool{}
	for _, r := range ranges {
		pos := r.From()
		if seen[pos] {
			continue
		}
		seen[pos] = true
		changes = append(changes, core.TextChange(pos, pos, content))
	}
	if len(changes) == 0 {
		return nil
	}
	cs, err := core.NewChangeSetFromChanges(text, changes)
	if err != nil {
		return err
	}
	newSel, err := sel.Map(cs)
	if err != nil {
		return err
	}
	tx := core.NewTransaction(text).WithChanges(cs).WithSelection(newSel)
	return e.Apply(tx)
}

// ShellRunCommand runs a shell command and returns combined stdout+stderr.
// Nothing is inserted into the document; the caller displays the result
func ShellRunCommand(e *view.Editor, cmdStr string) (string, error) {
	cmd := makeShellCmd(e, cmdStr)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil && out.Len() == 0 {
		return "", err
	}
	return strings.TrimRight(out.String(), "\n"), nil
}

func makeShellCmd(e *view.Editor, cmdStr string) *exec.Cmd {
	sh := e.Config().Shell()
	args := append(sh[1:], cmdStr)
	return exec.Command(sh[0], args...)
}

func runShell(e *view.Editor, cmdStr, input string) (string, error) {
	cmd := makeShellCmd(e, cmdStr)
	cmd.Stdin = strings.NewReader(input)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out.String(), nil
}
