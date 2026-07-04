//go:build js

package main

import (
	"encoding/json"
	"syscall/js"
)

const storeKey = "vimquest.v1"

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
	var out map[string]LevelProgress
	if err := json.Unmarshal([]byte(raw.String()), &out); err != nil {
		return map[string]LevelProgress{}
	}
	return out
}

func (jsStore) Save(m map[string]LevelProgress) {
	ls := js.Global().Get("localStorage")
	if ls.IsUndefined() {
		return
	}
	b, err := json.Marshal(m)
	if err != nil {
		return
	}
	ls.Call("setItem", storeKey, string(b))
}
