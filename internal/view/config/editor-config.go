package config

import (
	"maps"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kode4food/toe/internal/core"
)

type (
	EditorConfig struct {
		IndentStyle            *core.IndentStyle
		TabWidth               *int
		LineEnding             *core.LineEnding
		TrimTrailingWhitespace *bool
		InsertFinalNewline     *bool
		MaxLineLength          *int
	}

	editorConfigFile struct {
		dir string
		cfg editorConfigINI
	}

	editorConfigINI struct {
		root     bool
		sections []editorConfigSection
	}

	editorConfigSection struct {
		pattern string
		pairs   map[string]string
	}
)

func FindEditorConfig(file string) *EditorConfig {
	var configs []editorConfigFile
	for dir := filepath.Dir(file); ; dir = filepath.Dir(dir) {
		cfg, ok := loadEditorConfigFile(filepath.Join(dir, ".editorconfig"))
		if ok {
			configs = append(configs, editorConfigFile{dir: dir, cfg: cfg})
			if cfg.root {
				break
			}
		}
		next := filepath.Dir(dir)
		if next == dir {
			break
		}
	}
	pairs := map[string]string{}
	for i := len(configs) - 1; i >= 0; i-- {
		cfg := configs[i]
		rel, err := filepath.Rel(cfg.dir, file)
		if err != nil {
			continue
		}
		rel = filepath.ToSlash(rel)
		for _, section := range cfg.cfg.sections {
			if editorConfigSectionMatches(section.pattern, rel) {
				maps.Copy(pairs, section.pairs)
			}
		}
	}
	return editorConfigFromPairs(pairs)
}

func loadEditorConfigFile(file string) (editorConfigINI, bool) {
	data, err := os.ReadFile(file)
	if err != nil {
		return editorConfigINI{}, false
	}
	return parseEditorConfig(string(data)), true
}

func parseEditorConfig(data string) editorConfigINI {
	var ini editorConfigINI
	var current *editorConfigSection
	for raw := range strings.SplitSeq(data, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") ||
			strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			ini.sections = append(ini.sections, editorConfigSection{
				pattern: strings.TrimSpace(line[1 : len(line)-1]),
				pairs:   map[string]string{},
			})
			current = &ini.sections[len(ini.sections)-1]
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.ToLower(strings.TrimSpace(value))
		if current == nil {
			if key == "root" && value == "true" {
				ini.root = true
			}
			continue
		}
		current.pairs[key] = value
	}
	return ini
}

func editorConfigSectionMatches(pattern, rel string) bool {
	// Per spec: **.ext → **/*.ext to match .ext recursively
	pattern = strings.ReplaceAll(pattern, "**.", "**/*.")
	// Non-relative patterns (no '/' outside brackets) match at any depth
	if !isEditorConfigGlobRelative(pattern) {
		pattern = "**/" + pattern
	}
	return globMatch(pattern, rel)
}

// isEditorConfigGlobRelative reports whether a glob pattern contains '/'
// outside of bracket expressions, making it relative to the config file
func isEditorConfigGlobRelative(pattern string) bool {
	inBracket := false
	for _, ch := range pattern {
		switch ch {
		case '[':
			inBracket = true
		case ']':
			inBracket = false
		case '/':
			if !inBracket {
				return true
			}
		}
	}
	return false
}

func editorConfigFromPairs(pairs map[string]string) *EditorConfig {
	var cfg EditorConfig
	indentSize := pairs["indent_size"]
	tabWidth := parsePositiveEditorConfigInt(pairs["tab_width"])
	if tabWidth == nil && indentSize != "" && indentSize != "tab" {
		tabWidth = parsePositiveEditorConfigInt(indentSize)
	}
	cfg.TabWidth = tabWidth
	cfg.IndentStyle = editorConfigIndentStyle(pairs, tabWidth)
	cfg.LineEnding = editorConfigLineEnding(pairs["end_of_line"])
	cfg.TrimTrailingWhitespace = editorConfigBool(
		pairs["trim_trailing_whitespace"],
	)
	cfg.InsertFinalNewline = editorConfigBool(pairs["insert_final_newline"])
	cfg.MaxLineLength = parsePositiveEditorConfigInt(pairs["max_line_length"])
	return &cfg
}

func editorConfigIndentStyle(
	pairs map[string]string, tabWidth *int,
) *core.IndentStyle {
	switch pairs["indent_style"] {
	case "tab":
		return new(core.Tabs())
	case "space":
		width := 4
		if pairs["indent_size"] == "tab" && tabWidth != nil {
			width = *tabWidth
		} else if n := parsePositiveEditorConfigInt(pairs["indent_size"]); n != nil {
			width = *n
		}
		return new(core.Spaces(uint8(width)))
	default:
		return nil
	}
}

func editorConfigLineEnding(value string) *core.LineEnding {
	var le core.LineEnding
	switch value {
	case "lf":
		le = core.LineEndingLF
	case "crlf":
		le = core.LineEndingCRLF
	default:
		return nil
	}
	return &le
}

func editorConfigBool(value string) *bool {
	var b bool
	switch value {
	case "true":
		b = true
	case "false":
		b = false
	default:
		return nil
	}
	return &b
}

func parsePositiveEditorConfigInt(value string) *int {
	n, err := strconv.Atoi(value)
	if err != nil || n < 1 {
		return nil
	}
	return &n
}
