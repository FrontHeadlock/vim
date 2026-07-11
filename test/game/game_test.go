// Package gametest 는 게임 규칙의 블랙박스 테스트다 — 공개 API 만 쓴다.
// 키는 실제 프론트엔드와 동일하게 Input() 으로만 흘려보내므로, 상태 전환·
// strokes·bell 같은 규칙이 "실제 입력 경로 그대로" 검증된다.
package gametest

import (
	"math/rand"
	"strings"
	"testing"
	"unicode/utf8"

	"vimquest/internal/engine"
	. "vimquest/internal/game"
	"vimquest/internal/store"
)

// newGame 은 인메모리 저장소를 주입한 게임을 만든다 — 테스트가 이 머신의
// 실제 저장 파일(진행 상황)을 읽거나 쓰는 일이 없도록 명시적으로 격리한다
// (예전엔 store.New() 안의 testing.Testing() 암묵 감지가 맡던 보장).
func newGame() *Game {
	return New(store.NewMem())
}

// feedKeys 는 입력 문자열을 파싱해 엔진에 직접 흘려보낸다(게임 규칙을 거치지
// 않는 순수 엔진 검증용 — 게임 경유는 playKeys 를 쓸 것).
func feedKeys(e *engine.Editor, s string) {
	for _, k := range engine.ParseKeys(s) {
		e.Feed(k)
	}
}

// playKeys 는 키 문자열을 게임의 단일 입력 진입점(Input)으로 흘려보낸다 —
// 승리 판정·strokes 카운트까지 실제 플레이와 동일하게 돈다.
func playKeys(g *Game, s string) {
	for _, k := range engine.ParseKeys(s) {
		g.Input(k)
	}
}

// ex 는 ":cmd<cr>" 를 실제 키 입력으로 실행한다.
func ex(g *Game, cmd string) {
	g.Input(engine.RuneKey(':'))
	for _, r := range cmd {
		g.Input(engine.RuneKey(r))
	}
	g.Input(engine.SpecialKey("cr"))
}

// exActive 는 ex-command 입력 중인지 여부.
func exActive(g *Game) bool {
	_, active := g.ExLine()
	return active
}

// levelIndexByID 는 커리큘럼에서 해당 ID 의 인덱스를 찾는다(없으면 -1).
func levelIndexByID(id string) int {
	for i := 0; i < LevelCount(); i++ {
		if LevelAt(i).ID == id {
			return i
		}
	}
	return -1
}

// reachAllClear 는 마지막 레벨을 Solution 으로 클리어해 StateAllClear 에
// 도달한 게임을 돌려준다.
func reachAllClear(t *testing.T) *Game {
	t.Helper()
	g := newGame()
	last := LevelCount() - 1
	g.LoadLevel(last)
	playKeys(g, LevelAt(last).Solution)
	if g.State() != StateAllClear {
		t.Fatalf("마지막 레벨 Solution 으로 AllClear 도달 실패: state=%v", g.State())
	}
	return g
}

// TestEditLevelsSolvable 은 모든 edit 레벨이 의도된 Solution 으로 Target 에
// 정확히 도달하는지 검증한다 — 풀이 불가능한 퍼즐을 출시 전에 잡아낸다.
// 여러 줄 charwise 비주얼 선택이 "줄 단위로 대체 처리"되는 가드도 겸한다 —
// 이는 알려진 부정확성이라, 레벨 저작이 실수로 이 경로를 밟으면 여기서
// 즉시 실패해야 한다.
func TestEditLevelsSolvable(t *testing.T) {
	engine.ResetMultilineCharwiseFallbackCount()
	for i := 0; i < LevelCount(); i++ {
		lv := LevelAt(i)
		if lv.Kind != "edit" {
			continue
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
		t.Errorf("레벨 Solution 이 여러 줄 charwise 대체 경로를 %d회 밟음 — "+
			"실제 Vim 과 다르게 동작하는 부정확한 경로이므로 레벨 설계를 바꿔야 함", n)
	}
}

// TestNavigateLevelsValid 은 navigate 레벨의 맵 구조를 검증한다.
func TestNavigateLevelsValid(t *testing.T) {
	for i := 0; i < LevelCount(); i++ {
		lv := LevelAt(i)
		if lv.Kind != "navigate" {
			continue
		}
		ats, dollars := 0, 0
		for _, row := range lv.Map {
			ats += strings.Count(row, "@")
			dollars += strings.Count(row, "$")
		}
		if ats != 1 {
			t.Errorf("[%s] '@' 시작 위치가 %d개 (1개여야 함)", lv.ID, ats)
		}
		if dollars < 1 {
			t.Errorf("[%s] '$' 출구가 없음", lv.ID)
		}
	}
}

// TestNavigateSolveLevel1 은 1-1 을 이동만으로 클리어해 클리어 화면으로 전환되는지 본다.
func TestNavigateSolveLevel1(t *testing.T) {
	g := newGame()
	if g.Level().Kind != "navigate" {
		t.Fatal("레벨 1-1 이 navigate 가 아님")
	}
	playKeys(g, "jjllll") // 열쇠(row2,col4) 획득
	if g.KeysLeft() != 0 {
		t.Fatalf("열쇠 미획득: 남은 %d", g.KeysLeft())
	}
	playKeys(g, "jjlllll") // 출구(row4,col9) 도달 → 클리어 화면
	if g.State() != StateLevelClear {
		t.Fatalf("출구 도달 후 클리어 상태 전환 실패: state=%v", g.State())
	}
}

// TestNavigateLevelsSolvable 은 navigate 레벨 전부가 Solution 키 시퀀스로
// 실제로 클리어(StateLevelClear/StateAllClear 전환)되는지 검증한다.
func TestNavigateLevelsSolvable(t *testing.T) {
	for i := 0; i < LevelCount(); i++ {
		lv := LevelAt(i)
		if lv.Kind != "navigate" {
			continue
		}
		g := newGame()
		g.LoadLevel(i)
		playKeys(g, lv.Solution)
		if g.State() != StateLevelClear && g.State() != StateAllClear {
			t.Errorf("[%s] Solution %q 로 클리어 실패 (state=%v)", lv.ID, lv.Solution, g.State())
		}
	}
}

// TestLevel16NaiveSolveIsWorse 는 "1-6" 보너스 레벨이 f/t 랜드마크 없이
// (hjkl 만으로) 풀면 par 대비 1.5배를 넘는 타수가 드는지 확인한다 — 카운트·
// find 조합 없이는 par 급 클리어가 불가능하도록 설계됐음을 실측으로 보증한다.
func TestLevel16NaiveSolveIsWorse(t *testing.T) {
	idx := levelIndexByID("1-6")
	if idx < 0 {
		t.Fatal("레벨 1-6 을 찾을 수 없음")
	}
	lv := LevelAt(idx)
	par := len(engine.ParseKeys(lv.Solution))

	// naive: hjkl 만으로 왼쪽 끝(뒤쪽) 열쇠까지 갔다가 오른쪽 끝 출구까지
	// — 중간의 두 번째 열쇠는 지나가는 길에 자동 습득된다.
	row := lv.Map[0]
	startCol := strings.IndexRune(row, '@')
	e := engine.NewEditor([]string{row})
	e.SetCursor(0, startCol)
	naiveKeys := strings.Repeat("h", startCol) + strings.Repeat("l", len([]rune(row))-1)
	feedKeys(e, naiveKeys)
	if e.Col() != len([]rune(row))-1 {
		t.Fatalf("naive 해가 출구 칸에 도달 못함: col=%d want %d", e.Col(), len([]rune(row))-1)
	}
	naive := len(naiveKeys)
	if float64(naive) <= float64(par)*1.5 {
		t.Fatalf("naive(%d) 가 par(%d)*1.5=%.1f 를 넘지 않음 — 카운트/find 없이도 2★ 이상 가능",
			naive, par, float64(par)*1.5)
	}
}

// naiveFromMacroSolution 은 "qa<body>q<count>@a" 형태의 W9 Solution 에서
// <body>(기록된 편집 한 줄 분)를 뽑아 total 회(=레벨의 줄 수) 반복한
// "매크로 없이 손타이핑" 시퀀스를 만든다. W9 레벨 전부가 레지스터 a 하나만
// 쓰고, 기록 본문 안에 리터럴 'q' 문자가 없다는 저작 관례를 전제한다(둘 다
// TestEditLevelsSolvable/이 함수 자체가 앞으로 깨지면 바로 드러난다).
func naiveFromMacroSolution(t *testing.T, id, sol string, total int) string {
	t.Helper()
	if !strings.HasPrefix(sol, "qa") {
		t.Fatalf("[%s] Solution이 \"qa\"로 시작하지 않음(W9 저작 관례 위반): %q", id, sol)
	}
	rest := sol[len("qa"):]
	idx := strings.IndexByte(rest, 'q')
	if idx < 0 {
		t.Fatalf("[%s] Solution에서 기록 종료 \"q\"를 못 찾음: %q", id, sol)
	}
	body := rest[:idx]
	return strings.Repeat(body, total)
}

// TestW9NaiveSolveIsWorse 는 W9(매크로) 레벨을 매크로 없이 손타이핑(같은
// 편집을 줄마다 직접 반복)해도 클리어는 되지만, par(매크로 사용 기준) 대비
// 1.5배를 넘는 타수가 드는지 확인한다 — "매크로 없이는 par 급이 불가능"함을
// 실측으로 보증한다(패턴은 TestLevel16NaiveSolveIsWorse와 동일).
func TestW9NaiveSolveIsWorse(t *testing.T) {
	for i := 0; i < LevelCount(); i++ {
		lv := LevelAt(i)
		if !strings.HasPrefix(lv.ID, "9-") {
			continue
		}
		par := len(engine.ParseKeys(lv.Solution))
		naiveKeys := naiveFromMacroSolution(t, lv.ID, lv.Solution, len(lv.Map))

		e := engine.NewEditor(append([]string(nil), lv.Map...))
		feedKeys(e, naiveKeys)
		got := strings.Join(e.Lines(), "\n")
		want := strings.Join(lv.Target, "\n")
		if got != want {
			t.Errorf("[%s] naive(매크로 없는) 시퀀스가 Target에 도달 못함\n  got:  %q\n  want: %q", lv.ID, got, want)
			continue
		}

		naive := len(engine.ParseKeys(naiveKeys))
		if float64(naive) <= float64(par)*1.5 {
			t.Errorf("[%s] naive(%d)가 par(%d)*1.5=%.1f를 넘지 않음 — 매크로 없이도 2★ 이상 가능",
				lv.ID, naive, par, float64(par)*1.5)
		}
	}
}

// TestBonusLandmarkLevelsNaiveSolveIsWorse 는 1-6 과 같은 패턴으로 추가된
// 보너스 navigate 레벨(2-4/5-5/6-6 — K 가 시작 뒤에 있어 hjkl 만으로는 왼쪽
// 끝까지 갔다가 오른쪽 끝까지 되짚어야 하는 구조)이 전부 hjkl 만으로 풀면
// par 대비 1.5배를 넘는 타수가 드는지 확인한다(로직은 TestLevel16NaiveSolveIsWorse
// 와 동일 — 대상 레벨만 여러 개로 확장).
func TestBonusLandmarkLevelsNaiveSolveIsWorse(t *testing.T) {
	for _, id := range []string{"2-4", "5-5", "6-6"} {
		idx := levelIndexByID(id)
		if idx < 0 {
			t.Fatalf("레벨 %s 을 찾을 수 없음", id)
		}
		lv := LevelAt(idx)
		par := len(engine.ParseKeys(lv.Solution))

		row := lv.Map[0]
		startCol := strings.IndexRune(row, '@')
		e := engine.NewEditor([]string{row})
		e.SetCursor(0, startCol)
		naiveKeys := strings.Repeat("h", startCol) + strings.Repeat("l", len([]rune(row))-1)
		feedKeys(e, naiveKeys)
		if e.Col() != len([]rune(row))-1 {
			t.Errorf("[%s] naive 해가 출구 칸에 도달 못함: col=%d want %d", id, e.Col(), len([]rune(row))-1)
			continue
		}
		naive := len(naiveKeys)
		if float64(naive) <= float64(par)*1.5 {
			t.Errorf("[%s] naive(%d) 가 par(%d)*1.5=%.1f 를 넘지 않음 — hjkl 만으로도 2★ 이상 가능",
				id, naive, par, float64(par)*1.5)
		}
	}
}

// TestBonusEditLevelsNaiveSolveIsWorse 는 W3/W4/W7/W8 에 추가된 보너스
// edit 레벨(count/text object/visual-select/yank-paste 를 안 쓰고 손타이핑만
// 반복하면 par 대비 1.5배를 넘는지) 을 확인한다 — 각 레벨의 "naive" 시퀀스는
// 해당 레벨이 가르치려는 효율 기법(count·text object·visual·yank)을 의도적으로
// 배제한, 그래도 Target 엔 도달하는 손타이핑 경로다.
func TestBonusEditLevelsNaiveSolveIsWorse(t *testing.T) {
	cases := []struct {
		id    string
		naive string // count/text-object/visual/yank 없이 손타이핑만으로 Target 도달
	}{
		// 3-7: "d6w"(count) 대신 "dw" 를 6번.
		{"3-7", "w" + strings.Repeat("dw", 6)},
		// 4-6: "di("(text object) 대신 괄호 안 23글자를 x 로 하나씩.
		{"4-6", "f(l" + strings.Repeat("x", 23)},
		// 7-5: "v" + count 모션(4e) 대신 4단어치(18글자) 를 x 로 하나씩.
		{"7-5", "w" + strings.Repeat("x", 18)},
		// 8-7: yank+paste 대신 매번 새 줄을 열어 직접 타이핑 3회.
		{"8-7", strings.Repeat("otemplate line<esc>", 3)},
	}
	for _, c := range cases {
		idx := levelIndexByID(c.id)
		if idx < 0 {
			t.Fatalf("레벨 %s 을 찾을 수 없음", c.id)
		}
		lv := LevelAt(idx)
		par := len(engine.ParseKeys(lv.Solution))

		e := engine.NewEditor(append([]string(nil), lv.Map...))
		feedKeys(e, c.naive)
		got := strings.Join(e.Lines(), "\n")
		want := strings.Join(lv.Target, "\n")
		if got != want {
			t.Errorf("[%s] naive 시퀀스가 Target에 도달 못함\n  got:  %q\n  want: %q", c.id, got, want)
			continue
		}

		naive := len(engine.ParseKeys(c.naive))
		if float64(naive) <= float64(par)*1.5 {
			t.Errorf("[%s] naive(%d)가 par(%d)*1.5=%.1f를 넘지 않음 — 효율 기법 없이도 2★ 이상 가능",
				c.id, naive, par, float64(par)*1.5)
		}
	}
}

// TestNavigateBlocksEditing 은 navigate 레벨에서 편집키가 막히는지 확인한다.
func TestNavigateBlocksEditing(t *testing.T) {
	g := newGame() // 1-1
	before := strings.Join(g.Editor().Lines(), "\n")
	playKeys(g, "dd") // dd 시도 — 막혀야 함
	after := strings.Join(g.Editor().Lines(), "\n")
	if before != after {
		t.Errorf("navigate 레벨에서 편집이 허용됨:\n  before %q\n  after  %q", before, after)
	}
}

// TestNavigateAllowsSearch 는 navigate 레벨에서 검색(/ ? n N)이 막히지 않는지 확인한다.
func TestNavigateAllowsSearch(t *testing.T) {
	g := newGame() // 1-1: "@........." 등 5줄
	g.Input(engine.RuneKey('/'))
	if !g.Editor().Searching() {
		t.Fatal("navigate 레벨에서 '/' 가 막힘 — searching 진입 실패")
	}
	playKeys(g, "K")
	g.Input(engine.SpecialKey("cr"))
	if g.Editor().Row() != 2 || g.Editor().Col() != 4 {
		t.Fatalf("navigate 레벨에서 검색 이동 실패: row=%d col=%d want 2,4", g.Editor().Row(), g.Editor().Col())
	}
}

// firstMeaningfulRuneOfCmdToken 은 Cmd.K 토큰("f{char}", "{N}G", "<cr>",
// "(type)" 등)에서 실제로 입력되는 첫 키를 추출한다. 특수키 표기(<...>)나
// 설명용 텍스트(예: "(type)")는 0을 돌려준다(호출자가 건너뜀).
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

// TestNavigateAllowsAllTaughtKeys 는 모든 navigate 레벨이 가르치는 키(메타의
// Cmds)가 실제 입력 경로에서 막히지(visual bell) 않는지 확인한다 — 새
// navigate 레벨이 가르치는 키를 허용 목록에 추가하는 걸 잊는 사고("가르치는
// 키가 막히는" 버그)를 실제 Input 경유 동작으로 잡는다.
func TestNavigateAllowsAllTaughtKeys(t *testing.T) {
	for i := 0; i < LevelCount(); i++ {
		lv := LevelAt(i)
		if lv.Kind != "navigate" {
			continue
		}
		for _, cmd := range MetaFor(lv.ID).Cmds {
			for _, tok := range strings.Fields(cmd.K) {
				r := firstMeaningfulRuneOfCmdToken(tok)
				if r == 0 {
					continue
				}
				g := newGame()
				g.LoadLevel(i) // 키마다 새 게임 — 이전 키의 대기 상태(f 인자 등) 격리
				g.Input(engine.RuneKey(r))
				if g.BellActive() {
					t.Errorf("[%s] 가르치는 키 %q(토큰 %q)가 차단됨(visual bell)",
						lv.ID, string(r), tok)
				}
			}
		}
	}
}

// TestLevelMetaComplete 는 모든 커리큘럼 레벨에 표시 데이터(제목·힌트·명령
// 팔레트)가 빠짐없이 있는지 확인한다 — 표시 데이터가 levels_meta.go 로
// 분리되면서(웹은 생성 파일 levels_meta.js 로 소비) 새 레벨 추가 시 메타
// 등록을 잊는 사고를 잡는다.
func TestLevelMetaComplete(t *testing.T) {
	for i := 0; i < LevelCount(); i++ {
		lv := LevelAt(i)
		m := MetaFor(lv.ID)
		if m.Title == "" {
			t.Errorf("[%s] 메타에 Title 없음", lv.ID)
		}
		if m.Hint == "" {
			t.Errorf("[%s] 메타에 Hint 없음", lv.ID)
		}
		if len(m.Cmds) == 0 {
			t.Errorf("[%s] 메타에 Cmds 없음", lv.ID)
		}
	}
}

// TestVisualBellOnBlockedKey 는 막힌 키 입력 시 visual bell 이 켜지는지 확인한다.
func TestVisualBellOnBlockedKey(t *testing.T) {
	g := newGame()               // 1-1, navigate
	g.Input(engine.RuneKey('d')) // 편집키 — 막혀야 함
	if !g.BellActive() {
		t.Fatal("막힌 키 입력인데 visual bell 이 켜지지 않음")
	}
}

// TestNoBellOnValidKey 는 정상 입력에서는 visual bell 이 발동하지 않는지 확인한다.
func TestNoBellOnValidKey(t *testing.T) {
	g := newGame()
	g.Input(engine.RuneKey('l')) // 정상 이동
	if g.BellActive() {
		t.Fatal("정상 입력인데 visual bell 이 켜짐")
	}
}

// TestBugKillFiresEffect 는 버그 처치 시 문자 치환 이펙트가 생성되는지 확인한다.
func TestBugKillFiresEffect(t *testing.T) {
	g := newGame()
	g.LoadLevel(levelIndexByID("1-5")) // 1-5: (0,3)에 버그
	playKeys(g, "lllx")
	if _, ok := g.EffectAt(0, 3); !ok {
		t.Fatal("버그 처치 후 (0,3)에 이펙트가 생성되지 않음")
	}
}

// TestNavigateBugKillPreservesCoordinates 는 회귀 테스트: 버그가 같은 줄에서
// 열쇠/출구보다 왼쪽에 있을 때 x 로 처치해도 그 줄의 다른 좌표가 밀리지
// 않는지 확인한다. 물리적 삭제를 타면 버그 오른쪽의 모든 문자가 한 칸씩
// 왼쪽으로 밀려, keyPos(레벨 로드 시 고정 캡처)와 라이브 판정 좌표가 어긋난다.
func TestNavigateBugKillPreservesCoordinates(t *testing.T) {
	lv := Level{ID: "x-desync-regress", Kind: "navigate", Map: []string{"@.*..K...$"}}
	g := newGame()
	g.LoadCustomLevel(lv)

	playKeys(g, "llx") // 버그(col2)로 이동해 처치

	line := g.Editor().Lines()[0]
	if len(line) != len(lv.Map[0]) {
		t.Fatalf("버그 처치 후 줄 길이가 바뀜: got %q(len %d) want len %d", line, len(line), len(lv.Map[0]))
	}
	if ch, _ := g.Editor().Cell(0, 2); ch != '.' {
		t.Fatalf("버그 위치(col2)가 '.'로 치환되지 않음: got %q", ch)
	}
	if ch, _ := g.Editor().Cell(0, 5); ch != 'K' {
		t.Fatalf("열쇠 위치(col5)가 밀림: got %q want 'K'", ch)
	}
	if !g.HasKeyAt(0, 5) {
		t.Fatal("keyPos(0,5) 가 더 이상 유효하지 않음 — 실제 K 위치와 desync")
	}
	if ch, _ := g.Editor().Cell(0, 9); ch != '$' {
		t.Fatalf("출구 위치(col9)가 밀림: got %q want '$'", ch)
	}
}

// TestExCommandQ 는 :q 로 레벨 선택 화면으로 전환되는지 확인한다.
func TestExCommandQ(t *testing.T) {
	g := newGame()
	ex(g, "q")
	if g.State() != StateLevelSelect {
		t.Fatalf(":q 후 state=%v want StateLevelSelect", g.State())
	}
}

// TestExCommandQInDrillShowsSummary 는 :drill 세션 중 ":q"를 치면 바로 레벨
// 선택으로 나가는 대신 StateDrillSummary(세션 통계 요약)로 전환되고, 거기서
// 아무 키나 누르면 그제서야 레벨 선택으로 넘어가는지 확인한다.
func TestExCommandQInDrillShowsSummary(t *testing.T) {
	g := newGame()
	ex(g, "drill")
	lv := g.Level()
	playKeys(g, lv.Solution) // 한 문제 클리어해 통계를 누적(streak>=1)
	if g.State() != StateDrill {
		t.Fatalf("드릴 문제 클리어 후 state=%v want StateDrill(다음 문제로 이어짐)", g.State())
	}
	streak := g.DrillStreak()
	totalKeys, totalPar := g.DrillTotals()
	if streak == 0 {
		t.Fatal("드릴 클리어 후에도 streak 이 누적되지 않음")
	}

	ex(g, "q")
	if g.State() != StateDrillSummary {
		t.Fatalf(":drill 중 :q 후 state=%v want StateDrillSummary", g.State())
	}
	if s2, k2, p2 := g.DrillStreak(), totalKeys, totalPar; s2 != streak || k2 != totalKeys || p2 != totalPar {
		t.Fatalf("요약 화면 진입 시 통계가 바뀜: streak=%d(전 %d) keys=%d(전 %d) par=%d(전 %d)",
			s2, streak, k2, totalKeys, p2, totalPar)
	}

	g.Input(engine.RuneKey('x')) // 임의의 키 — 재도전/클리어 등 다른 의미가 없어야 함
	if g.State() != StateLevelSelect {
		t.Fatalf("요약 화면에서 아무 키 입력 후 state=%v want StateLevelSelect", g.State())
	}
}

// TestStrokesExemptExCommand 는 ':' 진입과 그 이후 ex-command 입력이 strokes
// 를 증가시키지 않는지 확인한다 — ':help' 를 열어봐도 별점 손해가 없어야 한다.
func TestStrokesExemptExCommand(t *testing.T) {
	g := newGame()
	before := g.Strokes()
	ex(g, "help")
	if g.Strokes() != before {
		t.Fatalf("':help<cr>' 이후 strokes 가 변함: before=%d after=%d", before, g.Strokes())
	}
}

// TestExCommandDrillKindDispatch 는 ":drill"/":drill w"/":drill f"/":drill x"
// 가 각각 올바른 생성기 유형으로 드릴을 시작하는지 확인한다(레벨 Title 의
// 대괄호 표기로 판별 — HUD 표시와 같은 소스).
func TestExCommandDrillKindDispatch(t *testing.T) {
	cases := []struct{ cmd, title string }{
		{"drill", "DRILL"},
		{"drill w", "DRILL [w]"},
		{"drill f", "DRILL [f]"},
		{"drill x", "DRILL [x]"},
	}
	for _, c := range cases {
		g := newGame()
		ex(g, c.cmd)
		if g.State() != StateDrill {
			t.Fatalf("[%q] 이후 state=%v want StateDrill", c.cmd, g.State())
		}
		if g.Level().Title != c.title {
			t.Fatalf("[%q] 레벨 Title=%q want %q", c.cmd, g.Level().Title, c.title)
		}
	}
}

// TestExCommandRestart 는 :restart 로 현재 레벨이 리로드되는지 확인한다.
func TestExCommandRestart(t *testing.T) {
	g := newGame()
	playKeys(g, "jjllll") // 열쇠 획득해 strokes/keyPos 변화를 만든다
	ex(g, "restart")
	if g.Strokes() != 0 || g.KeysLeft() == 0 {
		t.Fatalf(":restart 후 레벨이 리로드되지 않음: strokes=%d keysLeft=%d", g.Strokes(), g.KeysLeft())
	}
}

// TestExCommandRestartInDrillStaysInDrill 은 :drill 중에 :restart 를 치면
// 같은 드릴 문제가 strokes=0 으로 재시작되고 StateDrill 이 유지되는지 확인한다.
func TestExCommandRestartInDrillStaysInDrill(t *testing.T) {
	g := newGame()
	g.LoadLevel(2) // levelIdx=2 로 이동 — :restart 가 여기로 새는지 구분하기 위함
	ex(g, "drill")
	lv := g.Level() // 드릴이 생성한 문제(맵/해)를 기억해둔다
	g.Input(engine.RuneKey('l'))
	before := g.Strokes()
	ex(g, "restart")
	if g.State() != StateDrill {
		t.Fatalf(":drill 중 :restart 가 드릴을 벗어남: state=%v want StateDrill", g.State())
	}
	if g.LevelIndex() != 2 {
		t.Fatalf(":drill 중 :restart 가 커리큘럼 레벨로 샘: levelIdx=%d", g.LevelIndex())
	}
	if g.Strokes() != 0 {
		t.Fatalf(":restart 후 strokes 가 리셋되지 않음: before=%d after=%d", before, g.Strokes())
	}
	if g.Level().Solution != lv.Solution {
		t.Fatalf(":restart 가 같은 드릴 문제를 유지하지 않음: got %q want %q", g.Level().Solution, lv.Solution)
	}
}

// TestRestartCurrentIsDrillAware 는 RestartCurrent() 자체를 직접 호출해도
// (즉 :restart ex-command 경로를 거치지 않고, 웹 RESET 버튼이 부르는 것과
// 동일하게) 드릴 인식이 유지되는지 확인한다 — RestartCurrent 가 재시작의
// 유일한 진입점이라는 계약을 이 테스트가 직접 지킨다.
func TestRestartCurrentIsDrillAware(t *testing.T) {
	g := newGame()
	g.LoadLevel(2)
	ex(g, "drill")
	lv := g.Level()
	g.Input(engine.RuneKey('l'))
	g.RestartCurrent()
	if g.State() != StateDrill || g.LevelIndex() != 2 || g.Strokes() != 0 || g.Level().Solution != lv.Solution {
		t.Fatalf("RestartCurrent() 가 드릴을 벗어남: state=%v levelIdx=%d strokes=%d solution=%q want StateDrill,2,0,%q",
			g.State(), g.LevelIndex(), g.Strokes(), g.Level().Solution, lv.Solution)
	}
}

// TestExCommandGotoLine 은 :{N} 이 실제로 N번째 줄로 이동하는지 확인한다.
func TestExCommandGotoLine(t *testing.T) {
	g := newGame()
	g.LoadLevel(levelIndexByID("3-2")) // 4줄짜리 edit 레벨
	ex(g, "3")
	if g.Editor().Row() != 2 {
		t.Fatalf(":3 후 row=%d want 2", g.Editor().Row())
	}
}

// TestExCommandEscCancels 는 esc 로 ex-command 입력을 취소하면 상태가 그대로인지 확인한다.
func TestExCommandEscCancels(t *testing.T) {
	g := newGame()
	before := g.State()
	g.Input(engine.RuneKey(':'))
	g.Input(engine.RuneKey('q'))
	g.Input(engine.SpecialKey("esc"))
	if exActive(g) || g.State() != before {
		t.Fatal("esc 취소 후 exMode 가 남아있거나 state 가 바뀜")
	}
}

// TestExCommandUnknownIgnored 는 인식 못하는 명령이 조용히 무시되는지 확인한다.
func TestExCommandUnknownIgnored(t *testing.T) {
	g := newGame()
	before := g.State()
	ex(g, "bogus")
	if exActive(g) || g.State() != before {
		t.Fatalf("알 수 없는 명령 처리 후 상태 이상: exActive=%v state=%v", exActive(g), g.State())
	}
}

// TestColonInInsertModeIsLiteral 은 Insert 모드에서 ':' 이 ex-command 로 가로채이지
// 않고 그냥 문자로 입력되는지 확인한다.
func TestColonInInsertModeIsLiteral(t *testing.T) {
	g := newGame()
	g.LoadLevel(levelIndexByID("3-4")) // edit 레벨(Insert 모드 진입 가능)
	g.Input(engine.RuneKey('A'))
	g.Input(engine.RuneKey(':'))
	if exActive(g) {
		t.Fatal("Insert 모드에서 ':' 이 ex-command 로 가로채임")
	}
	if !strings.Contains(strings.Join(g.Editor().Lines(), "\n"), ":") {
		t.Fatal("Insert 모드에서 ':' 이 문자로 입력되지 않음")
	}
}

// TestInputLevelClearEnterAdvances 는 클리어 화면에서 Input(cr) 이 다음 레벨로
// 넘어가는지 확인한다.
func TestInputLevelClearEnterAdvances(t *testing.T) {
	g := newGame()
	playKeys(g, "jjlllljjlllll") // 1-1 클리어 → StateLevelClear
	if g.State() != StateLevelClear {
		t.Fatalf("사전조건 실패: state=%v want StateLevelClear", g.State())
	}
	g.Input(engine.SpecialKey("cr"))
	if g.LevelIndex() != 1 || g.State() != StatePlaying {
		t.Fatalf("Enter 후 levelIdx=%d state=%v want 1,StatePlaying", g.LevelIndex(), g.State())
	}
}

// TestClearIsNewFlag 는 ClearStats.IsNew 가 최초 클리어/기록 미갱신 두
// 경우에서 올바른지 확인한다(렌더러가 직접 재계산하지 않고 이 값만 읽는다).
func TestClearIsNewFlag(t *testing.T) {
	g := newGame()
	playKeys(g, "jjlllljjlllll")
	if !g.LastClear().IsNew {
		t.Fatal("최초 클리어인데 IsNew=false")
	}
	firstStrokes := g.LastClear().Strokes

	g.LoadLevel(0)
	playKeys(g, "h") // 낭비 키 — col0 에서 no-op 이동이지만 strokes 는 증가시킴
	playKeys(g, "jjlllljjlllll")
	if g.LastClear().Strokes <= firstStrokes {
		t.Fatalf("사전조건 실패: 재클리어 타수(%d)가 최초(%d)보다 작지 않음", g.LastClear().Strokes, firstStrokes)
	}
	if g.LastClear().IsNew {
		t.Fatalf("이전 기록(%d)보다 느린 재클리어(%d)인데 IsNew=true", firstStrokes, g.LastClear().Strokes)
	}
}

// TestClearYoursRecordsMyKeys 는 클리어 화면의 "yours" 가 실제로 입력한 키
// 시퀀스를 반영하고, ex-command 로 진입한 키(:q 등)는 포함하지 않는지
// 확인한다(strokes 와 같은 기준).
func TestClearYoursRecordsMyKeys(t *testing.T) {
	g := newGame()
	playKeys(g, "jjllll")
	ex(g, "q")     // ex-command 는 strokes 에서 빠지므로 yours 에도 없어야 함
	g.LoadLevel(0) // :q 가 레벨 선택으로 보냈으므로 다시 로드
	playKeys(g, "jjlllljjlllll")

	yours := g.LastClear().Yours
	if strings.Contains(yours, ":") {
		t.Fatalf("yours 에 ex-command 흔적이 남음: %q", yours)
	}
	if yours != "jjlllljjlllll" {
		t.Fatalf("yours=%q want %q", yours, "jjlllljjlllll")
	}
	if len(yours) != g.LastClear().Strokes {
		t.Fatalf("yours 길이(%d)가 strokes(%d)와 다름", len(yours), g.LastClear().Strokes)
	}
}

// TestInputLevelClearRetry 는 클리어 화면에서 Input('r') 이 같은 레벨을
// strokes=0 으로 리로드하는지 확인한다.
func TestInputLevelClearRetry(t *testing.T) {
	g := newGame()
	playKeys(g, "jjlllljjlllll")
	if g.State() != StateLevelClear {
		t.Fatalf("사전조건 실패: state=%v want StateLevelClear", g.State())
	}
	g.Input(engine.RuneKey('r'))
	if g.LevelIndex() != 0 || g.State() != StatePlaying || g.Strokes() != 0 {
		t.Fatalf("'r' 후 levelIdx=%d state=%v strokes=%d want 0,StatePlaying,0", g.LevelIndex(), g.State(), g.Strokes())
	}
}

// TestInputLevelSelectNavigation 은 레벨 선택 화면에서 h/j/k/l 로 커서를 옮기고
// 잠긴 레벨은 Enter 로 입장되지 않으며 unlocked 레벨은 입장되는지 확인한다.
func TestInputLevelSelectNavigation(t *testing.T) {
	g := newGame()
	g.EnterLevelSelect() // 1-1 위치, W1 selLevel=0

	g.Input(engine.RuneKey('l')) // W2 로 이동 — 2-1 은 아직 잠김
	g.Input(engine.SpecialKey("cr"))
	if g.State() != StateLevelSelect {
		t.Fatalf("잠긴 레벨(2-1) 진입이 막히지 않음: state=%v", g.State())
	}

	g.Input(engine.RuneKey('h')) // W1 로 복귀 — 1-1 은 unlocked
	g.Input(engine.SpecialKey("cr"))
	if g.State() != StatePlaying || g.LevelIndex() != 0 {
		t.Fatalf("unlocked 레벨(1-1) 진입 실패: state=%v levelIdx=%d", g.State(), g.LevelIndex())
	}
}

// TestInputNoStrokesOutsidePlaying 은 StatePlaying 이 아닌 상태에서의 Input 이
// strokes 를 증가시키지 않는지 확인한다(레벨 클리어 화면에서 의미 없는 키 입력).
func TestInputNoStrokesOutsidePlaying(t *testing.T) {
	g := newGame()
	playKeys(g, "jjlllljjlllll") // StateLevelClear
	before := g.Strokes()
	g.Input(engine.RuneKey('x')) // 클리어 화면에서 'r'/'cr' 도 아닌 키
	if g.Strokes() != before {
		t.Fatalf("비-플레이 상태 입력이 strokes 를 증가시킴: before=%d after=%d", before, g.Strokes())
	}
}

// TestInputAllClearReturnsToSelect 는 StateAllClear 화면에서 cr(또는 esc) 이
// 레벨 선택으로 복귀시키는지 확인한다 — 안 그러면 이 상태에서 입력이 전부
// 무시돼 데스크톱에서 앱 종료 외 탈출 수단이 없다(소프트락).
func TestInputAllClearReturnsToSelect(t *testing.T) {
	g := reachAllClear(t)
	g.Input(engine.SpecialKey("cr"))
	if g.State() != StateLevelSelect {
		t.Fatalf("AllClear 에서 cr 후 state=%v want StateLevelSelect", g.State())
	}

	g2 := reachAllClear(t)
	g2.Input(engine.SpecialKey("esc"))
	if g2.State() != StateLevelSelect {
		t.Fatalf("AllClear 에서 esc 후 state=%v want StateLevelSelect", g2.State())
	}
}

// TestParseKeyUTF8 은 웹 입력 경로(ParseKey)가 멀티바이트 UTF-8 토큰을 바이트
// 단위로 잘라 깨진 rune 을 만들지 않는지 확인한다.
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

// TestDrillCapsSessionLength 는 DrillMaxRounds 에 도달하면 :drill 이 새 문제를
// 계속 생성하는 대신 레벨 선택으로 빠지는지 확인한다 — -gc=leaking 웹 빌드에서
// 문제 생성마다 나오는 쓰레기가 세션 내내 무한정 쌓이는 걸 막는 상한.
// 상한까지 실제로 문제를 풀어 도달한다(드릴은 클리어 즉시 다음 문제를 만들고,
// 현재 문제의 Solution 은 Level() 로 읽을 수 있다).
func TestDrillCapsSessionLength(t *testing.T) {
	g := newGame()
	ex(g, "drill")
	rounds := 0
	for g.State() == StateDrill && rounds < DrillMaxRounds+5 {
		// 해를 키 단위로 넣다가 클리어 즉시 끊는다 — 이동 경로가 다른 열쇠·
		// 출구를 우연히 지나며 조기 클리어되면 다음 문제가 곧바로 로드되는데
		// (strokes 0 리셋으로 감지), 남은 키를 계속 흘리면 새 문제의 커서를
		// 오염시켜 다음 라운드 해가 어긋난다.
		for _, k := range engine.ParseKeys(g.Level().Solution) {
			g.Input(k)
			if g.State() != StateDrill || g.Strokes() == 0 {
				break
			}
		}
		rounds++
	}
	if g.State() != StateLevelSelect {
		t.Fatalf("드릴 상한(%d) 도달 후 레벨 선택으로 빠지지 않음: state=%v (rounds=%d)",
			DrillMaxRounds, g.State(), rounds)
	}
	if rounds != DrillMaxRounds {
		t.Fatalf("상한 도달까지 %d 라운드 — want %d", rounds, DrillMaxRounds)
	}
}

// TestDrillGeneratorsSolvable 은 4가지 드릴 생성기("", w, f, x)가 만드는
// 무작위 문제 각 100개가 전부 자신의 Solution 으로 실제 클리어되는지
// 확인한다(property 테스트 — 시드 고정으로 재현 가능). LoadCustomLevel +
// Input 이라는 실제 프로덕션 경로를 그대로 태우므로, x 드릴의 제자리 치환
// 처리(버그가 출구와 같은 줄 왼쪽에 있는 경우)까지 함께 검증된다.
func TestDrillGeneratorsSolvable(t *testing.T) {
	kinds := []struct {
		kind string
		seed int64
	}{{"", 1}, {"w", 2}, {"f", 3}, {"x", 4}}
	for _, k := range kinds {
		rng := rand.New(rand.NewSource(k.seed))
		for i := 0; i < 100; i++ {
			lv := GenerateDrillLevel(k.kind, rng)
			g := newGame()
			g.LoadCustomLevel(lv)
			for _, key := range engine.ParseKeys(lv.Solution) {
				g.Input(key)
				if g.State() != StatePlaying {
					break // 조기 클리어 — 남은 키가 클리어 화면으로 새지 않게
				}
			}
			if g.State() != StateLevelClear {
				t.Fatalf("[kind=%q iter %d] 생성기 해로 클리어 안 됨: state=%v map=%v sol=%q",
					k.kind, i, g.State(), lv.Map, lv.Solution)
			}
		}
	}
}

// TestLoadCustomLevelSetsCursorDcol 은 '@' 가 col>0 인 navigate 레벨에서 첫
// 입력이 수직 모션(j/k)이어도 col0 으로 튀지 않는지 확인한다 — 레벨 로드가
// 커서만 옮기고 dcol(수직 모션의 목표 열)을 안 맞추면 j/k 가 엉뚱한 열로
// 이동한다(:drill 생성기의 무작위 시작열에서 실제로 재현해 잡아낸 결함).
func TestLoadCustomLevelSetsCursorDcol(t *testing.T) {
	g := newGame()
	g.LoadCustomLevel(Level{
		Kind: "navigate",
		Map:  []string{"....@....", ".........", "....$...."},
	})
	if g.Editor().Col() != 4 {
		t.Fatalf("초기 col=%d want 4", g.Editor().Col())
	}
	g.Input(engine.RuneKey('j')) // 수직 모션 — dcol 이 4 로 맞춰져 있어야 col 유지
	if g.Editor().Col() != 4 {
		t.Fatalf("j 이후 col=%d want 4 (dcol 이 시작 열에 맞춰지지 않음)", g.Editor().Col())
	}
}
