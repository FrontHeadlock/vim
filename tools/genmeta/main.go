// genmeta 는 커리큘럼 레벨의 표시 데이터(levels_meta.go — !js 빌드 태그)를
// 웹 렌더러가 읽는 JS 테이블(web/src/levels_meta.js)로 변환해 stdout 에 쓴다.
// 빌드마다 scripts/build.sh 가 실행하므로 두 소스가 어긋날 수 없다 — 단일
// 진실은 levels_meta.go 하나다. encoding/json 은 이 도구(호스트 Go)에서만
// 쓰이고 wasm 페이로드와 무관하다.
//
// 사용: go run ./tools/genmeta > web/src/levels_meta.js
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"vimquest/internal/game"
)

func main() {
	type jsCmd struct {
		K string `json:"k"`
		D string `json:"d"`
	}
	type jsMeta struct {
		Title string  `json:"title"`
		Hint  string  `json:"hint"`
		Cmds  []jsCmd `json:"cmds"`
	}

	meta := make(map[string]jsMeta, game.LevelCount())
	for i := 0; i < game.LevelCount(); i++ {
		lv := game.LevelAt(i)
		m := game.MetaFor(lv.ID)
		if m.Title == "" {
			fmt.Fprintf(os.Stderr, "genmeta: 레벨 %s 의 메타가 비어 있음(levels_meta.go 등록 누락)\n", lv.ID)
			os.Exit(1)
		}
		cmds := make([]jsCmd, len(m.Cmds))
		for j, c := range m.Cmds {
			cmds[j] = jsCmd{K: c.K, D: c.D}
		}
		meta[lv.ID] = jsMeta{Title: m.Title, Hint: m.Hint, Cmds: cmds}
	}

	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "genmeta:", err)
		os.Exit(1)
	}
	fmt.Println("// 생성 파일 — 직접 편집 금지. 원본: internal/game/levels_meta.go")
	fmt.Println("// 재생성: go run ./tools/genmeta > web/src/levels_meta.js (build.sh 가 자동 실행)")
	fmt.Println("'use strict';")
	fmt.Printf("const LEVEL_META = %s;\n", b)
}
