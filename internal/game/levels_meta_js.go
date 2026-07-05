//go:build js

package game

// 웹(wasm) 빌드에는 커리큘럼 표시 데이터가 없다 — 같은 내용을 tools/genmeta
// 가 web/src/levels_meta.js 로 생성해 JS 쪽이 직접 읽는다(레벨 ID 로 조회).
// 표시 문자열을 wasm 에 이중으로 싣지 않기 위한 스텁이다.
func MetaFor(id string) LevelMeta { return LevelMeta{} }
