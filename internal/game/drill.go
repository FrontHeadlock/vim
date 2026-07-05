package game

// drill.go — :drill 절차 생성 연습 모드. 무작위 navigate 문제를 끝없이
// 생성하되, 항상 생성기 자신이 만든 hjkl 해(Solution)로 검증 가능하다.
// 진행은 세션 한정이며 store 에 저장하지 않는다.

import (
	"math/rand"
	"strings"
	"time"

	"vimquest/internal/platform"
)

const (
	drillRows = 5
	drillCols = 20
)

// drillMaxRounds 는 한 :drill 세션에서 생성할 문제 수의 상한. 웹 빌드는
// 크기를 줄이려고 GC 를 꺼놨으므로(-gc=leaking, build.sh) 문제를 생성할
// 때마다 나오는 자잘한 쓰레기(격자·Editor·해 문자열)가 세션 내내 전혀
// 회수되지 않는다 — 정확히 "반복 연습"이라는 이 기능의 용도에서 무한정
// 늘어날 수 있다는 뜻이라, 라운드 수에 상한을 둬 최악의 경우 메모리 증가를
// 유한하게 묶는다. 이 값(대략 문제당 수 KB 기준 총 수 MB)은 실제 연습
// 세션에선 거의 도달하지 않을 만큼 넉넉하다.
const drillMaxRounds = 1000

// enterDrill 은 :drill 모드로 전환하고 첫 문제를 생성한다.
func (g *Game) enterDrill() {
	if g.drillRng == nil {
		g.drillRng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	g.drillStreak = 0
	g.drillTotalKeys = 0
	g.drillTotalPar = 0
	g.state = StateDrill
	g.loadLevelData(generateDrill(g.drillRng))
}

// advanceDrill 은 :drill 문제를 클리어했을 때 통계를 누적하고 즉시 다음
// 문제를 생성한다 — 클리어 화면을 생략해 템포를 유지한다(반복 훈련이 목적).
func (g *Game) advanceDrill() {
	g.drillStreak++
	g.drillTotalKeys += g.strokes
	g.drillTotalPar += g.Par()
	platform.Sfx("clear")
	if g.drillStreak >= drillMaxRounds {
		g.EnterLevelSelect()
		return
	}
	g.loadLevelData(generateDrill(g.drillRng))
}

// generateDrill 은 무작위 navigate 문제를 만들고, 그리디 해(hjkl 만 사용해
// 항상 유효한 경로)를 Solution 에 채운다 — par 산출과 자동 검증(생성기 기준
// 해로 항상 클리어되어야 함) 양쪽에 쓰인다.
func generateDrill(rng *rand.Rand) Level {
	type pos struct{ r, c int }

	numKeys := 1 + rng.Intn(3) // 1~3

	all := make([]pos, 0, drillRows*drillCols)
	for r := 0; r < drillRows; r++ {
		for c := 0; c < drillCols; c++ {
			all = append(all, pos{r, c})
		}
	}
	rng.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })

	start := all[0]
	exit := all[1]
	keys := append([]pos(nil), all[2:2+numKeys]...)

	grid := make([][]rune, drillRows)
	for r := range grid {
		grid[r] = make([]rune, drillCols)
		for c := range grid[r] {
			grid[r][c] = '.'
		}
	}
	grid[start.r][start.c] = '@'
	grid[exit.r][exit.c] = '$'
	for _, k := range keys {
		grid[k.r][k.c] = 'K'
	}

	lines := make([]string, drillRows)
	for r, row := range grid {
		lines[r] = string(row)
	}

	var sol strings.Builder
	cur := start
	moveTo := func(target pos) {
		for cur.r < target.r {
			sol.WriteByte('j')
			cur.r++
		}
		for cur.r > target.r {
			sol.WriteByte('k')
			cur.r--
		}
		for cur.c < target.c {
			sol.WriteByte('l')
			cur.c++
		}
		for cur.c > target.c {
			sol.WriteByte('h')
			cur.c--
		}
	}
	for _, k := range keys {
		moveTo(k)
	}
	moveTo(exit)

	return Level{
		ID:       "drill",
		Kind:     "navigate",
		Title:    "DRILL",
		Hint:     "Grab every key, then reach the exit — as fast as you can.",
		Map:      lines,
		Solution: sol.String(),
	}
}
