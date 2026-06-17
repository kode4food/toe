package defaults

import (
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/action"
)

func registerInsertCommands(r *registry) {
	r.RegisterCommand(actNormalMode, command.Command{
		DocString: "Enter normal mode",
		Run:       Continuation(normalModeAction),
		Modes:     []string{"INS"},
		Keys:      keyBinding(special("esc")),
	})
	r.RegisterCommand(actInsertRegister, command.Command{
		DocString: "Insert register",
		Run:       Continuation(insertRegisterAction),
		Modes:     []string{"INS"},
		Keys:      keyBinding(ctrl('r')),
	})
	r.RegisterCommand(actCommitUndoCheckpoint, command.Command{
		DocString: "Commit changes to new checkpoint",
		Run:       Runner(action.CommitUndoCheckpoint),
		Modes:     []string{"INS"},
		Keys:      keyBinding(ctrl('s')),
	})
	r.RegisterCommand(actDeleteWordBackward, command.Command{
		DocString: "Delete previous word",
		Run:       Runner(action.DeleteWordBackward),
		Modes:     []string{"INS"},
		Keys:      keyBinding(ctrl('w'), altSpecial("backspace")),
	})
	r.RegisterCommand(actDeleteWordForward, command.Command{
		DocString: "Delete next word",
		Run:       Runner(action.DeleteWordForward),
		Modes:     []string{"INS"},
		Keys:      keyBinding(alt('d'), altSpecial("del")),
	})
	r.RegisterCommand(actKillToLineStart, command.Command{
		DocString: "Delete till start of line",
		Run:       Runner(action.KillToLineStart),
		Modes:     []string{"INS"},
		Keys:      keyBinding(ctrl('u')),
	})
	r.RegisterCommand(actKillToLineEnd, command.Command{
		DocString: "Delete till end of line",
		Run:       Runner(action.KillToLineEnd),
		Modes:     []string{"INS"},
		Keys:      keyBinding(ctrl('k')),
	})
	r.RegisterCommand(actDeleteCharBackward, command.Command{
		DocString: "Delete previous char",
		Run:       Runner(action.DeleteCharBackward),
		Modes:     []string{"INS"},
		Keys:      keyBinding(ctrl('h'), special("backspace"), shift("backspace")),
	})
	r.RegisterCommand(actDeleteCharForward, command.Command{
		DocString: "Delete next char",
		Run:       Runner(action.DeleteCharForward),
		Modes:     []string{"INS"},
		Keys:      keyBinding(ctrl('d'), special("del")),
	})
	r.RegisterCommand(actInsertNewline, command.Command{
		DocString: "Insert newline char",
		Run:       Runner(action.InsertNewline),
		Modes:     []string{"INS"},
		Keys:      keyBinding(ctrl('j'), special("ret")),
	})
	r.RegisterCommand(actSmartTab, command.Command{
		DocString: "Insert tab if all cursors have all whitespace to their " +
			"left; otherwise, run a separate command",
		Run:   Runner(action.SmartTab),
		Modes: []string{"INS"},
		Keys:  keyBinding(special("tab")),
	})
	r.RegisterCommand(actMoveUp, command.Command{
		DocString: "Move up",
		Run:       Runner(action.MoveUp),
		Modes:     []string{"INS"},
		Keys:      keyBinding(special("up")),
	})
	r.RegisterCommand(actMoveDown, command.Command{
		DocString: "Move down",
		Run:       Runner(action.MoveDown),
		Modes:     []string{"INS"},
		Keys:      keyBinding(special("down")),
	})
	r.RegisterCommand(actMoveLeft, command.Command{
		DocString: "Move left",
		Run:       Runner(action.MoveLeft),
		Modes:     []string{"INS"},
		Keys:      keyBinding(special("left")),
	})
	r.RegisterCommand(actMoveRight, command.Command{
		DocString: "Move right",
		Run:       Runner(action.MoveRight),
		Modes:     []string{"INS"},
		Keys:      keyBinding(special("right")),
	})
	r.RegisterCommand(actPageUp, command.Command{
		DocString: "Move page up",
		Run:       Runner(action.PageUp),
		Modes:     []string{"INS"},
		Keys:      keyBinding(special("pageup")),
	})
	r.RegisterCommand(actPageDown, command.Command{
		DocString: "Move page down",
		Run:       Runner(action.PageDown),
		Modes:     []string{"INS"},
		Keys:      keyBinding(special("pagedown")),
	})
	r.RegisterCommand(actGotoLineStart, command.Command{
		DocString: "Goto line start",
		Run:       Runner(action.MoveLineStart),
		Modes:     []string{"INS"},
		Keys:      keyBinding(special("home")),
	})
	r.RegisterCommand(actGotoLineEndNewline, command.Command{
		DocString: "Goto newline at line end",
		Run:       Runner(action.GotoLineEndNewline),
		Modes:     []string{"INS"},
		Keys:      keyBinding(special("end")),
	})
}

func insertRegisterAction(e *view.Editor) command.Continuation {
	e.SetHint("^r ...")
	return func(e *view.Editor, k command.KeyEvent) command.Continuation {
		if k.Code.Char != 0 && k.Mods == command.ModNone {
			action.PasteRegisterAtCursor(e, k.Code.Char)
		}
		e.SetHint("")
		return nil
	}
}
