//go:build js

// Package platform 은 게임 로직이 바깥 세상(브라우저 DOM·효과음)에 말을 거는
// 유일한 통로다. 웹 빌드는 syscall/js 로 실제 DOM 을 만지고, 데스크톱 빌드는
// no-op 이다(dom_other.go) — 게임 규칙 코드는 어느 쪽에서 도는지 몰라도 된다.
package platform

import "syscall/js"

// lastValues 는 SetText/SetHTML 이 마지막으로 쓴 값을 id 별로 캐시해, 같은
// 값이면 DOM 을 다시 건드리지 않는다(dedupe). 게임의 DOM 동기화가 매 입력마다
// 호출되므로 이 가드가 없으면 #hint 의 textContent 가 값이 같아도 매번
// 재설정되어 MutationObserver 기반 타자기 효과가 계속 리셋된다.
var lastValues = map[string]string{}

// SetText 는 id 에 해당하는 HTML 요소의 textContent 를 설정한다.
// 한국어 문자열은 JS 문자열로 그대로 전달되므로 캔버스 폰트 임베드가 필요 없다.
func SetText(id, value string) {
	if lastValues[id] == value {
		return
	}
	lastValues[id] = value
	if el := elementByID(id); !el.IsNull() {
		el.Set("textContent", value)
	}
}

// SetHTML 은 id 요소의 innerHTML 을 설정한다(우리가 만든 안전한 마크업 전용).
func SetHTML(id, html string) {
	cacheKey := "html:" + id
	if lastValues[cacheKey] == html {
		return
	}
	lastValues[cacheKey] = html
	if el := elementByID(id); !el.IsNull() {
		el.Set("innerHTML", html)
	}
}

// ShowOverlay 는 id 요소의 "hidden" 클래스를 제거해 오버레이를 다시 연다(:help 용).
func ShowOverlay(id string) {
	if el := elementByID(id); !el.IsNull() {
		el.Get("classList").Call("remove", "hidden")
	}
}

// Sfx 는 JS 쪽 인라인 신스(window.vimquestSfx)를 호출해 짧은 칩튠 효과음을 재생한다.
// name: "key"(열쇠 획득) · "bug"(버그 처치) · "blocked"(막힌 키) · "clear"(레벨 클리어).
func Sfx(name string) {
	fn := js.Global().Get("vimquestSfx")
	if fn.IsUndefined() {
		return
	}
	fn.Invoke(name)
}

// elementByID 는 document.getElementById 를 감싼다. document 가 없거나 요소가
// 없으면 js.Null() 을 돌려줘 호출부가 한 번만 검사하게 한다.
func elementByID(id string) js.Value {
	doc := js.Global().Get("document")
	if doc.IsUndefined() {
		return js.Null()
	}
	el := doc.Call("getElementById", id)
	if el.IsUndefined() {
		return js.Null()
	}
	return el
}
