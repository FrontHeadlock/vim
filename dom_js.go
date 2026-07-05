//go:build js

package main

import "syscall/js"

// lastDOMValues 는 domSet 이 마지막으로 쓴 값을 id 별로 캐시해, 같은 값이면
// DOM 을 다시 건드리지 않는다(dedupe). syncDOM() 이 매 프레임 호출되므로 이
// 가드가 없으면 #hint 의 textContent 가 매 프레임 재설정되어(값이 같아도)
// MutationObserver 기반 타자기 효과(2.4)가 계속 리셋된다.
var lastDOMValues = map[string]string{}

// domSet 은 id 에 해당하는 HTML 요소의 textContent 를 설정한다.
// 한국어 문자열은 JS 문자열로 그대로 전달되므로 캔버스 폰트 임베드가 필요 없다.
func domSet(id, value string) {
	if lastDOMValues[id] == value {
		return
	}
	lastDOMValues[id] = value
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
// domSet 과 동일하게 dedupe 캐시를 적용해 매 프레임 불필요한 DOM 갱신을 피한다.
func domSetHTML(id, html string) {
	cacheKey := "html:" + id
	if lastDOMValues[cacheKey] == html {
		return
	}
	lastDOMValues[cacheKey] = html
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

// domShow 는 id 요소의 "hidden" 클래스를 제거해 오버레이를 다시 연다(:help 용).
func domShow(id string) {
	doc := js.Global().Get("document")
	if doc.IsUndefined() {
		return
	}
	el := doc.Call("getElementById", id)
	if el.IsNull() || el.IsUndefined() {
		return
	}
	el.Get("classList").Call("remove", "hidden")
}

// jsSfx 는 JS 쪽 인라인 신스(window.vimquestSfx)를 호출해 짧은 칩튠 효과음을 재생한다.
// name: "key"(열쇠 획득) · "bug"(버그 처치) · "blocked"(막힌 키) · "clear"(레벨 클리어).
func jsSfx(name string) {
	fn := js.Global().Get("vimquestSfx")
	if fn.IsUndefined() {
		return
	}
	fn.Invoke(name)
}

// vimquestReset/Restart/LevelSelect 버튼 훅은 web_js.go 가 Game 을 직접 호출하는
// 형태로 노출한다(Phase 4 L2 전에는 request* 폴링 플래그를 거쳤다 — 이제
// 이벤트 구동이라 폴링이 필요 없다).
