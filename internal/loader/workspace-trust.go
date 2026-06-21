package loader

import (
	"os"
	"strings"
)

type TrustStatus int

const (
	TrustUntrusted TrustStatus = iota
	TrustTrusted
)

func QueryWorkspaceTrust(dir string, insecure bool) TrustStatus {
	if insecure {
		return TrustTrusted
	}
	root, _ := FindWorkspace(dir)
	path, ok := WorkspaceTrustFile()
	if !ok {
		return TrustUntrusted
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return TrustUntrusted
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		if line == root {
			return TrustTrusted
		}
	}
	return TrustUntrusted
}
