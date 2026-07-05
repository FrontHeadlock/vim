package engine

// insert.go — Insert 모드.

func (e *Editor) enterInsert(_ bool) {
	e.pushUndo()
	e.mode = ModeInsert
	e.clamp()
}

func (e *Editor) openLine(below bool) {
	e.pushUndo()
	at := e.row
	if below {
		at = e.row + 1
	}
	e.lines = append(e.lines[:at], append([][]rune{{}}, e.lines[at:]...)...)
	e.row = at
	e.col = 0
	e.mode = ModeInsert
}

func (e *Editor) feedInsert(k Key) {
	if !e.replaying {
		e.curKeys = append(e.curKeys, k)
	}
	switch k.S {
	case "esc":
		e.mode = ModeNormal
		if e.col > 0 {
			e.col--
		}
		e.clamp()
		e.finishInsertDot()
		return
	case "bs":
		if e.col > 0 {
			l := e.line()
			e.lines[e.row] = append(append([]rune(nil), l[:e.col-1]...), l[e.col:]...)
			e.col--
		} else if e.row > 0 {
			prev := e.lines[e.row-1]
			cur := e.line()
			e.col = len(prev)
			e.lines[e.row-1] = append(prev, cur...)
			e.lines = append(e.lines[:e.row], e.lines[e.row+1:]...)
			e.row--
		}
		return
	case "cr":
		l := e.line()
		rest := append([]rune(nil), l[e.col:]...)
		e.lines[e.row] = append([]rune(nil), l[:e.col]...)
		e.lines = append(e.lines[:e.row+1], append([][]rune{rest}, e.lines[e.row+1:]...)...)
		e.row++
		e.col = 0
		return
	}
	if k.R != 0 {
		l := e.line()
		nl := append([]rune(nil), l[:e.col]...)
		nl = append(nl, k.R)
		nl = append(nl, l[e.col:]...)
		e.lines[e.row] = nl
		e.col++
	}
}

// finishInsertDot 은 Insert 진입부터 esc 까지의 키 시퀀스를 dot 으로 저장한다
// — 단, 버퍼가 실제로 바뀌었을 때만(B2: "i<esc>" 같은 무변경 insert 가 dot 을
// "아무 일도 안 하는 반복"으로 오염시키지 않게 한다).
func (e *Editor) finishInsertDot() {
	if e.replaying {
		return
	}
	if e.commitUndoIfChanged() {
		e.dot = append([]Key(nil), e.curKeys...)
	}
	e.changed = false
}
