//go:build js

package main

import "syscall/js"

// domSet 은 id 에 해당하는 HTML 요소의 textContent 를 설정한다.
// 한국어 문자열은 JS 문자열로 그대로 전달되므로 캔버스 폰트 임베드가 필요 없다.
func domSet(id, value string) {
	doc := js.Global().Get("document")
	if doc.IsUndefined() {
		return
	}
	el := doc.Call("getElementById", id)
	if el.IsNull() || el.IsUndefined() {
		return
	}
	el.Set("textContent", value)
}

// domSetHTML 은 id 요소의 innerHTML 을 설정한다(우리가 만든 안전한 마크업 전용).
func domSetHTML(id, html string) {
	doc := js.Global().Get("document")
	if doc.IsUndefined() {
		return
	}
	el := doc.Call("getElementById", id)
	if el.IsNull() || el.IsUndefined() {
		return
	}
	el.Set("innerHTML", html)
}

// registerJSHooks 는 HTML 버튼이 호출할 전역 함수를 노출한다.
func registerJSHooks() {
	js.Global().Set("vimquestReset", js.FuncOf(func(js.Value, []js.Value) any {
		requestReset()
		return nil
	}))
	js.Global().Set("vimquestRestart", js.FuncOf(func(js.Value, []js.Value) any {
		requestRestart()
		return nil
	}))
}
