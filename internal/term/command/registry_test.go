package command_test

import (
	"strconv"
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
			Commands: []command.Command{
				{
					Name: "noop",
					Run:  registryCommand().Run,
				},
			},
		}))

		_, ok := km.ResolveCommand("noop")
		assert.True(t, ok)
	})

	t.Run("OptionKeys returns sorted option keys", func(t *testing.T) {
		reg := registryWithOptions(t)
		assert.Equal(t, []string{
			"cursorline",
			"scrolloff",
		}, reg.OptionKeys())
	})

	t.Run("BoolOptionKeys returns toggleable keys", func(t *testing.T) {
		reg := registryWithOptions(t)
		assert.Equal(t, []string{"cursorline"}, reg.BoolOptionKeys())
	})

	t.Run("OptionValues returns current values", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		e.Options().ScrollOff = 9
		e.Options().CursorLine = true
		reg := registryWithLiveOptions(t)

		values, err := reg.OptionValues(e)

		assert.NoError(t, err)
		assert.Equal(t, "true", values["cursorline"])
		assert.Equal(t, "9", values["scrolloff"])
	})

	t.Run("OptionValues returns get errors", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		reg := registryWithFailingOption(t)

		_, err := reg.OptionValues(e)

		assert.Error(t, err)
	})

	t.Run("ApplyOptionValues sets current values", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		reg := registryWithLiveOptions(t)

		err := reg.ApplyOptionValues(e, map[string]string{
			"cursorline": "true",
			"scrolloff":  "7",
		})

		assert.NoError(t, err)
		assert.True(t, e.Options().CursorLine)
		assert.Equal(t, 7, e.Options().ScrollOff)
	})

	t.Run("ApplyOptionValues rejects unknown keys", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		reg := registryWithLiveOptions(t)

		err := reg.ApplyOptionValues(e, map[string]string{
			"unknown": "true",
		})

		assert.Error(t, err)
	})

	t.Run("ApplyOptionValues returns set errors", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		reg := registryWithLiveOptions(t)

		err := reg.ApplyOptionValues(e, map[string]string{
			"cursorline": "maybe",
		})

		assert.Error(t, err)
	})

	t.Run("LookupOption is case-insensitive", func(t *testing.T) {
		reg := registryWithOptions(t)
		_, ok := reg.LookupOption(" ScrollOff ")
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
		results := completer(nil, "s")
		assert.NotEmpty(t, results)
	})

	t.Run("BoolOptionCompleter filters by prefix", func(t *testing.T) {
		reg := registryWithOptions(t)
		completer := reg.BoolOptionCompleter()
		results := completer(nil, "c")
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
			{Key: "scrolloff"},
			{
				Key: "cursorline",
				Toggle: func(*view.Editor) (string, error) {
					return "", nil
				},
			},
		},
	})
	assert.NoError(t, err)
	return reg
}

func registryWithLiveOptions(t *testing.T) *command.Registry {
	t.Helper()
	reg := command.NewRegistry(command.NewKeymaps())
	err := reg.RegisterModule(command.Module{
		Options: []command.Option{
			{
				Key: "scrolloff",
				Get: func(e *view.Editor) (string, error) {
					return strconv.Itoa(e.Options().ScrollOff), nil
				},
				Set: func(e *view.Editor, s string) error {
					n, err := strconv.Atoi(s)
					if err != nil {
						return err
					}
					e.Options().ScrollOff = n
					return nil
				},
			},
			{
				Key: "cursorline",
				Get: func(e *view.Editor) (string, error) {
					return strconv.FormatBool(e.Options().CursorLine), nil
				},
				Set: func(e *view.Editor, s string) error {
					v, err := strconv.ParseBool(s)
					if err != nil {
						return err
					}
					e.Options().CursorLine = v
					return nil
				},
				Toggle: func(e *view.Editor) (string, error) {
					e.Options().CursorLine = !e.Options().CursorLine
					return strconv.FormatBool(e.Options().CursorLine), nil
				},
			},
		},
	})
	assert.NoError(t, err)
	return reg
}

func registryWithFailingOption(t *testing.T) *command.Registry {
	t.Helper()
	reg := command.NewRegistry(command.NewKeymaps())
	err := reg.RegisterModule(command.Module{
		Options: []command.Option{
			{
				Key: "bad",
				Get: func(*view.Editor) (string, error) {
					return "", assert.AnError
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
