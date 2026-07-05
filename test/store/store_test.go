// Package storetest 는 진행 저장(코덱·파일 스토어)의 블랙박스 테스트다.
// 코덱(EncodeProgress/DecodeProgress)은 localStorage(glue.js 마이그레이션)와
// 파일 저장이 공유하는 직렬화 계약이라 공개 API 다.
package storetest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"vimquest/internal/store"
)

// TestProgressCodec 은 EncodeProgress/DecodeProgress 의 왕복 동일성과
// 손상 입력에 대한 내성을 확인한다(encoding/json 대체 코덱).
func TestProgressCodec(t *testing.T) {
	want := map[string]store.LevelProgress{
		"1-1": {Unlocked: true, BestStrokes: 13, Stars: 3},
		"3-2": {Unlocked: true, BestStrokes: 0, Stars: 2},
		"8-6": {Unlocked: false, BestStrokes: 0, Stars: 0},
	}
	got := store.DecodeProgress(store.EncodeProgress(want))
	if len(got) != len(want) {
		t.Fatalf("길이 불일치: got %d want %d (got=%v)", len(got), len(want), got)
	}
	for id, p := range want {
		if got[id] != p {
			t.Errorf("[%s] got %+v want %+v", id, got[id], p)
		}
	}
}

func TestProgressCodecEmpty(t *testing.T) {
	got := store.DecodeProgress(store.EncodeProgress(map[string]store.LevelProgress{}))
	if len(got) != 0 {
		t.Fatalf("빈 맵 왕복 실패: got %v", got)
	}
	if got2 := store.DecodeProgress(""); len(got2) != 0 {
		t.Fatalf("빈 문자열 디코드 실패: got %v", got2)
	}
}

// TestProgressCodecCorrupt 은 손상된 문자열이 파싱 실패로 전체를 무너뜨리지
// 않고, 유효한 항목만 건져 올리는지 확인한다.
func TestProgressCodecCorrupt(t *testing.T) {
	got := store.DecodeProgress("garbage;;1-1:1,13,3;no-colon-here;2-1:1,abc,2;;3-1:0,0,0")
	if len(got) != 2 {
		t.Fatalf("손상 항목 처리 실패: got %v (2개만 남아야 함: 1-1, 3-1)", got)
	}
	if got["1-1"] != (store.LevelProgress{Unlocked: true, BestStrokes: 13, Stars: 3}) {
		t.Errorf("1-1 파싱 오류: got %+v", got["1-1"])
	}
	if got["3-1"] != (store.LevelProgress{Unlocked: false, BestStrokes: 0, Stars: 0}) {
		t.Errorf("3-1 파싱 오류: got %+v", got["3-1"])
	}
	if _, ok := got["2-1"]; ok {
		t.Errorf("BestStrokes 파싱 실패해야 할 2-1 이 포함됨: %+v", got["2-1"])
	}
}

// TestDecodeProgressV1JSON 은 실제 encoding/json.Marshal 이 만든 옛 v1 포맷을
// DecodeProgressV1JSON(수제 폴백 파서)이 정확히 읽어내는지 확인한다 — Go 쪽
// 마이그레이션 폴백(store_js.go)이 실제 마샬 결과와 어긋나지 않는다는 보증.
func TestDecodeProgressV1JSON(t *testing.T) {
	want := map[string]store.LevelProgress{
		"1-1": {Unlocked: true, BestStrokes: 13, Stars: 3},
		"3-2": {Unlocked: true, BestStrokes: 0, Stars: 2},
		"8-6": {Unlocked: false, BestStrokes: 0, Stars: 0},
	}
	raw, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	got := store.DecodeProgressV1JSON(string(raw))
	if len(got) != len(want) {
		t.Fatalf("길이 불일치: got %d want %d (raw=%s got=%v)", len(got), len(want), raw, got)
	}
	for id, p := range want {
		if got[id] != p {
			t.Errorf("[%s] got %+v want %+v (raw=%s)", id, got[id], p, raw)
		}
	}
}

func TestDecodeProgressV1JSONEmpty(t *testing.T) {
	if got := store.DecodeProgressV1JSON("{}"); len(got) != 0 {
		t.Fatalf("빈 객체 디코드 실패: got %v", got)
	}
	if got := store.DecodeProgressV1JSON(""); len(got) != 0 {
		t.Fatalf("빈 문자열 디코드 실패: got %v", got)
	}
}

// TestFileStoreRoundTrip 은 데스크톱 파일 저장의 Save→Load 왕복을 임시
// 디렉토리에 주입해 확인한다(중간 디렉토리가 없어도 Save 가 만들어야 함).
func TestFileStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vimquest", "progress.txt")
	want := map[string]store.LevelProgress{
		"1-1": {Unlocked: true, BestStrokes: 13, Stars: 3},
		"3-2": {Unlocked: true, BestStrokes: 0, Stars: 2},
	}

	s := store.NewFileStoreAt(path)
	s.Save(want)

	s2 := store.NewFileStoreAt(path)
	got := s2.Load()
	if len(got) != len(want) {
		t.Fatalf("길이 불일치: got %d want %d (got=%v)", len(got), len(want), got)
	}
	for id, p := range want {
		if got[id] != p {
			t.Errorf("[%s] got %+v want %+v", id, got[id], p)
		}
	}
}

// TestFileStoreLoadMissingIsEmpty 는 저장 파일이 아직 없을 때 Load 가 빈
// 진행을 돌려주는지(에러로 죽지 않는지) 확인한다 — 최초 실행 시나리오.
func TestFileStoreLoadMissingIsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope", "progress.txt")
	if got := store.NewFileStoreAt(path).Load(); len(got) != 0 {
		t.Fatalf("존재하지 않는 파일 로드가 빈 맵이 아님: got %v", got)
	}
}

// TestFileStoreCorruptFileSalvages 는 손상된 저장 파일에서 유효 항목만
// 건져 올리는지 파일 경로로 확인한다(DecodeProgress 의 내성이 실제 파일
// 로드 경로에서도 동작하는지).
func TestFileStoreCorruptFileSalvages(t *testing.T) {
	path := filepath.Join(t.TempDir(), "progress.txt")
	if err := os.WriteFile(path, []byte("garbage;;1-1:1,13,3;broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := store.NewFileStoreAt(path).Load()
	if len(got) != 1 || got["1-1"].BestStrokes != 13 {
		t.Fatalf("손상 파일에서 유효 항목 복구 실패: got %v", got)
	}
}

// TestNewIsIsolatedUnderTest 는 go test 하에서 New() 가 실제 저장 파일을
// 건드리지 않는 인메모리 구현을 돌려주는지 행동으로 확인한다: 인스턴스 간
// 상태가 공유되지 않아야 한다(파일 스토어라면 Save 가 다음 New().Load() 에
// 보인다). 이게 깨지면 모든 헤드리스 테스트가 이 머신의 실제 저장 상태에
// 의존하게 된다.
func TestNewIsIsolatedUnderTest(t *testing.T) {
	s1 := store.New()
	s1.Save(map[string]store.LevelProgress{"1-1": {Unlocked: true, BestStrokes: 5, Stars: 3}})
	if got := store.New().Load(); len(got) != 0 {
		t.Fatalf("go test 하에서 New() 인스턴스 간 상태가 공유됨(파일 스토어로 의심): %v", got)
	}
}
