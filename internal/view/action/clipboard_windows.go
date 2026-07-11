//go:build windows

package action

const ttyDevice = "CONOUT$"

func detectClipboardProvider() clipboardProvider {
	if win32yank, ok := lookPath("win32yank.exe"); ok {
		return clipboardProvider{
			name:     "win32yank",
			readCmd:  []string{win32yank, "-o", "--lf"},
			writeCmd: []string{win32yank, "-i", "--crlf"},
		}
	}
	powershell, _ := lookPath("powershell")
	return clipboardProvider{
		name: "windows",
		readCmd: []string{
			powershell, "-NoProfile", "-Command", "Get-Clipboard -Raw",
		},
		writeCmd: []string{
			powershell, "-NoProfile", "-Command",
			"Set-Clipboard -Value ([Console]::In.ReadToEnd())",
		},
	}
}
