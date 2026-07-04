package main

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
