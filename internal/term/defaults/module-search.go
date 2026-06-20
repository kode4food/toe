package defaults

import (
	"strconv"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
	"github.com/kode4food/toe/internal/view/config"
)

type searchSection struct {
	Editor struct {
		Search struct {
			SmartCase  *bool `toml:"smart-case"`
			WrapAround *bool `toml:"wrap-around"`
		} `toml:"search"`
	} `toml:"editor"`
}

const (
	actCommandMode           = "enter_command_mode"
	actSearch                = "search_forward"
	actSearchReverse         = "search_backward"
	actSearchNext            = "search_next"
	actSearchPrev            = "search_prev"
	actSearchSelectionWord   = "search_selection_word"
	actMakeSearchWordBounded = "make_search_word_bounded"
	actSearchSelection       = "search_selection"
	actExtendSearchNext      = "extend_search_next"
	actExtendSearchPrev      = "extend_search_prev"
)

func searchModule(model ui.Model) command.Module {
	cfg := new(searchSection)
	z := prefixed(char('z'))
	Z := prefixed(char('Z'))

	return command.Module{
		Commands: map[string]command.Command{
			actCommandMode: {
				DocString: "Enter command mode",
				Run:       Continuation(model.CmdModeAction()),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char(':')),
			},
			actSearch: {
				DocString: "Search for regex pattern",
				Run:       Continuation(model.SearchAction(true)),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{
					"*": {{char('/')}, {z(char('/'))}, {Z(char('/'))}},
				},
			},
			actSearchReverse: {
				DocString: "Reverse search for regex pattern",
				Run:       Continuation(model.SearchAction(false)),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{
					"*": {{char('?')}, {z(char('?'))}, {Z(char('?'))}},
				},
			},
			actSearchNext: {
				DocString: "Select next search match",
				Run:       Runner(action.SearchNext),
				Modes:     []string{"NOR"},
				Keys: map[string][]command.KeyBinding{
					"*": {{char('n')}, {z(char('n'))}, {Z(char('n'))}},
				},
			},
			actSearchPrev: {
				DocString: "Select previous search match",
				Run:       Runner(action.SearchPrev),
				Modes:     []string{"NOR"},
				Keys: map[string][]command.KeyBinding{
					"*": {{char('N')}, {z(char('N'))}, {Z(char('N'))}},
				},
			},
			actSearchSelectionWord: {
				DocString: "Use current selection as the search pattern," +
					" automatically wrapping with `\\b` on word boundaries",
				Run:   Runner(action.SearchSelectionWord),
				Modes: []string{"NOR", "SEL"},
				Keys:  keys(char('*')),
			},
			actMakeSearchWordBounded: {
				DocString: "Modify current search to make it word bounded",
				Run:       Runner(action.MakeSearchWordBounded),
				Signature: sig(),
			},
			actSearchSelection: {
				DocString: "Use current selection as search pattern",
				Run:       Runner(action.SearchSelection),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt('*')),
			},
			actExtendSearchNext: {
				DocString: "Add next search match to selection",
				Run:       Runner(action.ExtendSearchNext),
				Modes:     []string{"SEL"},
				Keys: map[string][]command.KeyBinding{
					"*": {{char('n')}, {z(char('n'))}, {Z(char('n'))}},
				},
			},
			actExtendSearchPrev: {
				DocString: "Add previous search match to selection",
				Run:       Runner(action.ExtendSearchPrev),
				Modes:     []string{"SEL"},
				Keys: map[string][]command.KeyBinding{
					"*": {{char('N')}, {z(char('N'))}, {Z(char('N'))}},
				},
			},
		},
		Options: []command.Option{
			{
				Key: "editor.search.smart-case",
				Get: func(e *view.Editor) (string, error) {
					return strconv.FormatBool(e.Options().SearchSmartCase), nil
				},
				Set: func(e *view.Editor, s string) error {
					v, err := config.ParseBool(s)
					if err != nil {
						return err
					}
					e.Options().SearchSmartCase = v
					return nil
				},
				Toggle: func(e *view.Editor) (string, error) {
					v := !e.Options().SearchSmartCase
					e.Options().SearchSmartCase = v
					return strconv.FormatBool(v), nil
				},
			},
			{
				Key: "editor.search.wrap-around",
				Get: func(e *view.Editor) (string, error) {
					return strconv.FormatBool(e.Options().SearchWrapAround), nil
				},
				Set: func(e *view.Editor, s string) error {
					v, err := config.ParseBool(s)
					if err != nil {
						return err
					}
					e.Options().SearchWrapAround = v
					return nil
				},
				Toggle: func(e *view.Editor) (string, error) {
					v := !e.Options().SearchWrapAround
					e.Options().SearchWrapAround = v
					return strconv.FormatBool(v), nil
				},
			},
		},
		Section: &command.Section{
			Config: cfg,
			Reset:  func() { *cfg = searchSection{} },
			Apply: func(e *view.Editor) {
				opts := e.Options()
				opts.SearchSmartCase = boolOr(
					cfg.Editor.Search.SmartCase, true,
				)
				opts.SearchWrapAround = boolOr(
					cfg.Editor.Search.WrapAround, true,
				)
			},
		},
	}
}
