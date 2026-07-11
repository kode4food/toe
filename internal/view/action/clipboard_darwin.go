//go:build darwin

package action

func detectClipboardProvider() clipboardProvider {
	paste, okPaste := lookPath("pbpaste")
	copyBin, okCopy := lookPath("pbcopy")
	if okPaste && okCopy {
		return clipboardProvider{
			name:      "pasteboard",
			read:      []string{paste},
			write:     []string{copyBin},
			readPrim:  []string{paste},
			writePrim: []string{copyBin},
		}
	}
	return clipboardProvider{name: "none"}
}
