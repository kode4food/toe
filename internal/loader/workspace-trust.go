package loader

import (
	"os"
	"strings"
)

func QueryWorkspaceTrust(dir string, insecure bool) bool {
	if insecure {
		return true
	}
	root, _ := FindWorkspace(dir)
	path, ok := WorkspaceTrustFile()
	if !ok {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		if line == root {
			return true
		}
	}
	return false
}
