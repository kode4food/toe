package defaults

import (
	"bytes"
	"errors"
	"os/exec"
	"strconv"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/lsp"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
	"github.com/kode4food/toe/internal/view/language"
)

const (
	actFormat             = "format"
	actReflow             = "reflow"
	actSort               = "sort"
	actReindentSelections = "format_selections"
)

func formatModule() command.Module {
	return command.Module{
		Commands: []command.Command{
			{
				Name: actFormat,
				DocString: "Format the file using an external formatter or " +
					"language server",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					return runFormatter(e)
				},
				Aliases:   []string{"fmt"},
				Signature: sig(),
			},
			{
				Name:      actReindentSelections,
				DocString: "Format selection",
				Run:       runFormatSelection,
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char('=')),
			},
			{
				Name: actReflow,
				DocString: "Hard-wrap the current selection of lines to a " +
					"given width",
				Run: func(e *view.Editor, args *command.Args) command.Result {
					width := language.DefaultTextWidth
					if tw := e.Options().TextWidth; tw != nil {
						width = *tw
					}
					if args != nil && !args.Empty() {
						s, _ := args.First()
						n, err := strconv.Atoi(s)
						if err != nil || n < 1 {
							return command.Result{
								Message: "error: invalid width",
							}
						}
						width = n
					}
					action.ReflowSelections(e, width)
					return command.Result{}
				},
				Signature: optionalArg(),
			},
			{
				Name:      actSort,
				DocString: "Sort ranges in selection",
				Run: func(e *view.Editor, args *command.Args) command.Result {
					reverse := args != nil && args.HasFlag("reverse")
					insensitive := args != nil && args.HasFlag("insensitive")
					err := action.SortSelections(e, reverse, insensitive)
					if err != nil {
						return command.Result{Message: "error: " + err.Error()}
					}
					return command.Result{}
				},
				Signature: command.Signature{
					Flags: []command.Flag{
						{Name: "reverse", Alias: 'r'},
						{Name: "insensitive", Alias: 'i'},
					},
				},
			},
		},
	}
}

func runFormatter(e *view.Editor) command.Result {
	v, ok := e.FocusedView()
	if !ok {
		return command.Result{}
	}
	doc, ok := e.FocusedDocument()
	if !ok {
		return command.Result{}
	}
	if doc.ReadOnly() {
		return command.Result{Message: "error: buffer is read-only"}
	}

	lang := language.LoadLanguage(doc.Lang())
	if lang.Formatter == nil {
		return runLSPFormatter(e, doc, v.ID())
	}

	text := doc.Text().String()
	cmd := exec.Command(lang.Formatter.Command, lang.Formatter.Args...)
	cmd.Stdin = bytes.NewBufferString(text)
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		msg := lang.Formatter.Command + ": " + err.Error()
		if errOut.Len() > 0 {
			msg = lang.Formatter.Command + ": " + errOut.String()
		}
		return command.Result{Message: "error: " + msg}
	}

	formatted := out.String()
	if formatted == text {
		return command.Result{}
	}

	rope := doc.Text()
	n := rope.LenChars()
	cs, err := core.NewChangeSetFromChanges(rope, []core.Change{
		core.TextChange(0, n, formatted),
	})
	if err != nil {
		return command.Result{Message: "error: " + err.Error()}
	}
	sel := doc.SelectionFor(v.ID())
	tx := core.NewTransaction(rope).WithChanges(cs).WithSelection(sel)
	if err := e.Apply(tx); err != nil {
		return command.Result{Message: "error: " + err.Error()}
	}
	return command.Result{}
}

func runLSPFormatter(
	e *view.Editor, doc *view.Document, viewID view.Id,
) command.Result {
	ctl := e.LanguageServerController()
	if ctl == nil {
		return command.Result{
			Message: "no formatter configured for " + doc.Lang(),
		}
	}
	err := ctl.FormatDocument(doc, viewID)
	if errors.Is(err, lsp.ErrNoLanguageServer) {
		return command.Result{
			Message: "no formatter configured for " + doc.Lang(),
		}
	}
	if err != nil {
		return command.Result{Message: "error: " + err.Error()}
	}
	return command.Result{}
}

func autoFormat(e *view.Editor) {
	doc, ok := e.FocusedDocument()
	if !ok {
		return
	}
	if language.LoadLanguage(doc.Lang()).AutoFormat {
		runFormatter(e)
	}
}

func runFormatSelection(
	e *view.Editor, _ *command.Args,
) command.Result {
	doc, ok := e.FocusedDocument()
	if !ok {
		return command.Result{}
	}
	v, ok := e.FocusedView()
	if !ok {
		return command.Result{}
	}
	ctl := e.LanguageServerController()
	if ctl == nil {
		return command.Result{
			Message: "error: No configured language server supports " +
				"range formatting",
		}
	}
	err := ctl.FormatSelection(doc, v.ID())
	if errors.Is(err, lsp.ErrNoLanguageServer) {
		return command.Result{
			Message: "error: No configured language server supports " +
				"range formatting",
		}
	}
	if errors.Is(err, lsp.ErrFormatSelection) {
		return command.Result{
			Message: "error: format_selections only supports " +
				"a single selection for now",
		}
	}
	if err != nil {
		return command.Result{Message: "error: " + err.Error()}
	}
	return command.Result{}
}
