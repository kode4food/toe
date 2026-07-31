package picker

import (
	"fmt"
	"math"
	"strconv"
	"strings"

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
	splitRatiosPrefix = "picker.split-ratios."
)

// Module returns the generic, concern-independent pickers: the command
// palette and reopen-last-picker
func Module(model ui.Model) command.Module {
	cfg := new(section)

	return command.Module{
		Commands: []command.Command{
			{
				Name:      actCommandPalette,
				DocString: "Open command palette",
				Run:       kit.Continuation(model.CommandPaletteAction()),
				Modes:     command.PaneModes(),
				Keys:      kit.Leader('?'),
			},
			{
				Name:      actLastPicker,
				DocString: "Reopen the last picker",
				Run:       kit.Continuation(model.LastPickerAction()),
				Modes:     command.PaneModes(),
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
		},
	}
}

func splitRatiosOption(model ui.Model) command.Option {
	return command.Option{
		Key: splitRatiosPrefix,
		KeyGet: func(*view.Editor) (map[string]string, error) {
			ratios := model.PickerLayoutOptions().SplitRatios
			out := make(map[string]string, len(ratios))
			for key, ratio := range ratios {
				out[splitRatiosPrefix+key] = strconv.FormatFloat(
					ratio, 'f', -1, 64,
				)
			}
			return out, nil
		},
		KeySet: func(_ *view.Editor, key, s string) error {
			name := strings.TrimSpace(key)
			if len(name) <= len(splitRatiosPrefix) {
				return fmt.Errorf("%w: %s", config.ErrInvalidOption, key)
			}
			name = name[len(splitRatiosPrefix):]
			ratio, err := strconv.ParseFloat(s, 64)
			if err != nil || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
				return fmt.Errorf("%w: %s", config.ErrInvalidOption, s)
			}
			opts := model.PickerLayoutOptions()
			if opts.SplitRatios == nil {
				opts.SplitRatios = map[string]float64{}
			}
			opts.SplitRatios[name] = ratio
			model.SetPickerLayoutOptions(opts)
			return nil
		},
	}
}
