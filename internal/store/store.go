// Package store 는 레벨 진행 상황(잠금 해제·최고 기록·별점)의 영속화를 맡는다.
// 웹 빌드는 localStorage, 데스크톱 빌드는 os.UserConfigDir() 하위 파일로
// (둘 다 store_other.go/store_js.go, 빌드 태그로 구현이 갈린다) 저장하고,
// 직렬화 코덱(이 파일)은 양쪽이 공유한다. go test 하에서는 store_other.go 의
// New() 가 실제 파일을 건드리지 않는 인메모리 구현으로 자동 대체된다.
package store

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

// Store 는 진행 상황 영속화 인터페이스.
// dom_js.go/dom_other.go와 동일하게 빌드 태그로 구현을 분리한다
// (store_js.go: localStorage, store_other.go: 파일 — go test 하에서는 인메모리).
type Store interface {
	Load() map[string]LevelProgress // key = Level.ID
	Save(map[string]LevelProgress)
}

// EncodeProgress/DecodeProgress 는 진행 상황을 수제 텍스트 포맷으로 직렬화한다.
// encoding/json 은 TinyGo wasm 빌드에서 reflection 비용이 커서(89KB→458KB,
// gzip 33KB→170KB 실측) 레벨당 3필드뿐인 이 스키마엔 맞지 않는다.
//
// 포맷: "1-1:1,13,3;3-2:1,0,2"  (ID:Unlocked,BestStrokes,Stars — ';' 로 항목 구분)
func EncodeProgress(m map[string]LevelProgress) string {
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

// DecodeProgress 는 EncodeProgress 의 역변환. 손상된 항목은 건너뛰고 계속
// 진행한다 — 저장소 파싱 실패가 게임 진행을 막으면 안 된다.
func DecodeProgress(s string) map[string]LevelProgress {
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

// DecodeProgressV1JSON 은 옛 encoding/json 포맷
// (`{"1-1":{"Unlocked":true,"BestStrokes":13,"Stars":3}, ...}`)을 파싱하는
// Go 쪽 폴백이다. v1→v2 마이그레이션은 원래 glue.js 가 wasm 로드 전에
// JSON.parse 로 처리하지만, 그게 어떤 이유로든(스크립트 순서 어긋남,
// localStorage 예외 등) 실행되지 못했을 경우에도 기존 플레이어 진행이
// 사라지지 않도록 Go 쪽에서도 같은 v1 데이터를 읽을 수 있어야 한다.
// encoding/json 은 TinyGo 에서 무겁다는 게 이 파일의 전제이므로, 일반
// JSON 파서가 아니라 이 고정된 한 가지 모양만 문자열로 손수 스캔한다.
func DecodeProgressV1JSON(s string) map[string]LevelProgress {
	out := map[string]LevelProgress{}
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "{")
	s = strings.TrimSuffix(s, "}")
	if s == "" {
		return out
	}
	for _, entry := range strings.Split(s, "},") {
		entry = strings.TrimSuffix(entry, "}")
		colon := strings.IndexByte(entry, ':')
		if colon < 0 {
			continue
		}
		id := strings.Trim(entry[:colon], `"`)
		if id == "" {
			continue
		}
		body := strings.TrimPrefix(entry[colon+1:], "{")
		out[id] = LevelProgress{
			Unlocked:    strings.Contains(body, `"Unlocked":true`),
			BestStrokes: extractJSONInt(body, "BestStrokes"),
			Stars:       extractJSONInt(body, "Stars"),
		}
	}
	return out
}

// extractJSONInt 는 `"key":123` 형태에서 123 을 뽑는다(부호 있는 정수만).
func extractJSONInt(s, key string) int {
	marker := `"` + key + `":`
	idx := strings.Index(s, marker)
	if idx < 0 {
		return 0
	}
	rest := s[idx+len(marker):]
	end := 0
	for end < len(rest) && (rest[end] == '-' || (rest[end] >= '0' && rest[end] <= '9')) {
		end++
	}
	n, _ := strconv.Atoi(rest[:end])
	return n
}
