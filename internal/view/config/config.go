package config

import (
	"fmt"

	"github.com/kode4food/toe/internal/loader"
)

type (
	AutoSave struct {
		FocusLost  *bool              `toml:"focus-lost"`
		AfterDelay AutoSaveAfterDelay `toml:"after-delay"`
	}

	AutoSaveAfterDelay struct {
		Enable  *bool `toml:"enable"`
		Timeout *int  `toml:"timeout"`
	}

	Search struct {
		SmartCase  *bool `toml:"smart-case"`
		WrapAround *bool `toml:"wrap-around"`
	}
)

func (a *AutoSave) UnmarshalTOML(value any) error {
	if cfg, ok := decodeAutoSave(value); ok {
		*a = cfg
		return nil
	}
	return fmt.Errorf("%w: %v", ErrInvalidOption, value)
}

// LoadRawConfig returns the raw merged TOML map for the given config file path
func LoadRawConfig(path string) (map[string]any, bool) {
	return loader.LoadMergedTOML([]string{path}, 3)
}

// LoadRawConfigForDir returns user config merged with dir's trusted workspace
// config
func LoadRawConfigForDir(dir string) (map[string]any, bool) {
	if path, ok := loader.ConfigFile(); ok {
		return LoadRawConfigForWorkspace(
			path, loader.WorkspaceConfigFile(dir), dir,
		)
	}
	return nil, false
}

func LoadRawConfigForWorkspace(
	global, workspace, dir string,
) (map[string]any, bool) {
	paths := []string{global}
	insecure := false
	if globalRaw, ok := LoadRawConfig(global); ok {
		insecure = decodeInsecure(globalRaw)
	}
	if loader.QueryWorkspaceTrust(dir, insecure) {
		paths = append(paths, workspace)
	}
	return loader.LoadMergedTOML(paths, 3)
}

func decodeInsecure(m map[string]any) bool {
	if editor, ok := m["editor"].(map[string]any); ok {
		v, _ := editor["insecure"].(bool)
		return v
	}
	return false
}

func decodeAutoSave(value any) (AutoSave, bool) {
	switch v := value.(type) {
	case nil:
		return AutoSave{}, false
	case bool:
		return AutoSave{FocusLost: &v}, true
	case map[string]any:
		return AutoSave{
			FocusLost:  loader.BoolPtr(v["focus-lost"]),
			AfterDelay: decodeAutoSaveAfterDelay(v["after-delay"]),
		}, true
	default:
		return AutoSave{}, false
	}
}

func decodeAutoSaveAfterDelay(value any) AutoSaveAfterDelay {
	if m, ok := value.(map[string]any); ok {
		return AutoSaveAfterDelay{
			Enable:  loader.BoolPtr(m["enable"]),
			Timeout: loader.IntPtr(m["timeout"]),
		}
	}
	return AutoSaveAfterDelay{}
}
