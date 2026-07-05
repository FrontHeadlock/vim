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

	select {}
}
