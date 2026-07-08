package game

// arena_levels.go — Arena 시간공격 경쟁 모드. 모든 이용자에게 동일한 고정
// edit 문제 5개를 순서대로 풀고, 실측 시간(JS performance.now())으로
// 경쟁한다. 시간은 전적으로 JS 가 소유한다 — Go/엔진 어디에도 wall-clock
// 개념이 없고(strokes 뿐), "클라이언트 신고 시간 신뢰" 결정과 맞물려 여기서도
// 시간을 재지 않는다.
//
// ArenaLevels 는 커리큘럼 levels 슬라이스에 절대 추가하지 않는다 —
// WorldGroups()/tools/genmeta/저장 진행률과 완전히 격리되고, 진행은
// :drill 처럼 자기 전용 advance 경로(advanceArena)만 탄다. LoadCustomLevel
// 을 그대로 쓰면 클리어 시 advance()→recordClear() 가 남아있는 g.levelIdx
// 기준으로 엉뚱한 커리큘럼 레벨을 잠금 해제하는 버그가 생기기 때문이다.

import "vimquest/internal/platform"

// ArenaLevels 는 Arena 의 고정 5문제. Title/Hint 는 런타임 생성 레벨(:drill)과
// 같은 규칙으로 스냅샷에 실린다 — 커리큘럼이 아니므로 LEVEL_META(생성 파일)에
// 없다.
var ArenaLevels = []Level{
	{
		ID:       "arena-1",
		Kind:     "edit",
		Title:    "ARENA 1/5 — warm-up",
		Hint:     "Operator + motion: dw deletes a word, f jumps, x kills a char.",
		Map:      []string{"the the quick broown fox"},
		Target:   []string{"the quick brown fox"},
		Solution: "dwfox",
	},
	{
		ID:       "arena-2",
		Kind:     "edit",
		Title:    "ARENA 2/5 — lines",
		Hint:     "Whole lines: dd deletes one, O opens a new one above.",
		Map:      []string{"alpha", "JUNK JUNK", "gamma"},
		Target:   []string{"alpha", "beta", "gamma"},
		Solution: "jddObeta<esc>",
	},
	{
		ID:       "arena-3",
		Kind:     "edit",
		Title:    "ARENA 3/5 — precision",
		Hint:     "f{char} to land exactly, cw to change a word, A to append at line end.",
		Map:      []string{"name: foo, age: 99", "print(msg)"},
		Target:   []string{"name: bar, age: 21", "print(msg);"},
		Solution: "ffcwbar<esc>f9cw21<esc>jA;<esc>",
	},
	{
		ID:       "arena-4",
		Kind:     "edit",
		Title:    "ARENA 4/5 — repetition",
		Hint:     "Counts and the dot: d3w once, then . repeats the whole change.",
		Map:      []string{"OLD OLD OLD alpha", "OLD OLD OLD beta", "OLD OLD OLD gamma"},
		Target:   []string{"alpha", "beta", "gamma"},
		Solution: "d3wj.j.",
	},
	{
		ID:       "arena-5",
		Kind:     "edit",
		Title:    "ARENA 5/5 — capstone",
		Hint:     `Combine everything: ci" and ci( text objects, plus dd.`,
		Map:      []string{`conf = "debug"`, "# drop this line", "value(old)"},
		Target:   []string{`conf = "release"`, "value(new)"},
		Solution: `f"ci"release<esc>jddf(ci(new<esc>`,
	},
}

// EnterArena 는 Arena 모드로 전환하고 첫 문제를 로드한다(enterDrill 과 동일한
// 모양). 진행/저장(store)은 전혀 건드리지 않는다.
func (g *Game) EnterArena() {
	g.arenaIdx = 0
	g.state = StateArena
	g.loadLevelData(ArenaLevels[0])
}

// advanceArena 는 Arena 문제를 클리어했을 때 즉시 다음 문제로 넘긴다
// (advanceDrill 과 동일한 모양 — 클리어 화면 없이 템포 유지). 5문제를 다
// 풀면 StateArenaDone 으로 전환하고, JS 가 그 프레임에 시간을 얼려 제출
// UI 를 띄운다. recordClear()/store.Save() 는 절대 부르지 않는다.
func (g *Game) advanceArena() {
	g.arenaIdx++
	platform.Sfx("clear")
	if g.arenaIdx >= len(ArenaLevels) {
		g.state = StateArenaDone
		return
	}
	g.loadLevelData(ArenaLevels[g.arenaIdx])
}
