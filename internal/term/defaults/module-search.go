package defaults

import (
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
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
		Commands: []command.Command{
			{
				Name:      actCommandMode,
				DocString: "Enter command mode",
				Run:       Continuation(model.CmdModeAction()),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(char(':')),
			},
			{
				Name:      actSearch,
				DocString: "Search for regex pattern",
				Run:       Continuation(model.SearchAction(true)),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{
					"*": {{char('/')}, {z(char('/'))}, {Z(char('/'))}},
				},
			},
			{
				Name:      actSearchReverse,
				DocString: "Reverse search for regex pattern",
				Run:       Continuation(model.SearchAction(false)),
				Modes:     []string{"NOR", "SEL"},
				Keys: map[string][]command.KeyBinding{
					"*": {{char('?')}, {z(char('?'))}, {Z(char('?'))}},
				},
			},
			{
				Name:      actSearchNext,
				DocString: "Select next search match",
				Run:       Runner(action.SearchNext),
				Modes:     []string{"NOR"},
				Keys: map[string][]command.KeyBinding{
					"*": {{char('n')}, {z(char('n'))}, {Z(char('n'))}},
				},
			},
			{
				Name:      actSearchPrev,
				DocString: "Select previous search match",
				Run:       Runner(action.SearchPrev),
				Modes:     []string{"NOR"},
				Keys: map[string][]command.KeyBinding{
					"*": {{char('N')}, {z(char('N'))}, {Z(char('N'))}},
				},
			},
			{
				Name: actSearchSelectionWord,
				DocString: "Use current selection as the search pattern," +
					" automatically wrapping with `\\b` on word boundaries",
				Run:   Runner(action.SearchSelectionWord),
				Modes: []string{"NOR", "SEL"},
				Keys:  keys(char('*')),
			},
			{
				Name:      actMakeSearchWordBounded,
				DocString: "Modify current search to make it word bounded",
				Run:       Runner(action.MakeSearchWordBounded),
				Signature: sig(),
			},
			{
				Name:      actSearchSelection,
				DocString: "Use current selection as search pattern",
				Run:       Runner(action.SearchSelection),
				Modes:     []string{"NOR", "SEL"},
				Keys:      keys(alt('*')),
			},
			{
				Name:      actExtendSearchNext,
				DocString: "Add next search match to selection",
				Run:       Runner(action.ExtendSearchNext),
				Modes:     []string{"SEL"},
				Keys: map[string][]command.KeyBinding{
					"*": {{char('n')}, {z(char('n'))}, {Z(char('n'))}},
				},
			},
			{
				Name:      actExtendSearchPrev,
				DocString: "Add previous search match to selection",
				Run:       Runner(action.ExtendSearchPrev),
				Modes:     []string{"SEL"},
				Keys: map[string][]command.KeyBinding{
					"*": {{char('N')}, {z(char('N'))}, {Z(char('N'))}},
				},
			},
		},
		Options: []command.Option{
			editorBoolOption("search.smart-case",
				func(e *view.Editor) bool {
					return e.Options().SearchSmartCase
				},
				func(e *view.Editor, v bool) {
					e.Options().SearchSmartCase = v
				},
			),
			editorBoolOption("search.wrap-around",
				func(e *view.Editor) bool {
					return e.Options().SearchWrapAround
				},
				func(e *view.Editor, v bool) {
					e.Options().SearchWrapAround = v
				},
			),
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
