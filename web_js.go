//go:build js

package main

import "syscall/js"

// web_js.go — TinyGo 웹 프론트엔드 진입점.
//
// 이벤트 구동 모델: 키 입력마다 vqInput 을 호출하고 반환된 스냅샷으로 즉시
// 다시 그린다. 60fps 상시 루프는 없다 — effects/bell 이 살아있는 동안만
// glue.js 가 requestAnimationFrame 으로 vqTick 을 돌린다.

var vqGame *Game

func main() {
	vqGame = NewGame()

	// 키 토큰 하나(문자 또는 "<cr>"/"<esc>"/"<bs>"/"<c-r>")를 받아 즉시 적용하고
	// 새 스냅샷을 돌려준다. 토큰 문법은 keys.go 의 parseKeys 와 동일 — 테스트와
	// 웹 프론트엔드가 같은 파서를 공유한다.
	js.Global().Set("vqInput", js.FuncOf(func(_ js.Value, args []js.Value) any {
		for _, k := range parseKeys(args[0].String()) {
			vqGame.Input(k)
		}
		return vqGame.snapshot()
	}))
	js.Global().Set("vqState", js.FuncOf(func(js.Value, []js.Value) any {
		return vqGame.snapshot()
	}))
	js.Global().Set("vqTick", js.FuncOf(func(js.Value, []js.Value) any {
		vqGame.Tick()
		return vqGame.snapshot()
	}))

	// 기존 HTML 버튼(RESET/RESTART/LEVELS)이 호출하는 전역 함수.
	// Ebiten 쪽의 request* 폴링 플래그 대신 Game 을 직접 호출한다. vqInput 등과
	// 마찬가지로 스냅샷을 돌려줘야 glue.js 가 클릭 직후 캔버스를 다시 그릴 수
	// 있다 — nil 을 돌려주면 다음 키 입력 전까지 화면이 그대로 멈춰 있다.
	//
	// RESET 은 ":restart"/":e!" 와 의미가 같은 "지금 하던 것 다시" 버튼이라
	// restartCurrent() 를 그대로 써야 한다 — 여기서 g.loadLevel(g.levelIdx) 를
	// 직접 부르면 :drill 인식 분기가 빠져 드릴 중 RESET 을 누르면 커리큘럼으로
	// 튕겨나가는 버그가 재발한다(:restart 만 고치고 이 버튼을 놓쳐서 실제로
	// 겪은 회귀). RESTART 는 "게임 전체를 처음부터" 라는 더 큰 동작이라
	// :drill 을 벗어나는 게 맞으므로 loadLevel(0) 그대로 둔다.
	js.Global().Set("vimquestReset", js.FuncOf(func(js.Value, []js.Value) any {
		vqGame.restartCurrent()
		return vqGame.snapshot()
	}))
	js.Global().Set("vimquestRestart", js.FuncOf(func(js.Value, []js.Value) any {
		vqGame.loadLevel(0)
		return vqGame.snapshot()
	}))
	js.Global().Set("vimquestLevelSelect", js.FuncOf(func(js.Value, []js.Value) any {
		vqGame.enterLevelSelect()
		return vqGame.snapshot()
	}))

	select {}
}
