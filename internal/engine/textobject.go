package engine

// textobject.go — iw/aw, i"/a", i(/a( 등 텍스트 객체.

func (e *Editor) applyTextObject(qual, obj rune) {
	c1, c2, ok := e.textObjectSpan(qual, obj)
	if !ok {
		e.clearPending()
		return
	}
	op := e.op
	if e.mode == ModeVisual || e.mode == ModeVisualLine {
		// 비주얼에서 텍스트객체: 선택 확장
		e.col = c2 - 1
		e.vcol = c1
		return
	}
	e.applyCharRange(op, c1, c2)
	e.clearPending()
}

// textObjectSpan: iw aw, i"/a", i'/a', 괄호류 i(/a( i[/a[ i{/a{ i</a<.
func (e *Editor) textObjectSpan(qual, obj rune) (int, int, bool) {
	l := e.line()
	switch obj {
	case 'w':
		return wordObject(l, e.col, qual == 'a')
	case '"', '\'', '`':
		return quoteObject(l, e.col, obj, qual == 'a')
	case '(', ')', 'b':
		return pairObject(l, e.col, '(', ')', qual == 'a')
	case '[', ']':
		return pairObject(l, e.col, '[', ']', qual == 'a')
	case '{', '}', 'B':
		return pairObject(l, e.col, '{', '}', qual == 'a')
	case '<', '>':
		return pairObject(l, e.col, '<', '>', qual == 'a')
	}
	return 0, 0, false
}

func wordObject(l []rune, col int, around bool) (int, int, bool) {
	if len(l) == 0 || col >= len(l) {
		return 0, 0, false
	}
	k := charClass(l[col])
	s, en := col, col
	for s > 0 && charClass(l[s-1]) == k {
		s--
	}
	for en+1 < len(l) && charClass(l[en+1]) == k {
		en++
	}
	c1, c2 := s, en+1
	if around {
		// 뒤따르는 공백 포함, 없으면 앞 공백
		ext := c2
		for ext < len(l) && charClass(l[ext]) == 0 {
			ext++
		}
		if ext > c2 {
			c2 = ext
		} else {
			for c1 > 0 && charClass(l[c1-1]) == 0 {
				c1--
			}
		}
	}
	return c1, c2, true
}

func quoteObject(l []rune, col int, q rune, around bool) (int, int, bool) {
	// 줄에서 col 을 감싸는(또는 다음) 한 쌍의 q 를 찾는다.
	open := -1
	for i := 0; i < len(l); i++ {
		if l[i] == q {
			// 짝 찾기
			j := -1
			for k := i + 1; k < len(l); k++ {
				if l[k] == q {
					j = k
					break
				}
			}
			if j < 0 {
				return 0, 0, false
			}
			if col <= j {
				open = i
				if around {
					return open, j + 1, true
				}
				return open + 1, j, true
			}
			i = j
		}
	}
	return 0, 0, false
}

func pairObject(l []rune, col int, open, close rune, around bool) (int, int, bool) {
	// 빈 줄(길이 0)이면 clamp() 가 col=0 을 허용하는데(lastCol 이 빈 줄에서 0을
	// 돌려줌), 그 col 은 "한 칸도 없는 줄" 의 유효하지 않은 인덱스라 아래
	// l[i] 인덱싱이 패닉한다 — wordObject 와 동일한 가드(F3 fuzz 로 발견).
	if len(l) == 0 || col >= len(l) {
		return 0, 0, false
	}
	// open 위치: col 에서 왼쪽으로 균형 탐색
	depth := 0
	o := -1
	for i := col; i >= 0; i-- {
		if l[i] == close && i != col {
			depth++
		} else if l[i] == open {
			if depth == 0 {
				o = i
				break
			}
			depth--
		}
	}
	if o < 0 {
		return 0, 0, false
	}
	depth = 0
	c := -1
	for i := o + 1; i < len(l); i++ {
		if l[i] == open {
			depth++
		} else if l[i] == close {
			if depth == 0 {
				c = i
				break
			}
			depth--
		}
	}
	if c < 0 {
		return 0, 0, false
	}
	if around {
		return o, c + 1, true
	}
	return o + 1, c, true
}
