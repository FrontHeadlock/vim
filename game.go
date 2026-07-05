package main

// game.go — Ebiten 과 완전히 무관한 순수 게임 로직(상태 머신·규칙·저장·DOM 동기화).
// editor.go 와 같은 이유로 여기 있는 모든 것은 headless 로 테스트 가능해야 하고,
// ebiten/image-color 를 import 해선 안 된다(Phase 4 L2 의 TinyGo 웹 빌드 전제).
//
// 렌더링(Draw/draw*)과 Ebiten 폴링→Key 변환(Update)은 main.go 에 남는다.
// 프론트엔드(Ebiten/JS 공용)는 Input(k Key) 하나만 호출하면 된다.

import (
	"math/rand"
	"strconv"
	"strings"
	"time"
)

// gameState 는 게임 전체의 화면 상태를 나타낸다.
type gameState int

const (
	statePlaying     gameState = iota // 레벨 플레이 중
	stateLevelClear                   // 레벨 클리어 요약 화면
	stateLevelSelect                  // 레벨 선택 화면
	stateAllClear                     // 전체 클리어
	stateDrill                        // :drill 절차 생성 연습 모드
)

type Game struct {
	levelIdx int
	lv       Level
	ed       *Editor

	// navigate 상태
	keyPos   map[[2]int]bool // 아직 안 주운 열쇠 위치
	keysNeed int

	state gameState

	strokes int // 이번 레벨에서 누른 키 수(막힌 키 포함)

	// 레벨 클리어 화면에 표시할 스냅샷(advance() 시점에 고정)
	clearStrokes int
	clearPar     int
	clearStars   int
	clearBest    int // 갱신 전 best (0 이면 이전 클리어 기록 없음)

	// 레벨 선택 화면 커서
	selRow, selCol int

	store    ProgressStore
	progress map[string]LevelProgress

	// 터미널식 피드백(2.2): 문자 치환/반전 연출
	effects []Effect
	bellTTL int // > 0 이면 이번 프레임 visual bell(막힌 키) 발동 중

	// : ex-command 라인(2.3)
	exMode bool
	exBuf  []rune

	// :drill 절차 생성 연습 모드(Phase 4 L3) — 세션 한정, 진행 저장 없음.
	drillRng       *rand.Rand
	drillStreak    int
	drillTotalKeys int
	drillTotalPar  int
}

// Effect 는 몇 프레임 동안 표시되는 터미널식 문자 치환/반전 연출 한 건.
// 실제 버퍼 내용은 바꾸지 않고 렌더링에서만 겹쳐 그린다.
type Effect struct {
	Row, Col int
	Glyph    rune // 0 이면 문자 치환 없음(Invert 만 적용)
	Invert   bool
	TTL      int // 남은 프레임 수
}

// fireEvent 는 게임 이벤트(열쇠 획득/버그 처치/막힌 키/레벨 클리어)를
// 사운드(jsSfx)와 화면 연출(spawnEffect) 양쪽에 동시에 통지한다.
func (g *Game) fireEvent(name string, row, col int) {
	jsSfx(name)
	g.spawnEffect(name, row, col)
}

func (g *Game) spawnEffect(name string, row, col int) {
	switch name {
	case "bug":
		g.effects = append(g.effects, Effect{Row: row, Col: col, Glyph: 'x', TTL: 10})
	case "key":
		g.effects = append(g.effects, Effect{Row: row, Col: col, Invert: true, TTL: 6})
	case "blocked":
		g.bellTTL = 2
	}
}

// Tick 은 매 프레임 이펙트 TTL 을 줄이고 만료된 것을 정리한다.
// 키 입력과 무관하게 프론트엔드가 프레임마다(또는 이펙트가 살아있는 동안만) 호출한다.
func (g *Game) Tick() {
	if g.bellTTL > 0 {
		g.bellTTL--
	}
	live := g.effects[:0]
	for _, e := range g.effects {
		e.TTL--
		if e.TTL > 0 {
			live = append(live, e)
		}
	}
	g.effects = live
}

// effectAt 은 (r,c) 에 걸린 활성 이펙트를 돌려준다(없으면 ok=false).
func (g *Game) effectAt(r, c int) (Effect, bool) {
	for _, e := range g.effects {
		if e.Row == r && e.Col == c {
			return e, true
		}
	}
	return Effect{}, false
}

func NewGame() *Game {
	g := &Game{}
	g.store = newProgressStore()
	g.progress = g.store.Load()
	if len(levels) > 0 {
		first := g.progress[levels[0].ID]
		first.Unlocked = true
		g.progress[levels[0].ID] = first
	}
	g.loadLevel(0)
	return g
}

// loadLevel 은 정규 커리큘럼(levels[idx])을 로드하고 statePlaying 으로 돌아간다.
func (g *Game) loadLevel(idx int) {
	g.levelIdx = idx
	g.state = statePlaying
	g.loadLevelData(levels[idx])
}

// loadLevelData 는 정규 레벨과 :drill 문제가 공유하는 버퍼 초기화 로직.
// state 는 건드리지 않는다 — 호출자가 statePlaying/stateDrill 등을 정한다.
func (g *Game) loadLevelData(lv Level) {
	g.lv = lv
	g.keyPos = map[[2]int]bool{}
	g.keysNeed = 0
	g.strokes = 0
	g.effects = nil
	g.bellTTL = 0
	g.exMode = false
	g.exBuf = nil

	if g.lv.Kind == "navigate" {
		lines := make([]string, len(g.lv.Map))
		sr, sc := 0, 0
		for r, row := range g.lv.Map {
			b := []rune(row)
			for c, ch := range b {
				switch ch {
				case '@':
					sr, sc = r, c
					b[c] = '.'
				case 'K':
					g.keyPos[[2]int{r, c}] = true
					g.keysNeed++
				}
			}
			lines[r] = string(b)
		}
		g.ed = NewEditor(lines)
		g.ed.row, g.ed.col = sr, sc
		g.ed.dcol = sc // j/k 의 목표 열도 시작 위치에 맞춘다(안 하면 dcol=0 이 남아
		// @ 가 col>0 인 레벨에서 첫 입력이 j/k 일 때 col0 으로 튐 — drill 생성기가
		// 열어 놓은 무작위 시작열에서 실제로 발생을 확인)
	} else {
		g.ed = NewEditor(append([]string(nil), g.lv.Map...))
	}
	g.syncDOM()
}

func (g *Game) pestsLeft() int {
	n := 0
	for _, l := range g.ed.Lines() {
		n += strings.Count(l, "*")
	}
	return n
}

// navigateAllows 는 navigate 레벨에서 편집 명령을 막고 이동+x 만 허용한다.
func navigateAllows(e *Editor, k Key) bool {
	if e.await != "" || e.op != 0 || e.pendObj != 0 || e.mode != ModeNormal || e.searching {
		return true // 명령 진행 중(찾기 대상/검색 쿼리 입력 등)
	}
	if k.R == 0 {
		return true // esc 등 특수키
	}
	if k.R >= '1' && k.R <= '9' {
		return true
	}
	switch k.R {
	case 'h', 'j', 'k', 'l', 'w', 'b', 'e', 'W', 'B', 'E',
		'0', '^', '$', 'f', 'F', 't', 'T', ';', ',', 'g', 'G', 'x',
		'/', '?', 'n', 'N':
		return true
	}
	return false
}

func (g *Game) feed(k Key) {
	g.strokes++
	if g.exMode {
		g.feedEx(k)
		return
	}
	if k.R == ':' && g.ed.mode == ModeNormal && g.ed.isCmdStart() && !g.ed.searching {
		g.exMode = true
		g.exBuf = nil
		return
	}
	wasBug := g.lv.Kind == "navigate" && k.R == 'x' && g.cellAt(g.ed.row, g.ed.col) == '*'
	if g.lv.Kind == "navigate" && !navigateAllows(g.ed, k) {
		g.fireEvent("blocked", g.ed.row, g.ed.col)
		return
	}
	g.ed.Feed(k)
	if wasBug {
		g.fireEvent("bug", g.ed.row, g.ed.col)
	}
}

// feedEx 는 ":" 명령줄 입력을 처리하는 Game 레벨 pseudo-mode.
// 검색(1.4)과 같은 골격(bool 플래그 + 버퍼 + esc/cr/bs)을 재사용하되,
// :q/:restart 처럼 Editor 밖의 Game 상태를 조작해야 해서 Editor가 아닌
// Game에 둔다.
func (g *Game) feedEx(k Key) {
	switch k.S {
	case "esc":
		g.exMode = false
		g.exBuf = nil
		return
	case "cr":
		cmd := string(g.exBuf)
		g.exMode = false
		g.exBuf = nil
		g.runExCommand(cmd)
		return
	case "bs":
		if len(g.exBuf) > 0 {
			g.exBuf = g.exBuf[:len(g.exBuf)-1]
		}
		return
	}
	if k.R != 0 {
		g.exBuf = append(g.exBuf, k.R)
	}
}

// restartCurrent 는 "지금 하던 것을 strokes=0 으로 다시"에 해당하는 단일
// 진입점이다 — :restart/:e! ex-command 와 RESET 버튼(웹 web_js.go 의
// vimquestReset)이 반드시 이 함수 하나를 거쳐야 한다. 예전에 이 로직을
// runExCommand 안에만 넣었더니, web_js.go 의 vimquestReset 이 별도로
// g.loadLevel(g.levelIdx) 를 직접 불러 :drill 인식 분기가 빠진 채로
// 똑같은 버그(드릴 중 리셋하면 커리큘럼으로 튕겨나감)를 반복한 적이 있다.
func (g *Game) restartCurrent() {
	if g.state == stateDrill {
		g.loadLevelData(g.lv) // 같은 드릴 문제를 strokes=0 으로 재시작(다음 문제로 넘기지 않는다)
	} else {
		g.loadLevel(g.levelIdx)
	}
}

// runExCommand 는 확정된 ex-command 문자열을 해석해 실행한다.
// 인식하지 못한 명령은 조용히 무시한다(터미널처럼 무반응 — 에러 팝업 없음).
func (g *Game) runExCommand(cmd string) {
	switch {
	case cmd == "q" || cmd == "levels":
		g.enterLevelSelect()
	case cmd == "restart" || cmd == "e!":
		g.restartCurrent()
	case cmd == "help":
		domShow("intro")
	case cmd == "drill":
		g.enterDrill()
	default:
		if n, err := strconv.Atoi(cmd); err == nil && n > 0 {
			g.ed.gotoLine(n)
		}
	}
}

// currentPar 는 현재 레벨의 검증된 Solution 기준 타수(par)를 반환한다.
func (g *Game) currentPar() int {
	return len(parseKeys(g.lv.Solution))
}

// stars 는 par 대비 타수로 별점(1~3)을 계산한다.
func stars(strokes, par int) int {
	if par <= 0 {
		return 1
	}
	switch {
	case strokes <= par:
		return 3
	case float64(strokes) <= float64(par)*1.5:
		return 2
	default:
		return 1
	}
}

// recordClear 는 현재 레벨의 클리어 기록을 진행 상황에 반영하고 저장한다.
// 갱신 전 best(BestStrokes)를 반환한다(0 이면 최초 클리어).
func (g *Game) recordClear() int {
	prog := g.progress[g.lv.ID]
	prevBest := prog.BestStrokes
	if prevBest == 0 || g.strokes < prevBest {
		prog.BestStrokes = g.strokes
	}
	if s := stars(g.strokes, g.currentPar()); s > prog.Stars {
		prog.Stars = s
	}
	prog.Unlocked = true
	g.progress[g.lv.ID] = prog

	if g.levelIdx+1 < len(levels) {
		next := levels[g.levelIdx+1]
		np := g.progress[next.ID]
		np.Unlocked = true
		g.progress[next.ID] = np
	}
	g.store.Save(g.progress)
	return prevBest
}

// Input 은 프론트엔드(Ebiten/JS 공통)가 호출하는 단일 입력 진입점.
// 상태별로 키를 해석한다 — strokes 카운트/checkWin/syncDOM 은 statePlaying 의
// feed() 경로 안에서만 일어난다. 프레임 구동(이펙트 TTL 감소)은 Tick() 이 별도로
// 맡으므로 여기서는 호출하지 않는다.
func (g *Game) Input(k Key) {
	switch g.state {
	case stateLevelClear:
		if k.S == "cr" {
			g.loadLevel(g.levelIdx + 1)
		} else if k.R == 'r' {
			g.loadLevel(g.levelIdx)
		}
	case stateLevelSelect:
		g.inputLevelSelect(k)
	case stateAllClear:
		// 정적 화면 — 입력 무시(기존과 동일)
	default: // statePlaying
		g.feed(k)
		g.checkWin()
		g.syncDOM()
	}
}

// inputLevelSelect 는 레벨 선택 화면에서의 키 입력을 처리한다.
// h/l = 월드 이동, j/k = 레벨 이동, cr = 입장(잠금 시 무시), esc = 복귀.
func (g *Game) inputLevelSelect(k Key) {
	worlds := worldGroups()
	if len(worlds) == 0 {
		return
	}
	if g.selRow >= len(worlds) {
		g.selRow = len(worlds) - 1
	}
	if g.selCol >= len(worlds[g.selRow]) {
		g.selCol = len(worlds[g.selRow]) - 1
	}

	switch k.R {
	case 'h':
		if g.selRow > 0 {
			g.selRow--
			if g.selCol >= len(worlds[g.selRow]) {
				g.selCol = len(worlds[g.selRow]) - 1
			}
		}
	case 'l':
		if g.selRow < len(worlds)-1 {
			g.selRow++
			if g.selCol >= len(worlds[g.selRow]) {
				g.selCol = len(worlds[g.selRow]) - 1
			}
		}
	case 'j':
		if g.selCol < len(worlds[g.selRow])-1 {
			g.selCol++
		}
	case 'k':
		if g.selCol > 0 {
			g.selCol--
		}
	}
	if k.S == "cr" {
		idx := worlds[g.selRow][g.selCol]
		if g.progress[levels[idx].ID].Unlocked {
			g.loadLevel(idx)
		}
	}
	if k.S == "esc" {
		g.state = statePlaying
		g.syncDOM()
	}
}

// enterLevelSelect 는 레벨 선택 화면으로 전환하며 커서를 현재 레벨 위치로 맞춘다.
func (g *Game) enterLevelSelect() {
	g.state = stateLevelSelect
	g.selRow, g.selCol = 0, 0
	worlds := worldGroups()
	for wi, group := range worlds {
		for li, idx := range group {
			if idx == g.levelIdx {
				g.selRow, g.selCol = wi, li
			}
		}
	}
	g.syncDOM()
}

// worldGroupsCache 는 worldGroups() 의 계산 결과를 담아둔다. levels 는 런타임에
// 절대 바뀌지 않는 정적 슬라이스라 한 번만 계산하면 된다 — 캐싱이 없으면
// drawAllClear 처럼 매 프레임(최대 60Hz) 불리는 경로에서 매번 재스캔+재할당된다.
// 단일 고루틴(Ebiten Update/Draw, 또는 wasm/JS 단일 스레드 이벤트 루프)에서만
// 호출되므로 락 없이 캐싱해도 안전하다.
var worldGroupsCache [][]int

// worldGroups 는 levels 를 Level.ID 접두어(월드 번호) 기준으로 묶어
// [월드][월드 내 레벨] = levels 인덱스 형태로 반환한다.
func worldGroups() [][]int {
	if worldGroupsCache != nil {
		return worldGroupsCache
	}
	var groups [][]int
	var cur []int
	curWorld := ""
	for i, lv := range levels {
		world := lv.ID
		if idx := strings.IndexByte(lv.ID, '-'); idx >= 0 {
			world = lv.ID[:idx]
		}
		if world != curWorld {
			if len(cur) > 0 {
				groups = append(groups, cur)
			}
			cur = nil
			curWorld = world
		}
		cur = append(cur, i)
	}
	if len(cur) > 0 {
		groups = append(groups, cur)
	}
	worldGroupsCache = groups
	return groups
}

func (g *Game) checkWin() {
	if g.lv.Kind == "navigate" {
		pos := [2]int{g.ed.row, g.ed.col}
		if g.keyPos[pos] {
			delete(g.keyPos, pos) // 열쇠 위로 오면 획득
			g.fireEvent("key", pos[0], pos[1])
		}
		cell := g.cellAt(g.ed.row, g.ed.col)
		if len(g.keyPos) == 0 && g.pestsLeft() == 0 && cell == '$' {
			if g.state == stateDrill {
				g.advanceDrill()
			} else {
				g.advance()
			}
		}
		return
	}
	// edit: 목표와 완전 일치
	cur := g.ed.Lines()
	if len(cur) != len(g.lv.Target) {
		return
	}
	for i := range cur {
		if cur[i] != g.lv.Target[i] {
			return
		}
	}
	g.advance()
}

// advance 는 현재 레벨의 클리어 통계를 스냅샷하고 진행 상황을 저장한 뒤
// stateLevelClear(또는 마지막 레벨이면 stateAllClear)로 전환한다.
// 다음 레벨 로드는 Input() 이 stateLevelClear 에서 cr 을 받을 때 이뤄진다.
func (g *Game) advance() {
	g.clearStrokes = g.strokes
	g.clearPar = g.currentPar()
	g.clearStars = stars(g.strokes, g.clearPar)
	g.clearBest = g.recordClear()
	jsSfx("clear")

	if g.levelIdx+1 < len(levels) {
		g.state = stateLevelClear
	} else {
		g.state = stateAllClear
	}
	g.syncDOM()
}

const (
	drillRows = 5
	drillCols = 20
)

// enterDrill 은 :drill 모드로 전환하고 첫 문제를 생성한다. 진행은 세션
// 한정이라 store 에 저장하지 않는다.
func (g *Game) enterDrill() {
	if g.drillRng == nil {
		g.drillRng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	g.drillStreak = 0
	g.drillTotalKeys = 0
	g.drillTotalPar = 0
	g.state = stateDrill
	g.loadLevelData(generateDrill(g.drillRng))
}

// advanceDrill 은 :drill 문제를 클리어했을 때 통계를 누적하고 즉시 다음
// 문제를 생성한다 — 클리어 화면을 생략해 템포를 유지한다(반복 훈련이 목적).
// drillMaxRounds 는 한 :drill 세션에서 생성할 문제 수의 상한. 웹 빌드는
// 크기를 줄이려고 GC 를 꺼놨으므로(-gc=leaking, build.sh) 문제를 생성할
// 때마다 나오는 자잘한 쓰레기(격자·Editor·해 문자열)가 세션 내내 전혀
// 회수되지 않는다 — 정확히 "반복 연습"이라는 이 기능의 용도에서 무한정
// 늘어날 수 있다는 뜻이라, 라운드 수에 상한을 둬 최악의 경우 메모리 증가를
// 유한하게 묶는다. 이 값(대략 문제당 수 KB 기준 총 수 MB)은 실제 연습
// 세션에선 거의 도달하지 않을 만큼 넉넉하다.
const drillMaxRounds = 1000

func (g *Game) advanceDrill() {
	g.drillStreak++
	g.drillTotalKeys += g.strokes
	g.drillTotalPar += g.currentPar()
	jsSfx("clear")
	if g.drillStreak >= drillMaxRounds {
		g.enterLevelSelect()
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

func (g *Game) cellAt(r, c int) rune {
	ls := g.ed.lines
	if r < 0 || r >= len(ls) || c < 0 || c >= len(ls[r]) {
		return ' '
	}
	return ls[r][c]
}

// itoa 는 strconv.Itoa 의 얇은 별칭. game.go 전역에서 이미 이 이름으로 쓰이던
// 관용구를 유지하되(호출부 수십 곳을 다 고치지 않기 위함), stdlib 와 별개로
// 손으로 재구현해 유지하지는 않는다 — strconv 는 runExCommand 의 Atoi 때문에
// 이미 이 파일에 들어와 있어 추가 바이너리 비용이 없다.
func itoa(n int) string { return strconv.Itoa(n) }

// ───────────────────────── DOM(한국어 UI) ─────────────────────────

// cmdsHTML 은 명령어 목록을 우측 패널용 HTML 로 만든다.
func cmdsHTML(cmds []Cmd) string {
	var b strings.Builder
	for _, c := range cmds {
		b.WriteString(`<div class="cmd"><span class="k">`)
		b.WriteString(htmlEscape(c.K))
		b.WriteString(`</span><span class="d">`)
		b.WriteString(htmlEscape(c.D))
		b.WriteString(`</span></div>`)
	}
	return b.String()
}

func htmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;")
	return r.Replace(s)
}

func (g *Game) syncDOM() {
	switch g.state {
	case stateAllClear:
		domSet("level-title", "🎉 ALL CLEAR!")
		domSet("hint", "Congratulations! You cleared all "+itoa(len(levels))+" levels across W1-W"+itoa(len(worldGroups()))+". Now go practice in real Vim!")
		domSet("status", "")
		domSetHTML("solve-cmds", "")
		return
	case stateLevelClear:
		domSet("level-title", "LEVEL "+g.lv.ID+" CLEAR!")
		domSet("hint", "Press Enter for the next level, or r to retry this one.")
		domSet("status", "keys "+itoa(g.clearStrokes)+" / par "+itoa(g.clearPar))
		domSetHTML("solve-cmds", "")
		return
	case stateLevelSelect:
		domSet("level-title", "SELECT LEVEL")
		domSet("hint", "h/l move between worlds, j/k move within a world, Enter to play, Esc to go back.")
		domSet("status", "")
		domSetHTML("solve-cmds", "")
		return
	}

	domSet("level-title", g.lv.Title)
	domSet("hint", g.lv.Hint)
	domSetHTML("solve-cmds", cmdsHTML(g.lv.Cmds))
	parInfo := "   ·   keys " + itoa(g.strokes) + "/par " + itoa(g.currentPar())
	if g.state == stateDrill {
		parInfo += "   ·   streak " + itoa(g.drillStreak) + "   ·   total " + itoa(g.drillTotalKeys) + "/" + itoa(g.drillTotalPar)
	}
	if g.lv.Kind == "navigate" {
		s := "keys " + itoa(g.keysNeed-len(g.keyPos)) + "/" + itoa(g.keysNeed)
		if p := g.pestsLeft(); p > 0 {
			s += "   ·   " + itoa(p) + " bug(s) left"
		} else if len(g.keyPos) == 0 {
			s += "   ·   now head to $ (exit)!"
		}
		domSet("status", s+parInfo)
	} else {
		domSet("status", "Make CURRENT match TARGET — a line turns green when it matches!"+parInfo)
	}
}

// ───────────────────────── 스냅샷 (JS 렌더러 계약) ─────────────────────────

func toAnySlice(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

// snapshot 은 프론트엔드가 게임 상태를 그리는 데 필요한 전부를 담은
// 순수 데이터 스냅샷을 만든다. Ebiten(main.go 의 draw*)과 JS 렌더러
// (web/renderer.js) 양쪽이 이 계약만 보고 그린다 — 렌더러는 게임 규칙을
// 몰라도 된다(듀얼 프론트엔드 드리프트 방지).
//
// 3★ 미만이면 solution 을 빈 문자열로 내려보낸다 — 스포일러 방지 규칙은
// 엔진이 소유하고 클라이언트가 아니다.
func (g *Game) snapshot() map[string]any {
	base := map[string]any{
		"level":      g.levelIdx + 1,
		"levelCount": len(levels),
	}

	switch g.state {
	case stateAllClear:
		base["state"] = "allclear"
		base["worldCount"] = len(worldGroups())
		return base

	case stateLevelClear:
		base["state"] = "clear"
		base["id"] = g.lv.ID
		base["clearStrokes"] = g.clearStrokes
		base["clearPar"] = g.clearPar
		base["clearStars"] = g.clearStars
		base["clearBest"] = g.clearBest
		solution := ""
		if g.clearStars == 3 {
			solution = g.lv.Solution
		}
		base["solution"] = solution
		return base

	case stateLevelSelect:
		base["state"] = "select"
		worlds := worldGroups()
		wOut := make([]any, len(worlds))
		for wi, group := range worlds {
			lvOut := make([]any, len(group))
			for li, idx := range group {
				lv := levels[idx]
				prog := g.progress[lv.ID]
				lvOut[li] = map[string]any{
					"id":       lv.ID,
					"unlocked": prog.Unlocked,
					"stars":    prog.Stars,
				}
			}
			wOut[wi] = lvOut
		}
		base["worlds"] = wOut
		base["selRow"] = g.selRow
		base["selCol"] = g.selCol
		return base
	}

	// statePlaying / stateDrill
	base["state"] = "playing"
	if g.state == stateDrill {
		base["state"] = "drill"
		base["drill"] = map[string]any{
			"streak":    g.drillStreak,
			"totalKeys": g.drillTotalKeys,
			"totalPar":  g.drillTotalPar,
		}
	}
	base["kind"] = g.lv.Kind
	base["id"] = g.lv.ID
	base["title"] = g.lv.Title
	base["lines"] = toAnySlice(g.ed.Lines())
	base["row"] = g.ed.row
	base["col"] = g.ed.col
	base["mode"] = g.ed.ModeName()
	base["pending"] = g.ed.pendingStr
	base["last"] = g.ed.lastKey
	base["strokes"] = g.strokes
	base["par"] = g.currentPar()
	base["exMode"] = g.exMode
	base["exBuf"] = string(g.exBuf)
	base["bell"] = g.bellTTL > 0

	if g.lv.Kind == "navigate" {
		base["keys"] = g.keysNeed - len(g.keyPos)
		base["keysNeed"] = g.keysNeed
		base["bugs"] = g.pestsLeft()
		kp := make([]any, 0, len(g.keyPos))
		for pos := range g.keyPos {
			kp = append(kp, map[string]any{"row": pos[0], "col": pos[1]})
		}
		base["keyPos"] = kp
	} else {
		base["target"] = toAnySlice(g.lv.Target)
	}

	r1, c1, r2, c2, line, ok := g.ed.VisualSpan()
	base["visual"] = map[string]any{"r1": r1, "c1": c1, "r2": r2, "c2": c2, "line": line, "ok": ok}

	effs := make([]any, len(g.effects))
	for i, e := range g.effects {
		glyph := ""
		if e.Glyph != 0 {
			glyph = string(e.Glyph)
		}
		effs[i] = map[string]any{"row": e.Row, "col": e.Col, "glyph": glyph, "invert": e.Invert}
	}
	base["effects"] = effs
	base["effectsAlive"] = len(g.effects) > 0 || g.bellTTL > 0

	return base
}
