package main

import "strings"

// editor.go — Vim 편집 엔진(서브셋). Ebiten 과 무관한 순수 로직이라 headless 테스트가 가능하다.
//
// 지원: Normal / Insert / Visual / Visual-Line 모드,
//   모션 h j k l 0 ^ $ w b e W B E f F t T ; , gg G,
//   연산자 d c y (+모션 / +텍스트객체 / 중복 dd cc yy) + count,
//   x X r s S D C p P, i a o O A I, u / Ctrl-r, . 반복,
//   텍스트객체 iw aw i" a" i' a' i( a( i) a) ib i{ a{ iB i[ a[ i< a<,
//   검색 / ? n N (pseudo-mode).

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

type snapshot struct {
	lines [][]rune
	row   int
	col   int
}

type Editor struct {
	lines [][]rune
	row   int
	col   int
	dcol  int // j/k 용 목표 열
	mode  Mode

	// Normal 모드 파싱 상태
	count   int
	op      rune   // 0, 'd','c','y'
	await   string // "", "f","F","t","T","r","g"
	pendObj rune   // 0, 또는 'i'/'a' (텍스트 객체 한정자 대기)

	// Visual 앵커
	vrow, vcol int

	// 레지스터
	reg         []rune
	regLines    [][]rune
	regLinewise bool

	// f/t 반복
	lastFindCmd  rune
	lastFindChar rune

	// undo/redo
	undo []snapshot
	redo []snapshot

	// dot(.) 반복
	curKeys   []Key
	dot       []Key
	changed   bool // 현재 명령이 버퍼를 변경했는가
	replaying bool

	// 검색 (/ ? n N) — pseudo-mode
	searching     bool
	searchDir     rune   // '/' 정방향, '?' 역방향
	searchQuery   []rune // 입력 중인 쿼리
	lastSearch    string // 확정된 마지막 검색어 (n/N 반복용)
	lastSearchDir rune   // 확정 시점의 방향

	// 상태 표시
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

func cloneLines(src [][]rune) [][]rune {
	out := make([][]rune, len(src))
	for i, l := range src {
		c := make([]rune, len(l))
		copy(c, l)
		out[i] = c
	}
	return out
}

func (e *Editor) pushUndo() {
	e.undo = append(e.undo, snapshot{cloneLines(e.lines), e.row, e.col})
	e.redo = nil
	e.changed = true
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

// feedSearch 는 "/foo<cr>" 같은 검색 쿼리 입력을 처리하는 pseudo-mode.
// Normal/Insert/Visual 상태 머신과 완전히 분리해 Feed() 최상단에서 가로챈다.
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

func (e *Editor) updatePending() {
	if e.searching {
		e.pendingStr = string(e.searchDir) + string(e.searchQuery)
		return
	}
	s := ""
	if e.count > 0 {
		s += itoa(e.count)
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

func (e *Editor) isCmdStart() bool {
	return e.count == 0 && e.op == 0 && e.await == "" && e.pendObj == 0
}

// ---- Normal 모드 ----

func (e *Editor) feedNormal(k Key) {
	// dot(.) 반복은 별도 처리(녹화 안 함)
	if k.R == '.' && e.isCmdStart() {
		e.replayDot()
		return
	}
	if e.isCmdStart() && !e.replaying {
		e.curKeys = nil
		e.changed = false
	}
	if !e.replaying {
		e.curKeys = append(e.curKeys, k)
	}

	// 특수키 처리
	if k.S == "esc" {
		e.clearPending()
		return
	}
	if k.S == "c-r" {
		e.redoOp()
		e.clearPending()
		return
	}

	// 인자 대기 상태(f/t/r/g)
	if e.await != "" {
		e.handleAwait(k)
		e.finishIfBoundary()
		return
	}
	// 텍스트 객체 한정자 대기 (op 뒤 i/a)
	if e.pendObj != 0 {
		if k.R != 0 {
			e.applyTextObject(e.pendObj, k.R)
		}
		e.pendObj = 0
		e.finishIfBoundary()
		return
	}

	r := k.R
	if r == 0 {
		e.finishIfBoundary()
		return
	}

	// count 입력
	if r >= '1' && r <= '9' || (r == '0' && e.count > 0) {
		e.count = e.count*10 + int(r-'0')
		return
	}

	// 연산자 대기 중 같은 키 → 줄 단위(dd/cc/yy)
	if e.op != 0 {
		if r == e.op {
			e.applyLinewise(e.op)
			e.clearPending()
			e.finishIfBoundary()
			return
		}
		if r == 'i' || r == 'a' {
			e.pendObj = r
			return
		}
		if r == 'f' || r == 'F' || r == 't' || r == 'T' {
			e.await = string(r) // 다음 글자는 찾기 대상 (handleAwait 가 op 와 조합)
			return
		}
		// 연산자 + 모션
		if e.applyOpMotion(r) {
			e.clearPending()
		}
		e.finishIfBoundary()
		return
	}

	switch r {
	case 'h', 'l', 'j', 'k', 'w', 'W', 'b', 'B', 'e', 'E', '0', '^', '$':
		e.doMotion(r, e.takeCount())
	case 'f', 'F', 't', 'T':
		e.await = string(r)
		return
	case ';':
		e.repeatFind(false)
	case ',':
		e.repeatFind(true)
	case '/':
		e.searching = true
		e.searchDir = '/'
		e.searchQuery = nil
		return
	case '?':
		e.searching = true
		e.searchDir = '?'
		e.searchQuery = nil
		return
	case 'n':
		e.jumpToMatch(e.lastSearch, e.lastSearchDir, false)
	case 'N':
		e.jumpToMatch(e.lastSearch, e.lastSearchDir, true)
	case 'g':
		e.await = "g"
		return
	case 'G':
		n := e.count // takeCount() 이전에 원본 보존(0 = count 없음 = 마지막 줄)
		e.count = 0
		e.gotoLine(n)
	case 'd', 'c', 'y':
		e.op = r
		return
	case 'x':
		e.deleteChars(e.takeCount())
	case 'X':
		e.deleteBefore(e.takeCount())
	case 'D':
		e.deleteToEOL('d')
	case 'C':
		e.deleteToEOL('c')
	case 's':
		e.substituteChar(e.takeCount())
	case 'S':
		e.applyLinewise('c')
	case 'r':
		e.await = "r"
		return
	case '~':
		e.toggleCase(e.takeCount())
	case 'p':
		e.paste(true)
	case 'P':
		e.paste(false)
	case 'i':
		e.enterInsert(false)
	case 'a':
		e.col++
		e.enterInsert(false)
	case 'I':
		e.col = firstNonBlank(e.line())
		e.enterInsert(false)
	case 'A':
		e.col = len(e.line())
		e.enterInsert(false)
	case 'o':
		e.openLine(true)
	case 'O':
		e.openLine(false)
	case 'u':
		e.undoOp()
	case 'v':
		e.mode = ModeVisual
		e.vrow, e.vcol = e.row, e.col
	case 'V':
		e.mode = ModeVisualLine
		e.vrow, e.vcol = e.row, e.col
	}
	e.clearPending()
	e.finishIfBoundary()
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

// finishIfBoundary 는 명령이 끝났고(Normal 복귀) 버퍼가 바뀌었으면 dot 저장.
func (e *Editor) finishIfBoundary() {
	if e.replaying {
		return
	}
	if e.mode == ModeNormal && e.isCmdStart() && e.changed {
		e.dot = append([]Key(nil), e.curKeys...)
		e.changed = false
	}
}

func (e *Editor) replayDot() {
	if len(e.dot) == 0 || e.replaying {
		return
	}
	e.replaying = true
	keys := e.dot
	for _, k := range keys {
		e.Feed(k)
	}
	e.replaying = false
}

func (e *Editor) handleAwait(k Key) {
	switch e.await {
	case "g":
		if k.R == 'g' {
			n := e.count // takeCount() 이전에 원본 보존(0 = count 없음 = 첫 줄)
			e.count = 0
			e.gotoLineTop(n)
		}
		e.clearPending()
	case "r":
		if k.R != 0 {
			e.replaceChar(k.R, e.takeCount())
		}
		e.clearPending()
	case "f", "F", "t", "T":
		if k.R != 0 {
			cmd := rune(e.await[0])
			if e.op != 0 {
				e.opFind(cmd, k.R)
				e.clearPending()
			} else {
				e.lastFindCmd, e.lastFindChar = cmd, k.R
				e.findChar(cmd, k.R, e.takeCount())
				e.await = ""
			}
		} else {
			e.clearPending()
		}
	}
}

// ---- 모션(커서 이동) ----

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
	cls := func(ch rune) int {
		if big {
			if ch == ' ' || ch == '\t' {
				return 0
			}
			return 1
		}
		return charClass(ch)
	}
	l := e.lines[r]
	if c < len(l) {
		start := cls(l[c])
		for c < len(l) && cls(l[c]) == start && start != 0 {
			c++
		}
	}
	for {
		for c < len(l) && cls(l[c]) == 0 {
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
		if len(l) > 0 && cls(l[0]) != 0 {
			return r, 0
		}
	}
}

func (e *Editor) prevWordStart(r, c int, big bool) (int, int) {
	cls := func(ch rune) int {
		if big {
			if ch == ' ' || ch == '\t' {
				return 0
			}
			return 1
		}
		return charClass(ch)
	}
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
		if c >= 0 && cls(l[c]) != 0 {
			k := cls(l[c])
			for c > 0 && cls(l[c-1]) == k {
				c--
			}
			return r, c
		}
		c--
	}
}

func (e *Editor) nextWordEnd(r, c int, big bool) (int, int) {
	cls := func(ch rune) int {
		if big {
			if ch == ' ' || ch == '\t' {
				return 0
			}
			return 1
		}
		return charClass(ch)
	}
	c++
	for {
		l := e.lines[r]
		for c < len(l) && cls(l[c]) == 0 {
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
		k := cls(l[c])
		for c+1 < len(l) && cls(l[c+1]) == k {
			c++
		}
		return r, c
	}
}

func (e *Editor) findChar(cmd, ch rune, count int) {
	c := e.col
	l := e.line()
	for i := 0; i < count; i++ {
		switch cmd {
		case 'f':
			p := -1
			for j := c + 1; j < len(l); j++ {
				if l[j] == ch {
					p = j
					break
				}
			}
			if p < 0 {
				return
			}
			c = p
		case 't':
			p := -1
			for j := c + 2; j < len(l); j++ {
				if l[j] == ch {
					p = j - 1
					break
				}
			}
			if p < 0 {
				return
			}
			c = p
		case 'F':
			p := -1
			for j := c - 1; j >= 0; j-- {
				if l[j] == ch {
					p = j
					break
				}
			}
			if p < 0 {
				return
			}
			c = p
		case 'T':
			p := -1
			for j := c - 2; j >= 0; j-- {
				if l[j] == ch {
					p = j + 1
					break
				}
			}
			if p < 0 {
				return
			}
			c = p
		}
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

func (e *Editor) gotoLine(count int) {
	if count <= 0 {
		e.row = len(e.lines) - 1
	} else {
		e.row = count - 1
	}
	e.clamp()
	e.col = firstNonBlank(e.line())
	e.dcol = e.col
}

func (e *Editor) gotoLineTop(count int) {
	if count > 0 {
		e.row = count - 1
	} else {
		e.row = 0
	}
	e.clamp()
	e.col = firstNonBlank(e.line())
	e.dcol = e.col
}

// ---- 연산자 + 모션 ----

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

func (e *Editor) opFind(cmd, ch rune) {
	e.takeCount() // count 는 아직 지원 안 함(단발 find만) — 대기 중이던 count 를 소비/리셋
	l := e.line()
	start := e.col
	var c1, c2 int
	switch cmd {
	case 'f':
		p := -1
		for j := start + 1; j < len(l); j++ {
			if l[j] == ch {
				p = j
				break
			}
		}
		if p < 0 {
			return
		}
		c1, c2 = start, p+1
	case 't':
		p := -1
		for j := start + 2; j < len(l); j++ {
			if l[j] == ch {
				p = j - 1
				break
			}
		}
		if p < 0 {
			return
		}
		c1, c2 = start, p+1
	case 'F':
		p := -1
		for j := start - 1; j >= 0; j-- {
			if l[j] == ch {
				p = j
				break
			}
		}
		if p < 0 {
			return
		}
		c1, c2 = p, start
	case 'T':
		p := -1
		for j := start - 2; j >= 0; j-- {
			if l[j] == ch {
				p = j + 1
				break
			}
		}
		if p < 0 {
			return
		}
		c1, c2 = p, start
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

// applyLinewise 는 줄 단위 연산자(dd cc yy / d+j 등).
func (e *Editor) applyLinewise(op rune) {
	count := e.takeCount()
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

// ---- 텍스트 객체 ----

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

// ---- 단순 편집 명령 ----

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

// ---- Insert 모드 ----

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

func (e *Editor) finishInsertDot() {
	if e.replaying {
		return
	}
	e.dot = append([]Key(nil), e.curKeys...)
	e.changed = false
}

// ---- Visual 모드 ----

func (e *Editor) feedVisual(k Key) {
	if k.S == "esc" {
		e.mode = ModeNormal
		e.clamp()
		return
	}
	if e.pendObj != 0 {
		if k.R != 0 {
			e.applyTextObject(e.pendObj, k.R)
		}
		e.pendObj = 0
		return
	}
	if e.await != "" {
		// 비주얼 모드 f/t
		if k.R != 0 {
			cmd := rune(e.await[0])
			e.lastFindCmd, e.lastFindChar = cmd, k.R
			e.findChar(cmd, k.R, 1)
		}
		e.await = ""
		return
	}
	r := k.R
	if r == 0 {
		return
	}
	switch r {
	case 'h', 'l', 'j', 'k', 'w', 'W', 'b', 'B', 'e', 'E', '0', '^', '$', 'G':
		if r == 'G' {
			e.row = len(e.lines) - 1
			e.clamp()
		} else {
			e.motionOnce(r)
			e.clamp()
		}
	case 'f', 'F', 't', 'T':
		e.await = string(r)
	case 'i', 'a':
		e.pendObj = r
	case 'd', 'x':
		e.visualOperate('d')
	case 'y':
		e.visualOperate('y')
	case 'c', 's':
		e.visualOperate('c')
	case 'o':
		e.row, e.vrow = e.vrow, e.row
		e.col, e.vcol = e.vcol, e.col
	}
}

func (e *Editor) visualOperate(op rune) {
	if e.mode == ModeVisualLine {
		r1, r2 := e.vrow, e.row
		if r1 > r2 {
			r1, r2 = r2, r1
		}
		e.row = r1
		e.count = r2 - r1 + 1
		e.mode = ModeNormal
		e.applyLinewise(op)
		e.changed = true
		e.dot = nil
		return
	}
	// charwise: 같은 줄만 지원(퍼즐 설계상 충분)
	r1, c1 := e.vrow, e.vcol
	r2, c2 := e.row, e.col
	if r1 != r2 {
		// 여러 줄이면 줄 단위로 대체 처리
		if r1 > r2 {
			r1, r2 = r2, r1
		}
		e.row = r1
		e.count = r2 - r1 + 1
		e.mode = ModeNormal
		e.applyLinewise(op)
		return
	}
	if c1 > c2 {
		c1, c2 = c2, c1
	}
	e.row = r1
	e.mode = ModeNormal
	e.applyCharRange(op, c1, c2+1) // 비주얼은 끝 포함
}

// ---- undo / redo ----

func (e *Editor) undoOp() {
	if len(e.undo) == 0 {
		return
	}
	e.redo = append(e.redo, snapshot{cloneLines(e.lines), e.row, e.col})
	s := e.undo[len(e.undo)-1]
	e.undo = e.undo[:len(e.undo)-1]
	e.lines = s.lines
	e.row, e.col = s.row, s.col
	e.clamp()
}

func (e *Editor) redoOp() {
	if len(e.redo) == 0 {
		return
	}
	e.undo = append(e.undo, snapshot{cloneLines(e.lines), e.row, e.col})
	s := e.redo[len(e.redo)-1]
	e.redo = e.redo[:len(e.redo)-1]
	e.lines = s.lines
	e.row, e.col = s.row, s.col
	e.clamp()
}

// VisualSpan 은 렌더링용 선택 범위를 반환(없으면 ok=false).
func (e *Editor) VisualSpan() (r1, c1, r2, c2 int, lineMode, ok bool) {
	if e.mode != ModeVisual && e.mode != ModeVisualLine {
		return 0, 0, 0, 0, false, false
	}
	r1, c1, r2, c2 = e.vrow, e.vcol, e.row, e.col
	if r1 > r2 || (r1 == r2 && c1 > c2) {
		r1, c1, r2, c2 = r2, c2, r1, c1
	}
	return r1, c1, r2, c2, e.mode == ModeVisualLine, true
}
