package engine

// operator.go — 연산자(d/c/y)와 모션이 결합한 charwise/linewise 범위 계산 및 적용.

// wordStartFwdLine: 같은 줄에서 다음 단어 시작 열(없으면 len). dw/yw 용.
func wordStartFwdLine(l []rune, col int, big bool) int {
	c := col
	if c < len(l) && classOf(l[c], big) != 0 {
		k := classOf(l[c], big)
		for c < len(l) && classOf(l[c], big) == k {
			c++
		}
	}
	for c < len(l) && classOf(l[c], big) == 0 {
		c++
	}
	return c
}

// wordEndFwdLine: 같은 줄에서 단어 끝 열(포함). e/ce 용.
func wordEndFwdLine(l []rune, col int, big bool) int {
	c := col + 1
	for c < len(l) && classOf(l[c], big) == 0 {
		c++
	}
	if c >= len(l) {
		if len(l) == 0 {
			return 0
		}
		return len(l) - 1
	}
	k := classOf(l[c], big)
	for c+1 < len(l) && classOf(l[c+1], big) == k {
		c++
	}
	return c
}

// wordStartBackLine: 같은 줄에서 이전 단어 시작 열. db 용.
func wordStartBackLine(l []rune, col int, big bool) int {
	c := col - 1
	for c >= 0 && classOf(l[c], big) == 0 {
		c--
	}
	if c < 0 {
		return 0
	}
	k := classOf(l[c], big)
	for c > 0 && classOf(l[c-1], big) == k {
		c--
	}
	return c
}

// motionSpan 은 cmd 모션의 결과 범위를 같은 줄 charwise [c1,c2) 로 돌려준다.
// ok=false 면 적용 불가. linewise 모션은 여기서 다루지 않는다.
func (e *Editor) motionSpan(cmd rune, count int) (c1, c2 int, ok bool) {
	l := e.line()
	start := e.col
	switch cmd {
	case 'l', ' ':
		end := start + count
		if end > len(l) {
			end = len(l)
		}
		return start, end, true
	case 'h':
		end := start - count
		if end < 0 {
			end = 0
		}
		return end, start, true
	case '0':
		return 0, start, true
	case '^':
		fb := firstNonBlank(l)
		if fb <= start {
			return fb, start, true
		}
		return start, fb, true
	case '$':
		return start, len(l), true // 줄 끝까지 포함
	case 'w', 'W':
		nc := start
		for i := 0; i < count; i++ {
			nc = wordStartFwdLine(l, nc, cmd == 'W')
		}
		return start, nc, true
	case 'e', 'E':
		nc := start
		for i := 0; i < count; i++ {
			nc = wordEndFwdLine(l, nc, cmd == 'E')
		}
		return start, nc + 1, true // 포함
	case 'b', 'B':
		nc := start
		for i := 0; i < count; i++ {
			nc = wordStartBackLine(l, nc, cmd == 'B')
		}
		return nc, start, true
	}
	return 0, 0, false
}

func (e *Editor) applyOpMotion(cmd rune) bool {
	count := e.takeCount()
	// 줄 단위 모션
	if cmd == 'G' || cmd == 'j' || cmd == 'k' {
		e.applyLinewise(e.op)
		return true
	}
	// vim 특수규칙: 단어 위에서 cw 는 ce 처럼 동작(뒤 공백 미포함).
	if e.op == 'c' && (cmd == 'w' || cmd == 'W') {
		l := e.line()
		if e.col < len(l) && classOf(l[e.col], cmd == 'W') != 0 {
			cmd = cmd - 'w' + 'e'
		}
	}
	c1, c2, ok := e.motionSpan(cmd, count)
	if !ok {
		return false
	}
	e.applyCharRange(e.op, c1, c2)
	return true
}

// opFind 는 d/c/y + f/F/t/T 조합을 처리한다. count 는 findChar(모션 단독 f/t)와
// 동일하게 반복 탐색으로 지원한다(B3 — 예전엔 e.takeCount() 로 소비만 하고
// 버려서 "d2fl" 이 "dfl" 처럼 동작했다).
func (e *Editor) opFind(cmd, ch rune) {
	count := e.takeCount()
	l := e.line()
	start := e.col
	c := start
	for i := 0; i < count; i++ {
		p, ok := findCharPos(l, c, cmd, ch)
		if !ok {
			return // count 번 다 못 찾으면 아무것도 적용하지 않는다(vim 과 동일)
		}
		c = p
	}
	var c1, c2 int
	switch cmd {
	case 'f', 't':
		c1, c2 = start, c+1
	case 'F', 'T':
		c1, c2 = c, start
	}
	e.applyCharRange(e.op, c1, c2)
}

// applyCharRange 는 현재 줄 [c1,c2) 에 연산자 적용.
func (e *Editor) applyCharRange(op rune, c1, c2 int) {
	if c1 > c2 {
		c1, c2 = c2, c1
	}
	l := e.line()
	if c1 < 0 {
		c1 = 0
	}
	if c2 > len(l) {
		c2 = len(l)
	}
	e.reg = append([]rune(nil), l[c1:c2]...)
	e.regLinewise = false
	switch op {
	case 'y':
		e.col = c1
	case 'd':
		e.pushUndo()
		e.lines[e.row] = append(append([]rune(nil), l[:c1]...), l[c2:]...)
		e.col = c1
		e.clamp()
	case 'c':
		e.pushUndo()
		e.lines[e.row] = append(append([]rune(nil), l[:c1]...), l[c2:]...)
		e.col = c1
		e.mode = ModeInsert
	}
}

// applyLinewise 는 줄 단위 연산자(dd cc yy / d+j 등). count 는 e.count(현재
// 대기 중인 카운트)에서 소비한다 — 호출 전에 카운트가 아직 안 쓰였어야 하는
// 암묵 계약이 있다(B5). 이미 count 를 손에 쥔 호출부는 applyLinewiseN 을 직접 쓸 것.
func (e *Editor) applyLinewise(op rune) {
	e.applyLinewiseN(op, e.takeCount())
}

// applyLinewiseN 은 applyLinewise 의 명시적 count 버전. e.count 상태를 암묵
// 인자로 쓰지 않는다 — visualOperate 가 예전엔 e.count = r2-r1+1 로 상태
// 필드를 임시 채운 뒤 이 함수(당시 applyLinewise)가 takeCount 로 꺼내 쓰는
// 식이라, 호출 순서에 묶인 계약이 리팩터링 시 깨지기 쉬웠다(B5).
func (e *Editor) applyLinewiseN(op rune, count int) {
	r1 := e.row
	r2 := e.row + count - 1
	if r2 >= len(e.lines) {
		r2 = len(e.lines) - 1
	}
	e.regLines = cloneLines(e.lines[r1 : r2+1])
	e.regLinewise = true
	switch op {
	case 'y':
		e.row = r1
		// col 을 새 현재 줄 길이에 맞춰 다시 clamp — Visual 다중 줄 yank
		// 폴백(visualOperate)처럼 row 가 이전보다 훨씬 짧은 줄로 바뀔 수 있어,
		// 안 하면 col 이 그 줄 길이를 넘어선 채로 남는다(F3 fuzz 로 발견:
		// "wvEEEy" 가 col 범위 밖 패닉을 냄). firstNonBlank 로 강제 이동하지
		// 않는 이유는 yank 는 실제 vim 에서도 커서 열을 바꾸지 않기 때문 —
		// clamp() 는 이미 유효한 col 은 그대로 두고 넘친 경우만 자른다.
		e.clamp()
	case 'd':
		e.pushUndo()
		e.lines = append(e.lines[:r1], e.lines[r2+1:]...)
		if len(e.lines) == 0 {
			e.lines = [][]rune{{}}
		}
		if e.row >= len(e.lines) {
			e.row = len(e.lines) - 1
		}
		e.col = firstNonBlank(e.line())
	case 'c':
		e.pushUndo()
		for i := r1; i <= r2; i++ {
			e.lines[i] = []rune{}
		}
		e.lines = append(e.lines[:r1+1], e.lines[r2+1:]...)
		e.row = r1
		e.col = 0
		e.mode = ModeInsert
	}
}
