package gametest

// arena_test.go — Arena 시간공격 모드의 게임 규칙 검증. 시간 측정·제출은
// 전부 JS 몫이라 여기서 다루지 않는다 — Go 쪽 보증은 (1) 고정 5문제가 전부
// Solution 으로 풀리고, (2) Arena 진행이 커리큘럼 진행(잠금/별점/저장)을
// 절대 건드리지 않으며, (3) 문제가 1→5→완주로 정확히 전진한다는 세 가지다.

import (
	"reflect"
	"strings"
	"testing"

	"vimquest/internal/engine"
	. "vimquest/internal/game"
)

// arenaSnap 은 스냅샷의 arena 블록(num/count)을 읽는다.
func arenaSnap(t *testing.T, g *Game) map[string]any {
	t.Helper()
	st := g.Snapshot()
	blk, ok := st["arena"].(map[string]any)
	if !ok {
		t.Fatalf("snapshot에 arena 블록이 없음 (state=%v)", st["state"])
	}
	return blk
}

// TestArenaLevelsSolvable 은 ArenaLevels 전부가 자신의 Solution 으로 Target
// 에 도달하는지 검증한다(TestEditLevelsSolvable 과 동일 패턴). 덤으로 저작
// 규칙 — 전부 edit, ID 는 서로/커리큘럼과 안 겹침 — 도 고정한다.
func TestArenaLevelsSolvable(t *testing.T) {
	engine.ResetMultilineCharwiseFallbackCount()
	seen := map[string]bool{}
	for _, lv := range ArenaLevels {
		if lv.Kind != "edit" {
			t.Errorf("[%s] Kind=%q — Arena 는 전부 edit 이어야 함", lv.ID, lv.Kind)
		}
		if seen[lv.ID] {
			t.Errorf("[%s] ID 중복", lv.ID)
		}
		seen[lv.ID] = true
		if levelIndexByID(lv.ID) != -1 {
			t.Errorf("[%s] 커리큘럼 레벨과 ID 충돌", lv.ID)
		}
		e := engine.NewEditor(append([]string(nil), lv.Map...))
		feedKeys(e, lv.Solution)
		got := strings.Join(e.Lines(), "\n")
		want := strings.Join(lv.Target, "\n")
		if got != want {
			t.Errorf("[%s] Solution %q\n  got:  %q\n  want: %q", lv.ID, lv.Solution, got, want)
		}
	}
	if n := engine.MultilineCharwiseFallbackCount(); n != 0 {
		t.Errorf("Arena Solution 이 여러 줄 charwise 대체 경로를 %d회 밟음", n)
	}
}

// TestArenaRunLeavesProgressUntouched 는 Arena 5문제 완주 전후로 레벨 선택
// 스냅샷의 worlds(unlocked/stars)가 완전히 동일함을 확인한다 — advanceArena
// 가 recordClear()/store.Save() 를 절대 부르지 않는다는 이 설계의 핵심
// 회귀 가드다(LoadCustomLevel+advance() 경로였다면 여기서 바로 깨진다).
func TestArenaRunLeavesProgressUntouched(t *testing.T) {
	g := New()
	g.EnterLevelSelect()
	before := g.Snapshot()["worlds"]

	g.EnterArena()
	for _, lv := range ArenaLevels {
		playKeys(g, lv.Solution)
	}
	if st := g.Snapshot()["state"]; st != "arenaDone" {
		t.Fatalf("완주 후 state=%v, want arenaDone", st)
	}

	g.EnterLevelSelect()
	after := g.Snapshot()["worlds"]
	if !reflect.DeepEqual(before, after) {
		t.Errorf("Arena 완주가 커리큘럼 진행을 바꿈:\n before: %v\n after:  %v", before, after)
	}
}

// TestArenaAdvanceProgression 은 문제 번호가 1→5 로 전진하고 마지막에
// arenaDone 이 되는지, 도중 RESET(RestartCurrent)이 문제를 건너뛰지 않는지,
// 완주 화면에선 키 입력이 삼켜지는지 확인한다.
func TestArenaAdvanceProgression(t *testing.T) {
	g := New()
	g.EnterArena()

	if st := g.Snapshot()["state"]; st != "arena" {
		t.Fatalf("EnterArena 직후 state=%v, want arena", st)
	}
	if num := arenaSnap(t, g)["num"]; num != 1 {
		t.Fatalf("시작 문제 번호=%v, want 1", num)
	}

	// 1번 문제 도중 RESET — 같은 문제가 strokes=0 으로 다시 시작돼야 한다.
	playKeys(g, "dw")
	g.RestartCurrent()
	st := g.Snapshot()
	if st["state"] != "arena" || arenaSnap(t, g)["num"] != 1 {
		t.Fatalf("RESET 후 state=%v num=%v, want arena/1", st["state"], arenaSnap(t, g)["num"])
	}
	if st["strokes"] != 0 {
		t.Fatalf("RESET 후 strokes=%v, want 0", st["strokes"])
	}

	for i, lv := range ArenaLevels {
		if num := arenaSnap(t, g)["num"]; num != i+1 {
			t.Fatalf("문제 %d 시작 시 num=%v", i+1, num)
		}
		playKeys(g, lv.Solution)
	}
	if st := g.Snapshot()["state"]; st != "arenaDone" {
		t.Fatalf("5문제 후 state=%v, want arenaDone", st)
	}
	if cnt := g.Snapshot()["arenaCount"]; cnt != len(ArenaLevels) {
		t.Fatalf("arenaCount=%v, want %d", cnt, len(ArenaLevels))
	}

	// 완주 화면은 DOM 오버레이 소유 — 키가 삼켜져 상태가 안 바뀐다.
	playKeys(g, "x<cr><esc>j")
	if st := g.Snapshot()["state"]; st != "arenaDone" {
		t.Errorf("arenaDone 에서 키 입력이 상태를 바꿈: %v", st)
	}

	// 완주 화면의 RESET(RestartCurrent)은 런 전체 재시작이다 — default 분기로
	// 흘러 LoadLevel(levelIdx)이 커리큘럼 레벨로 튕겨나가면 안 된다(아키텍처
	// 리뷰가 실제로 잡았던 누수).
	g.RestartCurrent()
	if st := g.Snapshot()["state"]; st != "arena" {
		t.Fatalf("완주 후 RestartCurrent → state=%v, want arena(런 재시작)", st)
	}
	if num := arenaSnap(t, g)["num"]; num != 1 {
		t.Errorf("완주 후 RestartCurrent → num=%v, want 1", num)
	}
}

// TestArenaExCommandsRespectMode 는 아레나 도중의 ex-command 탈출구를
// 고정한다 — :drill(모드 전환)은 무시되고, :q(런 포기)만 레벨 선택으로
// 나간다. :drill 을 허용하면 아레나 패널·타이머 아래에 드릴이 깔리는 화면
// 분열이 생긴다(정합성 리뷰 #3).
func TestArenaExCommandsRespectMode(t *testing.T) {
	g := New()
	g.EnterArena()

	ex(g, "drill w")
	if st := g.Snapshot()["state"]; st != "arena" {
		t.Fatalf(":drill 이 아레나를 깼음: state=%v", st)
	}
	if num := arenaSnap(t, g)["num"]; num != 1 {
		t.Errorf(":drill 무시 후 num=%v, want 1(같은 문제 유지)", num)
	}

	ex(g, "restart") // :restart 는 같은 문제 재시작 — 모드 유지
	if st := g.Snapshot()["state"]; st != "arena" {
		t.Fatalf(":restart 가 아레나를 깼음: state=%v", st)
	}

	ex(g, "q") // 유일한 탈출구 — 런 포기
	if st := g.Snapshot()["state"]; st != "select" {
		t.Errorf(":q 후 state=%v, want select", st)
	}
}
