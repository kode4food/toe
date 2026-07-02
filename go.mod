module github.com/kode4food/toe

go 1.26

toolchain go1.26.4

require (
	charm.land/bubbletea/v2 v2.0.7
	charm.land/lipgloss/v2 v2.0.4
	github.com/BurntSushi/toml v1.6.0
	github.com/alecthomas/chroma/v2 v2.26.1
	github.com/charmbracelet/x/ansi v0.11.7
	github.com/fsnotify/fsnotify v1.10.1
	github.com/go-json-experiment/json v0.0.0-20260601182631-00ed12fed2a6
	github.com/mattn/go-runewidth v0.0.24
	github.com/rivo/uniseg v0.4.7
	github.com/sabhiram/go-gitignore v0.0.0-20210923224102-525f6e181f06
	github.com/smacker/go-tree-sitter v0.0.0-20240827094217-dd81d9e9be82
	github.com/stretchr/testify v1.11.1
	go.lsp.dev/jsonrpc2 v1.0.0
	go.lsp.dev/protocol v1.0.0
	go.lsp.dev/uri v1.0.0
)

require (
	github.com/charmbracelet/colorprofile v0.4.3 // indirect
	github.com/charmbracelet/ultraviolet v0.0.0-20260608091853-35bcb7319efa // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/charmbracelet/x/termios v0.1.1 // indirect
	github.com/charmbracelet/x/windows v0.2.2 // indirect
	github.com/clipperhouse/displaywidth v0.11.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dlclark/regexp2/v2 v2.2.1 // indirect
	github.com/lucasb-eyer/go-colorful v1.4.0 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	golang.org/x/exp v0.0.0-20260611194520-c48552f49976 // indirect
	golang.org/x/exp/typeparams v0.0.0-20260611194520-c48552f49976 // indirect
	golang.org/x/mod v0.37.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/telemetry v0.0.0-20260611141451-d61e87d5f4a3 // indirect
	golang.org/x/tools v0.46.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	honnef.co/go/tools v0.7.0 // indirect
)

tool (
	golang.org/x/tools/cmd/goimports
	honnef.co/go/tools/cmd/staticcheck
)
