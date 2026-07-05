package main

import "testing"

// TestProgressCodec 은 encodeProgress/decodeProgress 의 왕복 동일성과
// 손상 입력에 대한 내성을 확인한다(encoding/json 대체 코덱, Phase 4 L2).
func TestProgressCodec(t *testing.T) {
	want := map[string]LevelProgress{
		"1-1": {Unlocked: true, BestStrokes: 13, Stars: 3},
		"3-2": {Unlocked: true, BestStrokes: 0, Stars: 2},
		"8-6": {Unlocked: false, BestStrokes: 0, Stars: 0},
	}
	got := decodeProgress(encodeProgress(want))
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
	got := decodeProgress(encodeProgress(map[string]LevelProgress{}))
	if len(got) != 0 {
		t.Fatalf("빈 맵 왕복 실패: got %v", got)
	}
	if got2 := decodeProgress(""); len(got2) != 0 {
		t.Fatalf("빈 문자열 디코드 실패: got %v", got2)
	}
}

// TestProgressCodecCorrupt 은 손상된 문자열이 파싱 실패로 전체를 무너뜨리지
// 않고, 유효한 항목만 건져 올리는지 확인한다.
func TestProgressCodecCorrupt(t *testing.T) {
	got := decodeProgress("garbage;;1-1:1,13,3;no-colon-here;2-1:1,abc,2;;3-1:0,0,0")
	if len(got) != 2 {
		t.Fatalf("손상 항목 처리 실패: got %v (2개만 남아야 함: 1-1, 3-1)", got)
	}
	if got["1-1"] != (LevelProgress{Unlocked: true, BestStrokes: 13, Stars: 3}) {
		t.Errorf("1-1 파싱 오류: got %+v", got["1-1"])
	}
	if got["3-1"] != (LevelProgress{Unlocked: false, BestStrokes: 0, Stars: 0}) {
		t.Errorf("3-1 파싱 오류: got %+v", got["3-1"])
	}
	if _, ok := got["2-1"]; ok {
		t.Errorf("BestStrokes 파싱 실패해야 할 2-1 이 포함됨: %+v", got["2-1"])
	}
}
