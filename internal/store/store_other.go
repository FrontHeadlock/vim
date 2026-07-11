//go:build !js

package store

import (
	"os"
	"path/filepath"
)

// New 는 데스크톱 실행 파일에서 os.UserConfigDir()/vimquest/progress.txt 에
// 저장하는 파일 기반 구현을 반환한다. 호출자는 컴포지션 루트(cmd/desktop)
// 하나뿐이다 — 테스트는 이 함수를 부르지 않고 NewMem() 을 game.New 에 직접
// 주입하므로(예전의 testing.Testing() 암묵 감지를 대체), 실제 저장 파일이
// 헤드리스 테스트에 섞여 들어갈 경로 자체가 없다. UserConfigDir 조회 실패
// 시에는 인메모리로 폴백한다 — 저장소 문제가 게임 진행 자체를 막으면 안
// 된다는 이 패키지의 기존 철학과 동일.
func New() Store {
	dir, err := os.UserConfigDir()
	if err != nil {
		return NewMem()
	}
	return &fileStore{path: filepath.Join(dir, "vimquest", "progress.txt")}
}

// NewFileStoreAt 은 지정 경로에 저장하는 파일 스토어를 만든다 — 테스트가
// 임시 디렉토리를 주입해 Save→Load 왕복·손상 파일 내성을 검증하는 용도.
// 프로덕션 경로 결정은 New() 가 한다.
func NewFileStoreAt(path string) Store { return &fileStore{path: path} }

// fileStore 는 데스크톱 빌드의 실제 영속화 구현. EncodeProgress/DecodeProgress
// (store.go 의 수제 텍스트 코덱, 웹의 localStorage 구현과 공유)를 그대로 쓴다.
type fileStore struct{ path string }

func (s *fileStore) Load() map[string]LevelProgress {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return map[string]LevelProgress{} // 최초 실행/삭제됨/읽기 실패 — 빈 진행으로 시작
	}
	return DecodeProgress(string(data))
}

func (s *fileStore) Save(m map[string]LevelProgress) {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return // 쓰기 실패는 조용히 무시 — 진행 저장 불가가 플레이를 막으면 안 됨
	}
	_ = os.WriteFile(s.path, []byte(EncodeProgress(m)), 0o644)
}
