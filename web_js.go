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
	// Ebiten 쪽의 request* 폴링 플래그 대신 Game 을 직접 호출한다.
	js.Global().Set("vimquestReset", js.FuncOf(func(js.Value, []js.Value) any {
		vqGame.loadLevel(vqGame.levelIdx)
		return nil
	}))
	js.Global().Set("vimquestRestart", js.FuncOf(func(js.Value, []js.Value) any {
		vqGame.loadLevel(0)
		return nil
	}))
	js.Global().Set("vimquestLevelSelect", js.FuncOf(func(js.Value, []js.Value) any {
		vqGame.enterLevelSelect()
		return nil
	}))

	select {}
}
