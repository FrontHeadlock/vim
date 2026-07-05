// Package game 은 VimQuest 의 규칙 전부 — 레벨 진행, 승리 판정, 별점,
// : ex-command, :drill 연습 모드 — 를 소유하는 상태 머신이다.
// 렌더링과 입력 장치를 전혀 모른다: 프론트엔드(cmd/desktop 의 Ebiten,
// cmd/web 의 TinyGo wasm)는 engine.Key 를 Input() 에 넣고, view.go 의 읽기
// 전용 접근자 또는 snapshot.go 의 Snapshot() 계약만 보고 그린다.
//
// 의존 방향은 game → engine(편집 엔진) · store(진행 저장) · platform(DOM/SFX)
// 한 방향뿐이다. ebiten/syscall-js 를 여기서 import 하면 안 된다 — 그 경계가
// 곧 TinyGo 웹 빌드(빌드 크기 예산 100KB)의 전제다.
package game

import (
	"math/rand"
	"strconv"
	"strings"

	"vimquest/internal/engine"
	"vimquest/internal/platform"
	"vimquest/internal/store"
)

// State 는 게임 전체의 화면 상태를 나타낸다.
type State int

const (
	StatePlaying State = iota
	StateLevelClear
	StateLevelSelect
	StateAllClear
	StateDrill
)

// Game 은 한 판의 전체 상태. 상태를 바꾸는 공개 경로는 Input()/Tick() 과
// 프론트엔드 버튼이 쓰는 LoadLevel/RestartCurrent/EnterLevelSelect 뿐이다.
type Game struct {
	levelIdx int
	lv       Level
	ed       *engine.Editor

	keyPos   map[[2]int]bool
	keysNeed int

	state State
	strokes int

	clear ClearStats

	selRow, selCol int

	store    store.Store
	progress map[string]store.LevelProgress

	effects []Effect
	bellTTL int

	exMode bool
	exBuf  []rune

	drillRng       *rand.Rand
	drillStreak    int
	drillTotalKeys int
	drillTotalPar  int
}

// New 는 저장된 진행을 복원하고 첫 레벨을 로드한 게임을 만든다.
func New() *Game {
	g := &Game{}
	g.store = store.New()
	g.progress = g.store.Load()
	if len(levels) > 0 {
		first := g.progress[levels[0].ID]
		first.Unlocked = true
		g.progress[levels[0].ID] = first
	}
	g.LoadLevel(0)
	return g
}

// LoadLevel 은 정규 커리큘럼(levels[idx])을 로드하고 StatePlaying 으로 돌아간다.
func (g *Game) LoadLevel(idx int) {
	g.levelIdx = idx
	g.state = StatePlaying
	g.loadLevelData(levels[idx])
}

// loadLevelData 는 정규 레벨과 :drill 문제가 공유하는 버퍼 초기화 로직.
// state 는 건드리지 않는다 — 호출자가 StatePlaying/StateDrill 등을 정한다.
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
		g.ed = engine.NewEditor(lines)
		// SetCursor 는 j/k 의 목표 열(dcol)까지 함께 맞춘다 — row/col 만 옮기면
		// @ 가 col>0 인 레벨에서 첫 입력이 j/k 일 때 col0 으로 튄다(drill 생성기가
		// 열어 놓은 무작위 시작열에서 실제로 발생을 확인했던 결함).
		g.ed.SetCursor(sr, sc)
	} else {
		g.ed = engine.NewEditor(append([]string(nil), g.lv.Map...))
	}
	g.syncDOM()
}

// PestsLeft 는 버퍼에 남은 버그(*) 수. 승리 판정과 HUD 렌더 양쪽에서 쓴다.
func (g *Game) PestsLeft() int {
	n := 0
	for _, l := range g.ed.Lines() {
		n += strings.Count(l, "*")
	}
	return n
}

// navigateAllows 는 navigate 레벨에서 편집 명령을 막고 이동+x 만 허용한다.
func navigateAllows(e *engine.Editor, k engine.Key) bool {
	if e.MidCommand() || e.Mode() != engine.ModeNormal || e.Searching() {
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

func (g *Game) feed(k engine.Key) {
	g.strokes++
	if g.exMode {
		g.feedEx(k)
		return
	}
	if k.R == ':' && g.ed.Mode() == engine.ModeNormal && g.ed.IsCmdStart() && !g.ed.Searching() {
		g.exMode = true
		g.exBuf = nil
		return
	}
	wasBug := g.lv.Kind == "navigate" && k.R == 'x' && g.cellAt(g.ed.Row(), g.ed.Col()) == '*'
	if g.lv.Kind == "navigate" && !navigateAllows(g.ed, k) {
		g.fireEvent("blocked", g.ed.Row(), g.ed.Col())
		return
	}
	g.ed.Feed(k)
	if wasBug {
		g.fireEvent("bug", g.ed.Row(), g.ed.Col())
	}
}

// feedEx 는 ":" 명령줄 입력을 처리하는 Game 레벨 pseudo-mode.
// 검색과 같은 골격(bool 플래그 + 버퍼 + esc/cr/bs)을 재사용하되,
// :q/:restart 처럼 Editor 밖의 Game 상태를 조작해야 해서 엔진이 아닌
// 게임에 둔다.
func (g *Game) feedEx(k engine.Key) {
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

// RestartCurrent 는 "지금 하던 것을 strokes=0 으로 다시"에 해당하는 단일
// 진입점이다 — :restart/:e! ex-command 와 RESET 버튼(cmd/web 의
// vimquestReset)이 반드시 이 함수 하나를 거쳐야 한다. 예전에 이 로직을
// runExCommand 안에만 넣었더니, 웹 진입점이 별도로 LoadLevel 을 직접 불러
// :drill 인식 분기가 빠진 채로 똑같은 버그(드릴 중 리셋하면 커리큘럼으로
// 튕겨나감)를 반복한 적이 있다.
func (g *Game) RestartCurrent() {
	if g.state == StateDrill {
		g.loadLevelData(g.lv) // 같은 드릴 문제를 strokes=0 으로 재시작(다음 문제로 넘기지 않는다)
	} else {
		g.LoadLevel(g.levelIdx)
	}
}

// runExCommand 는 확정된 ex-command 문자열을 해석해 실행한다.
// 인식하지 못한 명령은 조용히 무시한다(터미널처럼 무반응 — 에러 팝업 없음).
func (g *Game) runExCommand(cmd string) {
	switch {
	case cmd == "q" || cmd == "levels":
		g.EnterLevelSelect()
	case cmd == "restart" || cmd == "e!":
		g.RestartCurrent()
	case cmd == "help":
		platform.ShowOverlay("intro")
	case cmd == "drill":
		g.enterDrill()
	default:
		if n, err := strconv.Atoi(cmd); err == nil && n > 0 {
			g.ed.GotoLine(n)
		}
	}
}

// Par 는 현재 레벨의 검증된 Solution 기준 타수(par)를 반환한다.
func (g *Game) Par() int {
	return len(engine.ParseKeys(g.lv.Solution))
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
	if s := stars(g.strokes, g.Par()); s > prog.Stars {
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
// 상태별로 키를 해석한다 — strokes 카운트/checkWin/syncDOM 은 StatePlaying 의
// feed() 경로 안에서만 일어난다. 프레임 구동(이펙트 TTL 감소)은 Tick() 이
// 별도로 맡으므로 여기서는 호출하지 않는다.
func (g *Game) Input(k engine.Key) {
	switch g.state {
	case StateLevelClear:
		if k.S == "cr" {
			g.LoadLevel(g.levelIdx + 1)
		} else if k.R == 'r' {
			g.LoadLevel(g.levelIdx)
		}
	case StateLevelSelect:
		g.inputLevelSelect(k)
	case StateAllClear:
		// 정적 화면 — 입력 무시
	default: // StatePlaying / StateDrill
		g.feed(k)
		g.checkWin()
		g.syncDOM()
	}
}

// inputLevelSelect 는 레벨 선택 화면에서의 키 입력을 처리한다.
// h/l = 월드 이동, j/k = 레벨 이동, cr = 입장(잠금 시 무시), esc = 복귀.
func (g *Game) inputLevelSelect(k engine.Key) {
	worlds := WorldGroups()
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
			g.LoadLevel(idx)
		}
	}
	if k.S == "esc" {
		g.state = StatePlaying
		g.syncDOM()
	}
}

// EnterLevelSelect 는 레벨 선택 화면으로 전환하며 커서를 현재 레벨 위치로 맞춘다.
func (g *Game) EnterLevelSelect() {
	g.state = StateLevelSelect
	g.selRow, g.selCol = 0, 0
	for wi, group := range WorldGroups() {
		for li, idx := range group {
			if idx == g.levelIdx {
				g.selRow, g.selCol = wi, li
			}
		}
	}
	g.syncDOM()
}

func (g *Game) checkWin() {
	if g.lv.Kind == "navigate" {
		pos := [2]int{g.ed.Row(), g.ed.Col()}
		if g.keyPos[pos] {
			delete(g.keyPos, pos) // 열쇠 위로 오면 획득
			g.fireEvent("key", pos[0], pos[1])
		}
		cell := g.cellAt(g.ed.Row(), g.ed.Col())
		if len(g.keyPos) == 0 && g.PestsLeft() == 0 && cell == '$' {
			if g.state == StateDrill {
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
// StateLevelClear(또는 마지막 레벨이면 StateAllClear)로 전환한다.
// 다음 레벨 로드는 Input() 이 StateLevelClear 에서 cr 을 받을 때 이뤄진다.
func (g *Game) advance() {
	par := g.Par()
	g.clear = ClearStats{
		Strokes: g.strokes,
		Par:     par,
		Stars:   stars(g.strokes, par),
		Best:    g.recordClear(),
	}
	platform.Sfx("clear")

	if g.levelIdx+1 < len(levels) {
		g.state = StateLevelClear
	} else {
		g.state = StateAllClear
	}
	g.syncDOM()
}

// cellAt 은 (r,c) 의 버퍼 문자를 돌려준다. 범위 밖이면 ' '.
func (g *Game) cellAt(r, c int) rune {
	ch, ok := g.ed.Cell(r, c)
	if !ok {
		return ' '
	}
	return ch
}

// ParseKey 는 웹 glue.js 에서 보낸 토큰을 engine.Key 로 변환한다.
func ParseKey(tok string) engine.Key {
	switch tok {
	case "<cr>":
		return engine.SpecialKey("cr")
	case "<esc>":
		return engine.SpecialKey("esc")
	case "<bs>":
		return engine.SpecialKey("bs")
	case "<c-r>":
		return engine.SpecialKey("c-r")
	default:
		if len(tok) == 1 {
			return engine.RuneKey(rune(tok[0]))
		}
		return engine.RuneKey(0)
	}
}
