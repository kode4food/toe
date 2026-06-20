package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

var (
	ErrUnknownOption = errors.New("unknown option")
	ErrInvalidOption = errors.New("invalid option")
)

func ParseBool(value string) (bool, error) {
	v, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%w: %s", ErrInvalidOption, value)
	}
	return v, nil
}

func ParsePositiveInt(value string) (int, error) {
	v, err := strconv.Atoi(value)
	if err != nil || v < 1 {
		return 0, fmt.Errorf("%w: %s", ErrInvalidOption, value)
	}
	return v, nil
}

func ParseNonNegInt(value string) (int, error) {
	v, err := strconv.Atoi(value)
	if err != nil || v < 0 {
		return 0, fmt.Errorf("%w: %s", ErrInvalidOption, value)
	}
	return v, nil
}

func ParseIntSlice(value string) ([]int, error) {
	var raw struct {
		Value []int `toml:"value"`
	}
	if _, err := toml.Decode("value = "+value, &raw); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidOption, value)
	}
	return raw.Value, nil
}

func ParseStringSlice(value string) ([]string, error) {
	var raw struct {
		Value []string `toml:"value"`
	}
	if _, err := toml.Decode("value = "+value, &raw); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidOption, value)
	}
	return raw.Value, nil
}

func ParseStringLiteral(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	switch value[0] {
	case '"', '\'':
		var raw struct {
			Value string `toml:"value"`
		}
		if _, err := toml.Decode("value = "+value, &raw); err != nil {
			return "", fmt.Errorf("%w: %s", ErrInvalidOption, value)
		}
		return raw.Value, nil
	default:
		return value, nil
	}
}

func FormatIntSlice(values []int) string {
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, strconv.Itoa(v))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func FormatStringSlice(values []string) string {
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, strconv.Quote(v))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
