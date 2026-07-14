package builtin

import (
	"github.com/kode4food/toe/internal/term/builtin/clipboard"
	"github.com/kode4food/toe/internal/term/builtin/config"
	"github.com/kode4food/toe/internal/term/builtin/editing"
	"github.com/kode4food/toe/internal/term/builtin/files"
	"github.com/kode4food/toe/internal/term/builtin/kit"
	"github.com/kode4food/toe/internal/term/builtin/motion"
	"github.com/kode4food/toe/internal/term/builtin/picker"
	"github.com/kode4food/toe/internal/term/builtin/shell"
	"github.com/kode4food/toe/internal/term/builtin/vcs"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
)

// Register installs the default command registry for an editor and
// returns the registry so callers can apply TOML config to module sections
func Register(model ui.Model, km *command.Keymaps) (*command.Registry, error) {
	r := command.NewRegistry(km)
	if err := registerDefaultCommands(r, model); err != nil {
		return nil, err
	}
	labelLeaders(km)
	return r, nil
}

func registerDefaultCommands(r *command.Registry, model ui.Model) error {
	modules := []command.Module{
		editing.InsertModule(),
		files.CompletionModule(model),
		motion.CursorModule(model),
		editing.EditModule(),
		editing.SelectionModule(model),
		motion.SearchModule(model),
		files.FileModule(),
		files.BufferModule(),
		files.DirectoryModule(),
		config.ConfigurationModule(r),
		clipboard.Module(),
		config.ViewModule(model),
		shell.Module(model),
		files.SessionModule(model),
		config.LifecycleModule(),
		files.FormatModule(),
		files.LspModule(model),
		files.PickerModule(model),
		motion.JumplistModule(model),
		files.DiagnosticsModule(model),
		config.SupportModule(),
		picker.Module(model),
		editing.CommentModule(),
		config.MacroModule(model),
		vcs.Module(model),
	}
	for _, m := range modules {
		if err := r.RegisterModule(m); err != nil {
			return err
		}
	}
	return nil

}

// labelLeaders names the top-level leader keys shared across many modules,
// which no single module owns
func labelLeaders(km *command.Keymaps) {
	for _, mode := range []string{"NOR", "SEL"} {
		km.LabelNode(mode, kit.Char(' '), "Space")
		km.LabelNode(mode, kit.Char('['), "Prev")
		km.LabelNode(mode, kit.Char(']'), "Next")
	}
}
