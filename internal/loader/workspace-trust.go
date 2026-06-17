package loader

import (
	"os"
	"strings"
)

type (
	TrustStatus int

	TrustUntrustStatus int
)

const (
	TrustUntrusted TrustStatus = iota
	TrustTrusted
)

const (
	TrustDenyAlways TrustUntrustStatus = iota
	TrustDenyOnce
	TrustAllowAlways
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

func QueryWorkspaceTrustWithExplicitUntrust(
	dir string, insecure bool,
) TrustUntrustStatus {
	if insecure {
		return TrustAllowAlways
	}
	root, _ := FindWorkspace(dir)
	if workspaceFileContains(WorkspaceTrustFile, root) {
		return TrustAllowAlways
	}
	if workspaceFileContains(WorkspaceExcludeFile, root) {
		return TrustDenyAlways
	}
	return TrustDenyOnce
}

func ExcludeWorkspace(dir string) error {
	if err := UntrustWorkspace(dir); err != nil {
		return err
	}
	path, ok := WorkspaceExcludeFile()
	if !ok {
		return ErrPathUnavailable
	}
	root, _ := FindWorkspace(dir)
	return updateWorkspaceSet(path, root, true)
}

func workspaceFileContains(
	pathFn func() (string, bool), workspace string,
) bool {
	path, ok := pathFn()
	if !ok {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		if line == workspace {
			return true
		}
	}
	return false
}
