//go:build windows

package action

const ttyDevice = "CONOUT$"

func detectClipboardProvider() clipboardProvider {
	if win32yank, ok := lookPath("win32yank.exe"); ok {
		return clipboardProvider{
			name:  "win32yank",
			read:  []string{win32yank, "-o", "--lf"},
			write: []string{win32yank, "-i", "--crlf"},
		}
	}
	powershell, _ := lookPath("powershell")
	return clipboardProvider{
		name: "windows",
		read: []string{
			powershell, "-NoProfile", "-Command", "Get-Clipboard -Raw",
		},
		write: []string{
			powershell, "-NoProfile", "-Command",
			"Set-Clipboard -Value ([Console]::In.ReadToEnd())",
		},
	}
}
