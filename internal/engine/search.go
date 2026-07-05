package engine

import "strings"

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

func (e *Editor) searchForward(query string) {
	n := len(e.lines)
	for i := 0; i <= n; i++ { // n+1 회: 현재 줄 포함 한 바퀴
		r := (e.row + i) % n
		line := string(e.lines[r])
		start := 0
		if i == 0 {
			start = e.col + 1
		}
		if start > len(line) {
			continue
		}
		if idx := strings.Index(line[start:], query); idx >= 0 {
			e.row, e.col = r, start+idx
			e.dcol = e.col
			return
		}
	}
}

func (e *Editor) searchBackward(query string) {
	n := len(e.lines)
	for i := 0; i <= n; i++ {
		r := ((e.row-i)%n + n) % n
		line := string(e.lines[r])
		end := len(line)
		if i == 0 {
			end = e.col
		}
		if end < 0 {
			continue
		}
		if idx := strings.LastIndex(line[:end], query); idx >= 0 {
			e.row, e.col = r, idx
			e.dcol = e.col
			return
		}
	}
}
