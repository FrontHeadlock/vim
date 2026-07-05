package game

// drill.go — :drill 절차 생성 연습 모드. 무작위 문제를 끝없이 생성하되,
// 항상 생성기 자신이 만든 해(Solution)로 검증 가능하다. 진행은 세션
// 한정이며 store 에 저장하지 않는다.
//
// 인자별로 생성기가 갈린다 — :drill(기본, hjkl 이동) · :drill w(단어 점프) ·
// :drill f(find 배치) · :drill x(버그 소탕). 세션 내내 같은 유형을 반복
// 생성한다(드릴 도중 유형이 바뀌지 않음).

import (
	"math/rand"
	"sort"
	"strings"
	"time"

	"vimquest/internal/platform"
)

const (
	drillRows = 5
	drillCols = 20
)

// DrillMaxRounds 는 한 :drill 세션에서 생성할 문제 수의 상한. 웹 빌드는
// 크기를 줄이려고 GC 를 꺼놨으므로(-gc=leaking, build.sh) 문제를 생성할
// 때마다 나오는 자잘한 쓰레기(격자·Editor·해 문자열)가 세션 내내 전혀
// 회수되지 않는다 — 정확히 "반복 연습"이라는 이 기능의 용도에서 무한정
// 늘어날 수 있다는 뜻이라, 라운드 수에 상한을 둬 최악의 경우 메모리 증가를
// 유한하게 묶는다. 이 값(대략 문제당 수 KB 기준 총 수 MB)은 실제 연습
// 세션에선 거의 도달하지 않을 만큼 넉넉하다.
const DrillMaxRounds = 1000

// enterDrill 은 :drill 모드로 전환하고 kind 에 맞는 생성기로 첫 문제를 만든다.
// kind: ""(기본, hjkl) · "w"(단어 점프) · "f"(find 배치) · "x"(버그 소탕).
// 알 수 없는 kind 는 기본(hjkl)으로 조용히 대체한다(터미널처럼 무반응 —
// runExCommand 의 미인식 명령 처리와 같은 원칙).
func (g *Game) enterDrill(kind string) {
	if g.drillRng == nil {
		g.drillRng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	g.drillKind = kind
	g.drillStreak = 0
	g.drillTotalKeys = 0
	g.drillTotalPar = 0
	g.state = StateDrill
	g.loadLevelData(generateDrillFor(g.drillKind, g.drillRng))
}

// advanceDrill 은 :drill 문제를 클리어했을 때 통계를 누적하고 즉시 같은
// 유형의 다음 문제를 생성한다 — 클리어 화면을 생략해 템포를 유지한다.
func (g *Game) advanceDrill() {
	g.drillStreak++
	g.drillTotalKeys += g.strokes
	g.drillTotalPar += g.Par()
	platform.Sfx("clear")
	if g.drillStreak >= DrillMaxRounds {
		g.EnterLevelSelect()
		return
	}
	g.loadLevelData(generateDrillFor(g.drillKind, g.drillRng))
}

// GenerateDrillLevel 은 kind("", "w", "f", "x")에 맞는 드릴 생성기로 문제
// 하나를 만든다 — 시드 고정 rng 를 주입해 property 테스트(생성 문제가 항상
// 자신의 Solution 으로 풀리는지)를 재현 가능하게 돌리기 위한 공개 진입점.
func GenerateDrillLevel(kind string, rng *rand.Rand) Level {
	return generateDrillFor(kind, rng)
}

// generateDrillFor 는 kind 에 맞는 생성기로 분기한다(B2).
func generateDrillFor(kind string, rng *rand.Rand) Level {
	switch kind {
	case "w":
		return generateDrillWord(rng)
	case "f":
		return generateDrillFind(rng)
	case "x":
		return generateDrillBug(rng)
	default:
		return generateDrill(rng)
	}
}

// generateDrill 은 무작위 navigate 문제(기본 hjkl 유형)를 만든다.
func generateDrill(rng *rand.Rand) Level {
	numKeys := 1 + rng.Intn(3) // 1~3
	lines, sol := drillGridBase(rng, numKeys, 'K', "")
	return Level{
		ID:       "drill",
		Kind:     "navigate",
		Title:    "DRILL",
		Hint:     "Grab every key, then reach the exit — as fast as you can.",
		Map:      lines,
		Solution: sol,
	}
}

// generateDrillBug 는 버그(*) 소탕 연습 문제를 만든다 — 열쇠 없이 버그를
// 전부 처치(x)한 뒤 출구로 가는, hjkl+x 반복 훈련. x 처치는 제자리 치환이라
// (game.go feed 참고) 다른 좌표를 밀지 않는다.
func generateDrillBug(rng *rand.Rand) Level {
	numBugs := 3 + rng.Intn(4) // 3~6
	lines, sol := drillGridBase(rng, numBugs, '*', "x")
	return Level{
		ID:       "drill",
		Kind:     "navigate",
		Title:    "DRILL [x]",
		Hint:     "Clear every bug with x, then reach the exit.",
		Map:      lines,
		Solution: sol,
	}
}

// drillGridBase 는 격자 기반 드릴(기본 hjkl, x 버그 소탕)이 공유하는 뼈대.
// 무작위 시작/출구/수집 대상 좌표를 뽑아 grid 를 채우고, 각 수집 지점에
// hjkl 로 도달할 때마다 perVisit 을 이어붙인 그리디 해를 함께 반환한다
// (perVisit 은 열쇠면 "", 버그면 "x" — 그 지점에서 한 번 더 눌러야 하는 키).
func drillGridBase(rng *rand.Rand, numItems int, itemGlyph rune, perVisit string) ([]string, string) {
	type pos struct{ r, c int }
	all := make([]pos, 0, drillRows*drillCols)
	for r := 0; r < drillRows; r++ {
		for c := 0; c < drillCols; c++ {
			all = append(all, pos{r, c})
		}
	}
	rng.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })

	start := all[0]
	exit := all[1]
	items := append([]pos(nil), all[2:2+numItems]...)

	grid := make([][]rune, drillRows)
	for r := range grid {
		grid[r] = make([]rune, drillCols)
		for c := range grid[r] {
			grid[r][c] = '.'
		}
	}
	grid[start.r][start.c] = '@'
	grid[exit.r][exit.c] = '$'
	for _, p := range items {
		grid[p.r][p.c] = itemGlyph
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
	for _, p := range items {
		moveTo(p)
		sol.WriteString(perVisit)
	}
	moveTo(exit)

	return lines, sol.String()
}

// generateDrillWord 는 단어 모션(w) 연습용 문제를 만든다 — 한 줄에 늘어선
// 여러 단어의 시작 칸에 열쇠/출구를 배치해, hjkl 로 한 칸씩 세는 대신 w 로
// 단어 단위 점프를 강제한다.
func generateDrillWord(rng *rand.Rand) Level {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	numWords := 4 + rng.Intn(3) // 4~6

	var b strings.Builder
	cols := make([]int, numWords) // 각 단어 시작 칸(rune 인덱스)
	for i := 0; i < numWords; i++ {
		if i > 0 {
			b.WriteByte(' ')
		}
		cols[i] = b.Len()
		wl := 2 + rng.Intn(4) // 단어 길이 2~5
		if i == 0 {
			// loadLevelData 가 '@' 를 '.'(구두점 class)로 치환하는데, 다글자
			// 단어의 첫 글자가 그 자리면 남은 글자(알파벳 class)와 클래스가
			// 갈려 단어 하나가 두 토큰으로 쪼개진다 — 첫 w 가 word1 이 아니라
			// word0 의 나머지로만 이동해 그리디 해(w 반복)가 어긋난다. 시작
			// 단어를 한 글자로 고정해 치환 후에도 공백으로 깔끔히 구분되는
			// 단일 토큰이 되게 한다.
			wl = 1
		}
		for j := 0; j < wl; j++ {
			b.WriteByte(letters[rng.Intn(len(letters))])
		}
	}
	line := []rune(b.String())

	// 시작=첫 단어, 출구=마지막 단어, 그 사이 임의 개수를 K 로.
	line[cols[0]] = '@'
	line[cols[numWords-1]] = '$'
	numKeys := 0
	for i := 1; i < numWords-1; i++ {
		if rng.Intn(2) == 0 {
			line[cols[i]] = 'K'
			numKeys++
		}
	}
	if numKeys == 0 {
		mid := cols[numWords/2]
		line[mid] = 'K'
	}

	return Level{
		ID:       "drill",
		Kind:     "navigate",
		Title:    "DRILL [w]",
		Hint:     "Words, not cells — jump with w to grab every key, then reach the exit.",
		Map:      []string{string(line)},
		Solution: strings.Repeat("w", numWords-1),
	}
}

// generateDrillFind 는 f{char} 연습용 문제를 만든다 — 필러 문자로 채운 한
// 줄에 K/$ 를 흩뿌려, l 로 한 칸씩 세는 대신 fK/f$ 로 바로 점프하게 만든다.
func generateDrillFind(rng *rand.Rand) Level {
	const width = 30
	const filler = "abcdefghijklmnopqrstuvwxyz"
	numKeys := 2 + rng.Intn(3) // 2~4

	line := make([]rune, width)
	for i := range line {
		line[i] = rune(filler[rng.Intn(len(filler))])
	}

	// 0번 칸은 시작(@) 전용 — 1..width-1 범위에서 서로 다른 (numKeys+1)개를
	// 뽑아 오름차순 정렬: 앞쪽 numKeys 개는 K, 마지막 하나는 $.
	perm := rng.Perm(width - 1)
	picked := append([]int(nil), perm[:numKeys+1]...)
	for i := range picked {
		picked[i]++
	}
	sort.Ints(picked)

	for _, c := range picked[:numKeys] {
		line[c] = 'K'
	}
	line[picked[numKeys]] = '$'
	line[0] = '@'

	return Level{
		ID:       "drill",
		Kind:     "navigate",
		Title:    "DRILL [f]",
		Hint:     "Don't count cells — f{char} leaps straight to it. Try fK, then f$.",
		Map:      []string{string(line)},
		Solution: strings.Repeat("fK", numKeys) + "f$",
	}
}
