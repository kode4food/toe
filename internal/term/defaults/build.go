package defaults

import (
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
)

// RegisterDefaults installs the default command registry for an editor
func RegisterDefaults(model ui.Model, km *command.Keymaps) {
	r := &registry{km: km}
	registerDefaultCommands(r, model)
}

func registerDefaultCommands(r *registry, model ui.Model) {
	registerInsertCommands(r)
	registerMotionCommands(r)
	registerEditCommands(r)
	registerSelectionCommands(r, model)
	registerSearchCommands(r, model)
	registerFileCommands(r)
	registerBufferCommands(r)
	registerDirectoryCommands(r)
	registerConfigCommands(r)
	registerClipboardCommands(r)
	registerViewCommands(r, model)
	registerShellCommands(r, model)
	registerSupportCommands(r, model)
	labelPrefixNodes(r.km)
}

func labelPrefixNodes(km *command.Keymaps) {
	gKey := []command.KeyEvent{{Code: command.KeyCode{Char: 'g'}}}
	spcKey := []command.KeyEvent{{Code: command.KeyCode{Char: ' '}}}
	mKey := []command.KeyEvent{{Code: command.KeyCode{Char: 'm'}}}
	prevKey := []command.KeyEvent{{Code: command.KeyCode{Char: '['}}}
	nextKey := []command.KeyEvent{{Code: command.KeyCode{Char: ']'}}}
	zKey := []command.KeyEvent{{Code: command.KeyCode{Char: 'z'}}}
	ZKey := []command.KeyEvent{{Code: command.KeyCode{Char: 'Z'}}}
	cwKey := []command.KeyEvent{
		{Code: command.KeyCode{Char: 'w'}, Mods: command.ModCtrl},
	}
	spcwKey := []command.KeyEvent{
		{Code: command.KeyCode{Char: ' '}},
		{Code: command.KeyCode{Char: 'w'}},
	}
	spcwnKey := []command.KeyEvent{
		{Code: command.KeyCode{Char: ' '}},
		{Code: command.KeyCode{Char: 'w'}},
		{Code: command.KeyCode{Char: 'n'}},
	}
	cwnKey := []command.KeyEvent{
		{Code: command.KeyCode{Char: 'w'}, Mods: command.ModCtrl},
		{Code: command.KeyCode{Char: 'n'}},
	}
	for _, mode := range []string{"NOR", "SEL"} {
		km.LabelNode(mode, gKey, "Goto")
		km.LabelNode(mode, spcKey, "Space")
		km.LabelNode(mode, mKey, "Match")
		km.LabelNode(mode, prevKey, "Prev")
		km.LabelNode(mode, nextKey, "Next")
		km.LabelNode(mode, zKey, "View")
		km.LabelNode(mode, ZKey, "View")
		km.LabelNode(mode, cwKey, "Window")
		km.LabelNode(mode, cwnKey, "New split scratch buffer")
		km.LabelNode(mode, spcwKey, "Window")
		km.LabelNode(mode, spcwnKey, "New split scratch buffer")
	}
}
