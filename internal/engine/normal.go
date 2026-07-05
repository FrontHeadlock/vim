package engine

// normal.go — Normal 모드 키 디스패치(editor.go 의 Feed 가 여기로 넘긴다).

// maxCount 는 count 접두사(예: "12dd")의 상한. 실제 레벨 어떤 것도 이 근처를
// 요구하지 않지만, 상한이 없으면 "2000000000B" 같은 입력이 doMotion/
// motionSpan(operator.go)/findChar 의 O(count) 반복 루프를 그대로 실행해
// 수 초~수십 초 동안 멈춘다 — 웹 빌드에선 브라우저 탭이 그대로 얼어붙는다는
// 뜻이라 더 심각하다(F3 fuzz 로 발견: "2000000000Bxxx..." 가 실제로 멈춤).
const maxCount = 9999

func (e *Editor) feedNormal(k Key) {
	// dot(.) 반복은 별도 처리(녹화 안 함)
	if k.R == '.' && e.IsCmdStart() {
		e.replayDot()
		return
	}
	if e.IsCmdStart() && !e.replaying {
		e.curKeys = nil
		e.changed = false
		// B2: dot 재생 중에는 finishIfBoundary/finishInsertDot 이 replaying
		// 가드로 commitUndoIfChanged 를 건너뛴다 — 재생이 끝난 뒤 undoPending 이
		// true 로 남아있으면 그 다음 진짜 커맨드가 그 오래된 스냅샷과 비교돼
		// dot 을 엉뚱한 키(예: 순수 모션 하나)로 덮어쓴다. 새 커맨드 시작점에서
		// 반드시 정리한다.
		e.undoPending = false
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
		if e.count > maxCount || e.count < 0 { // 오버플로 포함 상한 고정
			e.count = maxCount
		}
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
		e.GotoLine(n)
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

// finishIfBoundary 는 명령이 끝났고(Normal 복귀) 버퍼가 실제로 바뀌었으면
// dot 저장(B2: "무변경 커밋" — 예: `~`/`r` 가 결과적으로 아무것도 바꾸지
// 않았으면 undo 스택에도 dot 에도 남기지 않는다).
func (e *Editor) finishIfBoundary() {
	if e.replaying {
		return
	}
	if e.mode != ModeNormal || !e.IsCmdStart() {
		return
	}
	if e.commitUndoIfChanged() {
		e.changed = true
	}
	if e.changed {
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
