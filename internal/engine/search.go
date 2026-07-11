package engine

// search.go — "/foo<cr>" 검색 쿼리 입력을 처리하는 pseudo-mode. Normal/Insert/
// Visual 상태 머신과 완전히 분리해 Feed() 최상단에서 가로챈다(editor.go).

func (e *Editor) feedSearch(k Key) {
	switch k.S {
	case "esc":
		e.searching = false
		e.searchQuery = nil
		return
	case "cr":
		e.searching = false
		e.lastSearch = string(e.searchQuery)
		e.lastSearchDir = e.searchDir
		e.searchQuery = nil
		e.jumpToMatch(e.lastSearch, e.lastSearchDir, false)
		return
	case "bs":
		if len(e.searchQuery) > 0 {
			e.searchQuery = e.searchQuery[:len(e.searchQuery)-1]
		}
		return
	}
	if k.R != 0 {
		e.searchQuery = append(e.searchQuery, k.R)
	}
}

// jumpToMatch 는 query 를 dir 방향으로 커서 다음 위치부터 찾아 이동한다.
// reverse=true 면 dir 을 한 번 뒤집는다(N 용).
func (e *Editor) jumpToMatch(query string, dir rune, reverse bool) {
	if query == "" {
		return
	}
	effDir := dir
	if reverse {
		if dir == '/' {
			effDir = '?'
		} else {
			effDir = '/'
		}
	}
	if effDir == '/' {
		e.searchForward(query)
	} else {
		e.searchBackward(query)
	}
}

// runesIndex 는 l 에서 from 이후(포함) q 가 처음 나타나는 rune 인덱스를
// 돌려준다(없으면 -1). strings.Index 를 안 쓰는 이유: 버퍼는 [][]rune 이고
// e.col 은 rune 인덱스라, string 변환 후 byte 오프셋을 그대로 col 에 대입하면
// 멀티바이트 문자(한글 등)가 있는 줄에서 커서가 엉뚱한 열로 간다(감사 A2).
// 퍼즐 버퍼는 수십 rune 이라 단순 이중 루프로 충분하다.
func runesIndex(l, q []rune, from int) int {
	if from < 0 {
		from = 0
	}
	for c := from; c+len(q) <= len(l); c++ {
		if runesMatchAt(l, q, c) {
			return c
		}
	}
	return -1
}

// runesLastIndex 는 l[:end] 안에서 q 가 마지막으로 나타나는 rune 인덱스를
// 돌려준다(없으면 -1) — 매치는 end 이전에 끝나야 한다(strings.LastIndex 의
// line[:end] 슬라이싱과 동일한 의미).
func runesLastIndex(l, q []rune, end int) int {
	if end > len(l) {
		end = len(l)
	}
	for c := end - len(q); c >= 0; c-- {
		if runesMatchAt(l, q, c) {
			return c
		}
	}
	return -1
}

// runesMatchAt 은 l 의 c 위치에서 q 가 그대로 이어지는지 — 호출자가 c 의
// 범위(c+len(q) ≤ len(l))를 보장한다.
func runesMatchAt(l, q []rune, c int) bool {
	for i := range q {
		if l[c+i] != q[i] {
			return false
		}
	}
	return true
}

func (e *Editor) searchForward(query string) {
	q := []rune(query)
	n := len(e.lines)
	for i := 0; i <= n; i++ { // n+1 회: 현재 줄 포함 한 바퀴
		r := (e.row + i) % n
		start := 0
		if i == 0 {
			start = e.col + 1
		}
		if idx := runesIndex(e.lines[r], q, start); idx >= 0 {
			e.row, e.col = r, idx
			e.dcol = e.col
			return
		}
	}
}

func (e *Editor) searchBackward(query string) {
	q := []rune(query)
	n := len(e.lines)
	for i := 0; i <= n; i++ {
		r := ((e.row-i)%n + n) % n
		end := len(e.lines[r])
		if i == 0 {
			end = e.col
		}
		if idx := runesLastIndex(e.lines[r], q, end); idx >= 0 {
			e.row, e.col = r, idx
			e.dcol = e.col
			return
		}
	}
}
