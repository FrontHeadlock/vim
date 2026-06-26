package main

import "testing"

// step 은 Update 의 이동+진입 처리를 테스트용으로 재현한다.
func (g *Game) step(dx, dy int) {
	g.move(dx, dy)
	g.onEnter()
}

func TestLevelLoad(t *testing.T) {
	g := NewGame()
	if g.levelIdx != 0 {
		t.Fatalf("levelIdx = %d, want 0", g.levelIdx)
	}
	if g.cx != 0 || g.cy != 0 {
		t.Fatalf("cursor = (%d,%d), want (0,0)", g.cx, g.cy)
	}
	if g.keysNeed != 1 {
		t.Fatalf("keysNeed = %d, want 1", g.keysNeed)
	}
	if got := g.buf[0][0]; got != '.' {
		t.Fatalf("start cell = %q, want '.' (@ 는 바닥으로 치환되어야 함)", got)
	}
}

func TestCollectKeyAndClearLevel(t *testing.T) {
	g := NewGame() // 레벨 1-1
	// K 는 (row2, col4). j 두 번, l 네 번.
	g.step(0, 1)
	g.step(0, 1)
	for i := 0; i < 4; i++ {
		g.step(1, 0)
	}
	if g.keys != 1 {
		t.Fatalf("열쇠 획득 실패: keys = %d, want 1", g.keys)
	}
	// $ 는 (row4, col9). j 두 번, l 다섯 번 -> 도달 시 다음 레벨로 전환.
	g.step(0, 1)
	g.step(0, 1)
	for i := 0; i < 5; i++ {
		g.step(1, 0)
	}
	if g.levelIdx != 1 {
		t.Fatalf("출구 도달 후 레벨 전환 실패: levelIdx = %d, want 1", g.levelIdx)
	}
}

func TestExitLockedUntilKeyCollected(t *testing.T) {
	g := NewGame()
	// 열쇠 없이 곧장 $ 로: (0,0) -> (4,9)
	for i := 0; i < 4; i++ {
		g.step(0, 1)
	}
	for i := 0; i < 9; i++ {
		g.step(1, 0)
	}
	if g.levelIdx != 0 {
		t.Fatalf("열쇠 없이 출구가 열렸다: levelIdx = %d, want 0", g.levelIdx)
	}
}

func TestWordForwardJumpsByWord(t *testing.T) {
	g := NewGame()
	g.loadLevel(1) // "@start  the  long  ..." (공백 2칸 구분)
	// 시작은 col0. w 한 번 -> 다음 단어 "the" 시작.
	startCol := g.cx
	g.wordForward()
	if g.cx <= startCol {
		t.Fatalf("w 가 전진하지 않음: %d -> %d", startCol, g.cx)
	}
	// "@start" 는 한 단어이므로 'the' 의 't' 위치(col8)로 가야 한다.
	if g.cx != 8 {
		t.Fatalf("w 착지 col = %d, want 8 (the 의 시작)", g.cx)
	}
}

func TestDeleteBugWithX(t *testing.T) {
	g := NewGame()
	g.loadLevel(2) // 버그 잡기
	before := g.pests
	if before == 0 {
		t.Fatal("레벨 1-3 에 버그가 없음")
	}
	// 첫 버그는 (row0, col3). l 세 번 후 x.
	for i := 0; i < 3; i++ {
		g.step(1, 0)
	}
	if g.cellAt(g.cy, g.cx) != '*' {
		t.Fatalf("커서가 버그 위에 없음: cell = %q", g.cellAt(g.cy, g.cx))
	}
	g.deleteUnder()
	if g.pests != before-1 {
		t.Fatalf("x 로 버그 제거 실패: pests %d -> %d", before, g.pests)
	}
	if g.cellAt(0, 3) != '.' {
		t.Fatalf("제거된 칸이 바닥이 아님: %q", g.cellAt(0, 3))
	}
}
