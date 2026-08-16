package builtin_test

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/i18n"
	"github.com/kode4food/toe/internal/term/builtin"
	"github.com/kode4food/toe/internal/term/command"
	"github.com/kode4food/toe/internal/term/ui"
	"github.com/kode4food/toe/internal/view"
)

const maxOptionDoc = 60

func TestOptionDocStrings(t *testing.T) {
	t.Run("every settable option is described", func(t *testing.T) {
		r := registeredOptions(t)

		var undocumented []string
		for _, o := range r {
			if o.DocString == "" {
				undocumented = append(undocumented, o.Key)
			}
		}
		assert.Empty(t, undocumented)
	})

	// the popup shows the description beside the key, and a translation runs
	// longer than the English it replaces
	t.Run("descriptions stay under the cap", func(t *testing.T) {
		var tooLong []string
		for _, o := range registeredOptions(t) {
			if utf8.RuneCountInString(o.DocString) > maxOptionDoc {
				tooLong = append(tooLong, o.Key)
			}
		}
		assert.Empty(t, tooLong)
	})

	t.Run("private options are not offered", func(t *testing.T) {
		e := view.NewEditor(t.TempDir())
		km := command.NewKeymaps()
		r, err := builtin.Register(ui.New(e, km), km)
		assert.NoError(t, err)

		items := r.OptionCompleter()(e, nil, "")
		for _, item := range items {
			o := r.LookupOption(item.Text)
			if assert.NotNil(t, o) {
				assert.False(t, o.Private)
			}
		}
		assert.NotEmpty(t, items)
	})

	// a module's locale files must win over the English an option is
	// declared with, which they only do when the default is registered first
	t.Run("a locale file overrides the default", func(t *testing.T) {
		km := command.NewKeymaps()
		r := command.NewRegistry(km)
		assert.NoError(t, r.RegisterModule(command.Module{
			Options: []command.Option{{
				Key:       "probe.option",
				DocString: "English default",
				Get: func(*view.Editor) (string, error) {
					return "", nil
				},
				Set: func(*view.Editor, string) error { return nil },
			}},
			Translations: i18n.Translations{
				"option.probe.option.docstring": "translated",
			},
		}))

		o := r.LookupOption("probe.option")
		if assert.NotNil(t, o) {
			assert.Equal(t, "translated", o.DocString)
		}
	})
}

func registeredOptions(t *testing.T) []*command.Option {
	t.Helper()
	e := view.NewEditor(t.TempDir())
	km := command.NewKeymaps()
	r, err := builtin.Register(ui.New(e, km), km)
	assert.NoError(t, err)

	var out []*command.Option
	for _, key := range r.OptionKeys() {
		if o := r.LookupOption(key); o != nil && !o.Private {
			out = append(out, o)
		}
	}
	return out
}
