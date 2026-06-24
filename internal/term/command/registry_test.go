package command_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/view"
)

type (
	registryRoot struct {
		Editor registryEditor `toml:"editor"`
	}

	registryEditor struct {
		ScrollOff int `toml:"scrolloff"`
	}
)

const defaultRegistryScrollOff = 3

func TestRegistry(t *testing.T) {
	t.Run("RegisterCommand prepends the name", func(t *testing.T) {
		km := command.NewKeymaps()
		reg := command.NewRegistry(km)
		cmd := command.Command{
			Aliases: []string{"write"},
			Run: func(
				*view.Editor, *command.Args,
			) command.Result {
				return command.Result{}
			},
		}
		assert.NoError(t, reg.RegisterCommand("write_all", cmd))

		got, ok := km.ResolveCommand("write_all")
		assert.True(t, ok)
		assert.Equal(t, []string{"write_all", "write"}, got.Aliases)
	})

	t.Run("RegisterModule installs commands", func(t *testing.T) {
		km := command.NewKeymaps()
		reg := command.NewRegistry(km)
		assert.NoError(t, reg.RegisterModule(command.Module{
			Commands: map[string]command.Command{
				"noop": registryCommand(),
			},
		}))

		_, ok := km.ResolveCommand("noop")
		assert.True(t, ok)
	})

	t.Run("OptionKeys returns sorted option keys", func(t *testing.T) {
		reg := registryWithOptions(t)
		assert.Equal(t, []string{
			"editor.cursorline",
			"editor.scrolloff",
		}, reg.OptionKeys())
	})

	t.Run("BoolOptionKeys returns toggleable keys", func(t *testing.T) {
		reg := registryWithOptions(t)
		assert.Equal(t, []string{"editor.cursorline"}, reg.BoolOptionKeys())
	})

	t.Run("LookupOption is case-insensitive", func(t *testing.T) {
		reg := registryWithOptions(t)
		_, ok := reg.LookupOption(" Editor.ScrollOff ")
		assert.True(t, ok)
	})

	t.Run("LookupOption misses unknown keys", func(t *testing.T) {
		reg := registryWithOptions(t)
		_, ok := reg.LookupOption("no.such.option")
		assert.False(t, ok)
	})

	t.Run("OptionCompleter filters by prefix", func(t *testing.T) {
		reg := registryWithOptions(t)
		completer := reg.OptionCompleter()
		results := completer(nil, "editor.s")
		assert.NotEmpty(t, results)
	})

	t.Run("BoolOptionCompleter filters by prefix", func(t *testing.T) {
		reg := registryWithOptions(t)
		completer := reg.BoolOptionCompleter()
		results := completer(nil, "editor.c")
		assert.NotEmpty(t, results)
	})

	t.Run("ApplyTOML decodes sections", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		reg := registryWithSection(t)
		raw := map[string]any{
			"editor": map[string]any{"scrolloff": 8},
		}

		assert.NoError(t, reg.ApplyTOML(e, raw))
		assert.Equal(t, 8, e.Options().ScrollOff)
	})

	t.Run("ApplyTOML resets sections", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		reg := registryWithSection(t)
		raw := map[string]any{
			"editor": map[string]any{"scrolloff": 8},
		}

		assert.NoError(t, reg.ApplyTOML(e, raw))
		assert.NoError(t, reg.ApplyTOML(e, map[string]any{}))
		assert.Equal(t, defaultRegistryScrollOff, e.Options().ScrollOff)
	})

	t.Run("ApplyTOML rejects invalid values", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		reg := registryWithSection(t)
		raw := map[string]any{
			"editor": map[string]any{"scrolloff": "bad"},
		}

		assert.Error(t, reg.ApplyTOML(e, raw))
	})
}

func registryCommand() command.Command {
	return command.Command{
		Run: func(*view.Editor, *command.Args) command.Result {
			return command.Result{}
		},
	}
}

func registryWithOptions(t *testing.T) *command.Registry {
	t.Helper()
	reg := command.NewRegistry(command.NewKeymaps())
	err := reg.RegisterModule(command.Module{
		Options: []command.Option{
			{Key: "editor.scrolloff"},
			{
				Key: "editor.cursorline",
				Toggle: func(*view.Editor) (string, error) {
					return "", nil
				},
			},
		},
	})
	assert.NoError(t, err)
	return reg
}

func registryWithSection(t *testing.T) *command.Registry {
	t.Helper()
	cfg := registryRoot{
		Editor: registryEditor{ScrollOff: defaultRegistryScrollOff},
	}
	reg := command.NewRegistry(command.NewKeymaps())
	err := reg.RegisterModule(command.Module{
		Section: &command.Section{
			Config: &cfg,
			Reset: func() {
				cfg.Editor = registryEditor{
					ScrollOff: defaultRegistryScrollOff,
				}
			},
			Apply: func(e *view.Editor) {
				e.Options().ScrollOff = cfg.Editor.ScrollOff
			},
		},
	})
	assert.NoError(t, err)
	return reg
}
