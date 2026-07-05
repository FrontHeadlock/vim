package game

import (
	"math/rand"
	"strings"
	"testing"
	"unicode/utf8"

	"vimquest/internal/engine"
	"vimquest/internal/store"
)

// feedKeys 는 입력 문자열을 파싱해 엔진에 흘려보낸다(engine 패키지 테스트의
// 동명 헬퍼와 같은 3줄 — 패키지가 갈리면서 각자 소유한다).
func feedKeys(e *engine.Editor, s string) {
	for _, k := range engine.ParseKeys(s) {
		e.Feed(k)
	}
}

// TestEditLevelsSolvable 은 모든 edit 레벨이 의도된 Solution 으로 Target 에
// 정확히 도달하는지 검증한다 — 풀이 불가능한 퍼즐을 출시 전에 잡아낸다.
// B4 가드도 겸한다: 여러 줄 charwise 비주얼 선택은 "줄 단위로 대체 처리"되는
// 알려진 부정확성(refactor_code.md B4)이라, 레벨 저작이 실수로 이 경로를
// 밟으면 여기서 즉시 실패해야 한다.
func TestEditLevelsSolvable(t *testing.T) {
	engine.ResetMultilineCharwiseFallbackCount()
	for _, lv := range levels {
		if lv.Kind != "edit" {
			continue
		}
		e := engine.NewEditor(append([]string(nil), lv.Map...))
		feedKeys(e, lv.Solution)
		got := strings.Join(e.Lines(), "\n")
		want := strings.Join(lv.Target, "\n")
		if got != want {
			t.Errorf("[%s] Solution %q\n  got:  %q\n  want: %q", lv.Title, lv.Solution, got, want)
		}
	}
	if n := engine.MultilineCharwiseFallbackCount(); n != 0 {
		t.Errorf("레벨 Solution 이 여러 줄 charwise 대체 경로를 %d회 밟음 — "+
			"실제 Vim 과 다르게 동작하는 부정확한 경로(B4)이므로 레벨 설계를 바꿔야 함", n)
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
		g.feed(engine.RuneKey(r))
		g.checkWin()
	}
}

// TestNavigateSolveLevel1 은 1-1 을 이동만으로 클리어해 클리어 화면으로 전환되고,
// Enter 를 누르면 다음 레벨로 넘어가는지 본다.
func TestNavigateSolveLevel1(t *testing.T) {
	g := New()
	if g.lv.Kind != "navigate" {
		t.Fatal("레벨 1-1 이 navigate 가 아님")
	}
	playNav(g, "jjllll") // 열쇠(row2,col4) 획득
	if len(g.keyPos) != 0 {
		t.Fatalf("열쇠 미획득: 남은 %d", len(g.keyPos))
	}
	playNav(g, "jjlllll") // 출구(row4,col9) 도달 → 클리어 화면
	if g.state != StateLevelClear {
		t.Fatalf("출구 도달 후 클리어 상태 전환 실패: state=%v", g.state)
	}
}

// TestNavigateLevelsSolvable 은 navigate 레벨 전부가 Solution 키 시퀀스로
// 실제로 클리어(StateLevelClear/StateAllClear 전환)되는지 검증한다.
func TestNavigateLevelsSolvable(t *testing.T) {
	for idx, lv := range levels {
		if lv.Kind != "navigate" {
			continue
		}
		g := &Game{store: store.New()}
		g.progress = g.store.Load()
		g.LoadLevel(idx)
		for _, k := range engine.ParseKeys(lv.Solution) {
			g.feed(k)
			g.checkWin()
		}
		if g.state != StateLevelClear && g.state != StateAllClear {
			t.Errorf("[%s] Solution %q 로 클리어 실패 (state=%v)", lv.Title, lv.Solution, g.state)
		}
	}
}

// TestLevel16NaiveSolveIsWorse 는 B3: "1-6" 보너스 레벨이 f/t 랜드마크 없이
// (hjkl 만으로) 풀면 par 대비 1.5배를 넘는 타수가 드는지 확인한다 — 카운트·
// find 조합 없이는 par 급 클리어가 불가능하도록 설계됐음을 실측으로 보증한다.
func TestLevel16NaiveSolveIsWorse(t *testing.T) {
	var lv Level
	for _, l := range levels {
		if l.ID == "1-6" {
			lv = l
			break
		}
	}
	if lv.ID == "" {
		t.Fatal("레벨 1-6 을 찾을 수 없음")
	}
	par := len(engine.ParseKeys(lv.Solution))

	// naive: hjkl 만으로 왼쪽 끝(뒤쪽) 열쇠까지 갔다가 오른쪽 끝 출구까지
	// — 중간의 두 번째 열쇠는 지나가는 길에 자동 습득된다.
	row := lv.Map[0]
	startCol := strings.IndexRune(row, '@')
	e := engine.NewEditor([]string{row})
	e.SetCursor(0, startCol)
	naiveKeys := strings.Repeat("h", startCol) + strings.Repeat("l", len([]rune(row))-1)
	for _, k := range engine.ParseKeys(naiveKeys) {
		e.Feed(k)
	}
	if e.Col() != len([]rune(row))-1 {
		t.Fatalf("naive 해가 출구 칸에 도달 못함: col=%d want %d", e.Col(), len([]rune(row))-1)
	}
	naive := len(naiveKeys)
	if float64(naive) <= float64(par)*1.5 {
		t.Fatalf("naive(%d) 가 par(%d)*1.5=%.1f 를 넘지 않음 — 카운트/find 없이도 2★ 이상 가능",
			naive, par, float64(par)*1.5)
	}
}

// TestNavigateBlocksEditing 은 navigate 레벨에서 편집키가 막히는지 확인한다.
func TestNavigateBlocksEditing(t *testing.T) {
	g := New() // 1-1
	before := strings.Join(g.ed.Lines(), "\n")
	g.feed(engine.RuneKey('d'))
	g.feed(engine.RuneKey('d')) // dd 시도 — 막혀야 함
	after := strings.Join(g.ed.Lines(), "\n")
	if before != after {
		t.Errorf("navigate 레벨에서 편집이 허용됨:\n  before %q\n  after  %q", before, after)
	}
}

// TestNavigateAllowsSearch 는 navigate 레벨에서 검색(/ ? n N)이 막히지 않는지 확인한다.
func TestNavigateAllowsSearch(t *testing.T) {
	g := New() // 1-1: "@........." 등 5줄
	g.feed(engine.RuneKey('/'))
	if !g.ed.Searching() {
		t.Fatal("navigate 레벨에서 '/' 가 막힘 — searching 진입 실패")
	}
	for _, r := range "K" {
		g.feed(engine.RuneKey(r))
	}
	g.feed(engine.SpecialKey("cr"))
	if g.ed.Row() != 2 || g.ed.Col() != 4 {
		t.Fatalf("navigate 레벨에서 검색 이동 실패: row=%d col=%d want 2,4", g.ed.Row(), g.ed.Col())
	}
}

// firstMeaningfulRuneOfCmdToken 은 Cmd.K 토큰("f{char}", "{N}G", "<cr>",
// "(type)" 등)에서 실제로 입력되는 첫 키를 추출한다. 특수키 표기(<...>)나
// 설명용 텍스트(다음 오는 문자열, 예: "(type)")는 0을 돌려준다(호출자가 건너뜀).
func firstMeaningfulRuneOfCmdToken(tok string) rune {
	if tok == "" || tok[0] == '<' || tok[0] == '(' {
		return 0
	}
	if tok[0] == '{' {
		if i := strings.IndexByte(tok, '}'); i >= 0 {
			tok = tok[i+1:]
		}
	}
	if tok == "" {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(tok)
	return r
}

// TestNavigateAllowsAllTaughtKeys 는 C2: 모든 navigate 레벨의 Cmds 에 등장하는
// 키가 navigateAllows 화이트리스트를 통과하는지 확인한다. 화이트리스트는
// 하드코딩된 switch 로 유지하되(허용 정책은 코드에 보이는 게 낫다), 새
// navigate 레벨이 가르치는 키를 화이트리스트에 추가하는 걸 잊는 사고("가르치는
// 키가 막히는" 버그)를 이 테이블 테스트가 자동으로 잡는다.
func TestNavigateAllowsAllTaughtKeys(t *testing.T) {
	e := engine.NewEditor([]string{"x"})
	for _, lv := range levels {
		if lv.Kind != "navigate" {
			continue
		}
		for _, cmd := range lv.Cmds {
			for _, tok := range strings.Fields(cmd.K) {
				r := firstMeaningfulRuneOfCmdToken(tok)
				if r == 0 {
					continue
				}
				if !navigateAllows(e, engine.RuneKey(r)) {
					t.Errorf("[%s] Cmds %q(토큰 %q) 의 키 %q 가 navigateAllows 를 통과하지 못함",
						lv.ID, cmd.K, tok, string(r))
				}
			}
		}
	}
}

// TestVisualBellOnBlockedKey 는 막힌 키 입력 시 visual bell(bellTTL)이 켜지는지 확인한다.
func TestVisualBellOnBlockedKey(t *testing.T) {
	g := New()                  // 1-1, navigate
	g.feed(engine.RuneKey('d')) // 편집키 — 막혀야 함
	if g.bellTTL == 0 {
		t.Fatal("막힌 키 입력인데 bellTTL 이 설정되지 않음")
	}
}

// TestNoBellOnValidKey 는 정상 입력에서는 visual bell 이 발동하지 않는지 확인한다.
func TestNoBellOnValidKey(t *testing.T) {
	g := New()
	g.feed(engine.RuneKey('l')) // 정상 이동
	if g.bellTTL != 0 {
		t.Fatal("정상 입력인데 bellTTL 이 설정됨")
	}
}

// TestBugKillFiresEffect 는 버그 처치 시 문자 치환 이펙트가 생성되는지 확인한다.
func TestBugKillFiresEffect(t *testing.T) {
	g := New() // 1-1엔 버그가 없으므로 1-5(버그 존재)로 전환
	for i, lv := range levels {
		if lv.ID == "1-5" {
			g.LoadLevel(i)
			break
		}
	}
	// 1-5 맵의 첫 버그(row0,col3)로 이동 후 처치
	g.feed(engine.RuneKey('l'))
	g.feed(engine.RuneKey('l'))
	g.feed(engine.RuneKey('l'))
	g.feed(engine.RuneKey('x'))
	if len(g.effects) == 0 {
		t.Fatal("버그 처치 후 이펙트가 생성되지 않음")
	}
}

// TestNavigateBugKillPreservesCoordinates 는 A5 회귀 테스트: 버그가 같은 줄에서
// 열쇠/출구보다 왼쪽에 있을 때 x 로 처치해도 그 줄의 다른 좌표가 밀리지
// 않는지 확인한다. 예전엔 g.ed.Feed(k) 가 deleteChars(물리적 삭제)를 태워
// 버그 오른쪽의 모든 문자가 한 칸씩 왼쪽으로 밀렸다 — keyPos(레벨 로드 시
// 고정 캡처)와 라이브 판정 좌표(cellAt)가 어긋나는 결함이었다.
func TestNavigateBugKillPreservesCoordinates(t *testing.T) {
	lv := Level{ID: "x-desync-regress", Kind: "navigate", Map: []string{"@.*..K...$"}}
	g := &Game{store: store.New()}
	g.progress = g.store.Load()
	g.loadLevelData(lv)

	// 버그(col2)로 이동해 처치.
	g.feed(engine.RuneKey('l'))
	g.feed(engine.RuneKey('l'))
	g.feed(engine.RuneKey('x'))

	line := g.ed.Lines()[0]
	if len(line) != len(lv.Map[0]) {
		t.Fatalf("버그 처치 후 줄 길이가 바뀜: got %q(len %d) want len %d", line, len(line), len(lv.Map[0]))
	}
	if ch, _ := g.ed.Cell(0, 2); ch != '.' {
		t.Fatalf("버그 위치(col2)가 '.'로 치환되지 않음: got %q", ch)
	}
	if ch, _ := g.ed.Cell(0, 5); ch != 'K' {
		t.Fatalf("열쇠 위치(col5)가 밀림: got %q want 'K'", ch)
	}
	if !g.HasKeyAt(0, 5) {
		t.Fatal("keyPos(0,5) 가 더 이상 유효하지 않음 — 실제 K 위치와 desync")
	}
	if ch, _ := g.ed.Cell(0, 9); ch != '$' {
		t.Fatalf("출구 위치(col9)가 밀림: got %q want '$'", ch)
	}
}

// feedStr 은 문자열의 각 rune 을 g.feed(engine.RuneKey(r)) 로 순서대로 흘려보낸다.
func feedStr(g *Game, s string) {
	for _, r := range s {
		g.feed(engine.RuneKey(r))
	}
}

// TestExCommandQ 는 :q 로 레벨 선택 화면으로 전환되는지 확인한다.
func TestExCommandQ(t *testing.T) {
	g := New()
	g.feed(engine.RuneKey(':'))
	feedStr(g, "q")
	g.feed(engine.SpecialKey("cr"))
	if g.state != StateLevelSelect {
		t.Fatalf(":q 후 state=%v want StateLevelSelect", g.state)
	}
}

// TestStrokesExemptExCommand 는 B5: ':' 진입과 그 이후 ex-command 입력이
// strokes 를 증가시키지 않는지 확인한다 — ':help' 를 열어봐도 별점 손해가
// 없어야 한다(예전엔 feed() 최상단에서 무조건 strokes++ 했었다).
func TestStrokesExemptExCommand(t *testing.T) {
	g := New()
	before := g.strokes
	g.feed(engine.RuneKey(':'))
	feedStr(g, "help")
	g.feed(engine.SpecialKey("cr"))
	if g.strokes != before {
		t.Fatalf("':help<cr>' 이후 strokes 가 변함: before=%d after=%d", before, g.strokes)
	}
}

// TestExCommandRestart 는 :restart 로 현재 레벨이 리로드되는지 확인한다.
// TestExCommandDrillKindDispatch 는 B2: ":drill"/":drill w"/":drill f"/
// ":drill x" 가 각각 올바른 생성기 유형으로 드릴을 시작하는지 확인한다
// (레벨 Title 의 대괄호 표기로 판별 — HUD 표시와 같은 소스).
func TestExCommandDrillKindDispatch(t *testing.T) {
	cases := []struct{ cmd, title string }{
		{"drill", "DRILL"},
		{"drill w", "DRILL [w]"},
		{"drill f", "DRILL [f]"},
		{"drill x", "DRILL [x]"},
	}
	for _, c := range cases {
		g := New()
		g.feed(engine.RuneKey(':'))
		feedStr(g, c.cmd)
		g.feed(engine.SpecialKey("cr"))
		if g.state != StateDrill {
			t.Fatalf("[%q] 이후 state=%v want StateDrill", c.cmd, g.state)
		}
		if g.lv.Title != c.title {
			t.Fatalf("[%q] 레벨 Title=%q want %q", c.cmd, g.lv.Title, c.title)
		}
	}
}

func TestExCommandRestart(t *testing.T) {
	g := New()
	playNav(g, "jjllll") // 열쇠 획득해 strokes/keyPos 변화를 만든다
	g.feed(engine.RuneKey(':'))
	feedStr(g, "restart")
	g.feed(engine.SpecialKey("cr"))
	if g.strokes != 0 || len(g.keyPos) == 0 {
		t.Fatalf(":restart 후 레벨이 리로드되지 않음: strokes=%d keyPos=%v", g.strokes, g.keyPos)
	}
}

// TestExCommandRestartInDrillStaysInDrill 은 :drill 중에 :restart 를 치면
// (버그: 마지막 커리큘럼 레벨로 튕겨나가던 것과 달리) 같은 드릴 문제가
// strokes=0 으로 재시작되고 StateDrill 이 유지되는지 확인한다.
func TestExCommandRestartInDrillStaysInDrill(t *testing.T) {
	g := New()
	g.LoadLevel(2) // levelIdx=2 로 이동 — :restart 가 여기로 새는지 구분하기 위함
	g.enterDrill("")
	lv := g.lv // 드릴이 생성한 문제(맵/해)를 기억해둔다
	g.feed(engine.RuneKey('l'))
	before := g.strokes
	g.feed(engine.RuneKey(':'))
	feedStr(g, "restart")
	g.feed(engine.SpecialKey("cr"))
	if g.state != StateDrill {
		t.Fatalf(":drill 중 :restart 가 드릴을 벗어남: state=%v want StateDrill", g.state)
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
	g := New()
	g.LoadLevel(2)
	g.enterDrill("")
	lv := g.lv
	g.feed(engine.RuneKey('l'))
	g.RestartCurrent()
	if g.state != StateDrill || g.levelIdx != 2 || g.strokes != 0 || g.lv.Solution != lv.Solution {
		t.Fatalf("restartCurrent() 가 드릴을 벗어남: state=%v levelIdx=%d strokes=%d solution=%q want StateDrill,2,0,%q",
			g.state, g.levelIdx, g.strokes, g.lv.Solution, lv.Solution)
	}
}

// TestExCommandGotoLine 은 :{N} 이 실제로 N번째 줄로 이동하는지 확인한다
// (Phase3 §0에서 고친 gotoLine count 버그에 의존).
func TestExCommandGotoLine(t *testing.T) {
	g := New()
	for i, lv := range levels {
		if lv.ID == "3-2" { // "good line"/"DELETE THIS"/"another good"/"DELETE THIS" 4줄
			g.LoadLevel(i)
			break
		}
	}
	g.feed(engine.RuneKey(':'))
	feedStr(g, "3")
	g.feed(engine.SpecialKey("cr"))
	if g.ed.Row() != 2 {
		t.Fatalf(":3 후 row=%d want 2", g.ed.Row())
	}
}

// TestExCommandEscCancels 는 esc 로 ex-command 입력을 취소하면 상태가 그대로인지 확인한다.
func TestExCommandEscCancels(t *testing.T) {
	g := New()
	before := g.state
	g.feed(engine.RuneKey(':'))
	feedStr(g, "q")
	g.feed(engine.SpecialKey("esc"))
	if g.exMode || g.state != before {
		t.Fatal("esc 취소 후 exMode 가 남아있거나 state 가 바뀜")
	}
}

// TestExCommandUnknownIgnored 는 인식 못하는 명령이 조용히 무시되는지 확인한다.
func TestExCommandUnknownIgnored(t *testing.T) {
	g := New()
	before := g.state
	g.feed(engine.RuneKey(':'))
	feedStr(g, "bogus")
	g.feed(engine.SpecialKey("cr"))
	if g.exMode || g.state != before {
		t.Fatalf("알 수 없는 명령 처리 후 상태 이상: exMode=%v state=%v", g.exMode, g.state)
	}
}

// TestColonInInsertModeIsLiteral 은 Insert 모드에서 ':' 이 ex-command 로 가로채이지
// 않고 그냥 문자로 입력되는지 확인한다.
func TestColonInInsertModeIsLiteral(t *testing.T) {
	g := New()
	for i, lv := range levels {
		if lv.ID == "3-4" { // edit 레벨(Insert 모드 진입 가능)
			g.LoadLevel(i)
			break
		}
	}
	g.feed(engine.RuneKey('A'))
	g.feed(engine.RuneKey(':'))
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
	g := New()
	playNav(g, "jjllll")  // 열쇠 획득
	playNav(g, "jjlllll") // 출구 도달 → StateLevelClear
	if g.state != StateLevelClear {
		t.Fatalf("사전조건 실패: state=%v want StateLevelClear", g.state)
	}
	g.Input(engine.SpecialKey("cr"))
	if g.levelIdx != 1 || g.state != StatePlaying {
		t.Fatalf("Enter 후 levelIdx=%d state=%v want 1,StatePlaying", g.levelIdx, g.state)
	}
}

// TestClearIsNewFlag 는 C1: ClearStats.IsNew 가 최초 클리어/기록 미갱신
// 두 경우에서 올바른지 확인한다(렌더러가 직접 재계산하지 않고 이 값만 읽는다).
func TestClearIsNewFlag(t *testing.T) {
	g := New()
	playNav(g, "jjllll")
	playNav(g, "jjlllll")
	if !g.clear.IsNew {
		t.Fatal("최초 클리어인데 IsNew=false")
	}
	firstStrokes := g.clear.Strokes

	g.LoadLevel(0)
	playNav(g, "h") // 낭비 키 — col0 에서 no-op 이동이지만 strokes 는 증가시킴
	playNav(g, "jjllll")
	playNav(g, "jjlllll")
	if g.clear.Strokes <= firstStrokes {
		t.Fatalf("사전조건 실패: 재클리어 타수(%d)가 최초(%d)보다 작지 않음", g.clear.Strokes, firstStrokes)
	}
	if g.clear.IsNew {
		t.Fatalf("이전 기록(%d)보다 느린 재클리어(%d)인데 IsNew=true", firstStrokes, g.clear.Strokes)
	}
}

// TestClearYoursRecordsMyKeys 는 B4: 클리어 화면의 "yours" 가 실제로 입력한
// 키 시퀀스를 반영하고, ex-command 로 진입한 키(:q 등)는 포함하지 않는지
// 확인한다(strokes 와 같은 기준 — B5).
func TestClearYoursRecordsMyKeys(t *testing.T) {
	g := New()
	playNav(g, "jjllll")
	g.feed(engine.RuneKey(':')) // ex-command 는 strokes 에서 빠지므로 yours 에도 없어야 함
	feedStr(g, "q")
	g.feed(engine.SpecialKey("cr"))
	g.LoadLevel(0) // :q 가 레벨 선택으로 보냈으므로 다시 로드
	playNav(g, "jjllll")
	playNav(g, "jjlllll")

	got := engine.KeysString(g.myKeys)
	if strings.Contains(got, ":") {
		t.Fatalf("yours 에 ex-command 흔적이 남음: %q", got)
	}
	if len(got) != g.clear.Strokes {
		t.Fatalf("yours 길이(%d)가 strokes(%d)와 다름: %q", len(got), g.clear.Strokes, got)
	}
	if g.clear.Yours != got {
		t.Fatalf("ClearStats.Yours=%q want %q", g.clear.Yours, got)
	}
}

// TestInputLevelClearRetry 는 클리어 화면에서 Input('r') 이 같은 레벨을
// strokes=0 으로 리로드하는지 확인한다.
func TestInputLevelClearRetry(t *testing.T) {
	g := New()
	playNav(g, "jjllll")
	playNav(g, "jjlllll")
	if g.state != StateLevelClear {
		t.Fatalf("사전조건 실패: state=%v want StateLevelClear", g.state)
	}
	g.Input(engine.RuneKey('r'))
	if g.levelIdx != 0 || g.state != StatePlaying || g.strokes != 0 {
		t.Fatalf("'r' 후 levelIdx=%d state=%v strokes=%d want 0,StatePlaying,0", g.levelIdx, g.state, g.strokes)
	}
}

// TestInputLevelSelectNavigation 은 레벨 선택 화면에서 h/j/k/l 로 커서를 옮기고
// 잠긴 레벨은 Enter 로 입장되지 않으며 unlocked 레벨은 입장되는지 확인한다.
func TestInputLevelSelectNavigation(t *testing.T) {
	g := New()
	g.EnterLevelSelect() // 1-1 위치, W1(selRow=0) selCol=0

	g.Input(engine.RuneKey('l')) // W2 로 이동 — 2-1 은 아직 잠김
	g.Input(engine.SpecialKey("cr"))
	if g.state != StateLevelSelect {
		t.Fatalf("잠긴 레벨(2-1) 진입이 막히지 않음: state=%v", g.state)
	}

	g.Input(engine.RuneKey('h')) // W1 로 복귀 — 1-1 은 unlocked
	g.Input(engine.SpecialKey("cr"))
	if g.state != StatePlaying || g.levelIdx != 0 {
		t.Fatalf("unlocked 레벨(1-1) 진입 실패: state=%v levelIdx=%d", g.state, g.levelIdx)
	}
}

// TestInputNoStrokesOutsidePlaying 은 StatePlaying 이 아닌 상태에서의 Input 이
// strokes 를 증가시키지 않는지 확인한다(레벨 클리어 화면에서 의미 없는 키 입력).
func TestInputNoStrokesOutsidePlaying(t *testing.T) {
	g := New()
	playNav(g, "jjllll")
	playNav(g, "jjlllll") // StateLevelClear
	before := g.strokes
	g.Input(engine.RuneKey('x')) // 클리어 화면에서 'r'/'cr' 도 아닌 키
	if g.strokes != before {
		t.Fatalf("비-플레이 상태 입력이 strokes 를 증가시킴: before=%d after=%d", before, g.strokes)
	}
}

// TestInputAllClearReturnsToSelect 는 F2/A1: StateAllClear 화면에서 cr(또는 esc)
// 이 레벨 선택으로 복귀시키는지 확인한다 — 예전엔 이 상태에서 입력이 전부
// 무시돼 데스크톱에서 앱 종료 외 탈출 수단이 없었다(소프트락).
func TestInputAllClearReturnsToSelect(t *testing.T) {
	g := &Game{store: store.New(), state: StateAllClear}
	g.progress = g.store.Load()
	g.Input(engine.SpecialKey("cr"))
	if g.state != StateLevelSelect {
		t.Fatalf("AllClear 에서 cr 후 state=%v want StateLevelSelect", g.state)
	}

	g2 := &Game{store: store.New(), state: StateAllClear}
	g2.progress = g2.store.Load()
	g2.Input(engine.SpecialKey("esc"))
	if g2.state != StateLevelSelect {
		t.Fatalf("AllClear 에서 esc 후 state=%v want StateLevelSelect", g2.state)
	}
}

// TestParseKeyUTF8 은 A4: 웹 입력 경로(ParseKey)가 멀티바이트 UTF-8 토큰을
// 바이트 단위로 잘라 깨진 rune 을 만들지 않는지 확인한다 — engine.ParseKeys
// 와 같은 계열의 결함이 이 함수에도 별도로 있었다(rune(tok[0])).
func TestParseKeyUTF8(t *testing.T) {
	k := ParseKey("한")
	if k.R != '한' {
		t.Fatalf("ParseKey(%q)=%+v want RuneKey('한')", "한", k)
	}
	if got := ParseKey("<esc>"); got.S != "esc" {
		t.Fatalf("ParseKey(<esc>)=%+v want SpecialKey(esc)", got)
	}
	if got := ParseKey("x"); got.R != 'x' {
		t.Fatalf("ParseKey(x)=%+v want RuneKey('x')", got)
	}
}

// TestDrillCapsSessionLength 는 drillMaxRounds 에 도달하면 :drill 이 새 문제를
// 계속 생성하는 대신 레벨 선택으로 빠지는지 확인한다 — -gc=leaking 웹 빌드에서
// 문제 생성마다 나오는 쓰레기가 세션 내내 무한정 쌓이는 걸 막는 상한.
func TestDrillCapsSessionLength(t *testing.T) {
	g := New()
	g.enterDrill("")
	g.drillStreak = drillMaxRounds - 1
	g.advanceDrill()
	if g.state != StateLevelSelect {
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

		g := &Game{store: store.New()}
		g.progress = g.store.Load()
		g.loadLevelData(lv)

		keyPos := make(map[[2]int]bool, len(g.keyPos))
		for pos := range g.keyPos {
			keyPos[pos] = true
		}

		for _, k := range engine.ParseKeys(lv.Solution) {
			g.ed.Feed(k)
			delete(keyPos, [2]int{g.ed.Row(), g.ed.Col()})
		}

		cell := g.cellAt(g.ed.Row(), g.ed.Col())
		if len(keyPos) != 0 || cell != '$' {
			t.Fatalf("[iter %d] 생성기 해로 클리어 안 됨: 남은 keyPos=%v cell=%q map=%v sol=%q",
				i, keyPos, cell, lv.Map, lv.Solution)
		}
	}
}

// TestDrillWordGeneratorSolvable 은 B2: :drill w 가 생성하는 문제 100개가
// 전부 생성기 자신의 그리디 해(w 반복)로 클리어되는지 확인한다.
func TestDrillWordGeneratorSolvable(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	for i := 0; i < 100; i++ {
		lv := generateDrillWord(rng)

		g := &Game{store: store.New()}
		g.progress = g.store.Load()
		g.loadLevelData(lv)

		keyPos := make(map[[2]int]bool, len(g.keyPos))
		for pos := range g.keyPos {
			keyPos[pos] = true
		}
		for _, k := range engine.ParseKeys(lv.Solution) {
			g.ed.Feed(k)
			delete(keyPos, [2]int{g.ed.Row(), g.ed.Col()})
		}
		cell := g.cellAt(g.ed.Row(), g.ed.Col())
		if len(keyPos) != 0 || cell != '$' {
			t.Fatalf("[iter %d] :drill w 생성기 해로 클리어 안 됨: 남은 keyPos=%v cell=%q map=%v sol=%q",
				i, keyPos, cell, lv.Map, lv.Solution)
		}
	}
}

// TestDrillFindGeneratorSolvable 은 B2: :drill f 가 생성하는 문제 100개가
// 전부 생성기 자신의 그리디 해(fK 반복 + f$)로 클리어되는지 확인한다.
func TestDrillFindGeneratorSolvable(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	for i := 0; i < 100; i++ {
		lv := generateDrillFind(rng)

		g := &Game{store: store.New()}
		g.progress = g.store.Load()
		g.loadLevelData(lv)

		keyPos := make(map[[2]int]bool, len(g.keyPos))
		for pos := range g.keyPos {
			keyPos[pos] = true
		}
		for _, k := range engine.ParseKeys(lv.Solution) {
			g.ed.Feed(k)
			delete(keyPos, [2]int{g.ed.Row(), g.ed.Col()})
		}
		cell := g.cellAt(g.ed.Row(), g.ed.Col())
		if len(keyPos) != 0 || cell != '$' {
			t.Fatalf("[iter %d] :drill f 생성기 해로 클리어 안 됨: 남은 keyPos=%v cell=%q map=%v sol=%q",
				i, keyPos, cell, lv.Map, lv.Solution)
		}
	}
}

// TestDrillBugGeneratorSolvable 은 B2: :drill x 가 생성하는 문제 100개가
// 전부 생성기 자신의 그리디 해(hjkl + x)로 클리어되는지 확인한다. g.ed.Feed
// 가 아니라 g.feed(게임 레벨 디스패치)로 흘려 넣어야 A5(x 를 제자리 치환으로
// 처리)를 실제로 타서, 버그가 출구/다른 버그와 같은 줄에서 왼쪽에 있는
// 경우까지 좌표 desync 없이 검증된다.
func TestDrillBugGeneratorSolvable(t *testing.T) {
	rng := rand.New(rand.NewSource(4))
	for i := 0; i < 100; i++ {
		lv := generateDrillBug(rng)

		g := &Game{store: store.New()}
		g.progress = g.store.Load()
		g.loadLevelData(lv)

		for _, k := range engine.ParseKeys(lv.Solution) {
			g.feed(k)
		}
		if g.PestsLeft() != 0 {
			t.Fatalf("[iter %d] :drill x 생성기 해 이후 버그가 남음: map=%v sol=%q", i, lv.Map, lv.Solution)
		}
		if cell := g.cellAt(g.ed.Row(), g.ed.Col()); cell != '$' {
			t.Fatalf("[iter %d] :drill x 생성기 해로 출구 도달 못함: cell=%q map=%v sol=%q", i, cell, lv.Map, lv.Solution)
		}
	}
}

// TestLoadLevelDataSetsCursorDcol 은 '@' 가 col>0 인 navigate 레벨에서 첫
// 입력이 수직 모션(j/k)이어도 col0 으로 튀지 않는지 확인한다. loadLevelData
// 가 e.row/e.col 만 옮기고 e.dcol(수직 모션의 목표 열)은 그대로 두면 j/k 가
// dcol=0(SetLines 의 초기값)을 써서 커서가 엉뚱한 열로 이동한다 — :drill
// 생성기가 무작위 시작열을 만들면서 실제로 재현해 잡아낸 결함이다.
func TestLoadLevelDataSetsCursorDcol(t *testing.T) {
	g := &Game{store: store.New()}
	g.progress = g.store.Load()
	g.loadLevelData(Level{
		Kind: "navigate",
		Map:  []string{"....@....", ".........", "....$...."},
	})
	if g.ed.Col() != 4 {
		t.Fatalf("초기 col=%d want 4", g.ed.Col())
	}
	g.ed.Feed(engine.RuneKey('j')) // 수직 모션 — dcol 이 4 로 맞춰져 있어야 col 유지
	if g.ed.Col() != 4 {
		t.Fatalf("j 이후 col=%d want 4 (dcol 이 시작 열에 맞춰지지 않음)", g.ed.Col())
	}
}
