package defaults

import (
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
)

// RegisterDefaults installs the default command registry for an editor and
// returns the registry so callers can apply TOML config to module sections
func RegisterDefaults(
	model ui.Model, km *command.Keymaps,
) (*command.Registry, error) {
	r := command.NewRegistry(km)
	if err := registerDefaultCommands(r, model); err != nil {
		return nil, err
	}
	labelPrefixNodes(km)
	return r, nil
}

func registerDefaultCommands(r *command.Registry, model ui.Model) error {
	modules := []command.Module{
		insertModule(),
		completionModule(model),
		motionModule(),
		editModule(),
		selectionModule(model),
		searchModule(model),
		fileModule(),
		bufferModule(),
		directoryModule(),
		configModule(r),
		clipboardModule(),
		viewModule(),
		shellModule(model),
		sessionModule(),
		lifecycleModule(),
		formatModule(),
		lspModule(model),
		supportModule(),
		pickerModule(model),
		commentModule(),
		macroModule(model),
	}
	for _, m := range modules {
		if err := r.RegisterModule(m); err != nil {
			return err
		}
	}
	return nil

}

func labelPrefixNodes(km *command.Keymaps) {
	gKey := char('g')
	spcKey := char(' ')
	mKey := char('m')
	prevKey := char('[')
	nextKey := char(']')
	zKey := char('z')
	ZKey := char('Z')
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
