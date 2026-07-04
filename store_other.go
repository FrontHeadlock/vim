//go:build !js

package main

// memStore 는 데스크톱 빌드/헤드리스 테스트용 인메모리 구현.
// 프로세스 생존 동안만 유지된다(재시작 시 리셋) — dom_other.go 의 no-op 패턴과
// 달리 Save→Load 왕복을 실제로 검증할 수 있도록 값을 보관한다.
type memStore struct{ data map[string]LevelProgress }

func newProgressStore() ProgressStore { return &memStore{data: map[string]LevelProgress{}} }

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
