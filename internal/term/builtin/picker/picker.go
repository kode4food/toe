package picker

import (
	"embed"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/builtin/kit"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/config"
)

type section struct {
	Editor struct {
		Picker ui.PickerLayoutOptions `toml:"picker"`
	} `toml:"editor"`
}

const (
	actCommandPalette = "command_palette"
	actLastPicker     = "last_picker"
	splitPrefix       = "picker.split."
	scalesPrefix      = "picker.scales."
)

//go:embed i18n/picker.*.json
var pickerFS embed.FS

// Module returns the generic, concern-independent pickers: the command
// palette and reopen-last-picker
func Module(model ui.Model) command.Module {
	cfg := new(section)

	return command.Module{
		Translations: i18n.LoadTranslations(pickerFS),
		Commands: []command.Command{
			{
				Name:      actCommandPalette,
				DocString: "Open command palette",
				Run:       kit.Runner(model.CommandPaletteAction),
				Modes:     command.PaneModes,
				Keys:      kit.Leader('?'),
			},
			{
				Name:      actLastPicker,
				DocString: "Reopen the last picker",
				Run:       kit.Runner(model.LastPickerAction),
				Modes:     command.PaneModes,
				Keys:      kit.Leader('\''),
			},
		},
		Section: &command.Section{
			Config: cfg,
			Reset:  func() { *cfg = section{} },
			Apply: func(*view.Editor) {
				model.SetPickerLayoutOptions(cfg.Editor.Picker)
			},
		},
		Options: []command.Option{
			splitRatiosOption(model),
			scalesOption(model),
		},
	}
}

func splitRatiosOption(model ui.Model) command.Option {
	return floatMapOption(splitPrefix,
		func() map[string]float64 {
			return model.PickerLayoutOptions().SplitRatios
		},
		func(name string, value float64) {
			opts := model.PickerLayoutOptions()
			if opts.SplitRatios == nil {
				opts.SplitRatios = map[string]float64{}
			}
			opts.SplitRatios[name] = value
			model.SetPickerLayoutOptions(opts)
		},
	)
}

func scalesOption(model ui.Model) command.Option {
	return floatMapOption(scalesPrefix,
		func() map[string]float64 {
			return model.PickerLayoutOptions().Scales
		},
		func(name string, value float64) {
			opts := model.PickerLayoutOptions()
			if opts.Scales == nil {
				opts.Scales = map[string]float64{}
			}
			opts.Scales[name] = value
			model.SetPickerLayoutOptions(opts)
		},
	)
}

// a keyed map of floats behind a private option prefix, so each entry saves
// and restores with the session
func floatMapOption(
	prefix string, get func() map[string]float64, set func(string, float64),
) command.Option {
	return command.Option{
		Key:     prefix,
		Private: true,
		KeyGet: func(*view.Editor) (map[string]string, error) {
			values := get()
			out := make(map[string]string, len(values))
			for key, value := range values {
				out[prefix+key] = strconv.FormatFloat(value, 'f', -1, 64)
			}
			return out, nil
		},
		KeySet: func(_ *view.Editor, key, s string) error {
			name := strings.TrimSpace(key)
			if len(name) <= len(prefix) {
				return fmt.Errorf("%w: %s", config.ErrInvalidOption, key)
			}
			name = name[len(prefix):]
			value, err := strconv.ParseFloat(s, 64)
			if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
				return fmt.Errorf("%w: %s", config.ErrInvalidOption, s)
			}
			set(name, value)
			return nil
		},
	}
}
