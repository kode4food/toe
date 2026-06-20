package config

import (
	"bytes"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"

	"github.com/kode4food/toe/internal/loader"
)

type (
	Config struct {
		Theme  Theme  `toml:"theme"`
		Editor Editor `toml:"editor"`
	}

	Theme struct {
		Name     string
		Light    string
		Dark     string
		Fallback string
		Adaptive bool
	}

	Editor struct {
		Insecure *bool `toml:"insecure"`
	}

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

func DefaultConfig() *Config {
	return &Config{Theme: Theme{Name: "mocha"}}
}

func LoadUserConfig() (*Config, bool) {
	path, ok := UserConfigPath()
	if !ok {
		return DefaultConfig(), false
	}
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	cfg, ok := LoadConfigForWorkspace(path, WorkspaceConfigPath(cwd), cwd)
	if !ok {
		return DefaultConfig(), false
	}
	return cfg, true
}

func LoadConfig(path string) (*Config, bool) {
	merged, ok := loader.LoadMergedTOML([]string{path}, 3)
	if !ok {
		return nil, false
	}
	return decodeConfigMap(merged)
}

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
	return loadRawConfigForWorkspace(path, WorkspaceConfigPath(cwd), cwd)
}

func loadRawConfigForWorkspace(
	global, workspace, dir string,
) (map[string]any, bool) {
	paths := []string{global}
	globalCfg, _ := LoadConfig(global)
	insecure := false
	if globalCfg != nil {
		insecure = globalCfg.Insecure()
	}
	if loader.QueryWorkspaceTrust(dir, insecure) == loader.TrustTrusted {
		paths = append(paths, workspace)
	}
	return loader.LoadMergedTOML(paths, 3)
}

func LoadConfigForWorkspace(global, workspace, dir string) (*Config, bool) {
	paths := []string{global}
	globalCfg, _ := LoadConfig(global)
	insecure := false
	if globalCfg != nil {
		insecure = globalCfg.Insecure()
	}
	if loader.QueryWorkspaceTrust(dir, insecure) ==
		loader.TrustTrusted {
		paths = append(paths, workspace)
	}
	merged, ok := loader.LoadMergedTOML(paths, 3)
	if !ok {
		return nil, false
	}
	return decodeConfigMap(merged)
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

func WorkspaceTrustPath() (string, bool) {
	return loader.WorkspaceTrustFile()
}

func TrustWorkspace(dir string) error {
	return loader.TrustWorkspace(dir)
}

func UntrustWorkspace(dir string) error {
	return loader.UntrustWorkspace(dir)
}

func (c *Config) Insecure() bool {
	return boolValue(nil, c.Editor.Insecure, false)
}

func (a *AutoSave) UnmarshalTOML(value any) error {
	cfg, ok := decodeAutoSave(value)
	if !ok {
		return fmt.Errorf("%w: %v", ErrInvalidOption, value)
	}
	*a = cfg
	return nil
}

func (t Theme) Choose(light bool) string {
	if !t.Adaptive {
		return t.Name
	}
	if light {
		return t.Light
	}
	if t.Dark != "" {
		return t.Dark
	}
	return t.Fallback
}

func boolValue(_, editor *bool, fallback bool) bool {
	if editor != nil {
		return *editor
	}
	return fallback
}

func decodeConfigMap(m map[string]any) (*Config, bool) {
	var b bytes.Buffer
	if err := toml.NewEncoder(&b).Encode(m); err != nil {
		return nil, false
	}
	var raw struct {
		Editor Editor `toml:"editor"`
	}
	if _, err := toml.Decode(b.String(), &raw); err != nil {
		return nil, false
	}
	return &Config{
		Theme:  decodeTheme(m["theme"]),
		Editor: raw.Editor,
	}, true
}

func decodeTheme(value any) Theme {
	switch v := value.(type) {
	case string:
		return Theme{Name: v}
	case map[string]any:
		return Theme{
			Light:    stringValueFromMap(v, "light"),
			Dark:     stringValueFromMap(v, "dark"),
			Fallback: stringValueFromMap(v, "fallback"),
			Adaptive: true,
		}
	default:
		return Theme{Name: "mocha"}
	}
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

func stringValueFromMap(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}
