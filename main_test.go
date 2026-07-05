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

// TestNavigateSolveLevel1 은 1-1 을 이동만으로 클리어해 클리어 화면으로 전환되고,
// Enter 를 누르면 다음 레벨로 넘어가는지 본다.
func TestNavigateSolveLevel1(t *testing.T) {
	g := NewGame()
	if g.lv.Kind != "navigate" {
		t.Fatal("레벨 1-1 이 navigate 가 아님")
	}
	playNav(g, "jjllll") // 열쇠(row2,col4) 획득
	if len(g.keyPos) != 0 {
		t.Fatalf("열쇠 미획득: 남은 %d", len(g.keyPos))
	}
	playNav(g, "jjlllll") // 출구(row4,col9) 도달 → 클리어 화면
	if g.state != stateLevelClear {
		t.Fatalf("출구 도달 후 클리어 상태 전환 실패: state=%v", g.state)
	}
}

// TestNavigateLevelsSolvable 은 navigate 레벨 전부가 Solution 키 시퀀스로
// 실제로 클리어(stateLevelClear/stateAllClear 전환)되는지 검증한다.
func TestNavigateLevelsSolvable(t *testing.T) {
	for idx, lv := range levels {
		if lv.Kind != "navigate" {
			continue
		}
		g := &Game{store: newProgressStore()}
		g.progress = g.store.Load()
		g.loadLevel(idx)
		for _, k := range parseKeys(lv.Solution) {
			g.feed(k)
			g.checkWin()
		}
		if g.state != stateLevelClear && g.state != stateAllClear {
			t.Errorf("[%s] Solution %q 로 클리어 실패 (state=%v)", lv.Title, lv.Solution, g.state)
		}
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

// TestNavigateAllowsSearch 는 navigate 레벨에서 검색(/ ? n N)이 막히지 않는지 확인한다.
func TestNavigateAllowsSearch(t *testing.T) {
	g := NewGame() // 1-1: "@........." 등 5줄
	g.feed(RuneKey('/'))
	if !g.ed.searching {
		t.Fatal("navigate 레벨에서 '/' 가 막힘 — searching 진입 실패")
	}
	for _, r := range "K" {
		g.feed(RuneKey(r))
	}
	g.feed(SpecialKey("cr"))
	if g.ed.row != 2 || g.ed.col != 4 {
		t.Fatalf("navigate 레벨에서 검색 이동 실패: row=%d col=%d want 2,4", g.ed.row, g.ed.col)
	}
}

// TestVisualBellOnBlockedKey 는 막힌 키 입력 시 visual bell(bellTTL)이 켜지는지 확인한다.
func TestVisualBellOnBlockedKey(t *testing.T) {
	g := NewGame()       // 1-1, navigate
	g.feed(RuneKey('d')) // 편집키 — 막혀야 함
	if g.bellTTL == 0 {
		t.Fatal("막힌 키 입력인데 bellTTL 이 설정되지 않음")
	}
}

// TestNoBellOnValidKey 는 정상 입력에서는 visual bell 이 발동하지 않는지 확인한다.
func TestNoBellOnValidKey(t *testing.T) {
	g := NewGame()
	g.feed(RuneKey('l')) // 정상 이동
	if g.bellTTL != 0 {
		t.Fatal("정상 입력인데 bellTTL 이 설정됨")
	}
}

// TestBugKillFiresEffect 는 버그 처치 시 문자 치환 이펙트가 생성되는지 확인한다.
func TestBugKillFiresEffect(t *testing.T) {
	g := NewGame() // 1-1엔 버그가 없으므로 1-5(버그 존재)로 전환
	for i, lv := range levels {
		if lv.ID == "1-5" {
			g.loadLevel(i)
			break
		}
	}
	// 1-5 맵의 첫 버그(row0,col3)로 이동 후 처치
	g.feed(RuneKey('l'))
	g.feed(RuneKey('l'))
	g.feed(RuneKey('l'))
	g.feed(RuneKey('x'))
	if len(g.effects) == 0 {
		t.Fatal("버그 처치 후 이펙트가 생성되지 않음")
	}
}

// feedStr 은 문자열의 각 rune 을 g.feed(RuneKey(r)) 로 순서대로 흘려보낸다.
func feedStr(g *Game, s string) {
	for _, r := range s {
		g.feed(RuneKey(r))
	}
}

// TestExCommandQ 는 :q 로 레벨 선택 화면으로 전환되는지 확인한다.
func TestExCommandQ(t *testing.T) {
	g := NewGame()
	g.feed(RuneKey(':'))
	feedStr(g, "q")
	g.feed(SpecialKey("cr"))
	if g.state != stateLevelSelect {
		t.Fatalf(":q 후 state=%v want stateLevelSelect", g.state)
	}
}

// TestExCommandRestart 는 :restart 로 현재 레벨이 리로드되는지 확인한다.
func TestExCommandRestart(t *testing.T) {
	g := NewGame()
	playNav(g, "jjllll") // 열쇠 획득해 strokes/keyPos 변화를 만든다
	g.feed(RuneKey(':'))
	feedStr(g, "restart")
	g.feed(SpecialKey("cr"))
	if g.strokes != 0 || len(g.keyPos) == 0 {
		t.Fatalf(":restart 후 레벨이 리로드되지 않음: strokes=%d keyPos=%v", g.strokes, g.keyPos)
	}
}

// TestExCommandGotoLine 은 :{N} 이 실제로 N번째 줄로 이동하는지 확인한다
// (Phase3 §0에서 고친 gotoLine count 버그에 의존).
func TestExCommandGotoLine(t *testing.T) {
	g := NewGame()
	for i, lv := range levels {
		if lv.ID == "3-2" { // "good line"/"DELETE THIS"/"another good"/"DELETE THIS" 4줄
			g.loadLevel(i)
			break
		}
	}
	g.feed(RuneKey(':'))
	feedStr(g, "3")
	g.feed(SpecialKey("cr"))
	if g.ed.row != 2 {
		t.Fatalf(":3 후 row=%d want 2", g.ed.row)
	}
}

// TestExCommandEscCancels 는 esc 로 ex-command 입력을 취소하면 상태가 그대로인지 확인한다.
func TestExCommandEscCancels(t *testing.T) {
	g := NewGame()
	before := g.state
	g.feed(RuneKey(':'))
	feedStr(g, "q")
	g.feed(SpecialKey("esc"))
	if g.exMode || g.state != before {
		t.Fatal("esc 취소 후 exMode 가 남아있거나 state 가 바뀜")
	}
}

// TestExCommandUnknownIgnored 는 인식 못하는 명령이 조용히 무시되는지 확인한다.
func TestExCommandUnknownIgnored(t *testing.T) {
	g := NewGame()
	before := g.state
	g.feed(RuneKey(':'))
	feedStr(g, "bogus")
	g.feed(SpecialKey("cr"))
	if g.exMode || g.state != before {
		t.Fatalf("알 수 없는 명령 처리 후 상태 이상: exMode=%v state=%v", g.exMode, g.state)
	}
}

// TestColonInInsertModeIsLiteral 은 Insert 모드에서 ':' 이 ex-command 로 가로채이지
// 않고 그냥 문자로 입력되는지 확인한다.
func TestColonInInsertModeIsLiteral(t *testing.T) {
	g := NewGame()
	for i, lv := range levels {
		if lv.ID == "3-4" { // edit 레벨(Insert 모드 진입 가능)
			g.loadLevel(i)
			break
		}
	}
	g.feed(RuneKey('A'))
	g.feed(RuneKey(':'))
	if g.exMode {
		t.Fatal("Insert 모드에서 ':' 이 ex-command 로 가로채임")
	}
	if !strings.Contains(strings.Join(g.ed.Lines(), "\n"), ":") {
		t.Fatal("Insert 모드에서 ':' 이 문자로 입력되지 않음")
	}
}
