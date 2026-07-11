package ui

// cursor column paints first so rulers render over it
func (r *renderPass) paintContentOverlays(st *contentRenderState) {
	args := st.args
	buf := args.buf
	contentX := st.contentX
	format := st.format

	if st.cursorColumnEnabled && st.cursorLine < len(st.lineIdx)-1 {
		entry := st.lineIdx[st.cursorLine]
		next := st.lineIdx[st.cursorLine+1]
		end := next.byteStart - entry.endingLen
		cursorLStr := st.rawText[entry.byteStart:end]
		col := st.cursor - entry.charStart
		vcol := visualColOf(cursorLStr, col, format.TabWidth)
		rel := vcol - st.hOff
		if rel >= 0 && rel < format.ViewportWidth {
			sx := contentX + rel
			for row := args.y; row < args.y+args.height; row++ {
				buf.PatchBg(sx, row, st.cursorColumnBg)
			}
		}
	}
	if len(st.rulers) > 0 {
		applyRulers(
			buf, contentX, args.y, format.ViewportWidth, args.height, st.hOff,
			st.rulers, st.rulerBg,
		)
	}
}
