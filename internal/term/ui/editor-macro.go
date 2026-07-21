package ui

import (
	"slices"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

type macroSlot struct {
	recording bool
	reg       rune
	keys      []command.KeyEvent
	macros    map[rune][]command.KeyEvent
	replayReg rune
	replayN   int
	hasReplay bool
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
		ms.replayN = n
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
	e.replayMacro(cx, ms.macros[ms.replayReg], ms.replayN)
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
			modeStr := mode.String()
			if replaySkip(k, mode) {
				continue
			}
			act, found, _ := cx.Keymaps.Lookup(modeStr, []command.KeyEvent{k})
			if found {
				cont := act(cx.Editor)
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
