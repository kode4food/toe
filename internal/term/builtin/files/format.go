package files

import (
	"errors"
	"strconv"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/builtin/kit"
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

var (
	errInvalidWidth          = i18n.NewError(i18n.ErrorInvalidWidth)
	errBufferReadOnly        = i18n.NewError(i18n.ErrorBufferReadOnly)
	errNoFormatter           = i18n.NewError(i18n.StatusNoFormatter)
	errNoRangeFormatting     = i18n.NewError(i18n.ErrorNoRangeFormatting)
	errFormatSelectionSingle = i18n.NewError(i18n.ErrorFormatSelectionSingle)
)

// FormatModule returns the document and selection format commands
func FormatModule() command.Module {
	return command.Module{
		Commands: []command.Command{
			{
				Name: actFormat,
				DocString: "Format the file using an external formatter or " +
					"language server",
				Run: func(e *view.Editor, _ *command.Args) command.Result {
					return runFormatter(e)
				},
				Modes:     command.DocumentModes(),
				Aliases:   []string{"fmt"},
				Signature: kit.Sig(),
			},
			{
				Name:      actReindentSelections,
				DocString: "Format selection",
				Run:       runFormatSelection,
				Modes:     []string{"NOR", "SEL"},
				Keys:      kit.Keys(kit.Char('=')),
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
							return command.Result{Error: errInvalidWidth}
						}
						width = n
					}
					action.ReflowSelections(e, width)
					return command.Result{}
				},
				Modes:     []string{"NOR", "SEL"},
				Signature: kit.OptionalArg(),
			},
			{
				Name:      actSort,
				DocString: "Sort ranges in selection",
				Run: func(e *view.Editor, args *command.Args) command.Result {
					reverse := args != nil && args.HasFlag("reverse")
					insensitive := args != nil && args.HasFlag("insensitive")
					err := action.SortSelections(e, reverse, insensitive)
					if err != nil {
						return command.Result{Error: err}
					}
					return command.Result{}
				},
				Modes: []string{"NOR", "SEL"},
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
		return command.Result{Error: errBufferReadOnly}
	}

	lang := language.LoadLanguage(doc.Lang())
	if lang.Formatter == nil {
		return runLSPFormatter(e, doc, v.ID())
	}

	err := action.RunFormatter(action.RunFormatterArgs{
		Editor:  e,
		Doc:     doc,
		ViewID:  v.ID(),
		Command: lang.Formatter.Command,
		Argv:    lang.Formatter.Args,
	})
	if err != nil {
		return command.Result{Error: err}
	}
	return command.Result{}
}

func runLSPFormatter(
	e *view.Editor, doc *view.Document, viewID view.Id,
) command.Result {
	ctl := e.LanguageServerController()
	if ctl == nil {
		return command.Result{
			Error: errNoFormatter.WithVars(i18n.Vars{
				"lang": doc.Lang(),
			}),
		}
	}
	err := ctl.FormatDocument(doc, viewID)
	if errors.Is(err, view.ErrNoLanguageServer) {
		return command.Result{
			Error: errNoFormatter.WithVars(i18n.Vars{
				"lang": doc.Lang(),
			}),
		}
	}
	if err != nil {
		return command.Result{Error: err}
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

func runFormatSelection(e *view.Editor, _ *command.Args) command.Result {
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
		return command.Result{Error: errNoRangeFormatting}
	}
	err := ctl.FormatSelection(doc, v.ID())
	if errors.Is(err, view.ErrNoLanguageServer) {
		return command.Result{Error: errNoRangeFormatting}
	}
	if errors.Is(err, view.ErrFormatSelection) {
		return command.Result{Error: errFormatSelectionSingle}
	}
	if err != nil {
		return command.Result{Error: err}
	}
	return command.Result{}
}
