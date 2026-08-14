package ui

import (
	"path/filepath"
	"strings"

	"github.com/kode4food/toe/internal/term/theme"
	"github.com/kode4food/toe/internal/tui"
	"github.com/kode4food/toe/internal/view"
	"github.com/kode4food/toe/internal/view/language"
)

type (
	pickerFileMarker struct {
		glyph string
		color pickerFileColor
	}

	pickerFileColor uint8
)

const (
	pickerFileAzure pickerFileColor = iota
	pickerFileBlue
	pickerFileCyan
	pickerFileGreen
	pickerFileGrey
	pickerFileOrange
	pickerFilePurple
	pickerFileRed
	pickerFileYellow
)

var (
	pickerDefaultDirectoryIcon = pickerFileMarker{
		glyph: "\U000f024b",
		color: pickerFileAzure,
	}

	pickerDefaultFileIcon = pickerFileMarker{
		glyph: "\U000f0214",
		color: pickerFileGrey,
	}

	pickerFileColorScopes = [...]string{
		"ui.file-icon.azure",
		"ui.file-icon.blue",
		"ui.file-icon.cyan",
		"ui.file-icon.green",
		"ui.file-icon.grey",
		"ui.file-icon.orange",
		"ui.file-icon.purple",
		"ui.file-icon.red",
		"ui.file-icon.yellow",
	}
)

func pickerItemFileIcon(
	e *view.Editor, p *Picker, item *PickerItem,
) pickerFileMarker {
	if !e.Options().NerdFonts || item.Section {
		return pickerFileMarker{}
	}
	if pickerFileIconColumn(p, item) < 0 {
		return pickerFileMarker{}
	}
	if item.Directory {
		return pickerDefaultDirectoryIcon
	}
	path := item.Location.Target.Path
	return p.fileIcon(path, e.Document(item.Location.Target.ID))
}

func (p *Picker) fileIcon(path string, doc *view.Document) pickerFileMarker {
	lang := ""
	if doc != nil {
		if path == "" {
			path = doc.Path()
		}
		lang = doc.Lang()
	}
	if path == "" && doc != nil {
		return pickerDefaultFileIcon
	}
	if path == "" {
		return pickerFileMarker{}
	}
	if icon, ok := p.fileIcons[path]; ok {
		return icon
	}
	name := filepath.Base(path)
	icon, ok := pickerFileIcons[name]
	if !ok {
		if dot := strings.LastIndexByte(name, '.'); dot > 0 {
			ext := strings.ToLower(name[dot+1:])
			icon, ok = pickerExtensionIcons[ext]
		}
	}
	if !ok && lang == "" {
		lang = language.DetectLanguage(language.DetectLanguageArgs{Path: path})
	}
	if !ok {
		icon, ok = pickerFileTypeIcons[strings.ToLower(lang)]
	}
	if !ok {
		icon = pickerDefaultFileIcon
	}
	if p.fileIcons == nil {
		p.fileIcons = map[string]pickerFileMarker{}
	}
	p.fileIcons[path] = icon
	return icon
}

func pickerFileIconColumn(p *Picker, item *PickerItem) int {
	if len(item.Columns) <= 1 {
		return 0
	}
	for i := min(p.source.MatchColumn(), len(item.Columns)) - 1; i >= 0; i-- {
		if item.Columns[i] == "" {
			return i
		}
	}
	return -1
}

func pickerFileIconStyle(
	th *theme.Theme, base tui.Style, color pickerFileColor,
) tui.Style {
	scope := pickerFileColorScopes[color]
	if style, ok := th.TryGet(scope); ok {
		return applyAccentStyle(styleOverlay{base: base, overlay: style})
	}
	return base
}
