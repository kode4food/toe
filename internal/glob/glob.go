package glob

import (
	"path/filepath"
	"strings"
)

// Match reports whether path matches the glob pattern, expanding brace
// alternatives and matching with both native and slash separators
func Match(pattern, path string) bool {
	for _, p := range expandBraces(pattern) {
		if matchPath(p, filepath.ToSlash(path)) {
			return true
		}
		if matchPath(p, path) {
			return true
		}
	}
	return false
}

func expandBraces(pattern string) []string {
	start := strings.Index(pattern, "{")
	if start < 0 {
		return []string{pattern}
	}
	end := strings.Index(pattern[start:], "}")
	if end < 0 {
		return []string{pattern}
	}
	end += start
	pfx := pattern[:start]
	sfx := pattern[end+1:]
	alts := strings.Split(pattern[start+1:end], ",")
	out := make([]string, 0, len(alts))
	for _, alt := range alts {
		out = append(out, expandBraces(pfx+alt+sfx)...)
	}
	return out
}

func matchPath(pattern, path string) bool {
	parts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")
	if strings.HasPrefix(pattern, "*/") {
		for i := range len(pathParts) {
			if matchParts(parts, pathParts[i:]) {
				return true
			}
		}
	}
	return matchParts(parts, pathParts)
}

func matchParts(pattern, path []string) bool {
	for len(pattern) > 0 {
		part := pattern[0]
		pattern = pattern[1:]
		if part == "**" {
			if len(pattern) == 0 {
				return true
			}
			for i := range len(path) + 1 {
				if matchParts(pattern, path[i:]) {
					return true
				}
			}
			return false
		}
		if len(path) == 0 {
			return false
		}
		if ok, err := filepath.Match(part, path[0]); err != nil || !ok {
			return false
		}
		path = path[1:]
	}
	return len(path) == 0
}
