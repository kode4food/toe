package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kode4food/toe/internal/core"
	"github.com/kode4food/toe/internal/view"
)

var (
	statusElemFns = map[view.StatusLineElement]func(*statusElemCtx) statusElem{
		view.StatusLineMode:             statusElemMode,
		view.StatusLineSeparator:        statusElemSeparator,
		view.StatusLineFileName:         statusElemFileName,
		view.StatusLineFileBaseName:     statusElemFileBaseName,
		view.StatusLineFileAbsolutePath: statusElemFileAbsPath,
		view.StatusLineReadOnly:         statusElemReadOnly,
		view.StatusLineModified:         statusElemModified,
		view.StatusLineSelections:       statusElemSelections,
		view.StatusLinePrimaryLen:       statusElemPrimaryLen,
		view.StatusLinePosition:         statusElemPosition,
		view.StatusLinePercent:          statusElemPercent,
		view.StatusLineTotalLines:       statusElemTotalLines,
		view.StatusLineFileEncoding:     statusElemEncoding,
		view.StatusLineFileLineEnding:   statusElemLineEnding,
		view.StatusLineSpacer:           statusElemSpacer,
		view.StatusLineFileIndentStyle:  statusElemIndentStyle,
		view.StatusLineFileType:         statusElemFileType,
		view.StatusLineDiagnostics:      statusElemDiagnostics,
		view.StatusLineRegister:         statusElemRegister,
		view.StatusLineVersionControl:   statusElemVersionControl,
		view.StatusLineSpinner:          statusElemSpinner,
	}

	spinFrames = []string{
		"\u280b", // ⠋
		"\u2819", // ⠙
		"\u2839", // ⠹
		"\u2838", // ⠸
		"\u283c", // ⠼
		"\u2834", // ⠴
		"\u2826", // ⠦
		"\u2827", // ⠧
		"\u2807", // ⠇
		"\u280f", // ⠏
	}
)

func statusElemMode(s *statusElemCtx) statusElem {
	return statusElem{
		text:    " " + s.opts.ModeNameForMode(s.mode) + " ",
		style:   s.modeSt,
		compact: true,
	}
}

func statusElemSeparator(s *statusElemCtx) statusElem {
	return statusElem{text: s.sep, style: s.sepSt, compact: true}
}

func statusElemFileName(s *statusElemCtx) statusElem {
	return statusElem{
		text:  s.doc.RelativeName(s.cwd),
		style: s.baseTUI,
	}
}

func statusElemFileBaseName(s *statusElemCtx) statusElem {
	return statusElem{
		text:  filepath.Base(s.doc.Path()),
		style: s.baseTUI,
	}
}

func statusElemFileAbsPath(s *statusElemCtx) statusElem {
	return statusElem{text: s.doc.Path(), style: s.baseTUI}
}

func statusElemReadOnly(s *statusElemCtx) statusElem {
	if !s.doc.ReadOnly() {
		return statusElem{}
	}
	return statusElem{text: "[readonly]", style: s.baseTUI}
}

func statusElemModified(s *statusElemCtx) statusElem {
	if !s.doc.Modified() {
		return statusElem{}
	}
	return statusElem{text: "[modified]", style: s.baseTUI}
}

func statusElemSelections(s *statusElemCtx) statusElem {
	if s.nSel == 1 {
		return statusElem{text: "1 sel", style: s.baseTUI}
	}
	return statusElem{
		text:  fmt.Sprintf("%d/%d sels", s.primIdx+1, s.nSel),
		style: s.baseTUI,
	}
}

func statusElemPrimaryLen(s *statusElemCtx) statusElem {
	return statusElem{
		text:  fmt.Sprintf("%d", s.primLen),
		style: s.baseTUI,
	}
}

func statusElemPosition(s *statusElemCtx) statusElem {
	return statusElem{
		text:  fmt.Sprintf("%d:%d", s.row, s.col),
		style: s.baseTUI,
	}
}

func statusElemPercent(s *statusElemCtx) statusElem {
	pct := 0
	if s.totalLines > 0 {
		pct = (s.row * 100) / s.totalLines
	}
	return statusElem{
		text:  fmt.Sprintf("%d%%", pct),
		style: s.baseTUI,
	}
}

func statusElemTotalLines(s *statusElemCtx) statusElem {
	return statusElem{
		text:  fmt.Sprintf("%d", s.totalLines),
		style: s.baseTUI,
	}
}

func statusElemEncoding(s *statusElemCtx) statusElem {
	label := "utf-8"
	if s.doc.HasBOM() {
		label = "utf-8-bom"
	}
	return statusElem{text: label, style: s.baseTUI}
}

func statusElemSpacer(s *statusElemCtx) statusElem {
	return statusElem{text: " ", style: s.baseTUI, compact: true}
}

func statusElemLineEnding(s *statusElemCtx) statusElem {
	label := "lf"
	if s.doc.LineEnding() == core.LineEndingCRLF {
		label = "crlf"
	}
	return statusElem{text: label, style: s.baseTUI}
}

func statusElemIndentStyle(s *statusElemCtx) statusElem {
	indent := s.doc.IndentStyle()
	var label string
	if indent.IsTabs() {
		label = "tabs"
	} else {
		label = fmt.Sprintf("spaces:%d", indent.Width())
	}
	return statusElem{text: label, style: s.baseTUI}
}

func statusElemFileType(s *statusElemCtx) statusElem {
	lang := s.doc.Lang()
	if lang == "" {
		lang = "text"
	}
	return statusElem{text: lang, style: s.baseTUI}
}

func statusElemDiagnostics(s *statusElemCtx) statusElem {
	counts := s.doc.DiagnosticCounts()
	var parts []string
	if counts.Errors > 0 {
		parts = append(parts, fmt.Sprintf("E:%d", counts.Errors))
	}
	if counts.Warnings > 0 {
		parts = append(parts, fmt.Sprintf("W:%d", counts.Warnings))
	}
	if counts.Info > 0 {
		parts = append(parts, fmt.Sprintf("I:%d", counts.Info))
	}
	if counts.Hints > 0 {
		parts = append(parts, fmt.Sprintf("H:%d", counts.Hints))
	}
	if len(parts) == 0 {
		return statusElem{}
	}
	return statusElem{
		text:  strings.Join(parts, " "),
		style: s.baseTUI,
	}
}

func statusElemVersionControl(s *statusElemCtx) statusElem {
	if s.vcsHead == "" {
		return statusElem{}
	}
	return statusElem{text: s.vcsHead, style: s.baseTUI}
}

func statusElemSpinner(s *statusElemCtx) statusElem {
	if !s.busy {
		return statusElem{}
	}
	frame := spinFrames[s.spinFrame%len(spinFrames)]
	return statusElem{text: frame, style: s.baseTUI, compact: true}
}

func statusElemRegister(s *statusElemCtx) statusElem {
	if s.reg == 0 {
		return statusElem{}
	}
	return statusElem{
		text:  fmt.Sprintf("reg=%c", s.reg),
		style: s.baseTUI,
	}
}
