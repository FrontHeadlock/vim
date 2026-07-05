// Package engine 은 VimQuest 의 Vim 편집 엔진(서브셋)이다.
// 게임 규칙·렌더링·플랫폼과 완전히 무관한 순수 로직으로, 표준 라이브러리 외에
// 아무것도 import 하지 않는다 — headless 테스트와 TinyGo wasm 빌드의 전제.
//
// Editor 의 내부 상태는 전부 비공개다. 바깥 레이어(게임 규칙, 렌더러)는 파일
// 하단(api.go)의 읽기 전용 접근자와 SetCursor 만 쓸 수 있다.
//
// 파일 구성(F5: 1,585줄이던 editor.go 를 순수 이동으로 분해 — 로직 변경 없음):
// editor.go(이 파일, 타입·Feed 디스패치·공용 유틸) · search.go(검색 pseudo-mode) ·
// normal.go(Normal 모드 디스패치) · motion.go(커서 모션) ·
// operator.go(연산자+모션 스팬) · textobject.go(iw/i(/i" 등) ·
// edit.go(x/r/~/p 등 단순 편집) · insert.go(Insert 모드) · visual.go(Visual 모드) ·
// undo.go(undo/redo 스택) · api.go(패키지 밖 공개 API).
package engine

import (
	"strconv"
)

type Mode int

const (
	ModeNormal Mode = iota
	ModeInsert
	ModeVisual
	ModeVisualLine
)

// Key 는 엔진에 들어오는 한 번의 입력. 일반 문자는 R, 특수키는 S("esc","cr","bs","c-r").
type Key struct {
	R rune
	S string
}

func RuneKey(r rune) Key      { return Key{R: r} }
func SpecialKey(s string) Key { return Key{S: s} }

type Editor struct {
	lines [][]rune
	row   int
	col   int
	dcol  int // j/k 목표 열
	mode  Mode

	count   int
	op      rune
	await   string
	pendObj rune

	vrow, vcol int

	reg         []rune
	regLines    [][]rune
	regLinewise bool

	lastFindCmd  rune
	lastFindChar rune

	undo []snapshot
	redo []snapshot

	curKeys     []Key
	dot         []Key
	changed     bool
	replaying   bool
	undoPending bool // B2: pushUndo 가 이번 커맨드에서 호출됐는지(커밋 시점에 실제 변경 여부 확인)

	searching     bool
	searchDir     rune
	searchQuery   []rune
	lastSearch    string
	lastSearchDir rune

	lastKey    string
	pendingStr string
}

func NewEditor(lines []string) *Editor {
	e := &Editor{}
	e.SetLines(lines)
	return e
}

func (e *Editor) SetLines(lines []string) {
	e.lines = make([][]rune, len(lines))
	for i, l := range lines {
		e.lines[i] = []rune(l)
	}
	if len(e.lines) == 0 {
		e.lines = [][]rune{{}}
	}
	e.row, e.col, e.dcol = 0, 0, 0
	e.mode = ModeNormal
}

// Lines 는 현재 버퍼를 문자열 슬라이스로 반환(목표 비교/렌더용).
func (e *Editor) Lines() []string {
	out := make([]string, len(e.lines))
	for i, l := range e.lines {
		out[i] = string(l)
	}
	return out
}

func (e *Editor) ModeName() string {
	switch e.mode {
	case ModeInsert:
		return "-- INSERT --"
	case ModeVisual:
		return "-- VISUAL --"
	case ModeVisualLine:
		return "-- VISUAL LINE --"
	default:
		return "-- NORMAL --"
	}
}

// ---- 유틸 ----

func (e *Editor) line() []rune { return e.lines[e.row] }

func (e *Editor) lastCol(insert bool) int {
	n := len(e.lines[e.row])
	if insert {
		return n
	}
	if n == 0 {
		return 0
	}
	return n - 1
}

func (e *Editor) clamp() {
	if e.row < 0 {
		e.row = 0
	}
	if e.row >= len(e.lines) {
		e.row = len(e.lines) - 1
	}
	max := e.lastCol(e.mode == ModeInsert)
	if e.col > max {
		e.col = max
	}
	if e.col < 0 {
		e.col = 0
	}
}

func charClass(r rune) int {
	switch {
	case r == ' ' || r == '\t':
		return 0
	case r == '_' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		r >= 0x80: // 비ASCII 는 단어로 취급
		return 1
	default:
		return 2
	}
}

func firstNonBlank(l []rune) int {
	for i, r := range l {
		if r != ' ' && r != '\t' {
			return i
		}
	}
	return 0
}

// classOf 는 big(WORD) 여부에 맞춘 문자 분류기.
func classOf(r rune, big bool) int {
	if big {
		if r == ' ' || r == '\t' {
			return 0
		}
		return 1
	}
	return charClass(r)
}

// ---- 입력 진입점 ----

func (e *Editor) Feed(k Key) {
	if k.R != 0 {
		e.lastKey = string(k.R)
	} else {
		e.lastKey = k.S
	}
	if e.searching {
		e.feedSearch(k)
		e.updatePending()
		return
	}
	switch e.mode {
	case ModeInsert:
		e.feedInsert(k)
	case ModeVisual, ModeVisualLine:
		e.feedVisual(k)
	default:
		e.feedNormal(k)
	}
	e.updatePending()
}

func (e *Editor) updatePending() {
	if e.searching {
		e.pendingStr = string(e.searchDir) + string(e.searchQuery)
		return
	}
	s := ""
	if e.count > 0 {
		s += strconv.Itoa(e.count)
	}
	if e.op != 0 {
		s += string(e.op)
	}
	if e.pendObj != 0 {
		s += string(e.pendObj)
	}
	if e.await != "" {
		s += e.await
	}
	e.pendingStr = s
}

func (e *Editor) clearPending() {
	e.count = 0
	e.op = 0
	e.await = ""
	e.pendObj = 0
}

func (e *Editor) IsCmdStart() bool {
	return e.count == 0 && e.op == 0 && e.await == "" && e.pendObj == 0
}

// takeCount 은 누적 count 를 소비(없으면 1).
func (e *Editor) takeCount() int {
	c := e.count
	e.count = 0
	if c <= 0 {
		return 1
	}
	return c
}
