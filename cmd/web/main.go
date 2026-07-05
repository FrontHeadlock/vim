//go:build js

// VimQuest 웹 프론트엔드 진입점 — TinyGo 로 wasm 컴파일된다(build.sh).
//
// 의도적으로 함수 호출 하나뿐이다. 브리지 본체가 internal/jsbridge 에 있는
// 이유(TinyGo 링커의 main-패키지 경계 비용)는 그쪽 패키지 주석 참고.
package main

import "vimquest/internal/jsbridge"

func main() {
	jsbridge.Run()
}
