package config

import (
	"fmt"
	"strconv"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/builtin/kit"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
	"github.com/kode4food/toe/internal/view/config"
)

type (
	optionGetter[T any] func(*view.Options) T
	optionSetter[T any] func(*view.Options, T)

	wsRenderGetter func(*view.WhitespaceRender) view.WhitespaceRenderValue
	wsRenderSetter func(*view.WhitespaceRender, *view.WhitespaceRenderValue)
)

const (
	errorUnknownLineEndingKey i18n.Key = "error.unknownLineEnding"
	errorExpectedIndentKey    i18n.Key = "error.expectedIndent"
)

var (
	errNoDocument        = i18n.NewError(i18n.ErrorNoDocument)
	errUnknownLineEnding = i18n.NewError(errorUnknownLineEndingKey)
	errExpectedIndent    = i18n.NewError(errorExpectedIndentKey)
)

func formatCmds() []command.Command {
	return []command.Command{
		{
			Name: actSetLanguage,
			DocString: "Set the buffer's language (show current if not " +
				"specified)",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				doc := e.FocusedDocument()
				if doc == nil {
					return command.Result{Error: errNoDocument}
				}
				if args == nil || args.IsEmpty() {
					lang := doc.Lang()
					if lang == "" {
						lang = view.DefaultLanguage
					}
					return command.Result{Message: lang}
				}
				lang, _ := args.First()
				if lang == view.DefaultLanguage {
					lang = ""
				}
				doc.SetLang(lang)
				return command.Result{}
			},
			Modes:     command.DocModes,
			Aliases:   []string{"lang"},
			Signature: kit.StaticSig(kit.OptionalArg(), languageNames()...),
		},
		{
			Name:      actSetLineEnding,
			DocString: "Set the document's line ending: crlf, lf, or native",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				if args == nil || args.IsEmpty() {
					doc := e.FocusedDocument()
					if doc == nil {
						return command.Result{Error: errNoDocument}
					}
					switch doc.LineEnding() {
					case core.LineEndingCRLF:
						return command.Result{
							Message: core.LineEndingNameCRLF,
						}
					default:
						return command.Result{Message: "line feed"}
					}
				}
				name, _ := args.First()
				le, err := core.ParseLineEnding(name)
				if err != nil {
					return command.Result{
						Error: errUnknownLineEnding.WithVars(i18n.Vars{
							"name": name,
						}),
					}
				}
				if err := action.SetLineEnding(e, le); err != nil {
					return command.Result{Error: err}
				}
				return command.Result{}
			},
			Modes:   command.DocModes,
			Aliases: []string{"line-ending"},
			Signature: kit.StaticSig(
				kit.OptionalArg(), core.LineEndingNames()...,
			),
		},
		{
			Name: actIndentStyle,
			DocString: "Set the indentation style ('t' for tabs, or 1-16 " +
				"spaces)",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				doc := e.FocusedDocument()
				if doc == nil {
					return command.Result{Error: errNoDocument}
				}
				if args == nil || args.IsEmpty() {
					return command.Result{
						Message: doc.IndentStyle().AsStr(),
					}
				}
				arg, _ := args.First()
				switch arg {
				case "tabs", "tab", "t":
					doc.SetIndentStyle(core.Tabs())
				default:
					n, err := strconv.Atoi(arg)
					if err != nil || n < 1 || n > core.MaxIndent {
						return command.Result{Error: errExpectedIndent}
					}
					doc.SetIndentStyle(core.Spaces(uint8(n)))
				}
				return command.Result{Message: "indent style set"}
			},
			Modes: command.DocModes,
			Signature: kit.StaticSig(
				kit.OptionalArg(),
				"tabs", "tab", "t", "1", "2", "3", "4", "5", "6", "7", "8",
				"9", "10", "11", "12", "13", "14", "15", "16",
			),
		},
	}
}

func cursorShapeOption(
	key string, mode view.Mode, set optionSetter[view.CursorKind],
) command.Option {
	return command.Option{
		Key: key,
		Get: func(e *view.Editor) (string, error) {
			return string(e.Options().CursorShapeForMode(mode)), nil
		},
		Set: func(e *view.Editor, s string) error {
			v, err := view.ParseCursorKind(s)
			if err != nil {
				return fmt.Errorf("%w: %s", config.ErrInvalidOption, s)
			}
			set(e.Options(), v)
			return nil
		},
		Complete: command.StaticCompleter(view.CursorKindNames()...),
	}
}

func statuslineItemsOption(
	key string, get optionGetter[[]view.StatusLineItem],
	set optionSetter[[]view.StatusLineItem],
) command.Option {
	return command.Option{
		Key: key,
		Get: func(e *view.Editor) (string, error) {
			items := get(e.Options())
			values := make([]string, len(items))
			for i, item := range items {
				values[i] = string(item.Element)
				if item.Pinned {
					values[i] += "!"
				}
			}
			return config.FormatStringSlice(values), nil
		},
		Set: func(e *view.Editor, s string) error {
			values, err := config.ParseStringSlice(s)
			if err != nil {
				return err
			}
			items := make([]view.StatusLineItem, len(values))
			for i, value := range values {
				if err := items[i].UnmarshalText([]byte(value)); err != nil {
					return err
				}
			}
			set(e.Options(), items)
			return nil
		},
		Complete: sliceCompleter(view.StatusLineElementNames()...),
	}
}

func sliceCompleter[T ~string](items ...T) command.CompletionFunc {
	values := make([]string, len(items))
	for i, item := range items {
		values[i] = config.FormatStringSlice([]string{string(item)})
	}
	return command.StaticCompleter(values...)
}
