package engine

// motion.go — 커서 이동(모션). 연산자와 결합하는 스팬 계산은 operator.go 를 참고.

func (e *Editor) doMotion(cmd rune, count int) {
	for i := 0; i < count; i++ {
		e.motionOnce(cmd)
	}
	if cmd != 'j' && cmd != 'k' {
		e.dcol = e.col
	}
	e.clamp()
}

func (e *Editor) motionOnce(cmd rune) {
	switch cmd {
	case 'h':
		if e.col > 0 {
			e.col--
		}
	case 'l':
		if e.col < e.lastCol(false) {
			e.col++
		}
	case 'j':
		if e.row < len(e.lines)-1 {
			e.row++
			e.col = e.dcol
			e.clamp()
		}
	case 'k':
		if e.row > 0 {
			e.row--
			e.col = e.dcol
			e.clamp()
		}
	case '0':
		e.col = 0
	case '^':
		e.col = firstNonBlank(e.line())
	case '$':
		e.col = e.lastCol(false)
	case 'w':
		e.row, e.col = e.nextWordStart(e.row, e.col, false)
	case 'W':
		e.row, e.col = e.nextWordStart(e.row, e.col, true)
	case 'b':
		e.row, e.col = e.prevWordStart(e.row, e.col, false)
	case 'B':
		e.row, e.col = e.prevWordStart(e.row, e.col, true)
	case 'e':
		e.row, e.col = e.nextWordEnd(e.row, e.col, false)
	case 'E':
		e.row, e.col = e.nextWordEnd(e.row, e.col, true)
	}
}

func (e *Editor) nextWordStart(r, c int, big bool) (int, int) {
	l := e.lines[r]
	if c < len(l) {
		start := classOf(l[c], big)
		for c < len(l) && classOf(l[c], big) == start && start != 0 {
			c++
		}
	}
	for {
		for c < len(l) && classOf(l[c], big) == 0 {
			c++
		}
		if c < len(l) {
			return r, c
		}
		if r+1 >= len(e.lines) {
			if len(l) > 0 {
				return r, len(l) - 1
			}
			return r, 0
		}
		r++
		l = e.lines[r]
		c = 0
		if len(l) > 0 && classOf(l[0], big) != 0 {
			return r, 0
		}
	}
}

func (e *Editor) prevWordStart(r, c int, big bool) (int, int) {
	c--
	for {
		if c < 0 {
			if r == 0 {
				return 0, 0
			}
			r--
			c = len(e.lines[r]) - 1
			if c < 0 {
				continue
			}
		}
		l := e.lines[r]
		if c >= 0 && classOf(l[c], big) != 0 {
			k := classOf(l[c], big)
			for c > 0 && classOf(l[c-1], big) == k {
				c--
			}
			return r, c
		}
		c--
	}
}

func (e *Editor) nextWordEnd(r, c int, big bool) (int, int) {
	c++
	for {
		l := e.lines[r]
		for c < len(l) && classOf(l[c], big) == 0 {
			c++
		}
		if c >= len(l) {
			if r+1 >= len(e.lines) {
				if len(l) > 0 {
					return r, len(l) - 1
				}
				return r, 0
			}
			r++
			c = 0
			continue
		}
		k := classOf(l[c], big)
		for c+1 < len(l) && classOf(l[c+1], big) == k {
			c++
		}
		return r, c
	}
}

// findCharPos 는 l 에서 col 기준 f/t/F/T 한 번의 이동 결과 열을 계산한다
// (findChar/opFind 가 공유). ok=false 면 대상 문자를 못 찾은 것 — 호출자는
// 이전 위치를 그대로 유지해야 한다.
func findCharPos(l []rune, col int, cmd, ch rune) (int, bool) {
	switch cmd {
	case 'f':
		for j := col + 1; j < len(l); j++ {
			if l[j] == ch {
				return j, true
			}
		}
	case 't':
		for j := col + 2; j < len(l); j++ {
			if l[j] == ch {
				return j - 1, true
			}
		}
	case 'F':
		for j := col - 1; j >= 0; j-- {
			if l[j] == ch {
				return j, true
			}
		}
	case 'T':
		for j := col - 2; j >= 0; j-- {
			if l[j] == ch {
				return j + 1, true
			}
		}
	}
	return 0, false
}

func (e *Editor) findChar(cmd, ch rune, count int) {
	c := e.col
	l := e.line()
	for i := 0; i < count; i++ {
		p, ok := findCharPos(l, c, cmd, ch)
		if !ok {
			return // count 번 다 못 찾으면 커서를 전혀 옮기지 않는다(vim 과 동일)
		}
		c = p
	}
	e.col = c
	e.dcol = e.col
}

func (e *Editor) repeatFind(reverse bool) {
	if e.lastFindCmd == 0 {
		return
	}
	cmd := e.lastFindCmd
	if reverse {
		switch cmd {
		case 'f':
			cmd = 'F'
		case 'F':
			cmd = 'f'
		case 't':
			cmd = 'T'
		case 'T':
			cmd = 't'
		}
	}
	e.findChar(cmd, e.lastFindChar, e.takeCount())
}

// gotoLineOr 는 count>0 이면 count 번째 줄로, 아니면 fallbackRow 로 이동한다.
// GotoLine(G 용: count 없으면 마지막 줄)과 gotoLineTop(gg 용: count 없으면
// 첫 줄)이 기본값만 다르고 본문이 같아서 공유한다.
func (e *Editor) gotoLineOr(count, fallbackRow int) {
	if count > 0 {
		e.row = count - 1
	} else {
		e.row = fallbackRow
	}
	e.clamp()
	e.col = firstNonBlank(e.line())
	e.dcol = e.col
}

func (e *Editor) GotoLine(count int) {
	e.gotoLineOr(count, len(e.lines)-1)
}

func (e *Editor) gotoLineTop(count int) {
	e.gotoLineOr(count, 0)
}
