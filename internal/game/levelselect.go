package game

// levelselect.go — 레벨 선택 화면의 커서 이동/입장 규칙. 상태 전이의
// 진입(EnterLevelSelect)과 화면 내 입력(inputLevelSelect)만 소유한다 —
// excommand.go 와 같은 "하위 상태 머신은 자기 파일을 갖는다" 결.

import "vimquest/internal/engine"

// inputLevelSelect 는 레벨 선택 화면에서의 키 입력을 처리한다.
// h/l = 월드 이동, j/k = 레벨 이동, cr = 입장(잠금 시 무시), esc = 복귀.
func (g *Game) inputLevelSelect(k engine.Key) {
	worlds := WorldGroups()
	if len(worlds) == 0 {
		return
	}
	if g.selWorld >= len(worlds) {
		g.selWorld = len(worlds) - 1
	}
	if g.selLevel >= len(worlds[g.selWorld]) {
		g.selLevel = len(worlds[g.selWorld]) - 1
	}

	switch k.R {
	case 'h':
		if g.selWorld > 0 {
			g.selWorld--
			if g.selLevel >= len(worlds[g.selWorld]) {
				g.selLevel = len(worlds[g.selWorld]) - 1
			}
		}
	case 'l':
		if g.selWorld < len(worlds)-1 {
			g.selWorld++
			if g.selLevel >= len(worlds[g.selWorld]) {
				g.selLevel = len(worlds[g.selWorld]) - 1
			}
		}
	case 'j':
		if g.selLevel < len(worlds[g.selWorld])-1 {
			g.selLevel++
		}
	case 'k':
		if g.selLevel > 0 {
			g.selLevel--
		}
	}
	if k.S == "cr" {
		idx := worlds[g.selWorld][g.selLevel]
		if g.progress[levels[idx].ID].Unlocked {
			g.LoadLevel(idx)
		}
	}
	if k.S == "esc" {
		g.state = StatePlaying
	}
}

// EnterLevelSelect 는 레벨 선택 화면으로 전환하며 커서를 현재 레벨 위치로 맞춘다.
func (g *Game) EnterLevelSelect() {
	g.state = StateLevelSelect
	g.selWorld, g.selLevel = 0, 0
	for wi, group := range WorldGroups() {
		for li, idx := range group {
			if idx == g.levelIdx {
				g.selWorld, g.selLevel = wi, li
			}
		}
	}
}
