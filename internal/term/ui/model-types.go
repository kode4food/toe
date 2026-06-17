package ui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/highlight"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
)

type (
	// Model is the root Bubbletea model — a thin wrapper around Compositor
	Model struct {
		compositor *Compositor
		context    *Context
		component  *EditorComponent
		initCmd    tea.Cmd
	}

	renderCache struct {
		// docCaches holds per-document raw-text, highlight, and search-match
		// caches so multiple panes showing different documents do not evict
		// each other's tokenization every frame
		docCaches map[view.DocumentId]*docRenderCache

		// rebuilt only when theme or mode changes between frames
		stylesKey  string
		lgStyles   *lipglossStyles
		tuiStyles  *tuiStyles
		hlFn       func(string) lipgloss.Style
		hlTUICache map[string]tui.Style

		viewRowMaps map[view.Id][]viewRowEntry
	}

	// docRenderCache memoizes a single document's derived render state, keyed
	// internally by revision so it is recomputed only when the document changes
	docRenderCache struct {
		rawTextRev    int
		rawTextCached string

		hlRev   int
		hlLang  string
		hlSpans []highlight.Span

		smRev   int
		smPat   string
		smSpans []matchSpan
	}

	viewRowEntry struct {
		logLine int
		offset  int
		prefixW int
	}

	macroSlot struct {
		recording bool
		reg       rune
		keys      []command.KeyEvent
		macros    map[rune][]command.KeyEvent
		replayReg rune
		replayN   int
		hasReplay bool
	}

	saveGenSlot struct{ gen int }

	autoSaveMsg struct{ gen int }

	pickerFeedMsg struct {
		items []PickerItem
		feed  <-chan PickerItem
		done  <-chan struct{}
	}

	pickerDynamicTriggerMsg struct {
		gen   int
		query string
	}

	pickerDynamicFeedMsg struct {
		gen   int
		items []PickerItem
		feed  <-chan PickerItem
	}

	matchSpan struct{ from, to int }

	// statusElem is a single rendered piece of a status bar
	statusElem struct {
		text  string
		style tui.Style
	}

	lipglossStyles struct {
		background       lipgloss.Style
		text             lipgloss.Style
		line             lipgloss.Style
		lineSelected     lipgloss.Style
		selection        lipgloss.Style
		selectionPrim    lipgloss.Style
		cursor           lipgloss.Style
		cursorPrim       lipgloss.Style
		cursorlinePrim   lipgloss.Style
		cursorlineSec    lipgloss.Style
		cursorcolumnPrim lipgloss.Style
		cursorcolumnSec  lipgloss.Style
		whitespace       lipgloss.Style
		indentGuide      lipgloss.Style
		ruler            lipgloss.Style
		searchMatch      lipgloss.Style
	}

	tuiStyles struct {
		text             tui.Style
		selection        tui.Style
		selectionPrim    tui.Style
		cursor           tui.Style
		cursorPrim       tui.Style
		cursorlinePrim   tui.Style
		cursorlineSec    tui.Style
		cursorcolumnPrim tui.Style
		cursorcolumnSec  tui.Style
		whitespace       tui.Style
		indentGuide      tui.Style
		searchMatch      tui.Style
	}
)
