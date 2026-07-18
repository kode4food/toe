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
	viewcfg "github.com/kode4food/toe/internal/view/config"
)

type optionSetter[T any] func(*view.Options, T)

func configFormatCmds() []command.Command {
	return []command.Command{
		{
			Name: actSetLanguage,
			DocString: "Set the language of current buffer (show current " +
				"language if no value specified)",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				doc, ok := e.FocusedDocument()
				if !ok {
					return command.Result{
						Message: i18n.Text(i18n.ErrorNoDocument),
					}
				}
				if args == nil || args.Empty() {
					lang := doc.Lang()
					if lang == "" {
						lang = "text"
					}
					return command.Result{Message: lang}
				}
				lang, _ := args.First()
				if lang == "text" {
					lang = ""
				}
				doc.SetLang(lang)
				return command.Result{Message: ""}
			},
			Modes:     command.DocumentModes(),
			Aliases:   []string{"lang"},
			Signature: kit.StaticSig(kit.OptionalArg(), languageNames()...),
		},
		{
			Name: actSetLineEnding,
			DocString: "Set the document's default line ending. Options: " +
				"crlf, lf",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				if args == nil || args.Empty() {
					doc, ok := e.FocusedDocument()
					if !ok {
						return command.Result{
							Message: i18n.Text(i18n.ErrorNoDocument),
						}
					}
					switch doc.LineEnding() {
					case core.LineEndingCRLF:
						return command.Result{Message: "crlf"}
					default:
						return command.Result{Message: "line feed"}
					}
				}
				name, _ := args.First()
				var le core.LineEnding
				switch name {
				case "lf":
					le = core.LineEndingLF
				case "crlf":
					le = core.LineEndingCRLF
				default:
					return command.Result{
						Message: "error: unknown line ending: " + name,
					}
				}
				if err := action.SetLineEnding(e, le); err != nil {
					return command.Result{Message: "error: " + err.Error()}
				}
				return command.Result{Message: ""}
			},
			Modes:     command.DocumentModes(),
			Aliases:   []string{"line-ending"},
			Signature: kit.StaticSig(kit.OptionalArg(), "crlf", "lf"),
		},
		{
			Name: actIndentStyle,
			DocString: "Set the indentation style for editing. ('t' for tabs " +
				"or 1-16 for number of spaces)",
			Run: func(e *view.Editor, args *command.Args) command.Result {
				doc, ok := e.FocusedDocument()
				if !ok {
					return command.Result{
						Message: i18n.Text(i18n.ErrorNoDocument),
					}
				}
				if args == nil || args.Empty() {
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
						return command.Result{
							Message: "error: expected 'tab' or spaces " +
								"count (1-16)",
						}
					}
					doc.SetIndentStyle(core.Spaces(uint8(n)))
				}
				return command.Result{Message: "indent style set"}
			},
			Modes: command.DocumentModes(),
			Signature: kit.StaticSig(
				kit.Sig(),
				"tabs", "tab", "t", "1", "2", "3", "4", "5", "6", "7", "8",
				"9", "10", "11", "12", "13", "14", "15", "16",
			),
		},
		{
			Name:      actEncoding,
			DocString: "Set encoding",
			Run: func(_ *view.Editor, _ *command.Args) command.Result {
				return command.Result{Message: "utf-8"}
			},
			Modes:     command.DocumentModes(),
			Signature: kit.Sig(),
		},
	}
}

func cursorShapeOption(
	key, mode string, set optionSetter[view.CursorKind],
) command.Option {
	return command.Option{
		Key: key,
		Get: func(e *view.Editor) (string, error) {
			return string(e.Options().CursorShapeForMode(mode)), nil
		},
		Set: func(e *view.Editor, s string) error {
			v, err := view.ParseCursorKind(s)
			if err != nil {
				return fmt.Errorf("%w: %s", viewcfg.ErrInvalidOption, s)
			}
			set(e.Options(), v)
			return nil
		},
	}
}

func statuslineModeOption(
	key, mode string, set optionSetter[string],
) command.Option {
	return command.Option{
		Key: key,
		Get: func(e *view.Editor) (string, error) {
			return e.Options().ModeNameForMode(mode), nil
		},
		Set: func(e *view.Editor, s string) error {
			set(e.Options(), s)
			return nil
		},
	}
}
