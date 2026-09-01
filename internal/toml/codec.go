package toml

import (
	"strings"

	burntsushi "github.com/BurntSushi/toml"
)

// Decode decodes TOML text into target, discarding the decoder metadata
func Decode(text string, target any) error {
	_, err := burntsushi.Decode(text, target)
	return err
}

// DecodeFile decodes the TOML file at path into target
func DecodeFile(path string, target any) error {
	_, err := burntsushi.DecodeFile(path, target)
	return err
}

// Encode renders value as TOML text
func Encode(value any) (string, error) {
	var out strings.Builder
	if err := burntsushi.NewEncoder(&out).Encode(value); err != nil {
		return "", err
	}
	return out.String(), nil
}
