//go:build js

package main

import "syscall/js"

// vimquest.v1 → v2 마이그레이션은 wasm 로드 전 JS(glue.js)가 네이티브
// JSON.parse 로 처리한다(Go 바이너리에 encoding/json 을 다시 들이지 않기 위함).
// 여기서는 이미 v2 포맷으로 정리된 값만 다룬다.
const storeKey = "vimquest.v2"

type jsStore struct{}

func newProgressStore() ProgressStore { return jsStore{} }

func (jsStore) Load() map[string]LevelProgress {
	ls := js.Global().Get("localStorage")
	if ls.IsUndefined() {
		return map[string]LevelProgress{}
	}
	raw := ls.Call("getItem", storeKey)
	if raw.IsNull() || raw.IsUndefined() {
		return map[string]LevelProgress{}
	}
	return decodeProgress(raw.String())
}

func (jsStore) Save(m map[string]LevelProgress) {
	ls := js.Global().Get("localStorage")
	if ls.IsUndefined() {
		return
	}
	ls.Call("setItem", storeKey, encodeProgress(m))
}
