package action

import (
	"bytes"
	"errors"
	"os/exec"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

// RunFormatterArgs is a parameter bundle for RunFormatter
type RunFormatterArgs struct {
	Editor  *view.Editor
	Doc     *view.Document
	ViewID  view.Id
	Command string
	Argv    []string
}

// RunFormatter runs an external formatter as a subprocess over the document's
// text and applies its output as a single edit, replacing the document text
// only when the formatter changed it
func RunFormatter(a RunFormatterArgs) error {
	text := a.Doc.Text().String()
	cmd := exec.Command(a.Command, a.Argv...)
	cmd.Stdin = bytes.NewBufferString(text)
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		msg := a.Command + ": " + err.Error()
		if errOut.Len() > 0 {
			msg = a.Command + ": " + errOut.String()
		}
		return errors.New(msg)
	}

	formatted := out.String()
	if formatted == text {
		return nil
	}

	rope := a.Doc.Text()
	n := rope.LenChars()
	cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
		core.TextChange(0, n, formatted),
	})
	if err != nil {
		return err
	}
	sel := a.Doc.SelectionFor(a.ViewID)
	tx := core.NewTransaction(rope).WithChanges(cs).WithSelection(sel)
	return a.Editor.Apply(tx)
}
