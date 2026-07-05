package main

import (
	"math/rand"
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

// TestExCommandRestartInDrillStaysInDrill 은 :drill 중에 :restart 를 치면
// (버그: 마지막 커리큘럼 레벨로 튕겨나가던 것과 달리) 같은 드릴 문제가
// strokes=0 으로 재시작되고 stateDrill 이 유지되는지 확인한다.
func TestExCommandRestartInDrillStaysInDrill(t *testing.T) {
	g := NewGame()
	g.loadLevel(2) // levelIdx=2 로 이동 — :restart 가 여기로 새는지 구분하기 위함
	g.enterDrill()
	lv := g.lv // 드릴이 생성한 문제(맵/해)를 기억해둔다
	g.feed(RuneKey('l'))
	before := g.strokes
	g.feed(RuneKey(':'))
	feedStr(g, "restart")
	g.feed(SpecialKey("cr"))
	if g.state != stateDrill {
		t.Fatalf(":drill 중 :restart 가 드릴을 벗어남: state=%v want stateDrill", g.state)
	}
	if g.levelIdx != 2 {
		t.Fatalf(":drill 중 :restart 가 커리큘럼 레벨로 샘: levelIdx=%d", g.levelIdx)
	}
	if g.strokes != 0 {
		t.Fatalf(":restart 후 strokes 가 리셋되지 않음: before=%d after=%d", before, g.strokes)
	}
	if g.lv.Solution != lv.Solution {
		t.Fatalf(":restart 가 같은 드릴 문제를 유지하지 않음: got %q want %q", g.lv.Solution, lv.Solution)
	}
}

// TestRestartCurrentIsDrillAware 는 restartCurrent() 자체를 직접 호출해도
// (즉 :restart ex-command 경로를 거치지 않고, web_js.go 의 vimquestReset 이
// 부르는 것과 동일하게) 드릴 인식이 유지되는지 확인한다. 예전엔 이 로직이
// runExCommand 안에만 있어서 vimquestReset 이 별도로 g.loadLevel 을 직접
// 불러 같은 버그를 반복했다 — 이제 두 호출부 모두 restartCurrent() 하나만
// 거치므로, 이 테스트가 그 유일한 구현을 직접 지킨다.
func TestRestartCurrentIsDrillAware(t *testing.T) {
	g := NewGame()
	g.loadLevel(2)
	g.enterDrill()
	lv := g.lv
	g.feed(RuneKey('l'))
	g.restartCurrent()
	if g.state != stateDrill || g.levelIdx != 2 || g.strokes != 0 || g.lv.Solution != lv.Solution {
		t.Fatalf("restartCurrent() 가 드릴을 벗어남: state=%v levelIdx=%d strokes=%d solution=%q want stateDrill,2,0,%q",
			g.state, g.levelIdx, g.strokes, g.lv.Solution, lv.Solution)
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

// TestInputLevelClearEnterAdvances 는 클리어 화면에서 Input(cr) 이 다음 레벨로
// 넘어가는지 확인한다(Phase 4 L1: Update()/updateLevelClear() 를 대체한 Input 경로).
func TestInputLevelClearEnterAdvances(t *testing.T) {
	g := NewGame()
	playNav(g, "jjllll")  // 열쇠 획득
	playNav(g, "jjlllll") // 출구 도달 → stateLevelClear
	if g.state != stateLevelClear {
		t.Fatalf("사전조건 실패: state=%v want stateLevelClear", g.state)
	}
	g.Input(SpecialKey("cr"))
	if g.levelIdx != 1 || g.state != statePlaying {
		t.Fatalf("Enter 후 levelIdx=%d state=%v want 1,statePlaying", g.levelIdx, g.state)
	}
}

// TestInputLevelClearRetry 는 클리어 화면에서 Input('r') 이 같은 레벨을
// strokes=0 으로 리로드하는지 확인한다.
func TestInputLevelClearRetry(t *testing.T) {
	g := NewGame()
	playNav(g, "jjllll")
	playNav(g, "jjlllll")
	if g.state != stateLevelClear {
		t.Fatalf("사전조건 실패: state=%v want stateLevelClear", g.state)
	}
	g.Input(RuneKey('r'))
	if g.levelIdx != 0 || g.state != statePlaying || g.strokes != 0 {
		t.Fatalf("'r' 후 levelIdx=%d state=%v strokes=%d want 0,statePlaying,0", g.levelIdx, g.state, g.strokes)
	}
}

// TestInputLevelSelectNavigation 은 레벨 선택 화면에서 h/j/k/l 로 커서를 옮기고
// 잠긴 레벨은 Enter 로 입장되지 않으며 unlocked 레벨은 입장되는지 확인한다.
func TestInputLevelSelectNavigation(t *testing.T) {
	g := NewGame()
	g.enterLevelSelect() // 1-1 위치, W1(selRow=0) selCol=0

	g.Input(RuneKey('l')) // W2 로 이동 — 2-1 은 아직 잠김
	g.Input(SpecialKey("cr"))
	if g.state != stateLevelSelect {
		t.Fatalf("잠긴 레벨(2-1) 진입이 막히지 않음: state=%v", g.state)
	}

	g.Input(RuneKey('h')) // W1 로 복귀 — 1-1 은 unlocked
	g.Input(SpecialKey("cr"))
	if g.state != statePlaying || g.levelIdx != 0 {
		t.Fatalf("unlocked 레벨(1-1) 진입 실패: state=%v levelIdx=%d", g.state, g.levelIdx)
	}
}

// TestInputNoStrokesOutsidePlaying 은 statePlaying 이 아닌 상태에서의 Input 이
// strokes 를 증가시키지 않는지 확인한다(레벨 클리어 화면에서 의미 없는 키 입력).
func TestInputNoStrokesOutsidePlaying(t *testing.T) {
	g := NewGame()
	playNav(g, "jjllll")
	playNav(g, "jjlllll") // stateLevelClear
	before := g.strokes
	g.Input(RuneKey('x')) // 클리어 화면에서 'r'/'cr' 도 아닌 키
	if g.strokes != before {
		t.Fatalf("비-플레이 상태 입력이 strokes 를 증가시킴: before=%d after=%d", before, g.strokes)
	}
}

// TestDrillCapsSessionLength 는 drillMaxRounds 에 도달하면 :drill 이 새 문제를
// 계속 생성하는 대신 레벨 선택으로 빠지는지 확인한다 — -gc=leaking 웹 빌드에서
// 문제 생성마다 나오는 쓰레기가 세션 내내 무한정 쌓이는 걸 막는 상한.
func TestDrillCapsSessionLength(t *testing.T) {
	g := NewGame()
	g.enterDrill()
	g.drillStreak = drillMaxRounds - 1
	g.advanceDrill()
	if g.state != stateLevelSelect {
		t.Fatalf("드릴 상한(%d) 도달 후 레벨 선택으로 빠지지 않음: state=%v", drillMaxRounds, g.state)
	}
}

// TestDrillGeneratorSolvable 은 :drill 이 생성하는 무작위 문제 100개가 전부
// 생성기 자신이 산출한 Solution 으로 실제 클리어되는지 확인한다(property
// 테스트 — 시드 고정으로 재현 가능). loadLevelData(정규 레벨과 공유하는 실제
// 프로덕션 파싱 경로)를 그대로 재사용해 로직 중복/드리프트를 피한다.
// advanceDrill 을 거치면 클리어 즉시 다음 문제가 자동 생성돼 검증이 꼬이므로
// checkWin() 이 아니라 승리 조건만 직접 확인한다.
func TestDrillGeneratorSolvable(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 100; i++ {
		lv := generateDrill(rng)

		g := &Game{store: newProgressStore()}
		g.progress = g.store.Load()
		g.loadLevelData(lv)

		keyPos := make(map[[2]int]bool, len(g.keyPos))
		for pos := range g.keyPos {
			keyPos[pos] = true
		}

		for _, k := range parseKeys(lv.Solution) {
			g.ed.Feed(k)
			delete(keyPos, [2]int{g.ed.row, g.ed.col})
		}

		cell := g.cellAt(g.ed.row, g.ed.col)
		if len(keyPos) != 0 || cell != '$' {
			t.Fatalf("[iter %d] 생성기 해로 클리어 안 됨: 남은 keyPos=%v cell=%q map=%v sol=%q",
				i, keyPos, cell, lv.Map, lv.Solution)
		}
	}
}

// TestLoadLevelDataSetsCursorDcol 은 '@' 가 col>0 인 navigate 레벨에서 첫
// 입력이 수직 모션(j/k)이어도 col0 으로 튀지 않는지 확인한다. loadLevelData
// 가 e.row/e.col 만 옮기고 e.dcol(수직 모션의 목표 열)은 그대로 두면 j/k 가
// dcol=0(SetLines 의 초기값)을 써서 커서가 엉뚱한 열로 이동한다 — :drill
// 생성기가 무작위 시작열을 만들면서 실제로 재현해 잡아낸 결함이다.
func TestLoadLevelDataSetsCursorDcol(t *testing.T) {
	g := &Game{store: newProgressStore()}
	g.progress = g.store.Load()
	g.loadLevelData(Level{
		Kind: "navigate",
		Map:  []string{"....@....", ".........", "....$...."},
	})
	if g.ed.col != 4 {
		t.Fatalf("초기 col=%d want 4", g.ed.col)
	}
	g.ed.Feed(RuneKey('j')) // 수직 모션 — dcol 이 4 로 맞춰져 있어야 col 유지
	if g.ed.col != 4 {
		t.Fatalf("j 이후 col=%d want 4 (dcol 이 시작 열에 맞춰지지 않음)", g.ed.col)
	}
}
