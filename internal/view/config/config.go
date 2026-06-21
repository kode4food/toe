package config

import (
	"fmt"
	"os"

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

// LoadRawConfig returns the raw merged TOML map for the given config file path
func LoadRawConfig(path string) (map[string]any, bool) {
	return loader.LoadMergedTOML([]string{path}, 3)
}

// LoadRawUserConfig returns the raw merged TOML map for the user and trusted
// workspace config, for applying to module section structs via
// Registry.ApplyTOML
func LoadRawUserConfig() (map[string]any, bool) {
	path, ok := UserConfigPath()
	if !ok {
		return nil, false
	}
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	return LoadRawConfigForWorkspace(path, WorkspaceConfigPath(cwd), cwd)
}

func LoadRawConfigForWorkspace(
	global, workspace, dir string,
) (map[string]any, bool) {
	paths := []string{global}
	insecure := false
	if globalRaw, ok := LoadRawConfig(global); ok {
		insecure = decodeInsecure(globalRaw)
	}
	if loader.QueryWorkspaceTrust(dir, insecure) == loader.TrustTrusted {
		paths = append(paths, workspace)
	}
	return loader.LoadMergedTOML(paths, 3)
}

func UserConfigPath() (string, bool) {
	return loader.ConfigFile()
}

func WorkspaceConfigPath(dir string) string {
	return loader.WorkspaceConfigFile(dir)
}

func IgnorePath() string {
	return loader.ConfigIgnoreFile()
}

func LogFilePath() (string, bool) {
	return loader.LogFile()
}

func TrustWorkspace(dir string) error {
	return loader.TrustWorkspace(dir)
}

func UntrustWorkspace(dir string) error {
	return loader.UntrustWorkspace(dir)
}

func (a *AutoSave) UnmarshalTOML(value any) error {
	cfg, ok := decodeAutoSave(value)
	if !ok {
		return fmt.Errorf("%w: %v", ErrInvalidOption, value)
	}
	*a = cfg
	return nil
}

func decodeInsecure(m map[string]any) bool {
	editor, ok := m["editor"].(map[string]any)
	if !ok {
		return false
	}
	v, _ := editor["insecure"].(bool)
	return v
}

func decodeAutoSave(value any) (AutoSave, bool) {
	switch v := value.(type) {
	case nil:
		return AutoSave{}, false
	case bool:
		return AutoSave{FocusLost: &v}, true
	case map[string]any:
		return AutoSave{
			FocusLost:  boolPtr(v["focus-lost"]),
			AfterDelay: decodeAutoSaveAfterDelay(v["after-delay"]),
		}, true
	default:
		return AutoSave{}, false
	}
}

func decodeAutoSaveAfterDelay(value any) AutoSaveAfterDelay {
	m, ok := value.(map[string]any)
	if !ok {
		return AutoSaveAfterDelay{}
	}
	return AutoSaveAfterDelay{
		Enable:  boolPtr(m["enable"]),
		Timeout: intPtrOrNil(m["timeout"]),
	}
}

func boolPtr(value any) *bool {
	v, ok := value.(bool)
	if !ok {
		return nil
	}
	return &v
}

func intPtr(value any) (*int, bool) {
	switch v := value.(type) {
	case int:
		return &v, true
	case int64:
		return new(int(v)), true
	default:
		return nil, false
	}
}

func intPtrOrNil(value any) *int {
	v, _ := intPtr(value)
	return v
}
