package ui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/mattn/go-runewidth"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/tui"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

type macroSlot struct {
	recording   bool
	reg         rune
	keys        []command.KeyEvent
	macros      map[rune][]command.KeyEvent
	replayReg   rune
	replayCount int
	hasReplay   bool
}

// MacroRecordAction starts or stops macro recording. When not recording,
// prompts for a register key and begins recording. When already recording,
// stops and saves the macro to the chosen register
func (e *EditorComponent) MacroRecordAction(
	ed *view.Editor,
) command.Continuation {
	ms := e.macroSlot
	if ms.recording {
		if len(ms.keys) > 0 {
			ms.keys = ms.keys[:len(ms.keys)-1]
		}
		ms.macros[ms.reg] = slices.Clone(ms.keys)
		ms.recording = false
		ms.keys = nil
		ms.reg = 0
		return nil
	}
	ed.SetHint("Q ...")
	return func(ed *view.Editor, k command.KeyEvent) command.Continuation {
		if k.Code.Char == 0 || k.Mods != command.ModNone {
			ed.SetHint("")
			return nil
		}
		ms.recording = true
		ms.reg = k.Code.Char
		ms.keys = nil
		return nil
	}
}

// MacroReplayAction prompts for a register key and replays the macro stored
// there count times
func (e *EditorComponent) MacroReplayAction(
	ed *view.Editor,
) command.Continuation {
	ms := e.macroSlot
	if ms.recording {
		return nil
	}
	ed.SetHint("q ...")
	return func(ed *view.Editor, k command.KeyEvent) command.Continuation {
		if k.Code.Char == 0 || k.Mods != command.ModNone {
			ed.SetHint("")
			return nil
		}
		n := ed.Count()
		if n == 0 {
			n = 1
		}
		ms.replayReg = k.Code.Char
		ms.replayCount = n
		ms.hasReplay = true
		return nil
	}
}

func (e *EditorComponent) handleReplay(cx *Context) {
	ms := e.macroSlot
	if !ms.hasReplay {
		return
	}
	ms.hasReplay = false
	e.replayMacro(cx, ms.macros[ms.replayReg], ms.replayCount)
}

func (e *EditorComponent) replayMacro(
	cx *Context, keys []command.KeyEvent, n int,
) {
	for range n {
		i := 0
		for i < len(keys) {
			k := keys[i]
			i++
			mode := cx.Editor.Mode()
			if replaySkip(k, mode) {
				continue
			}
			lookup, ok := cx.Keymaps.Lookup(mode, []command.KeyEvent{k})
			if ok && lookup.Enabled(cx.Editor) {
				cont := lookup.Action(cx.Editor).Continuation
				for cont != nil && i < len(keys) {
					k = keys[i]
					i++
					cont = cont(cx.Editor, k)
				}
				cx.Editor.ResetCount()
			} else if mode == view.ModeInsert && k.IsTypable() {
				action.InsertChar(cx.Editor, k.Code.Char)
			}
		}
	}
}

func (e *EditorComponent) macroElems(
	cx *Context, base tui.Style,
) []statusElem {
	ms := e.macroSlot
	if !ms.recording {
		return nil
	}
	text := fmt.Sprintf("%s %c", i18n.Text(i18n.StatusMacroRecording), ms.reg)
	if e.macroBlink.phase%2 == 1 {
		return []statusElem{statusBadge(
			strings.Repeat(" ", runewidth.StringWidth(text)), base,
		)}
	}
	return []statusElem{statusBadge(
		text, cx.Theme().Get("ui.statusline.macro"),
	)}
}

func replaySkip(k command.KeyEvent, mode view.Mode) bool {
	if mode != view.ModeNormal && mode != view.ModeSelect {
		return false
	}
	if k.Mods == command.ModNone {
		switch k.Code.Char {
		case ':', '/', '?', 's', 'S', 'K', '|', '!', '$', 'Q', 'q':
			return true
		}
	}
	if k.Mods == command.ModAlt {
		switch k.Code.Char {
		case '|', '!', 'K':
			return true
		}
	}
	return false
}
