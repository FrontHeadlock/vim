//go:build js

package jsbridge

import (
	"syscall/js"

	"vimquest/internal/game"
)

var vq *game.Game

func toJS(v any) js.Value {
	switch x := v.(type) {
	case map[string]any:
		o := js.Global().Get("Object").New()
		for k, vv := range x {
			o.Set(k, toJS(vv))
		}
		return o
	case []any:
		a := js.Global().Get("Array").New(len(x))
		for i, vv := range x {
			a.SetIndex(i, toJS(vv))
		}
		return a
	case string:
		return js.ValueOf(x)
	case int:
		return js.ValueOf(x)
	case bool:
		return js.ValueOf(x)
	}
	return js.Null()
}

func Run() {
	vq = game.New()

	js.Global().Set("vqInput", js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) > 0 {
			tok := args[0].String()
			key := game.ParseKey(tok)
			vq.Input(key)
		}
		return toJS(vq.Snapshot())
	}))

	js.Global().Set("vqState", js.FuncOf(func(js.Value, []js.Value) any {
		return toJS(vq.Snapshot())
	}))

	js.Global().Set("vqTick", js.FuncOf(func(js.Value, []js.Value) any {
		vq.Tick()
		return toJS(vq.Snapshot())
	}))

	// Arena 진입 — 시간 측정·제출은 전부 JS(glue.js) 몫이고, wasm 경계는
	// 이 진입 호출 하나만 넓어진다(네트워킹은 wasm 을 거치지 않는다).
	js.Global().Set("vqArenaStart", js.FuncOf(func(js.Value, []js.Value) any {
		vq.EnterArena()
		return toJS(vq.Snapshot())
	}))

	// index.html 의 RESET/RESTART/LEVELS 버튼이 부르는 훅 3종 — 패키지 재편
	// (91712dc) 때 옛 web_js.go 와 함께 유실됐던 것을 복원한다. 의미는 유실
	// 이전과 동일: RESET = 지금 하던 것을 strokes=0 으로(드릴/아레나 인식),
	// RESTART = 1-1 부터, LEVELS = 레벨 선택.
	js.Global().Set("vimquestReset", js.FuncOf(func(js.Value, []js.Value) any {
		vq.RestartCurrent()
		return toJS(vq.Snapshot())
	}))
	js.Global().Set("vimquestRestart", js.FuncOf(func(js.Value, []js.Value) any {
		vq.LoadLevel(0)
		return toJS(vq.Snapshot())
	}))
	js.Global().Set("vimquestLevelSelect", js.FuncOf(func(js.Value, []js.Value) any {
		vq.EnterLevelSelect()
		return toJS(vq.Snapshot())
	}))

	select {}
}
