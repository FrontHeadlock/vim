//go:build !js

package store

import (
	"os"
	"path/filepath"
	"testing"
)

// New 는 데스크톱 실행 파일에서 os.UserConfigDir()/vimquest/progress.txt 에
// 저장하는 파일 기반 구현을 반환한다. `go test` 하네스 안에서는
// testing.Testing() 으로 감지해 인메모리 구현으로 대체한다 — 그러지 않으면
// 이 머신에 실제로 남아있는 저장 파일(예: 커밋된 vimquest 바이너리를 플레이해
// 생긴 진행 상황)이 헤드리스 테스트에 섞여 들어가 재현 불가능한 실패를 만든다
// (예: TestInputLevelSelectNavigation 의 "2-1 은 잠김" 가정이 실제 저장 상태에
// 따라 깨질 수 있음). UserConfigDir 조회 실패 시에도 인메모리로 폴백한다 —
// 저장소 문제가 게임 진행 자체를 막으면 안 된다는 이 패키지의 기존 철학과 동일.
func New() Store {
	if testing.Testing() {
		return &memStore{data: map[string]LevelProgress{}}
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return &memStore{data: map[string]LevelProgress{}}
	}
	return &fileStore{path: filepath.Join(dir, "vimquest", "progress.txt")}
}

// memStore 는 헤드리스 테스트용 인메모리 구현. 프로세스 생존 동안만 유지된다.
type memStore struct{ data map[string]LevelProgress }

func (s *memStore) Load() map[string]LevelProgress {
	out := make(map[string]LevelProgress, len(s.data))
	for k, v := range s.data {
		out[k] = v
	}
	return out
}

func (s *memStore) Save(m map[string]LevelProgress) {
	s.data = make(map[string]LevelProgress, len(m))
	for k, v := range m {
		s.data[k] = v
	}
}

// fileStore 는 데스크톱 빌드의 실제 영속화 구현. encodeProgress/decodeProgress
// (store.go 의 수제 텍스트 코덱, 웹의 localStorage 구현과 공유)를 그대로 쓴다.
type fileStore struct{ path string }

func (s *fileStore) Load() map[string]LevelProgress {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return map[string]LevelProgress{} // 최초 실행/삭제됨/읽기 실패 — 빈 진행으로 시작
	}
	return decodeProgress(string(data))
}

func (s *fileStore) Save(m map[string]LevelProgress) {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return // 쓰기 실패는 조용히 무시 — 진행 저장 불가가 플레이를 막으면 안 됨
	}
	_ = os.WriteFile(s.path, []byte(encodeProgress(m)), 0o644)
}
