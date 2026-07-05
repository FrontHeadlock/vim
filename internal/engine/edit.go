package engine

// edit.go — 단순 편집 명령: x/X/D/C/s/r/~/p 계열(연산자+모션 조합이 아닌 단발 명령).

func (e *Editor) deleteChars(count int) {
	l := e.line()
	if len(l) == 0 {
		return
	}
	end := e.col + count
	if end > len(l) {
		end = len(l)
	}
	e.reg = append([]rune(nil), l[e.col:end]...)
	e.regLinewise = false
	e.pushUndo()
	e.lines[e.row] = append(append([]rune(nil), l[:e.col]...), l[end:]...)
	e.clamp()
}

func (e *Editor) deleteBefore(count int) {
	if e.col == 0 {
		return
	}
	start := e.col - count
	if start < 0 {
		start = 0
	}
	l := e.line()
	e.pushUndo()
	e.lines[e.row] = append(append([]rune(nil), l[:start]...), l[e.col:]...)
	e.col = start
	e.clamp()
}

func (e *Editor) deleteToEOL(op rune) {
	l := e.line()
	if e.col > len(l) {
		return
	}
	e.reg = append([]rune(nil), l[e.col:]...)
	e.regLinewise = false
	e.pushUndo()
	e.lines[e.row] = append([]rune(nil), l[:e.col]...)
	if op == 'c' {
		e.mode = ModeInsert
	} else {
		e.clamp()
	}
}

func (e *Editor) substituteChar(count int) {
	l := e.line()
	if len(l) == 0 {
		e.mode = ModeInsert
		return
	}
	end := e.col + count
	if end > len(l) {
		end = len(l)
	}
	e.pushUndo()
	e.lines[e.row] = append(append([]rune(nil), l[:e.col]...), l[end:]...)
	e.mode = ModeInsert
}

func (e *Editor) replaceChar(ch rune, count int) {
	l := e.line()
	if e.col+count > len(l) {
		return
	}
	e.pushUndo()
	for i := 0; i < count; i++ {
		l[e.col+i] = ch
	}
	e.col += count - 1
	e.clamp()
}

func (e *Editor) toggleCase(count int) {
	l := e.line()
	e.pushUndo()
	for i := 0; i < count && e.col < len(l); i++ {
		r := l[e.col]
		switch {
		case r >= 'a' && r <= 'z':
			l[e.col] = r - 32
		case r >= 'A' && r <= 'Z':
			l[e.col] = r + 32
		}
		if e.col < len(l)-1 {
			e.col++
		}
	}
	e.clamp()
}

func (e *Editor) paste(after bool) {
	if e.regLinewise {
		e.pushUndo()
		ins := cloneLines(e.regLines)
		at := e.row
		if after {
			at = e.row + 1
		}
		tail := append([][]rune{}, e.lines[at:]...)
		e.lines = append(e.lines[:at], ins...)
		e.lines = append(e.lines, tail...)
		e.row = at
		e.col = firstNonBlank(e.line())
		return
	}
	if len(e.reg) == 0 {
		return
	}
	e.pushUndo()
	l := e.line()
	at := e.col
	if after && len(l) > 0 {
		at = e.col + 1
	}
	if at > len(l) {
		at = len(l)
	}
	nl := append([]rune(nil), l[:at]...)
	nl = append(nl, e.reg...)
	nl = append(nl, l[at:]...)
	e.lines[e.row] = nl
	e.col = at + len(e.reg) - 1
	e.clamp()
}
