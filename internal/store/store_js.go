//go:build js

package store

import "syscall/js"

// vimquest.v1 → v2 마이그레이션은 우선 wasm 로드 전 JS(glue.js)가 네이티브
// JSON.parse 로 처리한다(Go 바이너리에 encoding/json 을 다시 들이지 않기 위함).
// 그게 어떤 이유로든(스크립트 순서 어긋남, localStorage 예외 등) 실행되지
// 못했을 때를 대비해, v2 가 비어 있으면 Load() 가 v1 을 직접 읽어보는
// 폴백도 둔다 — 그래야 마이그레이션 실패가 기존 플레이어 진행을 조용히
// 지워버리지 않는다.
const storeKey = "vimquest.v2"
const legacyStoreKey = "vimquest.v1"

type jsStore struct{}

func New() Store { return jsStore{} }

func (jsStore) Load() map[string]LevelProgress {
	ls := js.Global().Get("localStorage")
	if ls.IsUndefined() {
		return map[string]LevelProgress{}
	}
	raw := ls.Call("getItem", storeKey)
	if !raw.IsNull() && !raw.IsUndefined() {
		return DecodeProgress(raw.String())
	}

	// v2 가 없다 — v1(구 JSON 포맷)이 아직 남아 있는지 폴백으로 확인한다.
	v1 := ls.Call("getItem", legacyStoreKey)
	if v1.IsNull() || v1.IsUndefined() {
		return map[string]LevelProgress{}
	}
	m := DecodeProgressV1JSON(v1.String())
	if len(m) > 0 {
		ls.Call("setItem", storeKey, EncodeProgress(m))
		ls.Call("removeItem", legacyStoreKey)
	}
	return m
}

func (jsStore) Save(m map[string]LevelProgress) {
	ls := js.Global().Get("localStorage")
	if ls.IsUndefined() {
		return
	}
	ls.Call("setItem", storeKey, EncodeProgress(m))
}
