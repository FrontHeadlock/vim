package main

import (
	"strconv"
	"strings"
)

// LevelProgress 는 레벨 하나의 진행 상태.
type LevelProgress struct {
	Unlocked    bool
	BestStrokes int // 0 = 아직 클리어 안 함
	Stars       int // 0~3
}

// ProgressStore 는 진행 상황 영속화 인터페이스.
// dom_js.go/dom_other.go와 동일하게 빌드 태그로 구현을 분리한다
// (store_js.go: localStorage, store_other.go: 인메모리).
type ProgressStore interface {
	Load() map[string]LevelProgress // key = Level.ID
	Save(map[string]LevelProgress)
}

// encodeProgress/decodeProgress 는 진행 상황을 수제 텍스트 포맷으로 직렬화한다.
// encoding/json 은 TinyGo wasm 빌드에서 reflection 비용이 커서(89KB→458KB,
// gzip 33KB→170KB 실측) 레벨당 3필드뿐인 이 스키마엔 맞지 않는다.
//
// 포맷: "1-1:1,13,3;3-2:1,0,2"  (ID:Unlocked,BestStrokes,Stars — ';' 로 항목 구분)
func encodeProgress(m map[string]LevelProgress) string {
	var b strings.Builder
	first := true
	for id, p := range m {
		if !first {
			b.WriteByte(';')
		}
		first = false
		b.WriteString(id)
		b.WriteByte(':')
		if p.Unlocked {
			b.WriteByte('1')
		} else {
			b.WriteByte('0')
		}
		b.WriteByte(',')
		b.WriteString(strconv.Itoa(p.BestStrokes))
		b.WriteByte(',')
		b.WriteString(strconv.Itoa(p.Stars))
	}
	return b.String()
}

// decodeProgress 는 encodeProgress 의 역변환. 손상된 항목은 건너뛰고 계속
// 진행한다 — 저장소 파싱 실패가 게임 진행을 막으면 안 된다.
func decodeProgress(s string) map[string]LevelProgress {
	out := map[string]LevelProgress{}
	if s == "" {
		return out
	}
	for _, entry := range strings.Split(s, ";") {
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 || parts[0] == "" {
			continue
		}
		id, fields := parts[0], strings.Split(parts[1], ",")
		if len(fields) != 3 {
			continue
		}
		best, errB := strconv.Atoi(fields[1])
		st, errS := strconv.Atoi(fields[2])
		if errB != nil || errS != nil {
			continue
		}
		out[id] = LevelProgress{
			Unlocked:    fields[0] == "1",
			BestStrokes: best,
			Stars:       st,
		}
	}
	return out
}
