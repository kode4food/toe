package motion

import (
	"embed"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/builtin/kit"
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

//go:embed i18n/search.*.json
var searchFS embed.FS

// SearchModule returns the search and search-navigation commands
func SearchModule(model ui.Model) command.Module {
	cfg := new(searchSection)
	z := kit.Prefixed(kit.Char('z'))
	Z := kit.Prefixed(kit.Char('Z'))

	return command.Module{
		Translations: i18n.LoadTranslations(searchFS),
		Commands: []command.Command{
			{
				Name:      actCommandMode,
				DocString: "Enter command mode",
				Run:       kit.Runner(model.CmdModeAction),
				Modes:     command.CmdKeyModes,
				Keys:      kit.Keys(kit.Char(':')),
			},
			{
				Name:      actSearch,
				DocString: "Search for regex pattern",
				Run:       kit.Runner(model.SearchAction(true)),
				Modes:     command.DocNormalModes,
				Keys: map[view.Mode]command.KeyBinding{
					view.ModeAny: kit.Or(
						kit.Char('/'), z(kit.Char('/')), Z(kit.Char('/')),
					),
				},
			},
			{
				Name:      actSearchReverse,
				DocString: "Reverse search for regex pattern",
				Run:       kit.Runner(model.SearchAction(false)),
				Modes:     command.DocNormalModes,
				Keys: map[view.Mode]command.KeyBinding{
					view.ModeAny: kit.Or(
						kit.Char('?'), z(kit.Char('?')), Z(kit.Char('?')),
					),
				},
			},
			{
				Name:      actSearchNext,
				DocString: "Select next search match",
				Run:       kit.Runner(action.SearchNext),
				Counted:   true,
				Modes:     view.ModeNormal,
				Keys: map[view.Mode]command.KeyBinding{
					view.ModeAny: kit.Or(
						kit.Char('n'), z(kit.Char('n')), Z(kit.Char('n')),
					),
				},
			},
			{
				Name:      actSearchPrev,
				DocString: "Select previous search match",
				Run:       kit.Runner(action.SearchPrev),
				Counted:   true,
				Modes:     view.ModeNormal,
				Keys: map[view.Mode]command.KeyBinding{
					view.ModeAny: kit.Or(
						kit.Char('N'), z(kit.Char('N')), Z(kit.Char('N')),
					),
				},
			},
			{
				Name: actSearchSelectionWord,
				DocString: "Search for the selection, wrapped on word " +
					"boundaries",
				Run:   kit.Runner(action.SearchSelectionWord),
				Modes: command.DocNormalModes,
				Keys:  kit.Keys(kit.Char('*')),
			},
			{
				Name:      actMakeSearchWordBounded,
				DocString: "Modify current search to make it word bounded",
				Run:       kit.Runner(action.MakeSearchWordBounded),
				Modes:     command.DocNormalModes,
			},
			{
				Name:      actSearchSelection,
				DocString: "Use current selection as search pattern",
				Run:       kit.Runner(action.SearchSelection),
				Modes:     command.DocNormalModes,
				Keys:      kit.Keys(kit.Alt('*')),
			},
			{
				Name:      actExtendSearchNext,
				DocString: "Add next search match to selection",
				Run:       kit.Runner(action.ExtendSearchNext),
				Counted:   true,
				Modes:     view.ModeSelect,
				Keys: map[view.Mode]command.KeyBinding{
					view.ModeAny: kit.Or(
						kit.Char('n'), z(kit.Char('n')), Z(kit.Char('n')),
					),
				},
			},
			{
				Name:      actExtendSearchPrev,
				DocString: "Add previous search match to selection",
				Run:       kit.Runner(action.ExtendSearchPrev),
				Counted:   true,
				Modes:     view.ModeSelect,
				Keys: map[view.Mode]command.KeyBinding{
					view.ModeAny: kit.Or(
						kit.Char('N'), z(kit.Char('N')), Z(kit.Char('N')),
					),
				},
			},
		},
		Options: []command.Option{
			kit.EditorBoolOption("search.smart-case",
				func(e *view.Editor) bool {
					return e.Options().SearchSmartCase
				},
				func(e *view.Editor, v bool) {
					e.Options().SearchSmartCase = v
				},
			).WithDoc("Smart case regex searching"),
			kit.EditorBoolOption("search.wrap-around",
				func(e *view.Editor) bool {
					return e.Options().SearchWrapAround
				},
				func(e *view.Editor, v bool) {
					e.Options().SearchWrapAround = v
				},
			).WithDoc("Wrap search after the last match"),
		},
		Section: &command.Section{
			Config: cfg,
			Reset:  func() { *cfg = searchSection{} },
			Apply: func(e *view.Editor) {
				opts := e.Options()
				opts.SearchSmartCase = kit.BoolOr(
					cfg.Editor.Search.SmartCase, true,
				)
				opts.SearchWrapAround = kit.BoolOr(
					cfg.Editor.Search.WrapAround, true,
				)
			},
		},
	}
}
