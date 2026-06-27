package main

import (
	"strings"
	"testing"
)

// TestEditLevelsSolvable 은 모든 edit 레벨이 의도된 Solution 으로 Target 에
// 정확히 도달하는지 검증한다 — 풀이 불가능한 퍼즐을 출시 전에 잡아낸다.
func TestEditLevelsSolvable(t *testing.T) {
	for _, lv := range levels {
		if lv.Kind != "edit" {
			continue
		}
		e := NewEditor(append([]string(nil), lv.Map...))
		feedKeys(e, lv.Solution)
		got := strings.Join(e.Lines(), "\n")
		want := strings.Join(lv.Target, "\n")
		if got != want {
			t.Errorf("[%s] Solution %q\n  got:  %q\n  want: %q", lv.Title, lv.Solution, got, want)
		}
	}
}

// TestNavigateLevelsValid 은 navigate 레벨의 맵 구조를 검증한다.
func TestNavigateLevelsValid(t *testing.T) {
	for _, lv := range levels {
		if lv.Kind != "navigate" {
			continue
		}
		ats, dollars := 0, 0
		for _, row := range lv.Map {
			ats += strings.Count(row, "@")
			dollars += strings.Count(row, "$")
		}
		if ats != 1 {
			t.Errorf("[%s] '@' 시작 위치가 %d개 (1개여야 함)", lv.Title, ats)
		}
		if dollars < 1 {
			t.Errorf("[%s] '$' 출구가 없음", lv.Title)
		}
	}
}

// playNav 는 navigate 레벨에서 키를 누르고 매 입력 후 승리 판정을 돌린다.
func playNav(g *Game, keys string) {
	for _, r := range keys {
		g.feed(RuneKey(r))
		g.checkWin()
	}
}

// TestNavigateSolveLevel1 은 1-1 을 이동만으로 클리어해 다음 레벨로 가는지 본다.
func TestNavigateSolveLevel1(t *testing.T) {
	g := NewGame()
	if g.lv.Kind != "navigate" {
		t.Fatal("레벨 1-1 이 navigate 가 아님")
	}
	playNav(g, "jjllll") // 열쇠(row2,col4) 획득
	if len(g.keyPos) != 0 {
		t.Fatalf("열쇠 미획득: 남은 %d", len(g.keyPos))
	}
	playNav(g, "jjlllll") // 출구(row4,col9) 도달 → 다음 레벨
	if g.levelIdx != 1 {
		t.Fatalf("출구 도달 후 레벨 전환 실패: levelIdx=%d want 1", g.levelIdx)
	}
}

// TestNavigateBlocksEditing 은 navigate 레벨에서 편집키가 막히는지 확인한다.
func TestNavigateBlocksEditing(t *testing.T) {
	g := NewGame() // 1-1
	before := strings.Join(g.ed.Lines(), "\n")
	g.feed(RuneKey('d'))
	g.feed(RuneKey('d')) // dd 시도 — 막혀야 함
	after := strings.Join(g.ed.Lines(), "\n")
	if before != after {
		t.Errorf("navigate 레벨에서 편집이 허용됨:\n  before %q\n  after  %q", before, after)
	}
}
