// Package engine 은 VimQuest 의 Vim 편집 엔진(서브셋)이다.
// 게임 규칙·렌더링·플랫폼과 완전히 무관한 순수 로직으로, 표준 라이브러리 외에
// 아무것도 import 하지 않는다 — headless 테스트와 TinyGo wasm 빌드의 전제.
//
// Editor 의 내부 상태는 전부 비공개다. 바깥 레이어(게임 규칙, 렌더러)는 파일
// 하단(api.go)의 읽기 전용 접근자와 SetCursor 만 쓸 수 있다.
//
// 파일 구성:
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
	undoPending bool // pushUndo 가 이번 커맨드에서 호출됐는지(커밋 시점에 실제 변경 여부 확인)

	searching     bool
	searchDir     rune
	searchQuery   []rune
	lastSearch    string
	lastSearchDir rune

	lastKey    string
	pendingStr string

	macros         map[rune][]Key // 레지스터별 완성된 매크로
	recording      rune           // 0=미기록, 아니면 기록 중인 레지스터
	recordBuf      []Key          // 기록 버퍼(curKeys 와는 별개 — dot 반복용이 아니다)
	lastMacroReg   rune           // "@@" 가 재생할 마지막 레지스터
	macroDepth     int            // 재생 중첩 가드(재귀 매크로 방지)
	macroStepsLeft int            // 최상위 playMacro 호출 트리 전체의 남은 재생 스텝
}

// maxMacroDepth 는 매크로 재생 중첩(재귀 매크로 등)의 상한 — 재귀 자체를
// 막는 안전망이다.
//
// maxMacroSteps 는 그와 별개로 훨씬 더 중요한 상한이다: depth 상한만으로는
// "레지스터 안에서 자기 자신을 count>1 로 재생"하는 패턴(예: "qa2@aq" 뒤
// "@a")을 못 막는다 — 재귀 한 단계 내려갈 때마다 남은 count 만큼 다시
// 갈라지므로 실행량이 depth 에 지수적으로 불어난다(depth=100, count=2 여도
// 2^100 번의 Feed 호출). fuzz(FuzzEditorNeverPanics)가 "qa2@aq@a" 로 이 행업을
// 실제로 찾아냈다. macroStepsLeft 는 재귀 트리 전체가 공유하는 단일 예산이라
// depth·count 조합과 무관하게 총 실행량을 유계로 만든다.
const (
	maxMacroDepth = 100
	maxMacroSteps = 1 << 16
)

func NewEditor(lines []string) *Editor {
	e := &Editor{macros: map[rune][]Key{}}
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
	// 매크로 녹화: Feed 가 유일한 모드 무관 진입점이라 여기 두면 검색
	// 시퀀스(/pattern<cr>)까지 포함해 임의 커맨드를 기록할 수 있다.
	// !e.replaying 은 dot(.) 재생 중 내부 전개가 이중으로 녹화되는 것을
	// 막고, macroDepth==0 은 매크로 재생 자체의 키가 (재생 중 우연히
	// 녹화가 걸려 있어도) 다시 녹화되지 않게 막는다. 시작/종료 "q"와
	// 레지스터 문자는 recording 이 아직/이미 0 인 시점이라 자연히 빠지고,
	// 종료 "q" 한 글자만 case 'q' 쪽에서 잘라낸다(normal.go).
	if e.recording != 0 && !e.replaying && e.macroDepth == 0 {
		e.recordBuf = append(e.recordBuf, k)
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

// maxCount 는 count 접두사(예: "12dd")의 상한. 상한이 없으면 매우 큰 수를
// 입력해도 doMotion/motionSpan/findChar 의 O(count) 루프가 그대로 실행돼
// 수초~수십초 멈춘다(웹 빌드는 브라우저 탭이 얼어붙는다).
const maxCount = 9999

// accumCount 는 숫자 키 하나를 count 에 누적한다(자릿수 쌓기 + 오버플로
// 포함 상한 클램프). 누적했으면 true — 호출자는 이때 더 처리하지 않고
// return 해야 한다. Normal/Visual 모드가 이 로직을 각각 복제해 상한
// 클램프를 한쪽에서 빠뜨린 전례가 있어(fuzz 로 실제 행업 발견) 하나로
// 통합했다.
func (e *Editor) accumCount(r rune) bool {
	isDigit := r >= '1' && r <= '9' || (r == '0' && e.count > 0)
	if !isDigit {
		return false
	}
	e.count = e.count*10 + int(r-'0')
	if e.count > maxCount || e.count < 0 {
		e.count = maxCount
	}
	return true
}
