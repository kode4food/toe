package loader

import (
	"os"
	"path/filepath"
)

// QueryKind names a runtime query file under a language query directory
type QueryKind string

// QueryInjections loads language injection captures
const QueryInjections QueryKind = "injections.scm"

// LoadRuntimeFile reads a runtime file from a language query directory
func LoadRuntimeFile(language, filename string) (string, error) {
	path := RuntimeFile(filepath.Join("queries", language, filename))
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// LoadQuery reads a known runtime query file for a language
func LoadQuery(language string, kind QueryKind) (string, error) {
	return LoadRuntimeFile(language, string(kind))
}
