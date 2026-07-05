//go:build !js

package store

import (
	"path/filepath"
	"testing"
)

// TestFileStoreRoundTrip 은 데스크톱 파일 저장의 Save→Load 왕복을 임시
// 디렉토리에 주입해 확인한다(중간 디렉토리가 없어도 Save 가 만들어야 함).
func TestFileStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vimquest", "progress.txt")
	want := map[string]LevelProgress{
		"1-1": {Unlocked: true, BestStrokes: 13, Stars: 3},
		"3-2": {Unlocked: true, BestStrokes: 0, Stars: 2},
	}

	s := &fileStore{path: path}
	s.Save(want)

	s2 := &fileStore{path: path}
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
	s := &fileStore{path: path}
	got := s.Load()
	if len(got) != 0 {
		t.Fatalf("존재하지 않는 파일 로드가 빈 맵이 아님: got %v", got)
	}
}

// TestNewReturnsMemStoreUnderTest 는 testing.Testing() 감지로 go test 하에서
// New() 가 실제 파일을 건드리지 않는 인메모리 구현을 반환하는지 확인한다 —
// 이게 깨지면 이후 모든 헤드리스 테스트가 이 머신의 실제 저장 파일 상태에
// 의존하게 된다.
func TestNewReturnsMemStoreUnderTest(t *testing.T) {
	s := New()
	if _, ok := s.(*memStore); !ok {
		t.Fatalf("go test 하에서 New() 가 memStore 가 아님: %T", s)
	}
}
