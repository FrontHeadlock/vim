package engine

// visual.go — Visual/VisualLine 모드.

// multilineCharwiseFallbacks 는 visualOperate 의 "여러 줄 charwise 선택은
// 줄 단위로 대체 처리"(B4) 분기가 실행된 횟수. 이 분기는 실제 Vim 과 다르게
// 동작하는 알려진 부정확성이라 — 레벨 Solution 이 이 경로를 절대 밟지
// 않음을 테스트로 보장하기 위한 훅이다(ResetMultilineCharwiseFallbackCount/
// MultilineCharwiseFallbackCount, 게임 패키지의 레벨 검증 테스트가 사용).
// 프로덕션 동작에는 영향 없는 카운터 하나뿐이다.
var multilineCharwiseFallbacks int

// ResetMultilineCharwiseFallbackCount 는 카운터를 0으로 되돌린다(테스트 시작 시 호출).
func ResetMultilineCharwiseFallbackCount() { multilineCharwiseFallbacks = 0 }

// MultilineCharwiseFallbackCount 는 마지막 리셋 이후 분기가 실행된 횟수.
func MultilineCharwiseFallbackCount() int { return multilineCharwiseFallbacks }

func (e *Editor) feedVisual(k Key) {
	if k.S == "esc" {
		e.mode = ModeNormal
		e.clamp()
		// B7: count 가 남아있으면 esc 로 지운다 — 안 그러면 "v2<esc>d" 처럼
		// 취소된 count 가 다음 Normal 커맨드로 새어 들어간다(count 는 Normal/
		// Visual 이 공유하는 필드).
		e.clearPending()
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
		// 비주얼 모드 f/t — count 는 Normal 모드와 동일하게 여기서 소비(예: "2fx").
		if k.R != 0 {
			cmd := rune(e.await[0])
			e.lastFindCmd, e.lastFindChar = cmd, k.R
			e.findChar(cmd, k.R, e.takeCount())
		}
		e.await = ""
		return
	}
	r := k.R
	if r == 0 {
		return
	}
	// B7: count 입력(2w, 3j 등) — Normal 모드(feedNormal)와 동일한 누적 규칙.
	// 예전엔 비주얼 모드가 count 를 전혀 안 봐서 "2w"/"3j" 가 한 칸만 이동해
	// 실제 Vim 과 다른 동작을 가르쳤다.
	if r >= '1' && r <= '9' || (r == '0' && e.count > 0) {
		e.count = e.count*10 + int(r-'0')
		// F3 fuzz 로 발견: normal.go 의 count 상한(maxCount)을 여기서도 걸지
		// 않으면 "V" + 긴 숫자열 + 모션이 아래 motionOnce 반복 루프를 그대로
		// O(count) 로 돌려 멈춘다 — 두 count 누적 로직이 복제되면서 상한만
		// 새어나간 사례.
		if e.count > maxCount || e.count < 0 {
			e.count = maxCount
		}
		return
	}
	switch r {
	case 'h', 'l', 'j', 'k', 'w', 'W', 'b', 'B', 'e', 'E', '0', '^', '$', 'G':
		if r == 'G' {
			n := e.count // takeCount() 이전에 원본 보존(0 = count 없음 = 마지막 줄)
			e.count = 0
			if n > 0 {
				e.row = n - 1
			} else {
				e.row = len(e.lines) - 1
			}
			e.clamp()
		} else {
			count := e.takeCount()
			for i := 0; i < count; i++ {
				e.motionOnce(r)
			}
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

// visualOperate 는 자신의 changed/dot 을 직접 관리한다(Normal 모드의
// finishIfBoundary/commitUndoIfChanged 경로를 타지 않는다) — 그래서 매
// 분기 끝에 e.undoPending 을 직접 소비해 둔다. 그러지 않으면 pushUndo 가
// 여기서 세운 "대기 중" 표시가 다음 Normal 커맨드의 finishIfBoundary 까지
// 새어 들어가, 서로 다른 두 커맨드의 버퍼 상태를 비교하는 사고가 난다.
func (e *Editor) visualOperate(op rune) {
	if e.mode == ModeVisualLine {
		r1, r2 := e.vrow, e.row
		if r1 > r2 {
			r1, r2 = r2, r1
		}
		e.row = r1
		e.mode = ModeNormal
		e.applyLinewiseN(op, r2-r1+1)
		e.changed = true
		e.dot = nil
		e.undoPending = false
		return
	}
	// charwise: 같은 줄만 지원(퍼즐 설계상 충분)
	r1, c1 := e.vrow, e.vcol
	r2, c2 := e.row, e.col
	if r1 != r2 {
		// 여러 줄이면 줄 단위로 대체 처리
		multilineCharwiseFallbacks++
		if r1 > r2 {
			r1, r2 = r2, r1
		}
		e.row = r1
		e.mode = ModeNormal
		e.applyLinewiseN(op, r2-r1+1)
		e.undoPending = false
		return
	}
	if c1 > c2 {
		c1, c2 = c2, c1
	}
	e.row = r1
	e.mode = ModeNormal
	e.applyCharRange(op, c1, c2+1) // 비주얼은 끝 포함
	e.undoPending = false
}
