package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/defaults"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/config"
)

func TestPickerConfig(t *testing.T) {
	t.Run("diagnostic pickers registered", func(t *testing.T) {
		e, km, _ := envWithRegistry(t, "")

		res := runCmd(t, km, e, "diagnostic_picker")
		assert.Empty(t, res.Message)

		res = runCmd(t, km, e, "workspace_diagnostics_picker")
		assert.Empty(t, res.Message)
	})

	t.Run("buffer picker config decodes", func(t *testing.T) {
		e, _, reg := envWithRegistry(t, "")
		err := reg.ApplyTOML(e, map[string]any{
			"editor": map[string]any{
				"buffer-picker": map[string]any{
					"start-position": "previous",
				},
			},
		})

		assert.NoError(t, err)
	})

	t.Run("buffer picker rejects invalid start", func(t *testing.T) {
		e, _, reg := envWithRegistry(t, "")
		err := reg.ApplyTOML(e, map[string]any{
			"editor": map[string]any{
				"buffer-picker": map[string]any{
					"start-position": "middle",
				},
			},
		})

		assert.Error(t, err)
	})

	t.Run("file explorer config decodes", func(t *testing.T) {
		e, km, reg := envWithRegistry(t, "")
		err := reg.ApplyTOML(e, map[string]any{
			"editor": map[string]any{
				"file-explorer": map[string]any{
					"hidden":          true,
					"follow-symlinks": true,
					"parents":         true,
					"ignore":          true,
					"git-ignore":      true,
					"git-global":      true,
					"git-exclude":     true,
					"flatten-dirs":    false,
				},
			},
		})

		assert.NoError(t, err)
		res := runCmd(t, km, e, "file_explorer")
		assert.Empty(t, res.Message)
	})

	t.Run("empty config decodes", func(t *testing.T) {
		e, _, reg := envWithRegistry(t, "")
		err := reg.ApplyTOML(e, map[string]any{})

		assert.NoError(t, err)
	})

	t.Run("picker split ratios config", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		reg, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)

		err = reg.ApplyTOML(e, map[string]any{
			"editor": map[string]any{
				"picker": map[string]any{
					"split-ratios": map[string]any{
						"Diagnostics": 0.65,
					},
				},
			},
		})

		assert.NoError(t, err)
		assert.Equal(
			t, 0.65,
			m.PickerLayoutOptions().SplitRatios["Diagnostics"],
		)
	})

	t.Run("picker split ratios option", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		reg, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)

		err = reg.ApplyOptionValues(e, map[string]string{
			"editor.picker.split-ratios.Diagnostics": "0.65",
		})
		values, valueErr := reg.OptionValues(e)

		assert.NoError(t, err)
		assert.NoError(t, valueErr)
		assert.Equal(
			t, 0.65,
			m.PickerLayoutOptions().SplitRatios["Diagnostics"],
		)
		assert.Equal(
			t, "0.65",
			values["editor.picker.split-ratios.Diagnostics"],
		)
	})

	t.Run("picker split ratios clamp out of range", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		reg, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)

		err = reg.ApplyOptionValues(e, map[string]string{
			"editor.picker.split-ratios.Diagnostics": "0.95",
		})

		assert.NoError(t, err)
		assert.Equal(t,
			ui.MaxPickerSplitRatio,
			m.PickerLayoutOptions().SplitRatios["Diagnostics"],
		)
	})

	t.Run("picker split ratios rejects garbage", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		m := ui.New(e, km)
		reg, err := defaults.RegisterDefaults(m, km)
		assert.NoError(t, err)

		err = reg.ApplyOptionValues(e, map[string]string{
			"editor.picker.split-ratios.Diagnostics": "nope",
		})

		assert.ErrorIs(t, err, config.ErrInvalidOption)
	})
}
