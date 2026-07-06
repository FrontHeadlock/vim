package engine

// macro.go — 매크로(q/@/@@) 재생. 녹화는 editor.go(Feed 캡처 훅)와
// normal.go(q/@ 디스패치, handleAwait)가 나눠 맡는다.

// playMacro 는 레지스터 reg 에 저장된 매크로를 count 번 재생한다. 미기록/빈
// 레지스터는 조용히 no-op — 실제 vim 도 빈 매크로 재생을 에러로 다루지
// 않는다. e.replaying(dot 전용 플래그)은 세팅하지 않는다: 매크로 안의 개별
// 커맨드가 "사용자가 직접 친 것"처럼 자기 자신의 dot/undo 경계를 정상적으로
// 갖게 하기 위함이다(재생 후 "."은 매크로 전체가 아니라 마지막 변경 커맨드를
// 반복 — 실제 vim과 동일).
func (e *Editor) playMacro(reg rune, count int) {
	keys, ok := e.macros[reg]
	if !ok || len(keys) == 0 || e.macroDepth >= maxMacroDepth {
		return
	}
	// macroStepsLeft 는 최상위 호출에서만 새로 채운다 — 재귀 호출(매크로가
	// 자기 자신 또는 다른 매크로를 부르는 경우)은 같은 예산을 공유해서 써야
	// depth·count 조합의 지수적 폭주를 막을 수 있다(editor.go 의 상수 주석 참고).
	if e.macroDepth == 0 {
		e.macroStepsLeft = maxMacroSteps
	}
	e.macroDepth++
	for i := 0; i < count; i++ {
		for _, k := range keys {
			if e.macroStepsLeft <= 0 {
				e.macroDepth--
				return
			}
			e.macroStepsLeft--
			e.Feed(k)
		}
	}
	e.macroDepth--
}
