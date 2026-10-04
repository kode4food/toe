package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// writeLockfile advertises the SSE endpoint to Claude Code, which discovers it
// by scanning ~/.claude/ide/<port>.lock
func writeLockfile(port int, cwd, token string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".claude", "ide")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, fmt.Sprintf("%d.lock", port))
	data, err := json.Marshal(map[string]any{
		"pid":              os.Getpid(),
		"workspaceFolders": []string{cwd},
		"ideName":          "toe",
		"transport":        "sse",
		"authToken":        token,
		"runningInWindows": false,
	})
	if err != nil {
		return "", err
	}
	return path, os.WriteFile(path, data, 0o600)
}

func removeLockfile(path string) {
	if path != "" {
		_ = os.Remove(path)
	}
}
