//go:build js

// Package platform 은 게임 로직이 바깥 세상(브라우저 오버레이·효과음)에 말을
// 거는 유일한 통로다. 웹 빌드는 syscall/js 로 실제 DOM 을 만지고, 데스크톱
// 빌드는 no-op 이다(dom_other.go) — 게임 규칙 코드는 어느 쪽에서 도는지
// 몰라도 된다.
//
// 예전엔 SetText/SetHTML 로 사이드패널(#hint 등)도 여기서 동기화했지만,
// 커리큘럼 표시 데이터를 wasm 에 싣지 않는 구조로 바꾸면서(levels_meta.go →
// tools/genmeta → levels_meta.js) 패널 동기화는 JS(glue.js vqUpdatePanel)가
// 스냅샷을 읽어 직접 한다. 남은 책임은 오버레이 열기와 효과음뿐이다.
package platform

import "syscall/js"

// ShowOverlay 는 id 요소의 "hidden" 클래스를 제거해 오버레이를 다시 연다(:help 용).
func ShowOverlay(id string) {
	doc := js.Global().Get("document")
	if doc.IsUndefined() {
		return
	}
	el := doc.Call("getElementById", id)
	if el.IsUndefined() || el.IsNull() {
		return
	}
	el.Get("classList").Call("remove", "hidden")
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
